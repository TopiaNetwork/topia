package transactionpool

import (
	"context"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"io/ioutil"
	"os"
	"sync"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/service"
	"github.com/TopiaNetwork/topia/transaction"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TransactionPoolServant interface {
	CurrentHeight() (uint64, error)
	GetNonce(tpcrtypes.Address) (uint64, error)
	GetLatestBlock() (*tpchaintypes.Block, error)
	GetBlockByHash(hash tpchaintypes.BlockHash) (*tpchaintypes.Block, error)
	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)
	PublishTx(ctx context.Context, tx *txbasic.Transaction) error

	savePoolConfig(path string, conf *tpconfig.TransactionPoolConfig, marshaler codec.Marshaler) error
	loadAndSetPoolConfig(path string, marshaler codec.Marshaler, setConf func(config *tpconfig.TransactionPoolConfig)) error

	saveLocalTxs(path string, marshaler codec.Marshaler, wrappedTxs []*wrappedTx) error
	saveAllLocalTxs(isLocalsNotNil func() bool, saveAllLocals func() error) error
	delLocalTxs(path string, marshaler codec.Marshaler, ids []txbasic.TxID) error
	loadAndAddLocalTxs(path string, marshaler codec.Marshaler, addLocalTx func(txID txbasic.TxID, txWrap *wrappedTx)) error

	Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error
	UnSubscribe(topic string) error

	getStateQuery() service.StateQueryService
}

func newTransactionPoolServant(stateQueryService service.StateQueryService, blockService service.BlockService,
	network tpnet.Network) TransactionPoolServant {
	txpoolservant := &transactionPoolServant{
		state:        stateQueryService,
		BlockService: blockService,
		Network:      network,
		mu:           sync.RWMutex{},
	}
	return txpoolservant
}

type transactionPoolServant struct {
	state service.StateQueryService
	service.BlockService
	tpnet.Network
	mu sync.RWMutex
}

func (servant *transactionPoolServant) getStateQuery() service.StateQueryService {
	return servant.state
}
func (servant *transactionPoolServant) CurrentHeight() (uint64, error) {
	curBlock, err := servant.GetLatestBlock()
	if err != nil {
		return 0, err
	}
	curHeight := curBlock.Head.Height
	return curHeight, nil
}

func (servant *transactionPoolServant) GetNonce(addr tpcrtypes.Address) (uint64, error) {
	return servant.state.GetNonce(addr)
}

func (servant *transactionPoolServant) GetLatestBlock() (*tpchaintypes.Block, error) {
	return servant.state.GetLatestBlock()

}
func (servant *transactionPoolServant) PublishTx(ctx context.Context, tx *txbasic.Transaction) error {
	if tx == nil {
		return ErrTxIsNil
	}
	msg := &TxMessage{}
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	data, err := marshaler.Marshal(tx)
	if err != nil {
		return err
	}
	msg.Data = data
	sendData, err := msg.Marshal()
	if err != nil {
		return err
	}
	var toModuleName []string
	toModuleName = append(toModuleName, txpooli.MOD_NAME)
	servant.Network.Publish(ctx, toModuleName, protocol.SyncProtocolID_Msg, sendData)

	return nil
}

func (servant *transactionPoolServant) Subscribe(ctx context.Context, topic string, localIgnore bool,
	validators ...message.PubSubMessageValidator) error {
	return servant.Network.Subscribe(ctx, topic, localIgnore, validators...)
}
func (servant *transactionPoolServant) UnSubscribe(topic string) error {
	return servant.Network.UnSubscribe(topic)
}

func (servant *transactionPoolServant) savePoolConfig(path string, conf *tpconfig.TransactionPoolConfig, marshaler codec.Marshaler) error {

	txsData, err := marshaler.Marshal(conf)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, txsData, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (servant *transactionPoolServant) loadAndSetPoolConfig(path string, marshaler codec.Marshaler, setConf func(config *tpconfig.TransactionPoolConfig)) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	conf := &tpconfig.TransactionPoolConfig{}
	err = marshaler.Unmarshal(data, &conf)
	if err != nil {
		return err
	}
	setConf(conf)
	return nil
}

