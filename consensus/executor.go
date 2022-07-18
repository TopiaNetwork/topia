package consensus

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/consensus/vrf"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
	lru "github.com/hashicorp/golang-lru"
)

type consensusExecutor struct {
	log                          tplog.Logger
	nodeID                       string
	priKey                       tpcrtypes.PrivateKey
	txPool                       txpooli.TransactionPool
	marshaler                    codec.Marshaler
	ledger                       ledger.Ledger
	exeScheduler                 execution.ExecutionScheduler
	epochService                 EpochService
	domainCSService              *domainConsensusService
	deliver                      *messageDeliver
	finishedMsgCh                chan *DKGFinishedMessage
	preparePackedMsgExeChan      chan *PreparePackedMessageExe
	preparePackedMsgExeIndicChan chan *PreparePackedMessageExeIndication
	commitMsgChan                chan *CommitMessage
	cryptService                 tpcrt.CryptService
	prepareInterval              time.Duration
	prepareMsgLunched            *lru.Cache //Key: state version
	remoteFinishedSync           sync.RWMutex
	remoteFinished               map[string]*DKGFinishedMessage //nodeID->*DKGFinishedMessage
}

func newConsensusExecutor(log tplog.Logger,
	nodeID string,
	priKey tpcrtypes.PrivateKey,
	txPool txpooli.TransactionPool,
	marshaler codec.Marshaler,
	ledger ledger.Ledger,
	exeScheduler execution.ExecutionScheduler,
	epochService EpochService,
	domainCSService *domainConsensusService,
	deliver *messageDeliver,
	finishedMsgCh chan *DKGFinishedMessage,
	preprePackedMsgExeChan chan *PreparePackedMessageExe,
	preparePackedMsgExeIndicChan chan *PreparePackedMessageExeIndication,
	commitMsgChan chan *CommitMessage,
	cryptService tpcrt.CryptService,
	prepareInterval time.Duration) *consensusExecutor {

	pMsgLCache, _ := lru.New(10)

	return &consensusExecutor{
		log:                          tplog.CreateSubModuleLogger("executor", log),
		nodeID:                       nodeID,
		priKey:                       priKey,
		txPool:                       txPool,
		marshaler:                    marshaler,
		ledger:                       ledger,
		exeScheduler:                 exeScheduler,
		epochService:                 epochService,
		domainCSService:              domainCSService,
		deliver:                      deliver,
		finishedMsgCh:                finishedMsgCh,
		preparePackedMsgExeChan:      preprePackedMsgExeChan,
		preparePackedMsgExeIndicChan: preparePackedMsgExeIndicChan,
		commitMsgChan:                commitMsgChan,
		cryptService:                 cryptService,
		prepareInterval:              prepareInterval,
		prepareMsgLunched:            pMsgLCache,
		remoteFinished:               make(map[string]*DKGFinishedMessage),
	}
}

func (e *consensusExecutor) startReceiveFinishedMsg(ctx context.Context) {
	go func() {
		for {
			select {
			case finishedMsg := <-e.finishedMsgCh:
				e.log.Infof("Executor receives dkg finished message: trigger number %d, self node %s", finishedMsg.TriggerNumber, e.nodeID)

				err := func() error {
					e.remoteFinishedSync.Lock()
					defer e.remoteFinishedSync.Unlock()

					if finishedMsg.TriggerNumber != e.domainCSService.triggerNumber {
						err := fmt.Errorf("Invalid finished message: expected trigger number %d, actual %d, self node %s", e.domainCSService.triggerNumber, finishedMsg.TriggerNumber, e.nodeID)
						e.log.Errorf("%v", err)
						return err
					}

					remoteID := string(finishedMsg.NodeID)
					if !e.domainCSService.ContainedInCandidateNodes(remoteID) {
						err := fmt.Errorf("Received finished message which not belongs to any candidate node: remote node %s, self node %s", remoteID, e.nodeID)
						e.log.Errorf("%v", err)
						return err
					}

					if _, ok := e.remoteFinished[remoteID]; !ok {
						e.remoteFinished[remoteID] = finishedMsg
						if len(e.remoteFinished) == e.domainCSService.CandidateNodesNumber() {
							e.log.Infof("Received message count reached the expected: %d", len(e.remoteFinished))
							e.domainCSService.ProduceAndSaveNodeDomain(0, nil, nil, nil)
							e.remoteFinished = make(map[string]*DKGFinishedMessage)
						}
						return nil
					} else {
						err := fmt.Errorf("Received the same dkg finished message: remote node %s, self node %s", remoteID, e.nodeID)
						e.log.Errorf("%v", err)
						return err
					}
				}()

				if err != nil {
					continue
				}
			case <-ctx.Done():
				e.log.Info("Executor receiving finished message loop stop")
				return
			}
		}
	}()
}

