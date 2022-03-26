package execution

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lazyledger/smt"
	"github.com/subchen/go-trylock/v2"
	"go.uber.org/atomic"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger/block"
	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/state"
	tx "github.com/TopiaNetwork/topia/transaction"
)

const (
	MOD_NAME = "excution"
)

type SchedulerState uint32

const (
	SchedulerState_Unknown SchedulerState = iota
	SchedulerState_Idle
	SchedulerState_Executing
	SchedulerState_Commiting
)

type ExecutionScheduler interface {
	State() SchedulerState

	MaxStateVersion(compState state.CompositionState) (uint64, error)

	PackedTxProofForValidity(ctx context.Context, stateVersion uint64, txHashs [][]byte, txResultHashes [][]byte) ([][]byte, [][]byte, error)

	ExecutePackedTx(ctx context.Context, txPacked *PackedTxs, compState state.CompositionState) (*PackedTxsResult, error)

	CommitPackedTx(ctx context.Context, stateVersion uint64, block *tpchaintypes.Block, blockStore block.BlockStore) error
}

type executionScheduler struct {
	log              tplog.Logger
	manager          *executionManager
	executeMutex     trylock.TryLocker
	schedulerState   *atomic.Uint32
	lastStateVersion *atomic.Uint64
	exePackedTxsList *list.List
}

func NewExecutionScheduler(log tplog.Logger) *executionScheduler {
	exeLog := tplog.CreateModuleLogger(logcomm.InfoLevel, MOD_NAME, log)
	return &executionScheduler{
		log:              exeLog,
		manager:          newExecutionManager(),
		executeMutex:     trylock.New(),
		lastStateVersion: atomic.NewUint64(0),
		schedulerState:   atomic.NewUint32(uint32(SchedulerState_Idle)),
		exePackedTxsList: list.New(),
	}
}

func (scheduler *executionScheduler) State() SchedulerState {
	return SchedulerState(scheduler.schedulerState.Load())
}

func (scheduler *executionScheduler) ExecutePackedTx(ctx context.Context, txPacked *PackedTxs, compState state.CompositionState) (*PackedTxsResult, error) {
	if ok := scheduler.executeMutex.TryLockTimeout(10 * time.Second); !ok {
		err := fmt.Errorf("A packedTxs is executing, try later again")
		scheduler.log.Errorf("%v", err)
		return nil, err
	}
	defer scheduler.executeMutex.Unlock()

	scheduler.schedulerState.Store(uint32(SchedulerState_Executing))
	defer scheduler.schedulerState.Store(uint32(SchedulerState_Idle))

	if scheduler.exePackedTxsList.Len() != 0 {
		exeTxsItem := scheduler.exePackedTxsList.Front()
		exeTxsF := exeTxsItem.Value.(*executionPackedTxs)
		exeTxsL := scheduler.exePackedTxsList.Back().Value.(*executionPackedTxs)

		if txPacked.StateVersion >= exeTxsF.StateVersion() && txPacked.StateVersion <= exeTxsL.StateVersion() {
			scheduler.log.Infof("Existed executed packedTxs stateVersion=%d", txPacked.StateVersion)
			for exeTxsItem.Value.(*executionPackedTxs).StateVersion() != txPacked.StateVersion {
				exeTxsItem = exeTxsItem.Next()
			}

			return exeTxsItem.Value.(*executionPackedTxs).PackedTxsResult(), nil
		}

		if txPacked.StateVersion-exeTxsL.StateVersion() != 1 {
			err := fmt.Errorf("Invalid packedTxs state version, expected %d, actual %d", exeTxsL.StateVersion()+1, txPacked.StateVersion)
			scheduler.log.Errorf("%v", err)
			return nil, err
		}
	} else {
		lastStateVersion := scheduler.lastStateVersion.Load()
		if lastStateVersion != 0 && txPacked.StateVersion <= lastStateVersion {
			err := fmt.Errorf("Invalid packedTxs state version, expected bigger than %d, actual %d", lastStateVersion, txPacked.StateVersion)
			scheduler.log.Errorf("%v", err)
			return nil, err
		}
	}

	exePackedTxs := newExecutionPackedTxs(txPacked, compState)
	scheduler.exePackedTxsList.PushBack(exePackedTxs)

	packedTxsRS, err := exePackedTxs.Execute(scheduler.log, ctx, tx.NewTansactionServant(compState, compState))
	if err != nil {
		scheduler.lastStateVersion.Store(txPacked.StateVersion)
	}

	return packedTxsRS, err
}

func (scheduler *executionScheduler) MaxStateVersion(compState state.CompositionState) (uint64, error) {
	scheduler.executeMutex.RLock()
	defer scheduler.executeMutex.RUnlock()

	maxStateVersion, err := compState.StateLatestVersion()
	if err != nil {
		scheduler.log.Errorf("Can't get the latest state version: %v", err)
		return 0, err
	}
	if scheduler.exePackedTxsList.Len() > 0 {
		exeTxsL := scheduler.exePackedTxsList.Back().Value.(*executionPackedTxs)
		maxStateVersion = exeTxsL.StateVersion()
	}

	return maxStateVersion, nil
}

