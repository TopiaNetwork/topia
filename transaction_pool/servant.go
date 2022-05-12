package transactionpool

import (
	"context"
	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	crypttypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state/account"
	"github.com/TopiaNetwork/topia/state/chain"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/libp2p/go-libp2p-core/peer"
)

type BlockAddedEvent struct{ Block *types.Block }

type TransactionPoolServant interface {
	StateQueryService
	BlockService
	network.Network
}
type StateQueryService interface {
	GetNonce(address crypttypes.Address) (uint64, error)
	CurrentHeight() uint64
	GetMaxGasLimit() uint64
	LocalPeerID() peer.ID
}
type BlockService interface {
	CurrentBlock() *types.Block
	GetBlockByHash(hash types.BlockHash) *types.Block
	PublishTx(ctx context.Context, tx *basic.Transaction) error
}

type transactionPoolServant struct {
	accState     account.AccountState
	chainState   chain.ChainState
	currentBlock *types.Block
	Network      network.Network
	marshaler    codec.Marshaler
}

func (servant *transactionPoolServant) GetNonce(address crypttypes.Address) (uint64, error) {
	return servant.accState.GetNonce(address)
}
func (servant *transactionPoolServant) CurrentHeight() uint64 {
	return servant.currentBlock.Head.Height
}
func (servant *transactionPoolServant) GetMaxGasLimit() uint64 {
	return 987654321
}
func (servant *transactionPoolServant) LocalPeerID() peer.ID {
	return peer.ID("TEST")
}

func (servant *transactionPoolServant) CurrentBlock() *types.Block {
	return servant.currentBlock
}
func (servant *transactionPoolServant) GetBlockByHash(hash types.BlockHash) *types.Block {
	block, _ := servant.chainState.GetLatestBlock()
	return block
}

func (servant *transactionPoolServant) PublishTx(ctx context.Context, tx *basic.Transaction) error {
	if tx == nil {
		return nil
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
	//comment for unit Test

	servant.Network.Publish(ctx, toModuleName, protocol.SyncProtocolID_Msg, sendData)

	return nil
}
