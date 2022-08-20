package consensus

import (
	"container/list"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/consensus/vrf"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
	lru "github.com/hashicorp/golang-lru"
)

const defaultLeaderCount = int(3)

var (
	notReachVoteThresholdErr = errors.New("Haven't reached threshold at present")
)

type PreparePackedMessagePropMap struct {
	StateVersion uint64
	PPMProps     map[string]*PreparePackedMessageProp //domainID->PreparePackedMessageProp
}

type consensusProposer struct {
	log                     tplog.Logger
	nodeID                  string
	priKey                  tpcrtypes.PrivateKey
	blockAddedCh            chan *tpchaintypes.Block
	preprePackedMsgPropChan chan *PreparePackedMessageProp
	voteMsgChan             chan *VoteMessage
	deliver                 messageDeliverI
	marshaler               codec.Marshaler
	ledger                  ledger.Ledger
	cryptService            tpcrt.CryptService
	epochService            EpochService
	dkgBls                  DKGBls
	proposeMaxInterval      time.Duration
	blockMaxCyclePeriod     time.Duration
	isProposing             uint32
	lastBlockHeight         uint64
	lastProposeHeight       uint64     //the block height which self-proposer launched based on last time
	newHeightStartTimeStamp time.Time  // the timestamp of new block height
	proposedRecord          *lru.Cache //height->context.CancelFunc
	votedRecord             *lru.Cache //Key: StateVersion@Voter
	syncPPMPropList         sync.RWMutex
	ppmPropList             *list.List
	validator               *consensusValidator
	voteCollector           *consensusVoteCollector
}

func newConsensusProposer(log tplog.Logger,
	nodeID string,
	priKey tpcrtypes.PrivateKey,
	preprePackedMsgPropChan chan *PreparePackedMessageProp,
	voteMsgChan chan *VoteMessage,
	blockAddedCh chan *tpchaintypes.Block,
	crypt tpcrt.CryptService,
	epochService EpochService,
	proposeMaxInterval time.Duration,
	blockMaxCyclePeriod time.Duration,
	deliver messageDeliverI,
	ledger ledger.Ledger,
	marshaler codec.Marshaler,
	validator *consensusValidator) *consensusProposer {

	proposedRecord, _ := lru.New(10)
	votedRecord, _ := lru.New(200)

	csProposer := &consensusProposer{
		log:                     tplog.CreateSubModuleLogger("proposer", log),
		nodeID:                  nodeID,
		priKey:                  priKey,
		blockAddedCh:            blockAddedCh,
		preprePackedMsgPropChan: preprePackedMsgPropChan,
		voteMsgChan:             voteMsgChan,
		deliver:                 deliver,
		marshaler:               marshaler,
		ledger:                  ledger,
		cryptService:            crypt,
		epochService:            epochService,
		proposeMaxInterval:      proposeMaxInterval,
		blockMaxCyclePeriod:     blockMaxCyclePeriod,
		isProposing:             0,
		proposedRecord:          proposedRecord,
		votedRecord:             votedRecord,
		ppmPropList:             list.New(),
		validator:               validator,
	}

	return csProposer
}

func (p *consensusProposer) updateDKGBls(dkgBls DKGBls) {
	p.dkgBls = dkgBls
}

func (p *consensusProposer) getVrfInputData(block *tpchaintypes.Block, proposeHeight uint64) ([]byte, error) {
	hasher := tpcmm.NewBlake2bHasher(0)

	if err := binary.Write(hasher.Writer(), binary.BigEndian, block.Head.Epoch); err != nil {
		return nil, err
	}
	if err := binary.Write(hasher.Writer(), binary.BigEndian, proposeHeight); err != nil {
		return nil, err
	}

	csProof := &tpchaintypes.ConsensusProof{
		ParentBlockHash: block.Head.ParentBlockHash,
		Height:          block.Head.Height,
		AggSign:         block.Head.VoteAggSignature,
	}
	csProofBytes, err := csProof.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err = hasher.Writer().Write(csProofBytes); err != nil {
		return nil, err
	}

	return hasher.Bytes(), nil
}