func (scheduler *executionScheduler) PackedTxProofForValidity(ctx context.Context, stateVersion uint64, txHashs [][]byte, txResultHashes [][]byte) ([][]byte, [][]byte, error) {
	scheduler.executeMutex.RLock()
	defer scheduler.executeMutex.RUnlock()

	if len(txHashs) == 0 || len(txResultHashes) == 0 || len(txHashs) != len(txResultHashes) {
		err := fmt.Errorf("Invalid tx count %d or tx result count %d", len(txHashs), len(txResultHashes))
		scheduler.log.Errorf("%v", err)
	}

	if scheduler.exePackedTxsList.Len() > 0 {
		exeTxsF := scheduler.exePackedTxsList.Front().Value.(*executionPackedTxs)
		if exeTxsF.StateVersion() != stateVersion {
			err := fmt.Errorf("Invalid state version: expected %d, actual %d", exeTxsF.StateVersion(), stateVersion)
			scheduler.log.Errorf("%v")
			return nil, nil, err
		}

		treeTx := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
		treeTxRS := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
		for i := 0; i < len(exeTxsF.packedTxs.TxList); i++ {
			txHashBytes, _ := exeTxsF.packedTxs.TxList[i].HashBytes()
			txRSHashBytes, _ := exeTxsF.packedTxsRS.TxsResult[i].HashBytes(exeTxsF.packedTxs.TxList[i].FromAddr)

			treeTx.Update(txHashBytes, txHashBytes)
			treeTxRS.Update(txRSHashBytes, txRSHashBytes)
		}

		var txProofs [][]byte
		var txRSProofs [][]byte
		for t := 0; t < len(txHashs); t++ {
			txProof, err := treeTx.Prove(txHashs[t])
			if err != nil {
				scheduler.log.Errorf("Can't get tx proof: t=%d, txHashBytes=v, err=%v", t, txHashs[t], err)
				return nil, nil, err
			}

			txRSProof, err := treeTxRS.Prove(txResultHashes[t])
			if err != nil {
				scheduler.log.Errorf("Can't get tx result proof: txRSHashBytes=v, err=%v", txResultHashes[t], err)
				return nil, nil, err
			}

			txProofBytes, _ := json.Marshal(txProof)
			txRSProofBytes, _ := json.Marshal(txRSProof)

			txProofs = append(txProofs, txProofBytes)
			txRSProofs = append(txRSProofs, txRSProofBytes)
		}

		return txProofs, txRSProofs, nil
	}

	return nil, nil, errors.New("Not exist packed txs")
}

func (scheduler *executionScheduler) CommitPackedTx(ctx context.Context, stateVersion uint64, block *tpchaintypes.Block, blockStore block.BlockStore) error {
	if ok := scheduler.executeMutex.TryLockTimeout(1 * time.Second); !ok {
		err := fmt.Errorf("A packedTxs is commiting, try later again")
		scheduler.log.Errorf("%v", err)
		return err
	}
	defer scheduler.executeMutex.Unlock()

	scheduler.schedulerState.Store(uint32(SchedulerState_Commiting))
	defer scheduler.schedulerState.Store(uint32(SchedulerState_Idle))

	if scheduler.exePackedTxsList.Len() != 0 {
		exeTxsItem := scheduler.exePackedTxsList.Front()
		exeTxsF := exeTxsItem.Value.(*executionPackedTxs)

		if stateVersion != exeTxsF.StateVersion() {
			err := fmt.Errorf("Invalid stateVersion to commit: expected %d, actual %d", exeTxsF.StateVersion(), stateVersion)
			scheduler.log.Errorf("%v", err)
			return err
		}

		errCMMState := exeTxsF.compState.Commit()
		if errCMMState != nil {
			scheduler.log.Errorf("Commit state version %d err: %s", stateVersion, errCMMState)
			return errCMMState
		}

		errCMMBlock := blockStore.CommitBlock(block)
		if errCMMState != nil {
			scheduler.log.Errorf("Commit block err: state version %d,  err: %s", stateVersion, errCMMBlock)
			return errCMMState
		}

		scheduler.exePackedTxsList.Remove(exeTxsItem)

		return nil
	} else {
		err := fmt.Errorf("Empty executed packedTxs stateVersion=%d", stateVersion)
		scheduler.log.Errorf("%v", err)
		return err
	}
}

func (s SchedulerState) String() string {
	switch s {
	case SchedulerState_Idle:
		return "Idle"
	case SchedulerState_Executing:
		return "Executing"
	case SchedulerState_Commiting:
		return "Commiting"
	default:
		return "Unknown"
	}
}

func (s SchedulerState) Value(state string) SchedulerState {
	switch state {
	case "Idle":
		return SchedulerState_Idle
	case "Executing":
		return SchedulerState_Executing
	case "Commiting":
		return SchedulerState_Commiting
	default:
		return SchedulerState_Unknown
	}
}