func (e *consensusExecutor) UnmarshalTxsOfPreparePackedMessage(txs [][]byte) ([]*txbasic.Transaction, error) {
	txSize := len(txs)
	receivedTxList := make([]*txbasic.Transaction, txSize)
	errRtn := error(nil)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	forceExitCh := make(chan bool)
	for i := 0; i < txSize; {
		startIndex := i
		endIndex := i + 500
		if endIndex >= txSize {
			endIndex = txSize
		}

		wg.Add(1)
		go func(startI, endI int) {
			defer wg.Done()

			for m := startI; m < endI; m++ {
				select {
				case <-ctx.Done():
					return
				default:
					var tx txbasic.Transaction
					err := e.marshaler.Unmarshal(txs[m], &tx)
					if err != nil {
						err = fmt.Errorf("Invalid tx from prepare packed msg exe: marshal err %v", err)
						e.log.Errorf("%v", err)
						errRtn = err
						forceExitCh <- true
						return
					}

					receivedTxList[m] = &tx
				}
			}
		}(startIndex, endIndex)

		i = endIndex
	}

	var waitWG sync.WaitGroup
	waitWG.Add(1)
	go func() {
		defer waitWG.Done()
		if forceExit := <-forceExitCh; forceExit {
			cancel()
		}
	}()

	wg.Wait()

	forceExitCh <- true

	waitWG.Wait()

	return receivedTxList, errRtn
}

