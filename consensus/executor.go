package consensus

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type consensusExecutor struct {
	log                          tplog.Logger
	nodeID                       string
	priKey                       tpcrtypes.PrivateKey
	txPool                       txpooli.TransactionPool
	marshaler                    codec.Marshaler
	ledger                       ledger.Ledger
	exeScheduler                 execution.ExecutionScheduler
	deliver                      *messageDeliver
	preparePackedMsgExeChan      chan *PreparePackedMessageExe
	preparePackedMsgExeIndicChan chan *PreparePackedMessageExeIndication
	commitMsgChan                chan *CommitMessage
	cryptService                 tpcrt.CryptService
	prepareInterval              time.Duration
}

func newConsensusExecutor(log tplog.Logger,
	nodeID string,
	priKey tpcrtypes.PrivateKey,
	txPool txpooli.TransactionPool,
	marshaler codec.Marshaler,
	ledger ledger.Ledger,
	exeScheduler execution.ExecutionScheduler,
	deliver *messageDeliver,
	preprePackedMsgExeChan chan *PreparePackedMessageExe,
	preparePackedMsgExeIndicChan chan *PreparePackedMessageExeIndication,
	commitMsgChan chan *CommitMessage,
	cryptService tpcrt.CryptService,
	prepareInterval time.Duration) *consensusExecutor {
	return &consensusExecutor{
		log:                          tplog.CreateSubModuleLogger("executor", log),
		nodeID:                       nodeID,
		priKey:                       priKey,
		txPool:                       txPool,
		marshaler:                    marshaler,
		ledger:                       ledger,
		exeScheduler:                 exeScheduler,
		deliver:                      deliver,
		preparePackedMsgExeChan:      preprePackedMsgExeChan,
		preparePackedMsgExeIndicChan: preparePackedMsgExeIndicChan,
		commitMsgChan:                commitMsgChan,
		cryptService:                 cryptService,
		prepareInterval:              prepareInterval,
	}
}

