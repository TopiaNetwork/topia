package execution

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lazyledger/smt"
	"github.com/subchen/go-trylock/v2"
	"go.uber.org/atomic"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
	txservant "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
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

const MaxAvail_Count = 5

type EpochUpdater interface {
	UpdateEpoch(ctx context.Context, newBH *tpchaintypes.BlockHead, compState state.CompositionState) error
}

type ExecutionScheduler interface {
	SetEpochUpdater(epochUpdater EpochUpdater)

	State() SchedulerState

	MaxStateVersion(latestBlock *tpchaintypes.Block) (uint64, error)

	CompositionStateAtVersion(stateVersion uint64) state.CompositionState

	PackedTxProofForValidity(ctx context.Context, stateVersion uint64, txHashs [][]byte, txResultHashes [][]byte) ([][]byte, [][]byte, error)

	CompositionStateOfExePackedTxs(stateVersion uint64) (state.CompositionState, error)

	ExecutePackedTx(ctx context.Context, txPacked *PackedTxs, compState state.CompositionState) (*PackedTxsResult, error)

	CommitBlock(ctx context.Context, stateVersion uint64, block *tpchaintypes.Block, blockRS *tpchaintypes.BlockResult, compState state.CompositionState, requester string) error

	CommitPackedTx(ctx context.Context, stateVersion uint64, refIndex uint32, blockHead *tpchaintypes.BlockHead, deltaHeight int, marshaler codec.Marshaler, network tpnet.Network, ledger ledger.Ledger) error
}

type executionScheduler struct {
	nodeID           string
	log              tplog.Logger
	manager          *executionManager
	executeMutex     trylock.TryLocker
	schedulerState   *atomic.Uint32
	lastStateVersion *atomic.Uint64
	exePackedCount   *atomic.Uint32
	exePackedTxsList *list.List
	config           *configuration.Configuration
	epochUpdater     EpochUpdater
	marshaler        codec.Marshaler
	txPool           txpooli.TransactionPool
	syncCommitBlock  sync.RWMutex
}

func NewExecutionScheduler(nodeID string, log tplog.Logger, config *configuration.Configuration, codecType codec.CodecType, txPool txpooli.TransactionPool) *executionScheduler {
	exeLog := tplog.CreateModuleLogger(logcomm.InfoLevel, MOD_NAME, log)
	return &executionScheduler{
		nodeID:           nodeID,
		log:              exeLog,
		manager:          newExecutionManager(),
		executeMutex:     trylock.New(),
		lastStateVersion: atomic.NewUint64(0),
		schedulerState:   atomic.NewUint32(uint32(SchedulerState_Idle)),
		exePackedCount:   atomic.NewUint32(0),
		exePackedTxsList: list.New(),
		config:           config,
		marshaler:        codec.CreateMarshaler(codecType),
		txPool:           txPool,
	}
}

func (scheduler *executionScheduler) SetEpochUpdater(epochUpdater EpochUpdater) {
	scheduler.epochUpdater = epochUpdater
}

func (scheduler *executionScheduler) State() SchedulerState {
	return SchedulerState(scheduler.schedulerState.Load())
}

func (scheduler *executionScheduler) updateTree(txPacked *PackedTxs, txRSPacked *PackedTxsResult) {
	//Change to async in future
	func(txPackedG *PackedTxs, txRSPackedG *PackedTxsResult) {
		treeTx := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
		treeTxRS := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
		for i := 0; i < len(txPackedG.TxList); i++ {
			txHashBytes, _ := txPackedG.TxList[i].HashBytes()
			txRSHashBytes, _ := txRSPackedG.TxsResult[i].HashBytes()

			treeTx.Update(txHashBytes, txHashBytes)
			treeTxRS.Update(txRSHashBytes, txRSHashBytes)
		}

		txPacked.treeTx = treeTx
		txRSPacked.treeTxRS = treeTxRS
	}(txPacked, txRSPacked)
}