func (e *consensusExecutor) receivePreparePackedMessageExeStart(ctx context.Context) {
	go func() {
		for {
			select {
			case perparePMExe := <-e.preparePackedMsgExeChan:
				if ok, _ := e.prepareMsgLunched.ContainsOrAdd(perparePMExe.StateVersion, struct{}{}); !ok {
					e.log.Infof("Received PreparePackedMessageExe from other executor, state version %d, self node %s", perparePMExe.StateVersion, e.nodeID)
				} else {
					e.log.Infof("Received PreparePackedMessageExe from other executor but existed, state version %d, self node %s", perparePMExe.StateVersion, e.nodeID)
					continue
				}

				compState := state.GetStateBuilder(state.CompStateBuilderType_Full).CreateCompositionState(e.log, e.nodeID, e.ledger, perparePMExe.StateVersion, "executor_exereceiver")

				waitCount := 1
				for compState == nil {
					e.log.Warnf("Can't CreateCompositionState when received new PreparePackedMessageExe, StateVersion %d, self node %s, waitCount %d", perparePMExe.StateVersion, e.nodeID, waitCount)
					compState = state.GetStateBuilder(state.CompStateBuilderType_Full).CreateCompositionState(e.log, e.nodeID, e.ledger, perparePMExe.StateVersion, "executor_exereceiver")
					waitCount++
					time.Sleep(50 * time.Millisecond)
				}

				latestBlock, err := compState.GetLatestBlock()
				if err != nil {
					e.log.Errorf("Can't get the latest bock when making prepare packed msg: %v", err)
					continue
				}

				latestBlockHash, _ := latestBlock.HashBytes()

				if bytes.Compare(latestBlockHash, perparePMExe.ParentBlockHash) != 0 {
					e.log.Errorf("Invalid parent block ref: expected %v, actual %v", latestBlockHash, perparePMExe.ParentBlockHash)
					//continue, tmp
				}

				receivedTxList, err := e.UnmarshalTxsOfPreparePackedMessage(perparePMExe.Txs)
				if err != nil {
					continue
				}
				receivedTxRoot := txbasic.TxRootByBytes(perparePMExe.Txs)
				if bytes.Compare(receivedTxRoot, perparePMExe.TxRoot) != 0 {
					e.log.Errorf("Invalid pepare packed msg exe: tx root expected %v, actual %v", perparePMExe.TxRoot, receivedTxRoot)
					break
				}

				txPacked := &execution.PackedTxs{
					StateVersion: perparePMExe.StateVersion,
					TxRoot:       perparePMExe.TxRoot,
					TxList:       receivedTxList,
				}

				e.log.Infof("Received packed txs begin to execute: state version %d, self node %s", perparePMExe.StateVersion, e.nodeID)
				_, err = e.exeScheduler.ExecutePackedTx(ctx, txPacked, compState)
				if err != nil {
					e.log.Errorf("Execute packed txs err from remote: %v, state version %d, self node %s", err, txPacked.StateVersion, e.nodeID)
				}
				e.log.Infof("Execute Packed tx successfully from remote, state version %d, self node %s", perparePMExe.StateVersion, e.nodeID)
				//go func(ctxdl context.Context) {
				msg := &PreparePackedMessageExeIndication{
					ChainID:      perparePMExe.ChainID,
					Version:      perparePMExe.Version,
					Epoch:        perparePMExe.Epoch,
					Round:        perparePMExe.Round,
					StateVersion: perparePMExe.StateVersion,
				}
				err = e.deliver.deliverPreparePackagedMessageExeIndication(ctx, string(perparePMExe.Launcher), msg)
				if err != nil {
					e.log.Errorf("Deliver PreparePackedMessageExeIndication err: state version %d, launcher %s, self node %s, err %v", perparePMExe.StateVersion, string(perparePMExe.Launcher), e.nodeID, err)
				}
				e.log.Errorf("Deliver PreparePackedMessageExeIndication successfully: state version %d, launcher %s, self node %s", perparePMExe.StateVersion, string(perparePMExe.Launcher), e.nodeID)
				//}(ctx)
			case <-ctx.Done():
				e.log.Info("Consensus executor receiveing prepare packed msg exe exit")
				return
			}
		}
	}()
}

func (e *consensusExecutor) receivePreparePackedMessageExeIndicationStart(ctx context.Context, requiredCount int, canForward chan bool) {
	go func() {
		recvCount := 1 //Contain self
		for {
			select {
			case perparePMExeIndic := <-e.preparePackedMsgExeIndicChan:
				recvCount++
				e.log.Infof("Received PreparePackedMessageExeIndication from other executor, state version %d, received %d required %d,  self node %s", perparePMExeIndic.StateVersion, recvCount, requiredCount, e.nodeID)
				if recvCount == requiredCount {
					canForward <- true
					return
				}
			case <-ctx.Done():
				e.log.Info("Consensus executor receiveing prepare packed msg exe indication exit")
				return
			}
		}
	}()
}