func (e *consensusExecutor) receivePreparePackedMessageExeStart(ctx context.Context) {
	go func() {
		for {
			select {
			case perparePMExe := <-e.preparePackedMsgExeChan:
				e.log.Infof("Received PreparePackedMessageExe from other executor, state version %d, self node %s", perparePMExe.StateVersion, e.nodeID)

				compState := state.GetStateBuilder().CreateCompositionState(e.log, e.nodeID, e.ledger, perparePMExe.StateVersion, "executor_exereceiver")

				waitCount := 1
				for compState == nil {
					e.log.Warnf("Can't CreateCompositionState when received new PreparePackedMessageExe, StateVersion %d, self node %s, waitCount %d", perparePMExe.StateVersion, e.nodeID, waitCount)
					compState = state.GetStateBuilder().CreateCompositionState(e.log, e.nodeID, e.ledger, perparePMExe.StateVersion, "executor_exereceiver")
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

				var receivedTxList []*txbasic.Transaction
				for i := 0; i < len(perparePMExe.Txs); i++ {
					var tx txbasic.Transaction

					err = e.marshaler.Unmarshal(perparePMExe.Txs[i], &tx)
					if err != nil {
						e.log.Errorf("Invalid tx from pepare packed msg exe: marshal err %v", err)
						break
					}
					receivedTxList = append(receivedTxList, &tx)
				}
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

				_, err = e.exeScheduler.ExecutePackedTx(ctx, txPacked, compState)
				if err != nil {
					e.log.Errorf("Execute packed txs err from remote: %v, state version %d self node %s", err, txPacked.StateVersion, e.nodeID)
				}
				e.log.Infof("Execute Packed tx successfully from remote, state version %d self node %s", perparePMExe.StateVersion, e.nodeID)
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

func (e *consensusExecutor) canPrepare() (bool, []byte, error) {
	if schedulerState := e.exeScheduler.State(); schedulerState != execution.SchedulerState_Idle {
		err := fmt.Errorf("Execution scheduler state %s not idle", schedulerState.String())
		e.log.Errorf("%v", err)
		return false, nil, err
	}

	csStateRN := state.CreateCompositionStateReadonly(e.log, e.ledger)
	defer csStateRN.Stop()

	roleSelector := newLeaderSelectorVRF(e.log, e.cryptService)

	maxStateVersion, _ := e.exeScheduler.MaxStateVersion(e.log, e.ledger)

	candInfo, vrfProof, err := roleSelector.Select(RoleSelector_ExecutionLauncher, maxStateVersion, e.priKey, csStateRN, 1)
	if err != nil {
		e.log.Errorf("Select executor err: %v", err)
		return false, nil, err
	}
	if len(candInfo) != 1 {
		err = fmt.Errorf("Invalid selected executor count: expected 1, actual %d", len(candInfo))
		return false, nil, err
	}

	if candInfo[0].nodeID == e.nodeID {
		return true, vrfProof, nil
	}

	return false, vrfProof, nil
}

func (e *consensusExecutor) prepareTimerStart(ctx context.Context) {
	e.log.Infof("Executor %s prepareTimerStart, timer=%fs", e.nodeID, e.prepareInterval.Seconds())

	go func() {
		timer := time.NewTicker(e.prepareInterval)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				isCan, vrfProof, _ := e.canPrepare()
				if isCan {
					e.log.Infof("Selected execution launcher %s can prepare", e.nodeID)
					e.Prepare(ctx, vrfProof)
				}
			case <-ctx.Done():
				e.log.Info("Consensus executor exit prepare timre")
				return
			}
		}
	}()
}

func (e *consensusExecutor) start(ctx context.Context) {
	e.receivePreparePackedMessageExeStart(ctx)
	e.prepareTimerStart(ctx)
	e.receiveCommitMsgStart(ctx)
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

	for i := 0; i < len(txList); i++ {
		txBytes, _ := e.marshaler.Marshal(txList[i])
		exePPM.Txs = append(exePPM.Txs, txBytes)

		txHashBytes, _ := txList[i].HashBytes()
		txRSHashBytes, _ := txResultList[i].HashBytes()
		proposePPM.TxHashs = append(proposePPM.TxHashs, txHashBytes)
		proposePPM.TxResultHashs = append(proposePPM.TxResultHashs, txRSHashBytes)
	}

	return exePPM, proposePPM, nil
}

func (e *consensusExecutor) Prepare(ctx context.Context, vrfProof []byte) error {
	pendTxs, err := e.txPool.Pending()
	if err != nil {
		e.log.Errorf("Can't get pending txs: %v", err)
		return err
	}

	if len(pendTxs) == 0 {
		e.log.Infof("Current pending txs'size 0")
		return nil
	}

	txRoot := txbasic.TxRoot(pendTxs)
	maxStateVer, err := e.exeScheduler.MaxStateVersion(e.log, e.ledger)
	if err != nil {
		e.log.Errorf("Can't get max state version: %v", err)
		return err
	}

	compState := state.GetStateBuilder().CreateCompositionState(e.log, e.nodeID, e.ledger, maxStateVer+1, "executor_exepreparer")
	if compState == nil {
		err = fmt.Errorf("Can't CreateCompositionState for Prepare: maxStateVer=%d", maxStateVer+1)
		e.log.Errorf("%v", err)
		return err
	}

	var packedTxs execution.PackedTxs

	packedTxs.StateVersion = maxStateVer + 1
	packedTxs.TxRoot = txRoot
	packedTxs.TxList = append(packedTxs.TxList, pendTxs...)

	txsRS, err := e.exeScheduler.ExecutePackedTx(ctx, &packedTxs, compState)
	if err != nil {
		e.log.Errorf("Execute state version %d packed txs err from local: %v", packedTxs.StateVersion, err)
		return err
	}
	txRSRoot := txbasic.TxResultRoot(txsRS.TxsResult, packedTxs.TxList)

	packedMsgExe, packedMsgProp, err := e.makePreparePackedMsg(vrfProof, txRoot, txRSRoot, packedTxs.StateVersion, packedTxs.TxList, txsRS.TxsResult, compState)
	if err != nil {
		return err
	}

	activeExecutors, _ := compState.GetActiveExecutorIDs()

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

	e.log.Infof("Deliver prepare packed message to propose network successfully: state version %d， self node %s", packedMsgProp.StateVersion, e.nodeID)

	return nil
}
