package service

import (
	"context"
	"fmt"
	"math/big"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplgblock "github.com/TopiaNetwork/topia/ledger/block"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/state"
	txfactory "github.com/TopiaNetwork/topia/transaction"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpool "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TransactionService interface {
	TxIDExists(txID txbasic.TxID) (bool, error)

	GetTransactionByID(txID txbasic.TxID) (*txbasic.Transaction, error)

	GetTransactionResultByID(txID txbasic.TxID) (*txbasic.TransactionResult, error)

	GetTransactionCount(addr tpcrtypes.Address, height uint64) (uint64, error)

	ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error

	SuggestGasPrice() (uint64, error)

	EstimateGas(ctx context.Context, tx *txbasic.Transaction) (*big.Int, error)

	ExecuteTxSim(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)
}

type transactionService struct {
	tplgblock.BlockStore
	execution.ExecutionForwarder
	nodeID     string
	log        tplog.Logger
	ledger     ledger.Ledger
	txPool     txpool.TransactionPool
	stateQuery StateQueryService
	marshaler  codec.Marshaler
	config     *tpconfig.Configuration
}

func newTransactionService(nodeID string,
	log tplog.Logger,
	marshaler codec.Marshaler,
	network tpnet.Network,
	ledger ledger.Ledger,
	txPool txpool.TransactionPool,
	stateQuery StateQueryService,
	config *tpconfig.Configuration) TransactionService {
	return &transactionService{
		BlockStore:         ledger.GetBlockStore(),
		ExecutionForwarder: execution.NewExecutionForwarder(nodeID, log, marshaler, network, ledger, txPool),
		nodeID:             nodeID,
		log:                log,
		ledger:             ledger,
		txPool:             txPool,
		stateQuery:         stateQuery,
		marshaler:          marshaler,
		config:             config,
	}
}

func (txs *transactionService) GetTransactionCount(addr tpcrtypes.Address, height uint64) (uint64, error) {
	block, err := txs.GetBlockByNumber(tpchaintypes.BlockNum(height))
	if err != nil {
		return 0, err
	}

	txTotalCount := uint32(0)
	for _, bhChunkBytes := range block.Head.HeadChunks {
		var bhChunk tpchaintypes.BlockHeadChunk
		bhChunk.Unmarshal(bhChunkBytes)
		txTotalCount += bhChunk.TxCount
	}

	return uint64(txTotalCount), nil
}

func (txs *transactionService) SuggestGasPrice() (uint64, error) {
	return txuni.NewGasPriceComputer(txs.marshaler, txs.txPool.Size, txs.stateQuery.GetLatestBlock, txs.config.GasConfig, txs.config.ChainConfig).ComputeGasPrice()
}

func (txs *transactionService) getTxServantMem() txbasic.TransactionServant {
	compStateRN := state.CreateCompositionStateReadonly(txs.log, txs.ledger)
	compStateMem := state.CreateCompositionStateMem(txs.log, compStateRN)

	return txbasic.NewTransactionServant(compStateMem, compStateMem, txs.marshaler, txs.txPool.Size)
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
