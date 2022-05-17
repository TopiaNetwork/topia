package transactionpool

import (
	"context"
	tplog "github.com/TopiaNetwork/topia/log"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tplgblock "github.com/TopiaNetwork/topia/ledger/block"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state/chain"
	"github.com/TopiaNetwork/topia/transaction"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionPoolServant interface {
	CurrentHeight() (uint64, error)
	GetLatestBlock() (*tpchaintypes.Block, error)
	GetBlockByHash(hash tpchaintypes.BlockHash) (*tpchaintypes.Block, error)
	PublishTx(ctx context.Context, tx *txbasic.Transaction) error
	Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error
	UnSubscribe(topic string) error
}

type transactionPoolServant struct {
	chainState chain.ChainState
	marshaler  codec.Marshaler
	blockStore tplgblock.BlockStore
	Network    tpnet.Network
}

func (servant *transactionPoolServant) CurrentHeight() (uint64, error) {
	curBlock, err := servant.chainState.GetLatestBlock()
	if err != nil {
		return 0, err
	}
	curHeight := curBlock.Head.Height
	return curHeight, nil
}

func (servant *transactionPoolServant) GetLatestBlock() (*tpchaintypes.Block, error) {
	return servant.chainState.GetLatestBlock()
}
func (servant *transactionPoolServant) GetBlockByHash(hash tpchaintypes.BlockHash) (*tpchaintypes.Block, error) {
	return servant.blockStore.GetBlockByHash(hash)
}

func (servant *transactionPoolServant) PublishTx(ctx context.Context, tx *txbasic.Transaction) error {
	if tx == nil {
		return ErrTxIsNil
	}
	msg := &TxMessage{}
	data, err := servant.marshaler.Marshal(tx)
	if err != nil {
		return err
	}
	msg.Data = data
	sendData, err := msg.Marshal()
	if err != nil {
		return err
	}
	var toModuleName []string
	toModuleName = append(toModuleName, MOD_NAME)
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
	msg := &TxMessage{}
	msg.Unmarshal(sendData)
	var tx *txbasic.Transaction
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	err := marshaler.Unmarshal(msg.Data, &tx)
	if err != nil {
		return message.ValidationReject
	}
	if isLocal {
		if numSegments(tx, DefaultTransactionPoolConfig.TxSegmentSize) > DefaultTransactionPoolConfig.TxMaxSegmentSize {
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
	if err := msgSub.txpool.ValidateTx(tx, false); err != nil {
		return err
	}
	category := txbasic.TransactionCategory(tx.Head.Category)
	msgSub.txpool.newTxListStructs(category)
	if err := msgSub.txpool.AddTx(tx, false); err != nil {
		return err
	}

	msgSub.txpool.txCache.Add(txId, StateTxAdded)
	return nil
}
