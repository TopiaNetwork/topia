package transactionpool

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
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

	savePoolConfig(path string, conf txpooli.TransactionPoolConfig, marshaler codec.Marshaler) error
	loadAndSetPoolConfig(path string, marshaler codec.Marshaler, setConf func(config txpooli.TransactionPoolConfig)) error

	saveLocalTxs(path string, marshaler codec.Marshaler, wrappedTxs []*wrappedTx) error
	saveAllLocalTxs(path string, marshaler codec.Marshaler, getLocals func() *tpcmm.ShrinkableMap) error
	delLocalTxs(path string, marshaler codec.Marshaler, ids []txbasic.TxID) error
	loadAndAddLocalTxs(path string, marshaler codec.Marshaler, addLocalTx func(txID txbasic.TxID, txWrap *wrappedTx)) error

	Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error
	UnSubscribe(topic string) error
}

func newTransactionPoolServant(stateQueryService service.StateQueryService, blockService service.BlockService,
	network tpnet.Network) TransactionPoolServant {
	txpoolservant := &transactionPoolServant{
		StateQueryService: stateQueryService,
		BlockService:      blockService,
		Network:           network,
	}
	return txpoolservant
}

type transactionPoolServant struct {
	service.StateQueryService
	service.BlockService
	tpnet.Network
}

func (servant *transactionPoolServant) CurrentHeight() (uint64, error) {
	curBlock, err := servant.GetLatestBlock()
	if err != nil {
		return 0, err
	}
	curHeight := curBlock.Head.Height
	return curHeight, nil
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

func (servant *transactionPoolServant) savePoolConfig(path string, conf txpooli.TransactionPoolConfig, marshaler codec.Marshaler) error {

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

func (servant *transactionPoolServant) loadAndSetPoolConfig(path string, marshaler codec.Marshaler, setConf func(config txpooli.TransactionPoolConfig)) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	conf := &txpooli.TransactionPoolConfig{}
	err = marshaler.Unmarshal(data, &conf)
	if err != nil {
		return err
	}
	setConf(*conf)
	return nil
}

func (servant *transactionPoolServant) delLocalTxs(path string, marshaler codec.Marshaler, ids []txbasic.TxID) error {
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

func (servant *transactionPoolServant) saveAllLocalTxs(path string, marshaler codec.Marshaler, getLocals func() *tpcmm.ShrinkableMap) error {

	TxsLocal := getLocals()
	if len(TxsLocal.AllKeys()) == 0 {
		return ErrTxNotExist
	}
	var allTxInfo []*wrappedTxData
	getAllTxInfo := func(k interface{}, v interface{}) {
		TxInfo := v.(*wrappedTx)
		txByte, _ := marshaler.Marshal(TxInfo.Tx)
		txData := &wrappedTxData{
			TxID:          TxInfo.TxID,
			IsLocal:       TxInfo.IsLocal,
			LastTime:      TxInfo.LastTime,
			LastHeight:    TxInfo.LastHeight,
			TxState:       TxInfo.TxState,
			IsRepublished: TxInfo.IsRepublished,
			Tx:            txByte,
		}
		allTxInfo = append(allTxInfo, txData)
	}

	TxsLocal.IterateCallback(getAllTxInfo)

	pathIndex := path + "index.json"
	pathData := path + "data.json"
	fileIndex, err := os.OpenFile(pathIndex, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	fileData, err := os.OpenFile(pathData, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)

	if err != nil {
		return err
	}
	txIndex := newTxStorageIndex()
	for _, v := range allTxInfo {
		ByteV, _ := marshaler.Marshal(v)
		n, err := fileData.Write(ByteV)
		if err != nil {
			fmt.Println("writeData error", err)
		}
		txIndex.set(string(v.TxID), [2]int{txIndex.LastPos, n})
	}
	txInByte, _ := marshaler.Marshal(txIndex)
	fileIndex.Write(txInByte)

	fileIndex.Close()
	fileData.Close()
	return nil
}

func (servant *transactionPoolServant) loadAndAddLocalTxs(path string, marshaler codec.Marshaler, addLocalTx func(txID txbasic.TxID, txWrap *wrappedTx)) error {
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

func (msgSub *txMsgSubProcessor) GetLoger() tplog.Logger {
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
	if int64(tx.Size()) > txpooli.DefaultTransactionPoolConfig.MaxSizeOfEachTx {
		msgSub.log.Errorf("transaction size is up to the TxMaxSize")
		return message.ValidationReject
	}
	if tx.Head.Nonce > MaxUint64 {
		msgSub.log.Errorf("transaction nonce is up to the MaxUint64")
		return message.ValidationReject
	}
	if txpooli.DefaultTransactionPoolConfig.GasPriceLimit > GasLimit(tx) {
		msgSub.log.Errorf("transaction gasLimit is lower to GasPriceLimit")
		return message.ValidationReject
	}
	if msgSub.txPool.PendingAccountTxCnt(tpcrtypes.Address(tx.Head.FromAddr)) > msgSub.txPool.config.MaxCntOfEachAccount {
		return message.ValidationReject
	}
	if !isLocal {
		ac := transaction.CreatTransactionAction(tx)
		verifyResult := ac.Verify(ctx, msgSub.GetLoger(), msgSub.GetNodeID(), nil)
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
	txId, _ := tx.TxID()
	err := tx.Unmarshal(subMsgTxMessage.Data)
	if err != nil {
		msgSub.log.Error("txmessage data error")
		return err
	}

	msgSub.txPool.AddTx(tx, false)
	if _, ok := msgSub.txPool.Get(txId); ok {
		msgSub.txPool.txCache.Add(txId, txpooli.StateTxAdded)
	}
	return nil
}