func (e *consensusExecutor) receiveCommitMsgStart(ctx context.Context) {
	go func() {
		for {
			select {
			case commitMsg := <-e.commitMsgChan:
				e.log.Infof("Received commit message, StateVersion %d, self node %s", commitMsg.StateVersion, e.nodeID)
				err := func() error {
					csStateRN := state.CreateCompositionStateReadonly(e.log, e.ledger)
					defer csStateRN.Stop()

					var bh tpchaintypes.BlockHead
					err := e.marshaler.Unmarshal(commitMsg.BlockHead, &bh)
					if err != nil {
						e.log.Errorf("Can't unmarshal received block head of commit message: %v", err)
						return err
					}

					latestBlock, err := csStateRN.GetLatestBlock()
					if err != nil {
						e.log.Errorf("Can't get the latest block: %v", err)
						return err
					}

					deltaHeight := int(bh.Height) - int(latestBlock.Head.Height)
					if deltaHeight <= 0 {
						err = fmt.Errorf("Received commit message is delayed, StateVersion %d, latest height %d", commitMsg.StateVersion, latestBlock.Head.Height)
						e.log.Infof("%s", err.Error())
						return err
					}

					err = e.exeScheduler.CommitPackedTx(ctx, commitMsg.StateVersion, &bh, deltaHeight, e.marshaler, e.deliver.network, e.ledger)
					if err != nil {
						e.log.Errorf("Commit packed tx err: %v", err)
						return err
					}

					e.log.Infof("Commit packed tx successfully, StateVersion %d", commitMsg.StateVersion)

					return nil
				}()

				if err != nil {
					continue
				}
			case <-ctx.Done():
				e.log.Info("Consensus executor receiveing prepare packed msg exe exit")
				return
			}
		}
	}()
}

func (e *consensusExecutor) canPrepare(stateVersion uint64, latestBlock *tpchaintypes.Block) (bool, []byte, error) {
	if schedulerState := e.exeScheduler.State(); schedulerState != execution.SchedulerState_Idle {
		err := fmt.Errorf("Execution scheduler state %s, self node %s", schedulerState.String(), e.nodeID)
		e.log.Errorf("%v", err)
		return false, nil, err
	}

	roleSelector := vrf.NewLeaderSelectorVRF(e.log, e.nodeID, e.cryptService)

	e.log.Infof("Begin Select: state version %d, self node %s", stateVersion, e.nodeID)
	candIDs, vrfProof, err := roleSelector.Select(vrf.RoleSelector_ExecutionLauncher, stateVersion, e.priKey, latestBlock, e.epochService, 1)
	if err != nil {
		e.log.Errorf("Select executor err: %v", err)
		return false, nil, err
	}
	e.log.Infof("End Select: state version %d, self node %s", stateVersion, e.nodeID)
	if len(candIDs) != 1 {
		err = fmt.Errorf("Invalid selected executor count: state version %d, expected 1, actual %d", stateVersion, len(candIDs))
		e.log.Errorf("%v", err)
		return false, nil, err
	}

	e.log.Infof("Selected launching node: state version %d, selected node %s, self node %s", stateVersion, candIDs[0], e.nodeID)

	if candIDs[0] == e.nodeID {
		return true, vrfProof, nil
	}

	return false, vrfProof, nil
}

func (e *consensusExecutor) prepareTimerStart(ctx context.Context) {
	e.log.Infof("Executor %s prepareTimerStart, timer=%fs", e.nodeID, e.prepareInterval.Seconds())

	go func() {
		timer := time.NewTicker(e.prepareInterval)
		defer timer.Stop()

		for !e.deliver.deliverNetwork().Ready() {
			time.Sleep(50 * time.Millisecond)
		}

		for {
			select {
			case <-timer.C:
				func() {
					e.log.Infof("Prepare timer starts: self node %s", e.nodeID)
					prepareStart := time.Now()
					compStatRN := state.CreateCompositionStateReadonly(e.log, e.ledger)
					defer compStatRN.Stop()

					latestBlock, err := compStatRN.GetLatestBlock()
					if err != nil {
						e.log.Errorf("Can't get the latest block: %v, self node %s", err, e.nodeID)
						return
					}

					e.log.Infof("Prepare timer starts to get max state version: self node %s", e.nodeID)

					var maxStateVersion uint64
					for maxStateVersion == 0 {
						maxStateVersion, err = e.exeScheduler.MaxStateVersion(latestBlock)
						if err != nil {
							//e.log.Warnf("Can't get max state version: %v, self node %s", err, e.nodeID)
							time.Sleep(50 * time.Millisecond)
						} else {
							break
						}
					}
					e.log.Infof("Prepare timer gets max state version: maxStateVersion %d, self node %s", maxStateVersion, e.nodeID)
					stateVersion := maxStateVersion + 1

					e.log.Infof("Prepare timer: state version %d, self node %s", stateVersion, e.nodeID)

					if e.prepareMsgLunched.Contains(stateVersion) {
						e.log.Infof("Have launched prepare message: state version %d, self node %s", stateVersion, e.nodeID)
						return
					}

					isCan, vrfProof, _ := e.canPrepare(stateVersion, latestBlock)
					if isCan {
						e.log.Infof("Selected execution launcher can prepare: state version %d, self node %s", stateVersion, e.nodeID)
						pStart := time.Now()
						e.Prepare(ctx, vrfProof, stateVersion)
						e.log.Infof("Prepare time: state version %d, cost %d ms, self node %s", stateVersion, time.Since(pStart).Milliseconds(), e.nodeID)
					}
					e.log.Infof("Prepare time total: isCan %v, state version %d, cost %d ms, self node %s", isCan, stateVersion, time.Since(prepareStart).Milliseconds(), e.nodeID)
				}()
			case <-ctx.Done():
				e.log.Infof("Consensus executor exit prepare timer: self node %s", e.nodeID)
				return
			}
		}
	}()
}

