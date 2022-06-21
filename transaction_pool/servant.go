package transactionpool

import (
	"context"

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

const MaxUint64 = 1<<64 - 1

type TransactionPoolServant interface {
	CurrentHeight() (uint64, error)
	GetNonce(tpcrtypes.Address) (uint64, error)
	GetLatestBlock() (*tpchaintypes.Block, error)
	GetBlockByHash(hash tpchaintypes.BlockHash) (*tpchaintypes.Block, error)
	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)
	PublishTx(ctx context.Context, tx *txbasic.Transaction) error
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

type TxMessageSubProcessor interface {
	Validate(ctx context.Context, isLocal bool, data []byte) message.ValidationResult
	Process(ctx context.Context, subMsgTxMessage *TxMessage) error
}

var TxMsgSubProcessor TxMessageSubProcessor

type txMessageSubProcessor struct {
	txpool *transactionPool
	log    tplog.Logger
	nodeID string
}

func (msgSub *txMessageSubProcessor) GetLoger() tplog.Logger {
	return msgSub.log
}
func (msgSub *txMessageSubProcessor) GetNodeID() string {
	return msgSub.nodeID
}

func (msgSub *txMessageSubProcessor) Validate(ctx context.Context, isLocal bool, sendData []byte) message.ValidationResult {
	if uint64(msgSub.txpool.Size()) > msgSub.txpool.config.TxPoolMaxSize {
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
	if uint64(tx.Size()) > txpooli.DefaultTransactionPoolConfig.TxMaxSize {
		msgSub.log.Errorf("transaction size is up to the TxMaxSize")
		return message.ValidationReject
	}
	if tx.Head.Nonce > MaxUint64 {
		msgSub.log.Errorf("transaction nonce is up to the MaxUint64")
		return message.ValidationReject
	}
	if txpooli.DefaultTransactionPoolConfig.GasPriceLimit < GasLimit(tx) {
		msgSub.log.Errorf("transaction gaslimit is up to GasPriceLimit")
		return message.ValidationReject
	}

	if isLocal {
		if uint64(tx.Size()) > txpooli.DefaultTransactionPoolConfig.MaxSizeOfEachPendingAccount {
			return message.ValidationReject
		}
		return message.ValidationAccept
	} else {
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
func (msgSub *txMessageSubProcessor) Process(ctx context.Context, subMsgTxMessage *TxMessage) error {
	var tx *txbasic.Transaction
	txId, _ := tx.TxID()
	err := tx.Unmarshal(subMsgTxMessage.Data)
	if err != nil {
		msgSub.log.Error("txmessage data error")
		return err
	}
	category := txbasic.TransactionCategory(tx.Head.Category)
	msgSub.txpool.newTxListStructs(category)

	if err := msgSub.txpool.AddTx(tx, false); err != nil {
		return err
	}

	msgSub.txpool.txCache.Add(txId, txpooli.StateTxAdded)
	return nil
}
