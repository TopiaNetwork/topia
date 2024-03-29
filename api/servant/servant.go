package servant

import (
	"context"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/currency"
	"math/big"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type APIServant interface {
	ChainID() tpchaintypes.ChainID

	NetworkType() common.NetworkType

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(symbol currency.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error)

	EstimateGas(tx *txbasic.Transaction) (*big.Int, error)

	GetContractCode(addr tpcrtypes.Address, height uint64) ([]byte, error)

	GetTransactionByHash(txHashHex string) (*txbasic.Transaction, error)

	GetTransactionCount(addr tpcrtypes.Address, height uint64) (uint64, error)

	GetTransactionResultByHash(txHashHex string) (*txbasic.TransactionResult, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetBlockByHeight(height uint64) (*tpchaintypes.Block, error)

	GetBlockByHash(hashHex string) (*tpchaintypes.Block, error)

	GetBlockByTxHash(txHashHex string) (*tpchaintypes.Block, error)

	ExecuteTxSim(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error
}

type apiServant struct{}

func NewAPIServant() APIServant {
	return &apiServant{}
}

func (s *apiServant) ChainID() tpchaintypes.ChainID {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) NetworkType() common.NetworkType {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetNonce(addr tpcrtypes.Address) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetBalance(symbol currency.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error) {
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

func (s *apiServant) GetBlockByHash(hashHex string) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (s *apiServant) GetBlockByTxHash(txHashHex string) (*tpchaintypes.Block, error) {
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