func (e *consensusExecutor) start(ctx context.Context) {
	e.startReceiveFinishedMsg(ctx)

	e.receivePreparePackedMessageExeStart(ctx)

	e.receiveCommitMsgStart(ctx)

	e.prepareTimerStart(ctx)
}

func (e *consensusExecutor) makePreparePackedMsg(vrfProof []byte, txRoot []byte, txRSRoot []byte, stateVersion uint64, txList []*txbasic.Transaction, txResultList []txbasic.TransactionResult, compState state.CompositionState) (*PreparePackedMessageExe, *PreparePackedMessageProp, error) {
	if len(txList) != len(txResultList) {
		err := fmt.Errorf("Mismatch tx list count %d and tx result count %d", len(txList), len(txResultList))
		e.log.Errorf("%v", err)
		return nil, nil, err
	}

	epochInfo, err := compState.GetLatestEpoch()
	if err != nil {
		e.log.Errorf("Can't get the latest epoch info when making prepare packed msg: %v", err)
		return nil, nil, err
	}

	latestBlock, err := compState.GetLatestBlock()
	if err != nil {
		e.log.Errorf("Can't get the latest bock when making prepare packed msg: %v", err)
		return nil, nil, err
	}
	parentBlockHash, _ := latestBlock.HashBytes()

	pubKey, _ := e.cryptService.ConvertToPublic(e.priKey)

	exePPM := &PreparePackedMessageExe{
		ChainID:         []byte(compState.ChainID()),
		Version:         CONSENSUS_VER,
		Epoch:           epochInfo.Epoch,
		Round:           latestBlock.Head.Height,
		ParentBlockHash: parentBlockHash,
		VRFProof:        vrfProof,
		VRFProofPubKey:  pubKey,
		Launcher:        []byte(e.nodeID),
		StateVersion:    stateVersion,
		TxRoot:          txRoot,
	}

	proposePPM := &PreparePackedMessageProp{
		ChainID:         []byte(compState.ChainID()),
		Version:         CONSENSUS_VER,
		Epoch:           epochInfo.Epoch,
		Round:           latestBlock.Head.Height,
		ParentBlockHash: parentBlockHash,
		VRFProof:        vrfProof,
		VRFProofPubKey:  pubKey,
		Launcher:        []byte(e.nodeID),
		StateVersion:    stateVersion,
		TxRoot:          txRoot,
		TxResultRoot:    txRSRoot,
	}

	var wg sync.WaitGroup
	txSize := len(txList)
	exePPM.Txs = make([][]byte, txSize)
	proposePPM.TxHashs = make([][]byte, txSize)
	proposePPM.TxResultHashs = make([][]byte, txSize)
	for i := 0; i < txSize; {
		startIndex := i
		endIndex := i + 500
		if endIndex >= txSize {
			endIndex = txSize
		}

		wg.Add(1)
		go func(startI, endI int) {
			defer wg.Done()

			for m := startI; m < endIndex; m++ {
				txBytes, _ := e.marshaler.Marshal(txList[m])
				txHashBytes, _ := txList[m].HashBytes()
				txRSHashBytes, _ := txResultList[m].HashBytes()

				exePPM.Txs[m] = txBytes
				proposePPM.TxHashs[m] = txHashBytes
				proposePPM.TxResultHashs[m] = txRSHashBytes
			}
		}(startIndex, endIndex)

		i = endIndex
	}

	wg.Wait()

	return exePPM, proposePPM, nil
}