func (servant *transactionPoolServant) delLocalTxs(path string, marshaler codec.Marshaler, ids []txbasic.TxID) error {
	servant.mu.Lock()
	defer servant.mu.Unlock()

	pathIndex := path + "index.json"
	txIndexFile, err := os.Open(pathIndex)
	if err != nil {
		return err
	}
	indexData, err := ioutil.ReadAll(txIndexFile)
	if err != nil {
		return err
	}
	txIndexFile.Close()
	var txIndex *txStorageIndex
	_ = marshaler.Unmarshal(indexData, &txIndex)
	for _, id := range ids {
		txIndex.del(string(id))
	}

	txIndexData, _ := marshaler.Marshal(txIndex)
	fileIndex, err := os.OpenFile(pathIndex, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	fileIndex.Write(txIndexData)
	fileIndex.Close()
	return nil
}

func (servant *transactionPoolServant) saveLocalTxs(path string, marshaler codec.Marshaler, wrappedTxs []*wrappedTx) error {
	servant.mu.Lock()
	defer servant.mu.Unlock()

	pathIndex := path + "index.json"
	pathData := path + "data.json"
	txIndexFile, _ := os.Open(pathIndex)
	indexData, _ := ioutil.ReadAll(txIndexFile)
	txIndexFile.Close()
	var txIndex *txStorageIndex
	_ = marshaler.Unmarshal(indexData, &txIndex)
	if txIndex == nil {
		txIndex = newTxStorageIndex()
	}
	fileData, err := os.OpenFile(pathData, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	for _, wrappedTx := range wrappedTxs {
		txByte, _ := marshaler.Marshal(wrappedTx.Tx)
		txData := &wrappedTxData{
			TxID:          wrappedTx.TxID,
			IsLocal:       wrappedTx.IsLocal,
			LastTime:      wrappedTx.LastTime,
			LastHeight:    wrappedTx.LastHeight,
			TxState:       wrappedTx.TxState,
			IsRepublished: wrappedTx.IsRepublished,
			Tx:            txByte,
		}
		byteV, _ := marshaler.Marshal(txData)
		n, err := fileData.Write(byteV)
		if err != nil {
			return err
		}
		txIndex.set(string(wrappedTx.TxID), [2]int{txIndex.LastPos, n})
	}
	listData, _ := marshaler.Marshal(txIndex)
	fileIndex, err := os.OpenFile(pathIndex, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	fileIndex.Write(listData)
	fileIndex.Close()
	fileData.Close()
	return nil
}

func (servant *transactionPoolServant) saveAllLocalTxs(isLocalsNil func() bool, saveAllLocals func() error) error {
	servant.mu.Lock()
	defer servant.mu.Unlock()

	if isLocalsNil() {
		return ErrTxNotExist
	}

	err := saveAllLocals()
	return err
}

func (servant *transactionPoolServant) loadAndAddLocalTxs(path string, marshaler codec.Marshaler, addLocalTx func(txID txbasic.TxID, txWrap *wrappedTx)) error {
	servant.mu.Lock()
	defer servant.mu.Unlock()

	if path == "" {
		return ErrTxStoragePath
	}
	pathIndex := path + "index.json"
	pathData := path + "data.json"
	txIndexFile, err := os.Open(pathIndex)
	if err != nil {
		return err
	}
	indexData, err := ioutil.ReadAll(txIndexFile)
	if err != nil {
		return err
	}
	var txIndex *txStorageIndex
	_ = marshaler.Unmarshal(indexData, &txIndex)
	txIndexFile.Close()
	txDataFile, err := os.OpenFile(pathData, os.O_CREATE|os.O_RDONLY, 0755)
	defer txDataFile.Close()
	if err != nil {
		return err
	}
	for _, startLen := range txIndex.TxBeginAndLen {
		txDataFile.Seek(int64(startLen[0]), 0)
		buf := make([]byte, startLen[1])
		_, err := txDataFile.Read(buf)
		if err != nil {
			return err
		}
		var txData *wrappedTxData
		_ = marshaler.Unmarshal(buf[:], &txData)
		if txData != nil {
			var tx *txbasic.Transaction
			_ = marshaler.Unmarshal(txData.Tx, &tx)
			wrapTx := &wrappedTx{
				TxID:          txData.TxID,
				IsLocal:       txData.IsLocal,
				Category:      txbasic.TransactionCategory(tx.Head.Category),
				LastTime:      txData.LastTime,
				LastHeight:    txData.LastHeight,
				TxState:       txData.TxState,
				IsRepublished: txData.IsRepublished,
				FromAddr:      tpcrtypes.Address(tx.Head.FromAddr),
				Nonce:         tx.Head.Nonce,
				Tx:            tx,
			}
			addLocalTx(txData.TxID, wrapTx)
		}
	}
	return nil
}

type TxMsgSubProcessor interface {
	Validate(ctx context.Context, isLocal bool, data []byte) message.ValidationResult
	Process(ctx context.Context, subMsgTxMessage *TxMessage) error
}

var TxMsgSub TxMsgSubProcessor

type txMsgSubProcessor struct {
	txPool *transactionPool
	log    tplog.Logger
	nodeID string
}

func (msgSub *txMsgSubProcessor) GetLogger() tplog.Logger {
	return msgSub.log
}
func (msgSub *txMsgSubProcessor) GetNodeID() string {
	return msgSub.nodeID
}

func (msgSub *txMsgSubProcessor) Validate(ctx context.Context, isLocal bool, sendData []byte) message.ValidationResult {
	if msgSub.txPool.Size() > msgSub.txPool.config.TxPoolMaxSize {
		return message.ValidationReject
	}
	msg := &TxMessage{}
	msg.Unmarshal(sendData)
	var tx *txbasic.Transaction
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	err := marshaler.Unmarshal(msg.Data, &tx)
	if err != nil {
		return message.ValidationReject
	}
	if int64(tx.Size()) > tpconfig.DefaultTransactionPoolConfig().MaxSizeOfEachTx {
		msgSub.log.Errorf("transaction size is up to the TxMaxSize")
		return message.ValidationReject
	}
	if tx.Head.Nonce > MaxUint64 {
		msgSub.log.Errorf("transaction nonce is up to the MaxUint64")
		return message.ValidationReject
	}
	if tpconfig.DefaultTransactionPoolConfig().GasPriceLimit > GasLimit(tx) {
		msgSub.log.Errorf("transaction gasLimit is lower to GasPriceLimit")
		return message.ValidationReject
	}
	if msgSub.txPool.PendingAccountTxCnt(tpcrtypes.Address(tx.Head.FromAddr)) > msgSub.txPool.config.MaxCntOfEachAccount {
		return message.ValidationReject
	}
	if !isLocal {
		ac := transaction.CreatTransactionAction(tx)
		verifyResult := ac.Verify(ctx, msgSub.GetLogger(), msgSub.GetNodeID(), nil)
		switch verifyResult {
		case txbasic.VerifyResult_Accept:
			return message.ValidationAccept
		case txbasic.VerifyResult_Ignore:
			return message.ValidationIgnore
		case txbasic.VerifyResult_Reject:
			return message.ValidationReject
		}
	}
	return message.ValidationAccept
}
func (msgSub *txMsgSubProcessor) Process(ctx context.Context, subMsgTxMessage *TxMessage) error {

	var tx *txbasic.Transaction
	if subMsgTxMessage == nil {
		return ErrTxIsNil
	}
	err := msgSub.txPool.marshaler.Unmarshal(subMsgTxMessage.Data, &tx)
	if err != nil {
		msgSub.log.Error("txMessage data error")
		return err
	}
	msgSub.txPool.AddTx(tx, false)
	return nil
}
