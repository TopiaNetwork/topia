package consensus

import (
	"bytes"
	"context"
	"fmt"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
	tx "github.com/TopiaNetwork/topia/transaction"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type consensusExecutor struct {
	log                     tplog.Logger
	txPool                  txpool.TransactionPool
	marshaler               codec.Marshaler
	ledger                  ledger.Ledger
	exeScheduler            execution.Executionscheduler
	deliver                 *messageDeliver
	preparePackedMsgExeChan chan *PreparePackedMessageExe
}

func newConsensusExecutor(log tplog.Logger, txPool txpool.TransactionPool, marshaler codec.Marshaler, ledger ledger.Ledger, exeScheduler execution.Executionscheduler, deliver *messageDeliver, preprePackedMsgExeChan chan *PreparePackedMessageExe) *consensusExecutor {
	return &consensusExecutor{
		log:                     log,
		txPool:                  txPool,
		marshaler:               marshaler,
		ledger:                  ledger,
		exeScheduler:            exeScheduler,
		deliver:                 deliver,
		preparePackedMsgExeChan: preprePackedMsgExeChan,
	}
}

func (e *consensusExecutor) start(ctx context.Context) {
	go func() {
		for {
			select {
			case perparePMExe := <-e.preparePackedMsgExeChan:
				compState := state.CreateCompositionState(e.log, e.ledger)

				latestBlock, err := compState.GetLatestBlock()
				if err != nil {
					e.log.Errorf("Can't get the latest bock when making prepare packed msg: %v", err)
					continue
				}

				latestBlockHash, _ := latestBlock.HashBytes(tpcmm.NewBlake2bHasher(0), e.marshaler)

				if bytes.Compare(latestBlockHash, perparePMExe.ParentBlockHash) != 0 {
					e.log.Errorf("Invalid parent block ref: expected %v, actual %v", latestBlockHash, perparePMExe.ParentBlockHash)
					continue
				}

				var receivedTxList []tx.Transaction
				for i := 0; i < len(perparePMExe.Txs); i++ {
					var tx tx.Transaction
					err = e.marshaler.Unmarshal(perparePMExe.Txs[i], &tx)
					if err != nil {
						e.log.Errorf("Invalid tx from pepare packed msg exe: marshal err %v", err)
						break
					}
					receivedTxList = append(receivedTxList, tx)
				}
				if err != nil {
					continue
				}
				receivedTxRoot := tx.TxRoot(receivedTxList)
				if bytes.Compare(receivedTxRoot, perparePMExe.TxRoot) != 0 {
					e.log.Errorf("Invalid pepare packed msg exe: tx root expected %v, actual %v", perparePMExe.TxRoot, receivedTxRoot)
					break
				}

				txPacked := &execution.PackedTxs{
					StateVersion: perparePMExe.StateVersion,
					TxRoot:       perparePMExe.TxRoot,
					TxList:       receivedTxList,
				}

				_, err = e.exeScheduler.ExecutePackedTx(ctx, txPacked, compState)
				if err != nil {
					e.log.Errorf("Execute state version %d packed txs err from remote: %v", txPacked.StateVersion, err)
				}
			case <-ctx.Done():
				e.log.Info("Consensus executor exit")
				return
			}
		}
	}()
}

func (e *consensusExecutor) makePreparePackedMsg(txRoot []byte, txRSRoot []byte, stateVersion uint64, txList []tx.Transaction, txResultList []tx.TransactionResult, compState state.CompositionState) (*PreparePackedMessageExe, *PreparePackedMessageProp, error) {
	if len(txList) != len(txResultList) {
		err := fmt.Errorf("Mismatch tx list count %d and tx result count %d", len(txList), len(txResultList))
		e.log.Errorf("%v", err)
		return nil, nil, err
	}

	latestBlock, err := compState.GetLatestBlock()
	if err != nil {
		e.log.Errorf("Can't get the latest bock when making prepare packed msg: %v", err)
		return nil, nil, err
	}
	parentBlockHash, _ := latestBlock.HashBytes(tpcmm.NewBlake2bHasher(0), e.marshaler)

	exePPM := &PreparePackedMessageExe{
		ChainID:         []byte(compState.ChainID()),
		Version:         CONSENSUS_VER,
		Epoch:           compState.GetCurrentEpoch(),
		Round:           compState.GetCurrentRound(),
		ParentBlockHash: parentBlockHash,
		StateVersion:    stateVersion,
		TxRoot:          txRoot,
	}

	proposePPM := &PreparePackedMessageProp{
		ChainID:         []byte(compState.ChainID()),
		Version:         CONSENSUS_VER,
		Epoch:           compState.GetCurrentEpoch(),
		Round:           compState.GetCurrentRound(),
		ParentBlockHash: parentBlockHash,
		StateVersion:    stateVersion,
		TxRoot:          txRoot,
		TxResultRoot:    txRSRoot,
	}

	for i := 0; i < len(txList); i++ {
		txBytes, _ := e.marshaler.Marshal(txList[i])
		exePPM.Txs = append(exePPM.Txs, txBytes)

		txHashBytes, _ := txList[i].HashBytes()
		txRSBytes, _ := e.marshaler.Marshal(txResultList[i])
		proposePPM.TxHashs = append(proposePPM.TxHashs, txHashBytes)
		proposePPM.TxResults = append(proposePPM.TxResults, txRSBytes)
	}

	return exePPM, proposePPM, nil
}

func (e *consensusExecutor) Prepare(ctx context.Context) error {
	pendTxs, err := e.txPool.Pending()
	if err != nil {
		e.log.Errorf("Can't get pending txs: %v", err)
		return err
	}

	if len(pendTxs) == 0 {
		e.log.Debug("Current pending txs'size 0")
		return nil
	}

	txRoot := tx.TxRoot(pendTxs)
	compState := state.CreateCompositionState(e.log, e.ledger)

	var packedTxs execution.PackedTxs

	maxStateVer, err := e.exeScheduler.MaxStateVersion(compState)
	if err != nil {
		e.log.Errorf("Can't get max state version: %v", err)
		return err
	}

	packedTxs.StateVersion = maxStateVer + 1
	packedTxs.TxRoot = txRoot
	packedTxs.TxList = append(packedTxs.TxList, pendTxs...)

	txsRS, err := e.exeScheduler.ExecutePackedTx(ctx, &packedTxs, compState)
	if err != nil {
		e.log.Errorf("Execute state version %d packed txs err from local: %v", packedTxs.StateVersion, err)
		return err
	}
	txRSRoot := tx.TxResultRoot(txsRS.TxsResult, packedTxs.TxList)

	packedMsgExe, packedMsgProp, err := e.makePreparePackedMsg(txRoot, txRSRoot, packedTxs.StateVersion, packedTxs.TxList, txsRS.TxsResult, compState)
	if err != nil {
		return err
	}

	err = e.deliver.deliverPreparePackagedMessageExe(ctx, packedMsgExe)
	if err != nil {
		e.log.Errorf("Deliver prepare packed message to execute network failed: err=%v", err)
		return err
	}

	err = e.deliver.deliverPreparePackagedMessageProp(ctx, packedMsgProp)
	if err != nil {
		e.log.Errorf("Deliver prepare packed message to propose network failed: err=%v", err)
		return err
	}

	return nil
}
