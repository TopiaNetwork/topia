package servant

import (
	"context"
	"math/big"

	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type APIServant interface {
	ChainID() chain.ChainID

	NetworkType() tpnet.NetworkType

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(symbol chain.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error)

	EstimateGas(tx *txbasic.Transaction) (*big.Int, error)

	GetContractCode(addr tpcrtypes.Address, height uint64) ([]byte, error)

	GetTransactionByHash(txHashHex string) (*txbasic.Transaction, error)

	GetTransactionCount(addr tpcrtypes.Address, height uint64) (uint64, error)

	GetTransactionResultByHash(txHashHex string) (*txbasic.TransactionResult, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetBlockByHeight(height uint64) (*tpchaintypes.Block, error)

	GetBlockByHash(txHashHex string) (*tpchaintypes.Block, error)

	ExecuteTxSim(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error
}

type apiServant struct{}

func NewAPIServant() APIServant {
	return &apiServant{}
}

func (s *apiServant) ChainID() chain.ChainID {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) NetworkType() tpnet.NetworkType {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetNonce(addr tpcrtypes.Address) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetBalance(symbol chain.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) EstimateGas(tx *txbasic.Transaction) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetContractCode(addr tpcrtypes.Address, height uint64) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetTransactionByHash(txHashHex string) (*txbasic.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetTransactionCount(addr tpcrtypes.Address, height uint64) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetTransactionResultByHash(txHashHex string) (*txbasic.TransactionResult, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetLatestBlock() (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetBlockByHeight(height uint64) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetBlockByHash(txHashHex string) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) ExecuteTxSim(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error {
	//TODO implement me
	panic("implement me")
}
