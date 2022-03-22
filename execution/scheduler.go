package execution

import (
	"container/list"
	"context"
	"fmt"
	"go.uber.org/atomic"
	"time"

	"github.com/subchen/go-trylock/v2"

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

type Executionscheduler interface {
	MaxStateVersion(compState state.CompositionState) (uint64, error)

	ExecutePackedTx(ctx context.Context, txPacked *PackedTxs, compState state.CompositionState) (*PackedTxsResult, error)

	CommitPackedTx(ctx context.Context, stateVersion uint64, block *tpchaintypes.Block, blockStore block.BlockStore) error
}

type executionScheduler struct {
	log              tplog.Logger
	manager          *executionManager
	executeMutex     trylock.TryLocker
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
		exePackedTxsList: list.New(),
	}
}

func (scheduler *executionScheduler) ExecutePackedTx(ctx context.Context, txPacked *PackedTxs, compState state.CompositionState) (*PackedTxsResult, error) {
	if ok := scheduler.executeMutex.TryLockTimeout(10 * time.Second); !ok {
		err := fmt.Errorf("A packedTxs is executing, try later again")
		scheduler.log.Errorf("%v", err)
		return nil, err
	}
	defer scheduler.executeMutex.Unlock()

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

func (scheduler *executionScheduler) CommitPackedTx(ctx context.Context, stateVersion uint64, block *tpchaintypes.Block, blockStore block.BlockStore) error {
	if ok := scheduler.executeMutex.TryLockTimeout(1 * time.Second); !ok {
		err := fmt.Errorf("A packedTxs is commiting, try later again")
		scheduler.log.Errorf("%v", err)
		return err
	}
	defer scheduler.executeMutex.Unlock()

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

		return nil
	} else {
		err := fmt.Errorf("Empty executed packedTxs stateVersion=%d", stateVersion)
		scheduler.log.Errorf("%v", err)
		return err
	}
}