func (p *consensusProposer) canProposeBlock(latestBlock *tpchaintypes.Block, proposeHeight uint64) (bool, []byte, []byte, error) {
	proposerSel := vrf.NewProposerSelector(vrf.ProposerSelectionType_Poiss, p.cryptService)

	vrfData, err := p.getVrfInputData(latestBlock, proposeHeight)
	if err != nil {
		p.log.Errorf("Can't get proposer selector vrf data: %v", err)
		return false, nil, nil, err
	}

	vrfProof, err := proposerSel.ComputeVRF(p.priKey, vrfData)
	if err != nil {
		p.log.Errorf("Compute proposer selector vrf err: %v", err)
		return false, nil, nil, err
	}

	totalActiveProposerWeight := p.epochService.GetActiveProposersTotalWeight()

	localNodeWeight, err := p.epochService.GetNodeWeight(p.nodeID)
	if err != nil {
		p.log.Errorf("Can't get local proposer weight: %v", err)
		return false, nil, nil, err
	}

	winCount := proposerSel.SelectProposer(vrfProof, big.NewInt(int64(localNodeWeight)), big.NewInt(int64(totalActiveProposerWeight)))
	p.log.Infof("Propose block based on the latest height %d: propose height %d, winCount %d, weight proportion %d/%d, self node=%s", latestBlock.Head.Height, proposeHeight, winCount, localNodeWeight, totalActiveProposerWeight, p.nodeID)
	if winCount >= 1 {
		maxPri := proposerSel.MaxPriority(vrfProof, winCount)

		bestPri, err := p.validator.judgeLocalMaxPriBestForProposer(maxPri, latestBlock)
		if !bestPri {
			return false, nil, nil, err
		}

		return true, vrfProof, maxPri, nil
	}

	return false, nil, nil, fmt.Errorf("Can't propose block at the epoch: winCount=%d", winCount)
}