func (scheduler *executionScheduler) ExecutePackedTx(ctx context.Context, txPacked *PackedTxs, compState state.CompositionState) (*PackedTxsResult, error) {
	scheduler.log.Infof("Scheduler try to fetch the execution lock: state version %d, self node %s", txPacked.StateVersion, scheduler.nodeID)
	if ok := scheduler.executeMutex.TryLockTimeout(60 * time.Second); !ok {
		err := fmt.Errorf("A packedTxs is executing, try later again")
		scheduler.log.Errorf("%v", err)
		return nil, err
	}
	defer scheduler.executeMutex.Unlock()

	scheduler.schedulerState.Store(uint32(SchedulerState_Executing))
	defer scheduler.schedulerState.Store(uint32(SchedulerState_Idle))

	scheduler.log.Infof("Scheduler try to fetch the comp state lock: state version %d, self node %s", txPacked.StateVersion, scheduler.nodeID)

	compState.Lock()
	defer compState.Unlock()

	scheduler.log.Infof("Scheduler begin to execute: state version %d, self node %s，txs'len %d", txPacked.StateVersion, scheduler.nodeID, scheduler.exePackedTxsList.Len())

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

	exePackedTxs := newExecutionPackedTxs(scheduler.nodeID, txPacked, compState)

	packedTxsRS, err := exePackedTxs.Execute(scheduler.log, ctx, txservant.NewTransactionServant(compState, compState, scheduler.marshaler, scheduler.txPool.Size))
	if err == nil {
		compState.UpdateCompSState(state.CompSState_Executed)
		scheduler.lastStateVersion.Store(txPacked.StateVersion)
		scheduler.exePackedCount.Add(1)
		exePackedTxs.packedTxsRS = packedTxsRS
		scheduler.exePackedTxsList.PushBack(exePackedTxs)

		scheduler.updateTree(txPacked, packedTxsRS)

		for _, tx := range txPacked.TxList {
			txID, _ := tx.TxID()
			scheduler.txPool.RemoveTxByKey(txID)
		}

		if scheduler.exePackedTxsList.Len() >= int(scheduler.config.CSConfig.MaxPrepareMsgCache) {
			iCycle := int(float32(scheduler.config.CSConfig.MaxPrepareMsgCache) * 0.2)
			for iCycle > 0 {
				topPackedTxs := scheduler.exePackedTxsList.Front()
				scheduler.exePackedTxsList.Remove(topPackedTxs)
				iCycle--
			}
		}

		return packedTxsRS, nil
	}

	scheduler.log.Errorf("State version %d packed tx eventually execute failed: %v", txPacked.StateVersion, err)

	return nil, err
}

func (scheduler *executionScheduler) MaxStateVersion(latestBlock *tpchaintypes.Block) (uint64, error) {
	exePackedCount := scheduler.exePackedCount.Load()
	if exePackedCount >= MaxAvail_Count {
		return 0, fmt.Errorf("Haved reached max avail state version: size=%d", exePackedCount)
	}

	maxStateVersion := latestBlock.Head.Height

	lastStateVersion := scheduler.lastStateVersion.Load()

	if maxStateVersion > lastStateVersion {
		scheduler.log.Warnf("The latest height %d bigger than the latest state version %d", latestBlock.Head.Height, lastStateVersion)
	} else {
		maxStateVersion = lastStateVersion
	}

	return maxStateVersion, nil
}

func (scheduler *executionScheduler) CompositionStateAtVersion(stateVersion uint64) state.CompositionState {
	scheduler.executeMutex.RLock()
	defer scheduler.executeMutex.RUnlock()

	if scheduler.exePackedTxsList.Len() > 0 {
		exeTxsItem := scheduler.exePackedTxsList.Front()
		for exeTxsItem != nil {
			exeTxsF := exeTxsItem.Value.(*executionPackedTxs)
			if exeTxsF.StateVersion() != stateVersion {
				exeTxsItem = exeTxsItem.Next()
			} else if exeTxsF.compState.CompSState() != state.CompSState_Commited {
				return exeTxsF.compState
			}
		}
		return nil
	} else {
		return nil
	}
}

