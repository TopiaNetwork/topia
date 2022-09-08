package transactionpool

import (
	"context"
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
	"github.com/TopiaNetwork/topia/network/protocol"
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

	PublishTx(ctx context.Context, tx *txbasic.Transaction) error

	savePoolConfig(conf *tpconfig.TransactionPoolConfig, marshaler codec.Marshaler) error

	loadPoolConfig(marshaler codec.Marshaler) (*tpconfig.TransactionPoolConfig, error)

	removeTxFromStore(txId txbasic.TxID)

	saveTxIntoStore(marshaler codec.Marshaler, txId txbasic.TxID, tx *txbasic.Transaction) error

	loadAllTxs(marshaler codec.Marshaler, addTx func(tx *txbasic.Transaction) (txpoolcore.TxWrapper, error)) error

	Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error

	UnSubscribe(topic string) error

	getStateQuery() service.StateQueryService
}

func newTransactionPoolServant(
	stateQueryService service.StateQueryService,
	blockService service.BlockService,
	network tpnet.Network,
	ledger ledger.Ledger) TransactionPoolServant {

	metaStore, _ := ledger.CreateMetaStore()

	metaStore.AddNamedStateStore(MetaData_TxPool_Tx, 50)
	metaStore.AddNamedStateStore(MetaData_TxPool_Config, 50)

	txpoolservant := &transactionPoolServant{
		ledger:       ledger,
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

func (servant *transactionPoolServant) PublishTx(ctx context.Context, tx *txbasic.Transaction) error {
	if tx == nil {
		return ErrTxIsNil
	}

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	data, err := marshaler.Marshal(tx)
	if err != nil {
		return err
	}

	msg := &TxMessage{
		Data: data,
	}
	sendData, err := msg.Marshal()
	if err != nil {
		return err
	}

	servant.Network.Publish(ctx, []string{txpooli.MOD_NAME}, protocol.SyncProtocolID_Msg, sendData)

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

func (servant *transactionPoolServant) saveTxIntoStore(marshaler codec.Marshaler, txId txbasic.TxID, tx *txbasic.Transaction) error {
	txBytes, _ := tx.Bytes()
	servant.MetaStore.Put(MetaData_TxPool_Tx, []byte(txId), txBytes)
	return nil
}

func (servant *transactionPoolServant) loadAllTxs(marshaler codec.Marshaler, addTx func(tx *txbasic.Transaction) (txpoolcore.TxWrapper, error)) error {
	servant.MetaStore.IterateAllMetaDataCB(MetaData_TxPool_Tx, func(key []byte, val []byte) {
		var tx txbasic.Transaction
		err := marshaler.Unmarshal(val, &tx)
		if err == nil {
			addTx(&tx)
		}
	})

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

func (msgSub *txMsgSubProcessor) getTxServantMem() txbasic.TransactionServant {
	compStateRN := state.CreateCompositionStateReadonly(msgSub.log, msgSub.txPool.txServant.GetLedger())
	compStateMem := state.CreateCompositionStateMem(msgSub.log, compStateRN)

	return txbasic.NewTransactionServant(compStateMem, compStateMem, msgSub.txPool.marshaler, msgSub.txPool.Size)
}

func (msgSub *txMsgSubProcessor) Validate(ctx context.Context, isLocal bool, sendData []byte) message.ValidationResult {
	if msgSub.txPool.Size() > msgSub.txPool.config.TxPoolMaxSize {
		return message.ValidationReject
	}
	msg := &TxMessage{}
	msg.Unmarshal(sendData)
	var tx txbasic.Transaction
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

	if msgSub.txPool.PendingAccountTxCnt(tpcrtypes.Address(tx.Head.FromAddr)) > msgSub.txPool.config.MaxCntOfEachAccount {
		return message.ValidationReject
	}
	if !isLocal {
		ac := transaction.CreatTransactionAction(&tx)

		verifyResult := ac.Verify(ctx, msgSub.GetLogger(), msgSub.GetNodeID(), msgSub.getTxServantMem())
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