func (p *consensusProposer) receivePreparePackedMessagePropStart(ctx context.Context) {
	go func() {
		for {
			select {
			case ppmProp := <-p.preprePackedMsgPropChan:
				if !p.epochService.SelfSelected() {
					p.log.Warnf("Not selected consensus node and should not receive prepare message from remote, state version %d self node %s", ppmProp.StateVersion, p.nodeID)
					continue
				}
				p.log.Infof("Received prepare message from remote, state version %d self node %s", ppmProp.StateVersion, p.nodeID)
				err := func() error {
					p.syncPPMPropList.Lock()
					defer p.syncPPMPropList.Unlock()

					latestBlock, err := state.GetLatestBlock(p.ledger)
					if err != nil {
						p.log.Errorf("Can't get the latest bock when receiving prepare packed msg prop: %v", err)
						return err
					}
					if ppmProp.StateVersion <= latestBlock.Head.Height {
						err = fmt.Errorf("Received outdated prepare packed msg prop: StateVersion=%d, latest block height=%d", ppmProp.StateVersion, latestBlock.Head.Height)
						p.log.Errorf("%v", err)
						return err
					}

					domainID := string(ppmProp.DomainID)

					if p.ppmPropList.Len() > 0 {
						latestPPMPropMap := p.ppmPropList.Back().Value.(*PreparePackedMessagePropMap)
						if ppmProp.StateVersion == latestPPMPropMap.StateVersion {
							if _, ok := latestPPMPropMap.PPMProps[domainID]; !ok {
								latestPPMPropMap.PPMProps[domainID] = ppmProp
							} else {
								err = fmt.Errorf("Received the same prepare packed msg prop: DomainID=%s, StateVersion=%d, latest block height=%d", domainID, ppmProp.StateVersion, latestBlock.Head.Height)
								return err
							}
						} else if ppmProp.StateVersion == latestPPMPropMap.StateVersion+1 {
							ppmPropMap := &PreparePackedMessagePropMap{
								StateVersion: ppmProp.StateVersion,
								PPMProps:     map[string]*PreparePackedMessageProp{domainID: ppmProp},
							}
							p.ppmPropList.PushBack(ppmPropMap)
						} else {
							err = fmt.Errorf("Received invalid prepare packed msg prop: expected state version %d, actual %d", latestPPMPropMap.StateVersion+1, ppmProp.StateVersion)
							p.log.Errorf("%v", err)
							return err
						}
					} else {
						ppmPropMap := &PreparePackedMessagePropMap{
							StateVersion: ppmProp.StateVersion,
							PPMProps:     map[string]*PreparePackedMessageProp{string(ppmProp.DomainID): ppmProp},
						}
						p.ppmPropList.PushBack(ppmPropMap)
					}

					return nil
				}()

				if err != nil {
					continue
				}
			case <-ctx.Done():
				p.log.Info("Consensus proposer receiveing prepare packed msg prop exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) produceCommitMsg(msg *VoteMessage, aggSign []byte) (*CommitMessage, *tpchaintypes.BlockHead, error) {
	var blockHead tpchaintypes.BlockHead
	err := p.marshaler.Unmarshal(msg.BlockHead, &blockHead)
	if err != nil {
		p.log.Errorf("Unmarshal block failed: %v", err)
		return nil, nil, err
	}

	pubKey, err := p.dkgBls.PubKey()
	if err != nil {
		p.log.Errorf("Fetch dkg public key failed: %v", err)
		return nil, nil, err
	}
	signInfo := tpcrtypes.SignatureInfo{
		PublicKey: pubKey,
		SignData:  aggSign,
	}
	blockHead.VoteAggSignature, _ = json.Marshal(&signInfo)

	newBlockHeadBytes, err := p.marshaler.Marshal(&blockHead)
	if err != nil {
		p.log.Errorf("Marshal block head failed: %v", err)
		return nil, nil, err
	}

	return &CommitMessage{
		ChainID:      msg.ChainID,
		Version:      msg.Version,
		Epoch:        msg.Epoch,
		Round:        msg.Round,
		StateVersion: msg.StateVersion,
		BlockHead:    newBlockHeadBytes,
	}, &blockHead, nil
}

func (p *consensusProposer) aggSignDisposeWhenRecvVoteMsg(ctx context.Context, voteMsg *VoteMessage, cancel context.CancelFunc) error {
	aggSign, err := p.voteCollector.tryAggregateSignAndAddVote(p.nodeID, voteMsg)
	if err != nil {
		p.log.Errorf("Try to aggregate sign and add vote faild: err=%v", err)
		return err
	}

	if aggSign == nil {
		p.log.Debugf("%s", notReachVoteThresholdErr.Error())
		return notReachVoteThresholdErr
	}

	commitMsg, bh, err := p.produceCommitMsg(voteMsg, aggSign)
	if err != nil {
		p.log.Errorf("Can't produce commit message: %v", err)
		return err
	}

	if p.validator.commitMsg.Contains(commitMsg.StateVersion) {
		p.log.Warnf("Have received commit message and not need to commit message again: state version %d, self node %s", voteMsg.StateVersion, p.nodeID)
		if cancel != nil {
			cancel()
		}
		return nil
	} else {
		p.validator.commitMsg.Add(commitMsg.StateVersion, struct{}{})
	}

	err = p.deliver.deliverCommitMessage(ctx, commitMsg)
	if err != nil {
		p.log.Errorf("Can't deliver commit message: %v", err)
		return err
	}

	if cancel != nil {
		cancel()
	}

	p.validator.commitMsgDispose(ctx, bh, commitMsg)

	p.log.Infof("Deliver commit message successfully, state version %d self node %s", voteMsg.StateVersion, p.nodeID)

	return nil
}

func (p *consensusProposer) setIfNotVoted(voteMsg *VoteMessage) bool {
	vKey := strings.Join([]string{strconv.FormatUint(voteMsg.StateVersion, 10),
		string(voteMsg.Voter),
	}, "@")

	ok, _ := p.votedRecord.ContainsOrAdd(vKey, struct{}{})

	return ok
}

func (p *consensusProposer) receiveVoteMessageStart(ctx context.Context) {
	go func() {
		for {
			select {
			case voteMsg := <-p.voteMsgChan:
				if !p.epochService.SelfSelected() {
					p.log.Warnf("Not selected consensus node and should not receive vote message from remote, state version %d self node %s", voteMsg.StateVersion, p.nodeID)
					continue
				}
				if p.validator.commitMsg.Contains(voteMsg.StateVersion) {
					p.log.Infof("Have received commit message and the vote message will be discarded: state version %d, self node %s", voteMsg.StateVersion, p.nodeID)
					continue
				}

				if p.setIfNotVoted(voteMsg) {
					p.log.Infof("Have received same vote and the vote message will be discarded: state version %d, voter %s, self node %s", voteMsg.StateVersion, string(voteMsg.Voter), p.nodeID)
					continue
				}
				p.log.Infof("Received vote message, state version %d self node %s", voteMsg.StateVersion, p.nodeID)

				if voteMsg.StateVersion != (p.voteCollector.latestHeight + 1) {
					p.log.Errorf("Received invalid vote message, state version %d, latest height %d, self node %s", voteMsg.StateVersion, p.voteCollector.latestHeight, p.nodeID)
					continue
				}

				err := p.dkgBls.Verify(voteMsg.BlockHead, voteMsg.Signature)
				if err != nil {
					p.log.Errorf("Received invalid vote message, state version %d, latest height %d, err %v, self node %s", voteMsg.StateVersion, p.voteCollector.latestHeight, err, p.nodeID)
					continue
				}

				cancel := context.CancelFunc(nil)
				cancelI, ok := p.proposedRecord.Get(p.lastBlockHeight)
				if ok {
					cancel = cancelI.(context.CancelFunc)
				}
				err = p.aggSignDisposeWhenRecvVoteMsg(ctx, voteMsg, cancel)
				if err != nil {
					continue
				}

				p.voteCollector.reset()
			case <-ctx.Done():
				p.log.Info("Consensus proposer receiveing prepare packed msg prop exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) bestProposeMsgTimerStart(ctx context.Context) context.CancelFunc {
	ctxControl, cancel := context.WithCancel(context.Background())
	go func(ctxSub context.Context) {
		p.log.Infof("Begin bestProposeMsgTimerStart, self node %s", p.nodeID)
		timer := time.NewTimer(3 * p.blockMaxCyclePeriod)
		defer timer.Stop()

		select {
		case <-timer.C:
			p.log.Infof("BestProposeMsgTimerStart timeout, self node %s", p.nodeID)
			err := p.validator.broadcastBestProposeMsg(ctx)
			if err != nil {
				p.log.Errorf("Broadcast best propose message err: %v, self node %s", err, p.nodeID)
				return
			}
		case <-ctxSub.Done():
			p.log.Infof("BestProposeMsgTimerStart end: self node %s", p.nodeID)
		}
	}(ctxControl)

	return cancel
}

func (p *consensusProposer) proposeBlockSpecification(ctx context.Context, addedBlock *tpchaintypes.Block) error {
	if !p.epochService.SelfSelected() {
		return fmt.Errorf("Not selected consensus node: self node %s", p.nodeID)
	}

	if !atomic.CompareAndSwapUint32(&p.isProposing, 0, 1) {
		err := fmt.Errorf("Self node %s is proposing", p.nodeID)
		p.log.Infof("%s", err.Error())
		return err
	}
	defer func() {
		p.isProposing = 0
	}()

	csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
	defer csStateRN.Stop()

	latestBlock := addedBlock

	latestEpoch, err := state.GetLatestEpoch(p.ledger)
	if err != nil {
		p.log.Errorf("Can't get the latest epoch: %v", err)
		return err
	}

	if latestBlock == nil {
		latestBlock, err = state.GetLatestBlock(p.ledger)
		if err != nil {
			p.log.Errorf("Can't get the latest block: %v", err)
			return err
		}
	}

	if p.lastProposeHeight == 0 || latestBlock.Head.Height > p.lastBlockHeight {
		p.lastBlockHeight = latestBlock.Head.Height
		p.lastProposeHeight = latestBlock.Head.Height
		p.newHeightStartTimeStamp = time.Now()

		if p.lastBlockHeight > 1 {
			if cancelI, ok := p.proposedRecord.Get(p.lastBlockHeight - 1); ok {
				cancel := cancelI.(context.CancelFunc)
				cancel()
			}
		}
	}

	if ok := p.proposedRecord.Contains(latestBlock.Head.Height); ok {
		err = fmt.Errorf("Have sucessfully proposed based on height %d, self node %s", latestBlock.Head.Height, p.nodeID)
		p.log.Warnf("%s", err.Error())
		if time.Since(p.newHeightStartTimeStamp).Milliseconds() > 60*p.blockMaxCyclePeriod.Milliseconds() {
			p.log.Warn("Must be time out")
		}
		return err
	}

	var proposeHeightNew uint64
	elapsedTime := time.Since(p.newHeightStartTimeStamp).Milliseconds()
	deltaHeight := uint64(elapsedTime)/uint64(p.blockMaxCyclePeriod.Milliseconds()) + 1

	proposeHeightNew = latestBlock.Head.Height + deltaHeight

	if p.lastProposeHeight > 0 && p.lastProposeHeight >= proposeHeightNew {
		stateVerPPMProp, lenPPMProp := p.currentPPMProp()
		err = fmt.Errorf("Have launched proposing block at propose height %d, latest height %d, last ProposeTimeStamp %s, ppmProp: stateVer %d,len %d，self node %s", p.lastProposeHeight, latestBlock.Head.Height, p.newHeightStartTimeStamp.String(), stateVerPPMProp, lenPPMProp, p.nodeID)
		p.log.Warnf("%s", err.Error())
		return err
	}

	p.lastProposeHeight = proposeHeightNew

	if p.validator.existProposeMsg(csStateRN.ChainID(), latestBlock.Head.Height, latestBlock.Head.Height+1, p.nodeID) {
		err = fmt.Errorf("Have existed proposed block: state version %d, latest height %d, self node %s", latestBlock.Head.Height+1, latestBlock.Head.Height, p.nodeID)
		p.log.Warnf("%s", err.Error())
		return err
	}

	canPropose, vrfProof, maxPri, err := p.canProposeBlock(latestBlock, proposeHeightNew)
	if !canPropose {
		err = fmt.Errorf("Can't propose block at the epoch : latest epoch=%d, latest height=%d, self node=%s, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, p.nodeID, err)
		p.log.Infof("%s", err.Error())
		return err
	}

	stateRoot, err := csStateRN.StateRoot()
	if err != nil {
		p.log.Errorf("Can't get state root: %v", err)
		return err
	}

	var pppPropMap *PreparePackedMessagePropMap
	for pppPropMap == nil {
		pppPropMap, err = p.getAvailPPMProp(latestBlock.Head.Height)
		if err != nil {
			//p.log.Warnf("%s", err.Error())
			time.Sleep(50 * time.Millisecond)
		} else {
			break
		}
	}
	p.log.Infof("Avail PPM prop state version %d, self node %s", pppPropMap.StateVersion, p.nodeID)

	proposeBlock, err := p.produceProposeBlock(csStateRN.ChainID(), latestEpoch, latestBlock, pppPropMap, vrfProof, maxPri, stateRoot, proposeHeightNew)
	if err != nil {
		p.log.Errorf("Produce propose block error: latest epoch=%d, latest height=%d, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, err)
		return err
	}

	if ok, _ := p.validator.validateAndCollectProposeMsg(ctx, maxPri, p.nodeID, proposeBlock); !ok {
		err = fmt.Errorf("Can't delive propose message: latest epoch=%d,latest height=%d", latestBlock.Head.Epoch, latestBlock.Head.Height)
		p.log.Infof("%s", err.Error())
		return err
	}

	waitCount := 1
	for !p.deliver.isReady() && waitCount <= 10 {
		p.log.Warnf("Message deliver not ready now for delivering propose message, wait 50ms, no. %d", waitCount)
		time.Sleep(50 * time.Millisecond)
		waitCount++
	}
	if waitCount > 10 {
		err = fmt.Errorf("Finally nil dkgBls and can't deliver propose message, self node %s", p.nodeID)
		p.log.Errorf("%v", err)
		return err
	}

	p.log.Infof("Message deliver ready:state version %d, propose Block height %d, self node %s", proposeBlock.StateVersion, latestBlock.Head.Height+1, p.nodeID)

	p.voteCollector = newConsensusVoteCollector(p.log, latestBlock.Head.Height, p.dkgBls)

	voteMsg, err := p.validator.produceVoteMsg(proposeBlock)
	if err != nil {
		p.log.Errorf("Can't produce vote msg: err=%v, self node %s", err, p.nodeID)
		return err
	}
	sigData, pubKey, err := p.dkgBls.Sign(voteMsg.BlockHead)
	if err != nil {
		p.log.Errorf("DKG sign VoteMessage err: %v, self node %s", err, p.nodeID)
		return err
	}
	voteMsg.Signature = sigData
	voteMsg.PubKey = pubKey

	cancel := context.CancelFunc(nil)
	cancelI, ok := p.proposedRecord.Get(p.lastBlockHeight)
	if ok {
		cancel = cancelI.(context.CancelFunc)
	}
	err = p.aggSignDisposeWhenRecvVoteMsg(ctx, voteMsg, cancel)
	if err != nil && err != notReachVoteThresholdErr {
		return err
	}

	elapsedTime = time.Since(p.newHeightStartTimeStamp).Milliseconds()
	if elapsedTime < 500 {
		p.log.Infof("Need sleep before delivering proposed block at the epoch : sleep time %d ms, latest epoch=%d, latest height=%d, self node=%s", 500-elapsedTime, latestBlock.Head.Epoch, latestBlock.Head.Height, p.nodeID, err)
		time.Sleep(time.Duration(500-elapsedTime) * time.Millisecond)
	}

	if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
		p.log.Errorf("Deliver propose message err: latest epoch =%d, latest height=%d, self node=%s, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, p.nodeID, err)
		return err
	}

	p.proposedRecord.Add(latestBlock.Head.Height, p.bestProposeMsgTimerStart(ctx))

	p.log.Infof("Propose block successfully: state version %d, propose Block height %d, self node %s", proposeBlock.StateVersion, latestBlock.Head.Height+1, p.nodeID)

	return nil
}

func (p *consensusProposer) proposeBlockStart(ctx context.Context) {
	go func() {
		timer := time.NewTicker(p.blockMaxCyclePeriod)
		defer timer.Stop()
		for !p.deliver.isReady() {
			time.Sleep(50 * time.Millisecond)
		}
		p.log.Infof("Proposer message deliver ready: self node %s", p.nodeID)

		for {
			select {
			case addedBlock := <-p.blockAddedCh:
				p.log.Infof("Proposer receives block added event: height %d, self node %s", addedBlock.Head.Height, p.nodeID)

				if err := p.proposeBlockSpecification(ctx, addedBlock); err != nil {
					continue
				}
			case <-timer.C:
				p.log.Infof("Proposer propose time out: self node %s", p.nodeID)
				if err := p.proposeBlockSpecification(ctx, nil); err != nil {
					continue
				}
			case <-ctx.Done():
				p.log.Info("Consensus proposer epoch exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) start(ctx context.Context) {
	p.receivePreparePackedMessagePropStart(ctx)

	p.receiveVoteMessageStart(ctx)

	p.proposeBlockStart(ctx)
}

func (p *consensusProposer) currentPPMProp() (uint64, int) {
	p.syncPPMPropList.RLock()
	defer p.syncPPMPropList.RUnlock()

	frontEle := p.ppmPropList.Front()
	if frontEle != nil {
		frontPPMProp := frontEle.Value.(*PreparePackedMessagePropMap)
		return frontPPMProp.StateVersion, p.ppmPropList.Len()
	}

	return 0, 0
}

func (p *consensusProposer) getAvailPPMProp(latestHeight uint64) (*PreparePackedMessagePropMap, error) {
	p.syncPPMPropList.RLock()
	defer p.syncPPMPropList.RUnlock()

	var frontPPMPropMap *PreparePackedMessagePropMap
	for p.ppmPropList.Len() > 0 {
		frontEle := p.ppmPropList.Front()
		frontPPMPropMap = frontEle.Value.(*PreparePackedMessagePropMap)
		if frontPPMPropMap.StateVersion == latestHeight+1 {
			break
		} else if frontPPMPropMap.StateVersion < latestHeight+1 {
			p.ppmPropList.Remove(frontEle)
		}
	}
	if frontPPMPropMap == nil || frontPPMPropMap.StateVersion == latestHeight {
		err := fmt.Errorf("Can't get prepare packed message prop: there is no expected state version %d, total PPM prop %d, self node=%s", latestHeight+1, p.ppmPropList.Len(), p.nodeID)
		return nil, err
	}

	return frontPPMPropMap, nil
}

func (p *consensusProposer) produceProposeBlock(chainID tpchaintypes.ChainID,
	latestEpoch *tpcmm.EpochInfo,
	latestBlock *tpchaintypes.Block,
	ppmPropMap *PreparePackedMessagePropMap,
	vrfProof []byte,
	maxPri []byte,
	stateRoot []byte,
	proposeHeight uint64) (*ProposeMessage, error) {
	blockHashBytes, err := latestBlock.HashBytes()
	if err != nil {
		p.log.Errorf("Can't get the hash bytes of block height %d: %v", latestBlock.Head.Height, err)
		return nil, err
	}

	csProof := &tpchaintypes.ConsensusProof{
		ParentBlockHash: latestBlock.Head.ParentBlockHash,
		Height:          latestBlock.Head.Height,
		AggSign:         latestBlock.Head.VoteAggSignature,
	}

	csProofBytes, err := p.marshaler.Marshal(csProof)
	if err != nil {
		p.log.Errorf("Marshal consensus proof failed: %v", err)
		return nil, err
	}

	var bhChunksBytes [][]byte
	var propDatasBytes [][]byte
	for domainID, ppmProp := range ppmPropMap.PPMProps {
		bhChunk := &tpchaintypes.BlockHeadChunk{
			Version:      tpchaintypes.BLOCK_VER,
			DomainID:     []byte(domainID),
			Launcher:     ppmProp.Launcher,
			TxCount:      uint32(len(ppmProp.TxHashs)),
			TxRoot:       ppmProp.TxRoot,
			TxResultRoot: ppmProp.TxResultRoot,
		}

		propData := &PropData{
			Version:       tpchaintypes.BLOCK_VER,
			DomainID:      []byte(domainID),
			TxHashs:       ppmProp.TxHashs,
			TxResultHashs: ppmProp.TxResultHashs,
		}

		bhChunkBytes, err := bhChunk.Marshal()
		if err != nil {
			p.log.Errorf("Marshal block head chunk failed: %v, DomainID=%s, Launcher %s, state Version %d, self node %s", err, domainID, string(ppmProp.Launcher), ppmPropMap.StateVersion, p.nodeID)
			continue
		}
		propDataBytes, err := propData.Marshal()
		if err != nil {
			p.log.Errorf("Marshal propose data failed: %v, DomainID=%s, Launcher %s, state Version %d, self node %s", err, domainID, string(ppmProp.Launcher), ppmPropMap.StateVersion, p.nodeID)
			continue
		}

		bhChunksBytes = append(bhChunksBytes, bhChunkBytes)
		propDatasBytes = append(propDatasBytes, propDataBytes)
	}

	if len(bhChunksBytes) == 0 {
		err := fmt.Errorf("Block head chunk size 0: state Version %d, self node %s", ppmPropMap.StateVersion, p.nodeID)
		p.log.Errorf("%v", err)
		return nil, err
	}

	bh := &tpchaintypes.BlockHead{
		ChainID:         []byte(chainID),
		Version:         tpchaintypes.BLOCK_VER,
		Height:          latestBlock.Head.Height + 1,
		Epoch:           latestEpoch.Epoch,
		Round:           latestBlock.Head.Height,
		ParentBlockHash: blockHashBytes,
		Proposer:        []byte(p.nodeID),
		Proof:           csProofBytes,
		VRFProof:        vrfProof,
		VRFProofHeight:  proposeHeight,
		MaxPri:          maxPri,
		ChunkCount:      uint32(len(bhChunksBytes)),
		StateRoot:       stateRoot,
		TimeStamp:       uint64(time.Now().UnixNano()),
	}
	bh.HeadChunks = append(bh.HeadChunks, bhChunksBytes...)

	newBlockHeadBytes, err := p.marshaler.Marshal(bh)
	if err != nil {
		p.log.Errorf("Marshal block head failed: %v", err)
		return nil, err
	}

	return &ProposeMessage{
		ChainID:      []byte(chainID),
		Version:      CONSENSUS_VER,
		Epoch:        latestEpoch.Epoch,
		Round:        latestBlock.Head.Height,
		StateVersion: ppmPropMap.StateVersion,
		MaxPri:       bh.MaxPri,
		Proposer:     bh.Proposer,
		PropDatas:    propDatasBytes,
		BlockHead:    newBlockHeadBytes,
	}, nil

}