func (scheduler *executionScheduler) PackedTxProofForValidity(ctx context.Context, stateVersion uint64, txHashs [][]byte, txResultHashes [][]byte) ([][]byte, [][]byte, error) {
	scheduler.executeMutex.RLock()
	defer scheduler.executeMutex.RUnlock()

	if len(txHashs) == 0 || len(txResultHashes) == 0 || len(txHashs) != len(txResultHashes) {
		err := fmt.Errorf("Invalid tx count %d or tx result count %d", len(txHashs), len(txResultHashes))
		scheduler.log.Errorf("%v", err)
	}

	if scheduler.exePackedTxsList.Len() > 0 {
		var scanedStateVer []uint64
		var exeTxsF *executionPackedTxs
		exeTxsItem := scheduler.exePackedTxsList.Front()
		for exeTxsItem != nil {
			exeTxsF = exeTxsItem.Value.(*executionPackedTxs)
			scanedStateVer = append(scanedStateVer, exeTxsF.StateVersion())
			if exeTxsF.StateVersion() != stateVersion {
				exeTxsItem = exeTxsItem.Next()
			} else {
				break
			}
		}
		if exeTxsItem == nil {
			err := fmt.Errorf("Can't find state version to verify: expected %d, existed packed txs state versions %v", stateVersion, scanedStateVer)
			scheduler.log.Errorf("%v", err)
			return nil, nil, err
		}

		if exeTxsF.packedTxsRS == nil {
			err := fmt.Errorf("No exist responding packed txs result: stateVersion=%d", stateVersion)
			scheduler.log.Errorf("%v", err)
			return nil, nil, err
		}

		var txProofs [][]byte
		var txRSProofs [][]byte
		for t := 0; t < len(txHashs); t++ {
			txProof, err := func(data []byte) (smt.SparseMerkleProof, error) {
				exeTxsF.packedTxs.sync.Lock()
				defer exeTxsF.packedTxs.sync.Unlock()
				return exeTxsF.packedTxs.treeTx.Prove(data)
			}(txHashs[t])
			if err != nil {
				scheduler.log.Errorf("Can't get tx proof: t=%d, txHashBytes=v, err=%v", t, txHashs[t], err)
				return nil, nil, err
			}

			txRSProof, err := func(data []byte) (smt.SparseMerkleProof, error) {
				exeTxsF.packedTxsRS.sync.Lock()
				defer exeTxsF.packedTxsRS.sync.Unlock()
				return exeTxsF.packedTxsRS.treeTxRS.Prove(data)
			}(txResultHashes[t])
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

	return nil, nil, fmt.Errorf("No any packed txs at present, state version %d to verify", stateVersion)
}

func (scheduler *executionScheduler) CompositionStateOfExePackedTxs(stateVersion uint64) (state.CompositionState, error) {
	scheduler.executeMutex.RLock()
	defer scheduler.executeMutex.RUnlock()

	exeTxsItem := scheduler.exePackedTxsList.Front()
	for exeTxsItem != nil {
		exeTxsF := exeTxsItem.Value.(*executionPackedTxs)
		if exeTxsF.StateVersion() != stateVersion {
			exeTxsItem = exeTxsItem.Next()
		} else {
			return exeTxsF.compState, nil
		}
	}

	return nil, fmt.Errorf("Can't find composition state: stateVersion %d, seld node %s", stateVersion, scheduler.nodeID)
}

func (scheduler *executionScheduler) constructBlockAndBlockResult(marshaler codec.Marshaler, refIndex uint32, blockHead *tpchaintypes.BlockHead, latestBlockResult *tpchaintypes.BlockResult, packedTxs *PackedTxs, packedTxsRS *PackedTxsResult) (*tpchaintypes.Block, *tpchaintypes.BlockResult, error) {
	dstBH, err := blockHead.DeepCopy(blockHead)
	if err != nil {
		err := fmt.Errorf("Block head deep copy err: %v", err)
		return nil, nil, err
	}

	blockData := &tpchaintypes.BlockData{
		Version: blockHead.Version,
	}
	bdChunk := &tpchaintypes.BlockDataChunk{
		Version:  blockHead.Version,
		RefIndex: refIndex,
	}
	for i := 0; i < len(packedTxs.TxList); i++ {
		txBytes, _ := packedTxs.TxList[i].Bytes()
		bdChunk.Txs = append(bdChunk.Txs, txBytes)
	}
	bdChunkBytes, _ := bdChunk.Marshal()
	blockData.DataChunks = append(blockData.DataChunks, bdChunkBytes)

	block := &tpchaintypes.Block{
		Head: dstBH,
		Data: blockData,
	}

	blockHash, err := block.HashBytes()
	if err != nil {
		scheduler.log.Errorf("Can't get current block hash: %v", err)
		return nil, nil, err
	}

	blockRSHash, err := latestBlockResult.HashBytes(tpcmm.NewBlake2bHasher(0), marshaler)
	if err != nil {
		scheduler.log.Errorf("Can't get latest block result hash: %v", err)
		return nil, nil, err
	}

	bRSHeadChunk := &tpchaintypes.BlockResultHeadChunk{
		Version:          blockHead.Version,
		TxResultHashRoot: tpcmm.BytesCopy(packedTxsRS.TxRSRoot),
	}
	bRSHeadChunkBytes, _ := bRSHeadChunk.Marshal()

	blockResultHead := &tpchaintypes.BlockResultHead{
		Version:         blockHead.Version,
		PrevBlockResult: blockRSHash,
		BlockHash:       blockHash,
		Status:          tpchaintypes.BlockResultHead_OK,
	}
	blockResultHead.ResultHeadChunks = append(blockResultHead.ResultHeadChunks, bRSHeadChunkBytes)

	blockResultData := &tpchaintypes.BlockResultData{
		Version: blockHead.Version,
	}
	bRSDataChunk := &tpchaintypes.BlockResultDataChunk{
		Version: blockHead.Version,
	}
	for r := 0; r < len(packedTxsRS.TxsResult); r++ {
		txRSBytes, _ := packedTxsRS.TxsResult[r].HashBytes()
		bRSDataChunk.TxResults = append(bRSDataChunk.TxResults, txRSBytes)
	}
	bRSDataChunkBytes, _ := bRSDataChunk.Marshal()
	blockResultData.ResultDataChunks = append(blockResultData.ResultDataChunks, bRSDataChunkBytes)

	blockResult := &tpchaintypes.BlockResult{
		Head: blockResultHead,
		Data: blockResultData,
	}

	return block, blockResult, nil
}

func (scheduler *executionScheduler) CommitBlock(ctx context.Context,
	stateVersion uint64,
	block *tpchaintypes.Block,
	blockRS *tpchaintypes.BlockResult,
	compState state.CompositionState,
	requester string) error {
	scheduler.log.Infof("Enter CommitBlock: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)

	scheduler.syncCommitBlock.Lock()
	defer scheduler.syncCommitBlock.Unlock()

	scheduler.log.Infof("CommitBlock gets lock: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)

	stateCommitErr := func() error {
		if compState == nil {
			return nil
		}

		compState.Lock()
		defer compState.Unlock()

		scheduler.log.Infof("CommitBlock gets compState lock: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)

		err := compState.SetLatestBlock(block)
		if err != nil {
			scheduler.log.Errorf("Set latest block err when CommitBlock: %v", err)
			return err
		}
		scheduler.log.Infof("CommitBlock set latest block: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)

		err = compState.SetLatestBlockResult(blockRS)
		if err != nil {
			scheduler.log.Errorf("Set latest block result err when CommitBlock: %v", err)
			return err
		}
		scheduler.log.Infof("CommitBlock set latest block result: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)

		scheduler.epochUpdater.UpdateEpoch(ctx, block.Head, compState)

		scheduler.log.Infof("CommitBlock begins committing: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)
		errCMMState := compState.Commit()
		if errCMMState != nil {
			scheduler.log.Errorf("Commit state version %d err: %v", compState.StateVersion(), errCMMState)
			return errCMMState
		}
		scheduler.log.Infof("CommitBlock finished committing: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)

		scheduler.log.Infof("CommitBlock begins updating comp state: stateVersion %d, height %d, requester %s, self node %s", stateVersion, block.Head.Height, requester, scheduler.nodeID)

		compState.UpdateCompSState(state.CompSState_Commited)

		scheduler.log.Infof("CompositionState changes to commited: state version %d, height %d, by %s, self node %s", compState.StateVersion(), block.Head.Height, requester, scheduler.nodeID)

		return nil
	}()
	if stateCommitErr != nil {
		return stateCommitErr
	}

	/*ToDo Save new block and block result to block store
	errCMMBlock := ledger.GetBlockStore().CommitBlock(block)
	if errCMMBlock != nil {
		scheduler.log.Errorf("Commit block err: state version %d, err: %v", stateVersion, errCMMBlock)
		return errCMMBlock
	}
	errCMMBlockRS := ledger.GetBlockStore().CommitBlockResult(blockRS)
	if errCMMBlockRS != nil {
		scheduler.log.Errorf("Commit block result err: state version %d, err: %v", stateVersion, errCMMBlockRS)
		return errCMMBlockRS
	}
	*/

	eventhub.GetEventHubManager().GetEventHub(scheduler.nodeID).Trig(ctx, eventhub.EventName_BlockAdded, block)

	return nil
}

func (scheduler *executionScheduler) CommitPackedTx(ctx context.Context,
	stateVersion uint64,
	refIndex uint32,
	blockHead *tpchaintypes.BlockHead,
	deltaHeight int,
	marshaler codec.Marshaler,
	network tpnet.Network,
	ledger ledger.Ledger) error {
	/*
		if ok := scheduler.executeMutex.TryLockTimeout(60 * time.Second); !ok {
			err := fmt.Errorf("A packedTxs is commiting, try later again")
			scheduler.log.Errorf("%v", err)
			return err
		}*/
	scheduler.executeMutex.Lock()
	defer scheduler.executeMutex.Unlock()

	scheduler.schedulerState.Store(uint32(SchedulerState_Commiting))
	defer scheduler.schedulerState.Store(uint32(SchedulerState_Idle))

	if scheduler.exePackedTxsList.Len() != 0 {
		var scanedStateVer []uint64
		var exeTxsF *executionPackedTxs
		exeTxsItem := scheduler.exePackedTxsList.Front()
		for exeTxsItem != nil {
			exeTxsF = exeTxsItem.Value.(*executionPackedTxs)
			scanedStateVer = append(scanedStateVer, exeTxsF.StateVersion())
			if stateVersion != exeTxsF.StateVersion() {
				exeTxsItem = exeTxsItem.Next()
			} else {
				break
			}
		}
		if exeTxsItem == nil {
			err := fmt.Errorf("Can't find state version to commit: expected %d, existed packed txs state versions %v", stateVersion, scanedStateVer)
			scheduler.log.Errorf("%v", err)
			return err
		}

		latestBlock, latestBlockResult, err := func() (*tpchaintypes.Block, *tpchaintypes.BlockResult, error) {
			latestBlock, err := state.GetLatestBlock(ledger)
			if err != nil {
				err = fmt.Errorf("Can't get the latest block: %v, can't coommit packed tx: height=%d", err, blockHead.Height)
				return nil, nil, err
			}

			latestBlockResult, err := state.GetLatestBlockResult(ledger)
			if err != nil {
				scheduler.log.Errorf("Can't get latest block result: %v", err)
				return nil, nil, err
			}

			return latestBlock, latestBlockResult, nil
		}()

		if latestBlock.Head.Height >= blockHead.Height {
			scheduler.log.Warnf("Receive delay committing packed txs request: height=%d, latest block height=%d, self node %s", blockHead.Height, latestBlock.Head.Height, scheduler.nodeID)
			return nil
		}

		block, blockRS, err := scheduler.constructBlockAndBlockResult(marshaler, refIndex, blockHead, latestBlockResult, exeTxsF.packedTxs, exeTxsF.PackedTxsResult())
		if err != nil {
			scheduler.log.Errorf("constructBlockAndBlockResult err: %v", err)
			return err
		}

		err = scheduler.CommitBlock(ctx, stateVersion, block, blockRS, exeTxsF.compState, "CommitPackedTx")
		if err != nil {
			scheduler.log.Errorf("CommitBlock err: %v, stateVersion=%d, self node %s", err, stateVersion, scheduler.nodeID)
			return err
		}

		//scheduler.exePackedTxsList.Remove(exeTxsItem)

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

		chainMsg := &tpchaintypes.ChainMessage{
			MsgType: tpchaintypes.ChainMessage_BlockInfo,
			Data:    pubsubBlockInfoBytes,
		}
		chainMsgBytes, err := marshaler.Marshal(chainMsg)
		if err != nil {
			scheduler.log.Errorf("Marshal chain msg err: stateVersion=%d, err=%v", stateVersion, err)
			return err
		}

		go func() {
			err := network.Publish(ctx, []string{tpchaintypes.MOD_NAME}, tpnetprotoc.PubSubProtocolID_BlockInfo, chainMsgBytes)
			if err != nil {
				scheduler.log.Errorf("Publish block info err: stateVersion=%d, err=%v", stateVersion, err)
			}
		}()

		scheduler.exePackedCount.Sub(1)

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
