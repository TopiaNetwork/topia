package execution

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/eventhub"
	"time"

	"github.com/lazyledger/smt"
	"github.com/subchen/go-trylock/v2"
	"go.uber.org/atomic"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/block"
	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
	"github.com/TopiaNetwork/topia/sync"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
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

	MaxStateVersion(log tplog.Logger, ledger ledger.Ledger) (uint64, error)

	PackedTxProofForValidity(ctx context.Context, stateVersion uint64, txHashs [][]byte, txResultHashes [][]byte) ([][]byte, [][]byte, error)

	ExecutePackedTx(ctx context.Context, txPacked *PackedTxs, compState state.CompositionState) (*PackedTxsResult, error)

	CommitPackedTx(ctx context.Context, stateVersion uint64, blockHead *tpchaintypes.BlockHead, marshaler codec.Marshaler, blockStore block.BlockStore, network tpnet.Network) error
}

type executionScheduler struct {
	nodeID           string
	log              tplog.Logger
	manager          *executionManager
	executeMutex     trylock.TryLocker
	schedulerState   *atomic.Uint32
	lastStateVersion *atomic.Uint64
	exePackedTxsList *list.List
}

func NewExecutionScheduler(nodeID string, log tplog.Logger) *executionScheduler {
	exeLog := tplog.CreateModuleLogger(logcomm.InfoLevel, MOD_NAME, log)
	return &executionScheduler{
		nodeID:           nodeID,
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

		if txPacked.StateVersion != exeTxsL.StateVersion()+1 {
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

	packedTxsRS, err := exePackedTxs.Execute(scheduler.log, ctx, txbasic.NewTansactionServant(compState, compState))
	if err == nil {
		scheduler.lastStateVersion.Store(txPacked.StateVersion)
		exePackedTxs.packedTxsRS = packedTxsRS
		scheduler.exePackedTxsList.PushBack(exePackedTxs)

		return packedTxsRS, nil
	}

	scheduler.log.Errorf("State version %d packed tx eventually execute failed: %v", txPacked.StateVersion, err)

	return nil, err
}

func (scheduler *executionScheduler) MaxStateVersion(log tplog.Logger, ledger ledger.Ledger) (uint64, error) {
	scheduler.executeMutex.RLock()
	defer scheduler.executeMutex.RUnlock()

	compStatRN := state.CreateCompositionStateReadonly(log, ledger)
	defer compStatRN.Stop()

	latestBlock, err := compStatRN.GetLatestBlock()
	if err != nil {
		scheduler.log.Errorf("Can't get the latest block: %v", err)
		return 0, nil
	}

	maxStateVersion := latestBlock.Head.Height

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

		if exeTxsF.packedTxsRS == nil {
			err := fmt.Errorf("No exist responding packed txs result: stateVersion=%d", stateVersion)
			scheduler.log.Errorf("%v", err)
			return nil, nil, err
		}

		treeTx := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
		treeTxRS := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
		for i := 0; i < len(exeTxsF.packedTxs.TxList); i++ {
			txHashBytes, _ := exeTxsF.packedTxs.TxList[i].HashBytes()
			txRSHashBytes, _ := exeTxsF.packedTxsRS.TxsResult[i].HashBytes()

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

func (scheduler *executionScheduler) ConstructBlockAndBlockResult(marshaler codec.Marshaler, blockHead *tpchaintypes.BlockHead, compState state.CompositionState, packedTxs *PackedTxs, packedTxsRS *PackedTxsResult) (*tpchaintypes.Block, *tpchaintypes.BlockResult, error) {
	dstBH, err := blockHead.DeepCopy(blockHead)
	if err != nil {
		err := fmt.Errorf("Block head deep copy err: %v", err)
		return nil, nil, err
	}

	blockData := &tpchaintypes.BlockData{
		Version: blockHead.Version,
	}
	for i := 0; i < len(packedTxs.TxList); i++ {
		txBytes, _ := packedTxs.TxList[i].HashBytes()
		blockData.Txs = append(blockData.Txs, txBytes)
	}

	block := &tpchaintypes.Block{
		Head: dstBH,
		Data: blockData,
	}

	blockHash, err := block.HashBytes()
	if err != nil {
		scheduler.log.Errorf("Can't get current block hash: %v", err)
		return nil, nil, err
	}

	latestBlockResult, err := compState.GetLatestBlockResult()
	if err != nil {
		scheduler.log.Errorf("Can't get latest block result: %v", err)
		return nil, nil, err
	}
	blockRSHash, err := latestBlockResult.HashBytes(tpcmm.NewBlake2bHasher(0), marshaler)
	if err != nil {
		scheduler.log.Errorf("Can't get latest block result hash: %v", err)
		return nil, nil, err
	}

	blockResultHead := &tpchaintypes.BlockResultHead{
		Version:          blockHead.Version,
		PrevBlockResult:  blockRSHash,
		BlockHash:        blockHash,
		TxResultHashRoot: tpcmm.BytesCopy(packedTxsRS.TxRSRoot),
		Status:           tpchaintypes.BlockResultHead_OK,
	}

	blockResultData := &tpchaintypes.BlockResultData{
		Version: blockHead.Version,
	}
	for r := 0; r < len(packedTxsRS.TxsResult); r++ {
		txRSBytes, _ := packedTxsRS.TxsResult[r].HashBytes()
		blockResultData.TxResults = append(blockResultData.TxResults, txRSBytes)
	}

	blockResult := &tpchaintypes.BlockResult{
		Head: blockResultHead,
		Data: blockResultData,
	}

	return block, blockResult, nil
}

func (scheduler *executionScheduler) CommitPackedTx(ctx context.Context, stateVersion uint64, blockHead *tpchaintypes.BlockHead, marshaler codec.Marshaler, blockStore block.BlockStore, network tpnet.Network) error {
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

		block, blockRS, err := scheduler.ConstructBlockAndBlockResult(marshaler, blockHead, exeTxsF.compState, exeTxsF.packedTxs, exeTxsF.PackedTxsResult())
		if err != nil {
			scheduler.log.Errorf("ConstructBlockAndBlockResult err: %v", err)
			return err
		}
		err = exeTxsF.compState.SetLatestBlock(block)
		if err != nil {
			scheduler.log.Errorf("Set latest block err when CommitPackedTx: %v", err)
			return err
		}
		err = exeTxsF.compState.SetLatestBlockResult(blockRS)
		if err != nil {
			scheduler.log.Errorf("Set latest block result err when CommitPackedTx: %v", err)
			return err
		}

		errCMMState := exeTxsF.compState.Commit()
		if errCMMState != nil {
			scheduler.log.Errorf("Commit state version %d err: %v", stateVersion, errCMMState)
			return errCMMState
		}

		errCMMBlock := blockStore.CommitBlock(block)
		if errCMMBlock != nil {
			scheduler.log.Errorf("Commit block err: state version %d, err: %v", stateVersion, errCMMBlock)
			return errCMMBlock
		}

		eventhub.GetEventHubManager().GetEventHub(scheduler.nodeID).Trig(ctx, eventhub.EventName_BlockAdded, block)

		scheduler.exePackedTxsList.Remove(exeTxsItem)

		blockBytes, err := marshaler.Marshal(block)
		if err != nil {
			scheduler.log.Errorf("Marshal commited block err: stateVersion=%d, err=%v", stateVersion, err)
		}
		blockRSBytes, err := marshaler.Marshal(blockRS)
		if err != nil {
			scheduler.log.Errorf("Marshal commited block result err: stateVersion=%d, err=%v", stateVersion, err)
		}

		pubsubBlockInfo := &tpchaintypes.PubSubMessageBlockInfo{
			Block:       blockBytes,
			BlockResult: blockRSBytes,
		}

		pubsubBlockInfoBytes, err := marshaler.Marshal(pubsubBlockInfo)
		if err != nil {
			scheduler.log.Errorf("Marshal pubsub block info err: stateVersion=%d, err=%v", stateVersion, err)
			return err
		}

		go func() {
			err := network.Publish(ctx, []string{sync.MOD_NAME}, tpnetprotoc.PubSubProtocolID_BlockInfo, pubsubBlockInfoBytes)
			if err != nil {
				scheduler.log.Errorf("Publish block info err: stateVersion=%d, err=%v", stateVersion, err)
			}
		}()

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