func (e *consensusExecutor) Prepare(ctx context.Context, vrfProof []byte, stateVersion uint64) error {
	pendTxs := e.txPool.PickTxs()

	compState := state.GetStateBuilder(state.CompStateBuilderType_Full).CreateCompositionState(e.log, e.nodeID, e.ledger, stateVersion, "executor_exepreparer")
	if compState == nil {
		err := fmt.Errorf("Can't create composition state for Prepare: maxStateVer=%d", stateVersion)
		e.log.Errorf("%v", err)
		return err
	}

	var packedTxs execution.PackedTxs

	var txRoot []byte
	var txList []*txbasic.Transaction
	if len(pendTxs) != 0 {
		txRoot = txbasic.TxRoot(pendTxs)
		txList = append(txList, pendTxs...)
	}

	packedTxs.StateVersion = stateVersion
	packedTxs.TxRoot = txRoot
	packedTxs.TxList = txList

	txsRS, err := e.exeScheduler.ExecutePackedTx(ctx, &packedTxs, compState)
	if err != nil {
		e.log.Errorf("Execute state version %d packed txs err from local: %v", packedTxs.StateVersion, err)
		return err
	}

	e.log.Infof("Executor starts making prepare packed message: state version %d, tx count %d, self node %s", stateVersion, e.nodeID)
	packedMsgExe, packedMsgProp, err := e.makePreparePackedMsg(vrfProof, txRoot, txsRS.TxRSRoot, packedTxs.StateVersion, packedTxs.TxList, txsRS.TxsResult, compState)
	if err != nil {
		return err
	}
	e.log.Infof("Executor finishes making prepare packed message: state version %d, tx count %d, self node %s", stateVersion, e.nodeID)

	activeExecutors := e.epochService.GetActiveExecutorIDs()

	var wg sync.WaitGroup
	wg.Add(1)
	go func(requiredCount int) {
		recvCount := 1 //Contain self
		defer wg.Done()
		for {
			select {
			case perparePMExeIndic := <-e.preparePackedMsgExeIndicChan:
				recvCount++
				e.log.Infof("Received PreparePackedMessageExeIndication from other executor, state version %d, received %d required %d,  self node %s", perparePMExeIndic.StateVersion, recvCount, requiredCount, e.nodeID)
				if recvCount == requiredCount {
					return
				}
			}
		}
	}(len(activeExecutors))

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = e.deliver.deliverPreparePackagedMessageExe(ctx, packedMsgExe)
		if err != nil {
			e.log.Errorf("Deliver prepare packed message to execute network failed: err=%v", err)
			return
		}
		e.log.Infof("Deliver prepare packed message to execute network successfully: state version %d， self node %s", packedMsgExe.StateVersion, e.nodeID)
	}()

	wg.Wait()

	err = e.deliver.deliverPreparePackagedMessageProp(ctx, packedMsgProp)
	if err != nil {
		e.log.Errorf("Deliver prepare packed message to propose network failed: err=%v", err)
		return err
	}

	e.prepareMsgLunched.ContainsOrAdd(stateVersion, struct{}{})

	e.log.Infof("Deliver prepare packed message to propose network successfully: state version %d， self node %s", packedMsgProp.StateVersion, e.nodeID)

	return nil
}
