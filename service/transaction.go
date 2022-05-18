package service

import (
	"context"
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/state"
	txfactory "github.com/TopiaNetwork/topia/transaction"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
	"math"
	"math/big"

	"github.com/TopiaNetwork/topia/execution"
	tplgblock "github.com/TopiaNetwork/topia/ledger/block"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionService interface {
	TxIDExists(txID txbasic.TxID) (bool, error)

	GetTransactionByID(txID txbasic.TxID) (*txbasic.Transaction, error)

	GetTransactionResultByID(txID txbasic.TxID) (*txbasic.TransactionResult, error)

	GetTransactionCount(addr tpcrtypes.Address, height uint64) (uint64, error)

	ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error

	ComputeGasPrice() (uint64, error)

	EstimateGas(ctx context.Context, tx *txbasic.Transaction) (*big.Int, error)

	ExecuteTxSim(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)
}

type transactionService struct {
	tplgblock.BlockStore
	execution.ExecutionForwarder
	nodeID         string
	log            tplog.Logger
	ledger         ledger.Ledger
	txPool         txpool.TransactionPool
	latestGasPrice uint64
	config         *tpconfig.Configuration
}

func newTransactionService(nodeID string,
	log tplog.Logger,
	marshaler codec.Marshaler,
	network tpnet.Network,
	ledger ledger.Ledger,
	txPool txpool.TransactionPool,
	config *tpconfig.Configuration) TransactionService {
	return &transactionService{
		BlockStore:         ledger.GetBlockStore(),
		ExecutionForwarder: execution.NewExecutionForwarder(nodeID, log, marshaler, network, ledger, txPool),
		nodeID:             nodeID,
		log:                log,
		ledger:             ledger,
		txPool:             txPool,
		config:             config,
		latestGasPrice:     config.GasConfig.MinGasPrice,
	}
}

func (txs *transactionService) GetTransactionCount(addr tpcrtypes.Address, height uint64) (uint64, error) {
	block, err := txs.GetBlockByNumber(tpchaintypes.BlockNum(height))
	if err != nil {
		return 0, err
	}

	return uint64(block.Head.TxCount), nil
}

func (txs *transactionService) ComputeGasPrice() (uint64, error) {
	pendingBlock := uint64(0)
	if uint64(txs.txPool.Size())%txs.config.ChainConfig.MaxTxSizeOfEachBlock > 0 {
		pendingBlock = uint64(txs.txPool.Size())/txs.config.ChainConfig.MaxTxSizeOfEachBlock + 1
	}

	if pendingBlock <= 1 { //idle
		txs.latestGasPrice = txs.config.GasConfig.MinGasPrice
		return txs.latestGasPrice, nil
	}

	tempGasPrice := float64(txs.latestGasPrice)
	if pendingBlock > 1 {
		for i := uint64(0); i < pendingBlock-1; i++ {
			tempGasPrice += math.Pow(tempGasPrice, txs.config.GasConfig.GasPriceMultiple)
		}
	}

	txs.latestGasPrice = uint64(tempGasPrice)

	if txs.latestGasPrice >= txs.config.GasConfig.MinGasPrice*5 {
		txs.latestGasPrice = txs.config.GasConfig.MinGasPrice * 5
	}

	return txs.latestGasPrice, nil
}

func (txs *transactionService) getTxServantMem() txbasic.TransactionServant {
	compStateRN := state.CreateCompositionStateReadonly(txs.log, txs.ledger)
	compStateMem := state.CreateCompositionStateMem(txs.log, compStateRN)

	return txbasic.NewTransactionServant(compStateMem, compStateMem)
}

func (txs *transactionService) EstimateGas(ctx context.Context, tx *txbasic.Transaction) (*big.Int, error) {
	if txbasic.TransactionCategory(tx.Head.Category) != txbasic.TransactionCategory_Topia_Universal {
		return nil, fmt.Errorf("Unsupport tx sim: category %s", txbasic.TransactionCategory(tx.Head.Category))
	}

	txAction := txfactory.CreatTransactionAction(tx)
	txServant := txs.getTxServantMem()

	return txAction.Estimate(ctx, txs.log, txs.nodeID, txServant)
}

func (txs *transactionService) ExecuteTxSim(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error) {
	if txbasic.TransactionCategory(tx.Head.Category) != txbasic.TransactionCategory_Topia_Universal {
		return nil, fmt.Errorf("Unsupport tx sim: category %s", txbasic.TransactionCategory(tx.Head.Category))
	}
	txAction := txfactory.CreatTransactionAction(tx)
	txServant := txs.getTxServantMem()

	if txAction.Verify(ctx, txs.log, txs.nodeID, txServant) == txbasic.VerifyResult_Reject {
		txID, _ := tx.TxID()
		return nil, fmt.Errorf("Sim tx %s verify failed", txID)
	}

	return txAction.Execute(ctx, txs.log, txs.nodeID, txServant), nil
}
