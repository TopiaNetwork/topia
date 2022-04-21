package consensus

import (
	"container/list"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

const defaultLeaderCount = int(3)

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
	dkgBls                  DKGBls
	proposeMaxInterval      time.Duration
	blockMaxCyclePeriod     time.Duration
	isProposing             uint32
	lastProposeHeight       uint64 //the block height which self-proposer launched based on last time
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
	proposeMaxInterval time.Duration,
	blockMaxCyclePeriod time.Duration,
	deliver messageDeliverI,
	ledger ledger.Ledger,
	marshaler codec.Marshaler,
	validator *consensusValidator) *consensusProposer {
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
		proposeMaxInterval:      proposeMaxInterval,
		blockMaxCyclePeriod:     blockMaxCyclePeriod,
		isProposing:             0,
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

	csProof := &ConsensusProof{
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

func (p *consensusProposer) canProposeBlock(csStateRN state.CompositionStateReadonly, latestBlock *tpchaintypes.Block, proposeHeight uint64) (bool, []byte, []byte, error) {
	proposerSel := NewProposerSelector(ProposerSelectionType_Poiss, p.cryptService)

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

	totalActiveProposerWeight, err := csStateRN.GetActiveProposersTotalWeight()
	if err != nil {
		p.log.Errorf("Can't get total active proposer weight: %v", err)
		return false, nil, nil, err
	}
	localNodeWeight, err := csStateRN.GetNodeWeight(p.nodeID)
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
				p.log.Infof("Received prepare message from remote, state version %d self node %s", ppmProp.StateVersion, p.nodeID)
				err := func() error {
					csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
					defer csStateRN.Stop()

					p.syncPPMPropList.Lock()
					defer p.syncPPMPropList.Unlock()

					latestBlock, err := csStateRN.GetLatestBlock()
					if err != nil {
						p.log.Errorf("Can't get the latest bock when receiving prepare packed msg prop: %v", err)
						return err
					}
					if ppmProp.StateVersion <= latestBlock.Head.Height {
						err = fmt.Errorf("Received outdated prepare packed msg prop: StateVersion=%d, latest block height=%d", ppmProp.StateVersion, latestBlock.Head.Height)
						p.log.Errorf("%v", err)
						return err
					}

					if p.ppmPropList.Len() > 0 {
						latestPPMProp := p.ppmPropList.Back().Value.(*PreparePackedMessageProp)
						if ppmProp.StateVersion != latestPPMProp.StateVersion+1 {
							err = fmt.Errorf("Received invalid prepare packed msg prop: expected state version %d, actual %d", latestPPMProp.StateVersion+1, ppmProp.StateVersion)
							p.log.Errorf("%v", err)
							return err
						}
					}

					p.ppmPropList.PushBack(ppmProp)

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

func (p *consensusProposer) produceCommitMsg(msg *VoteMessage, aggSign []byte) (*CommitMessage, error) {
	var blockHead tpchaintypes.BlockHead
	err := p.marshaler.Unmarshal(msg.BlockHead, &blockHead)
	if err != nil {
		p.log.Errorf("Unmarshal block failed: %v", err)
		return nil, err
	}
	blockHead.VoteAggSignature = aggSign

	newBlockHeadBytes, err := p.marshaler.Marshal(&blockHead)
	if err != nil {
		p.log.Errorf("Marshal block head failed: %v", err)
		return nil, err
	}

	return &CommitMessage{
		ChainID:      msg.ChainID,
		Version:      msg.Version,
		Epoch:        msg.Epoch,
		Round:        msg.Round,
		StateVersion: msg.StateVersion,
		BlockHead:    newBlockHeadBytes,
	}, nil
}

func (p *consensusProposer) receiveVoteMessagStart(ctx context.Context) {
	go func() {
		for {
			select {
			case voteMsg := <-p.voteMsgChan:
				p.log.Infof("Received vote message, state version %d self node %s", voteMsg.StateVersion, p.nodeID)

				if voteMsg.StateVersion != (p.voteCollector.latestHeight + 1) {
					p.log.Errorf("Received invalid vote message, state version %d, latest height %d, self node %s", voteMsg.StateVersion, p.voteCollector.latestHeight, p.nodeID)
					continue
				}

				aggSign, err := p.voteCollector.tryAggregateSignAndAddVote(voteMsg)
				if err != nil {
					p.log.Errorf("Try to aggregate sign and add vote faild: err=%v", err)
					continue
				}

				if aggSign == nil {
					p.log.Debug("Haven't reached threshold at present")
					continue
				}

				commitMsg, err := p.produceCommitMsg(voteMsg, aggSign)
				if err != nil {
					p.log.Errorf("Can't produce commit message: %v", err)
					continue
				}

				err = p.deliver.deliverCommitMessage(ctx, commitMsg)
				if err != nil {
					p.log.Errorf("Can't deliver commit message: %v", err)
					continue
				}

				p.log.Infof("Deliver commit message successfully, state version %d self node %s", voteMsg.StateVersion, p.nodeID)

				p.voteCollector.reset()

			case <-ctx.Done():
				p.log.Info("Consensus proposer receiveing prepare packed msg prop exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) proposeBlockSpecification(ctx context.Context, addedBlock *tpchaintypes.Block) error {
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

	latestEpoch, err := csStateRN.GetLatestEpoch()
	if err != nil {
		p.log.Errorf("Can't get the latest epoch: %v", err)
		return err
	}

	if latestBlock == nil {
		latestBlock, err = csStateRN.GetLatestBlock()
		if err != nil {
			p.log.Errorf("Can't get the latest block: %v", err)
			return err
		}
	}

	var proposeHeightNew uint64
	blTime := time.Unix(0, int64(latestBlock.Head.TimeStamp))
	elapsedTime := time.Since(blTime).Microseconds()
	deltaHeight := uint64(elapsedTime)/uint64(p.blockMaxCyclePeriod.Microseconds()) + 1
	if latestBlock.Head.Height > 1 {
		proposeHeightNew = latestBlock.Head.Height + deltaHeight
	} else {
		proposeHeightNew = 2
	}

	if p.lastProposeHeight > 0 && p.lastProposeHeight >= proposeHeightNew {
		err = fmt.Errorf("Have launched proposing block at propose height %d, latest height %d", p.lastProposeHeight, latestBlock.Head.Height)
		p.log.Warnf("%v", err)
		return err
	}

	pppProp, err := p.getAvailPPMProp(latestBlock.Head.Height)
	if err != nil {
		p.log.Errorf("%v", err)
		return err
	}

	p.log.Infof("Avail PPM prop state version %d", pppProp.StateVersion)

	p.lastProposeHeight = proposeHeightNew

	canPropose, vrfProof, maxPri, err := p.canProposeBlock(csStateRN, latestBlock, proposeHeightNew)
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
	proposeBlock, err := p.produceProposeBlock(csStateRN.ChainID(), latestEpoch, latestBlock, pppProp, vrfProof, maxPri, stateRoot, proposeHeightNew)
	if err != nil {
		p.log.Errorf("Produce propose block error: latest epoch=%d, latest height=%d, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, err)
		return err
	}

	if can := p.validator.canProcessForwardProposeMsg(ctx, maxPri, proposeBlock); !can {
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

	if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
		p.log.Errorf("Deliver propose message err: latest epoch =%d, latest height=%d, self node=%s, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, p.nodeID, err)
		return err
	}

	p.log.Infof("Propose block sucessfully: state version %d, propose Block height %d, self node %s", proposeBlock.StateVersion, latestBlock.Head.Height+1, p.nodeID)

	return nil
}

func (p *consensusProposer) proposeBlockStart(ctx context.Context) {
	go func() {
		timer := time.NewTicker(p.proposeMaxInterval)
		defer timer.Stop()
		for {
			select {
			case addedBlock := <-p.blockAddedCh:
				if err := p.proposeBlockSpecification(ctx, addedBlock); err != nil {
					continue
				}
			case <-timer.C:
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
	p.proposeBlockStart(ctx)

	p.receivePreparePackedMessagePropStart(ctx)

	p.receiveVoteMessagStart(ctx)
}

func (p *consensusProposer) createBlockHead(chainID tpchaintypes.ChainID, epoch uint64, vrfProof []byte, maxPri []byte, frontPPMProp *PreparePackedMessageProp, latestBlock *tpchaintypes.Block, stateRoot []byte, proposeHeight uint64) (*tpchaintypes.BlockHead, uint64, error) {
	blockHashBytes, err := latestBlock.HashBytes()
	if err != nil {
		p.log.Errorf("Can't get the hash bytes of block height %d: %v", latestBlock.Head.Height, err)
		return nil, 0, err
	}

	csProof := &ConsensusProof{
		ParentBlockHash: latestBlock.Head.ParentBlockHash,
		Height:          latestBlock.Head.Height,
		AggSign:         latestBlock.Head.VoteAggSignature,
	}

	csProofBytes, err := p.marshaler.Marshal(csProof)
	if err != nil {
		p.log.Errorf("Marshal consensus proof failed: %v", err)
		return nil, 0, err
	}

	return &tpchaintypes.BlockHead{
		ChainID:         []byte(chainID),
		Version:         tpchaintypes.BLOCK_VER,
		Height:          latestBlock.Head.Height + 1,
		Epoch:           epoch,
		Round:           latestBlock.Head.Height,
		ParentBlockHash: blockHashBytes,
		Launcher:        frontPPMProp.Launcher,
		Proposer:        []byte(p.nodeID),
		Proof:           csProofBytes,
		VRFProof:        vrfProof,
		VRFProofHeight:  proposeHeight,
		MaxPri:          maxPri,
		TxCount:         uint32(len(frontPPMProp.TxHashs)),
		TxRoot:          frontPPMProp.TxRoot,
		TxResultRoot:    frontPPMProp.TxResultRoot,
		StateRoot:       stateRoot,
		TimeStamp:       uint64(time.Now().UnixNano()),
	}, frontPPMProp.StateVersion, nil
}

func (p *consensusProposer) getAvailPPMProp(latestHeight uint64) (*PreparePackedMessageProp, error) {
	p.syncPPMPropList.RLock()
	defer p.syncPPMPropList.RUnlock()

	var frontPPMProp *PreparePackedMessageProp
	for p.ppmPropList.Len() > 0 {
		frontEle := p.ppmPropList.Front()
		frontPPMProp = frontEle.Value.(*PreparePackedMessageProp)
		if frontPPMProp.StateVersion == latestHeight+1 {
			break
		} else if frontPPMProp.StateVersion < latestHeight+1 {
			p.ppmPropList.Remove(frontEle)
		}
	}
	if frontPPMProp == nil || frontPPMProp.StateVersion == latestHeight {
		err := fmt.Errorf("Can't get prepare packed message prop: there is no expected state version %d, total PPM prop %d, self node=%s", latestHeight+1, p.ppmPropList.Len(), p.nodeID)
		return nil, err
	}

	return frontPPMProp, nil
}

func (p *consensusProposer) produceProposeBlock(chainID tpchaintypes.ChainID,
	latestEpoch *tpcmm.EpochInfo,
	latestBlock *tpchaintypes.Block,
	ppmProp *PreparePackedMessageProp,
	vrfProof []byte,
	maxPri []byte,
	stateRoot []byte,
	proposeHeight uint64) (*ProposeMessage, error) {
	newBlockHead, stateVersion, err := p.createBlockHead(chainID, latestEpoch.Epoch, vrfProof, maxPri, ppmProp, latestBlock, stateRoot, proposeHeight)
	if err != nil {
		p.log.Errorf("Create block failed: %v", err)
		return nil, err
	}
	newBlockHeadBytes, err := p.marshaler.Marshal(newBlockHead)
	if err != nil {
		p.log.Errorf("Marshal block head failed: %v", err)
		return nil, err
	}

	return &ProposeMessage{
		ChainID:       []byte(chainID),
		Version:       CONSENSUS_VER,
		Epoch:         latestEpoch.Epoch,
		Round:         latestBlock.Head.Height,
		StateVersion:  stateVersion,
		TxHashs:       ppmProp.TxHashs,
		TxResultHashs: ppmProp.TxResultHashs,
		BlockHead:     newBlockHeadBytes,
	}, nil
}
