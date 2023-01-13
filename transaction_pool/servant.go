package transactionpool

import (
	"context"
	"encoding/json"
	"sync"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplgms "github.com/TopiaNetwork/topia/ledger/meta"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/service"
	"github.com/TopiaNetwork/topia/state"
	"github.com/TopiaNetwork/topia/transaction"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpoolcore "github.com/TopiaNetwork/topia/transaction_pool/core"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

const (
	MetaData_TxPool_Tx     = "MD_TP_Tx"
	MetaData_TxPool_Config = "MD_TP_Config"
)

type TransactionPoolServant interface {
	CurrentHeight() (uint64, error)

	GetNonce(tpcrtypes.Address) (uint64, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetBlockByHash(hash tpchaintypes.BlockHash) (*tpchaintypes.Block, error)

	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)

	GetLedger() ledger.Ledger

	PublishTx(ctx context.Context, marshaler codec.Marshaler, topic string, domainID string, nodeID string, tx *txbasic.Transaction) error

	savePoolConfig(conf *tpconfig.TransactionPoolConfig, marshaler codec.Marshaler) error

	loadPoolConfig(marshaler codec.Marshaler) (*tpconfig.TransactionPoolConfig, error)

	removeTxFromStore(txId txbasic.TxID)

	saveTxIntoStore(marshaler codec.Marshaler, txId txbasic.TxID, height uint64, tx *txbasic.Transaction) error

	loadAllTxs(marshaler codec.Marshaler, addTx func(tx *txbasic.Transaction, height uint64) (txpoolcore.TxWrapper, error)) error

	stop()

	Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error

	UnSubscribe(topic string) error

	CreateTopic(topic string)

	getStateQuery() service.StateQueryService
}

func newTransactionPoolServant(
	stateQueryService service.StateQueryService,
	blockService service.BlockService,
	network tpnet.Network,
	l ledger.Ledger) TransactionPoolServant {

	metaStore, _ := l.CreateMetaStore()

	metaStore.AddNamedStateStore(MetaData_TxPool_Tx, 50)
	metaStore.AddNamedStateStore(MetaData_TxPool_Config, 50)

	txpoolservant := &transactionPoolServant{
		ledger:       l,
		state:        stateQueryService,
		BlockService: blockService,
		Network:      network,
		MetaStore:    metaStore,
	}
	return txpoolservant
}

type transactionPoolServant struct {
	ledger ledger.Ledger
	state  service.StateQueryService
	service.BlockService
	tpnet.Network
	tplgms.MetaStore
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

func (servant *transactionPoolServant) GetLedger() ledger.Ledger {
	return servant.ledger
}

func (servant *transactionPoolServant) PublishTx(ctx context.Context, marshaler codec.Marshaler, topic string, domainID string, nodeID string, tx *txbasic.Transaction) error {
	if tx == nil {
		return ErrTxIsNil
	}

	data, err := marshaler.Marshal(tx)
	if err != nil {
		return err
	}

	msgTx := &TxMessage{
		Data: data,
	}
	msgTxBytes, err := msgTx.Marshal()
	if err != nil {
		return err
	}

	msg := &TxPoolMessage{
		MsgType:    TxPoolMessage_Tx,
		FromDomain: []byte(domainID),
		FromNode:   []byte(nodeID),
		Data:       msgTxBytes,
	}
	msgBytes, err := msg.Marshal()
	if err != nil {
		return err
	}

	servant.Network.Publish(ctx, []string{txpooli.MOD_NAME}, topic, msgBytes)

	return nil
}

func (servant *transactionPoolServant) Subscribe(ctx context.Context, topic string, localIgnore bool,
	validators ...message.PubSubMessageValidator) error {
	return servant.Network.Subscribe(ctx, topic, localIgnore, validators...)
}
func (servant *transactionPoolServant) UnSubscribe(topic string) error {
	return servant.Network.UnSubscribe(topic)
}

func (servant *transactionPoolServant) savePoolConfig(conf *tpconfig.TransactionPoolConfig, marshaler codec.Marshaler) error {
	confDataBytes, err := marshaler.Marshal(conf)
	if err != nil {
		return err
	}

	return servant.MetaStore.Put(MetaData_TxPool_Config, []byte(MetaData_TxPool_Config), confDataBytes)
}

func (servant *transactionPoolServant) loadPoolConfig(marshaler codec.Marshaler) (*tpconfig.TransactionPoolConfig, error) {
	confDataBytes, err := servant.MetaStore.GetData(MetaData_TxPool_Config, []byte(MetaData_TxPool_Config))
	if err != nil {
		return nil, err
	}

	conf := &tpconfig.TransactionPoolConfig{}
	err = marshaler.Unmarshal(confDataBytes, &conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (servant *transactionPoolServant) removeTxFromStore(txId txbasic.TxID) {
	servant.MetaStore.Delete(MetaData_TxPool_Tx, []byte(txId))
}

func (servant *transactionPoolServant) saveTxIntoStore(marshaler codec.Marshaler, txId txbasic.TxID, height uint64, tx *txbasic.Transaction) error {
	txBytes, _ := tx.Bytes()
	sTx := struct {
		Height  uint64
		TxBytes []byte
	}{height, txBytes}
	sBytes, _ := json.Marshal(&sTx)

	servant.MetaStore.Put(MetaData_TxPool_Tx, []byte(txId), sBytes)

	return nil
}

func (servant *transactionPoolServant) loadAllTxs(marshaler codec.Marshaler, addTx func(tx *txbasic.Transaction, height uint64) (txpoolcore.TxWrapper, error)) error {
	servant.MetaStore.IterateAllMetaDataCB(MetaData_TxPool_Tx, func(key []byte, val []byte) {
		var sTx struct {
			Height  uint64
			TxBytes []byte
		}
		var tx txbasic.Transaction

		json.Unmarshal(val, &sTx)
		err := marshaler.Unmarshal(sTx.TxBytes, &tx)
		if err == nil {
			addTx(&tx, sTx.Height)
		}
	})

	return nil
}

func (servant *transactionPoolServant) stop() {
	servant.MetaStore.Commit()
	//	servant.MetaStore.Stop()
	servant.MetaStore.Close()
}

type TxMsgSubProcessor interface {
	Validate(ctx context.Context, isLocal bool, data []byte) message.ValidationResult
	Process(ctx context.Context, subMsgTxMessage *TxMessage) error
}

type txMsgSubProcessor struct {
	exeDomainID string
	nodeID      string
	log         tplog.Logger
	txPool      *transactionPool
}

func NewTxMsgSubProcessor(exeDomainID string, nodeID string, log tplog.Logger, txPool *transactionPool) TxMsgSubProcessor {
	return &txMsgSubProcessor{
		exeDomainID: exeDomainID,
		nodeID:      nodeID,
		log:         log,
		txPool:      txPool,
	}
}

func (msgSub *txMsgSubProcessor) getTxServantMem() txbasic.TransactionServant {
	compStateRN := state.CreateCompositionStateReadonly(msgSub.log, msgSub.txPool.txServant.GetLedger())
	compStateMem := state.CreateCompositionStateMem(msgSub.log, compStateRN)

	return txbasic.NewTransactionServant(compStateMem, compStateMem, msgSub.txPool.marshaler, msgSub.txPool.Size)
}

func (msgSub *txMsgSubProcessor) Validate(ctx context.Context, isLocal bool, data []byte) message.ValidationResult {
	if msgSub.txPool.Size() > msgSub.txPool.config.TxPoolMaxSize {
		return message.ValidationReject
	}
	msg := &TxPoolMessage{}
	if err := msg.Unmarshal(data); err != nil {
		msgSub.log.Errorf("Tx pool received invalid msg: isLocal %v, self node %s", isLocal, msgSub.nodeID)
		return message.ValidationReject
	}

	if string(msg.FromDomain) != msgSub.exeDomainID {
		msgSub.log.Errorf("Tx pool received invalid domain msg: isLocal %v, %s expected %s, self node %s", isLocal, string(msg.FromDomain), msgSub.exeDomainID, msgSub.nodeID)
		return message.ValidationReject
	}

	txMsg := &TxMessage{}
	if err := txMsg.Unmarshal(msg.Data); err != nil {
		msgSub.log.Errorf("Tx pool received invalid msg data: isLocal %v, self node %s", isLocal, msgSub.nodeID)
		return message.ValidationReject
	}

	var tx txbasic.Transaction
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	err := marshaler.Unmarshal(txMsg.Data, &tx)
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

	if msgSub.txPool.PendingAccountTxCnt(tpcrtypes.NewFromBytes(tx.Head.FromAddr)) > msgSub.txPool.config.MaxCntOfEachAccount {
		return message.ValidationReject
	}
	if !isLocal {
		ac := transaction.CreatTransactionAction(&tx)

		verifyResult := ac.Verify(ctx, msgSub.log, msgSub.nodeID, msgSub.getTxServantMem())
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
	if subMsgTxMessage == nil {
		return ErrTxIsNil
	}

	var tx txbasic.Transaction
	err := msgSub.txPool.marshaler.Unmarshal(subMsgTxMessage.Data, &tx)
	if err != nil {
		msgSub.log.Error("txMessage data error")
		return err
	}
	msgSub.txPool.AddTx(&tx, false)

	return nil
}
