package consensus

import (
	"container/list"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TopiaNetwork/topia/codec"
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
	blockAddedCh            chan *types.Block
	preprePackedMsgPropChan chan *PreparePackedMessageProp
	voteMsgChan             chan *VoteMessage
	deliver                 messageDeliverI
	marshaler               codec.Marshaler
	ledger                  ledger.Ledger
	cryptService            tpcrt.CryptService
	proposeMaxInterval      time.Duration
	isProposing             uint32
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
	blockAddedCh chan *types.Block,
	crypt tpcrt.CryptService,
	proposeMaxInterval time.Duration,
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
		isProposing:             0,
		ppmPropList:             list.New(),
		validator:               validator,
		voteCollector:           newConsensusVoteCollector(log),
	}

	return csProposer
}

func (p *consensusProposer) updateDKGBls(dkgBls DKGBls) {
	p.voteCollector.updateDKGBls(dkgBls)
}

func (p *consensusProposer) getVrfInputData(block *types.Block) ([]byte, error) {
	hasher := tpcmm.NewBlake2bHasher(0)

	if err := binary.Write(hasher.Writer(), binary.BigEndian, block.Head.Epoch); err != nil {
		return nil, err
	}
	if err := binary.Write(hasher.Writer(), binary.BigEndian, block.Head.Height); err != nil {
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

func (p *consensusProposer) canProposeBlock(csStateRN state.CompositionStateReadonly, block *types.Block) (bool, []byte, []byte, error) {
	proposerSel := NewProposerSelector(ProposerSelectionType_Poiss, p.cryptService)

	vrfData, err := p.getVrfInputData(block)
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
	if winCount >= 1 {
		maxPri := proposerSel.MaxPriority(vrfProof, winCount)

		bestPri, err := p.validator.judgeLocalMaxPriBestForProposer(maxPri)
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
	var blockHead types.BlockHead
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
				p.log.Infof("Received voite message, state version %d self node %s", voteMsg.StateVersion, p.nodeID)
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

func (p *consensusProposer) proposeBlockSpecification(ctx context.Context, addedBlock *types.Block) error {
	if !atomic.CompareAndSwapUint32(&p.isProposing, 0, 1) {
		err := fmt.Errorf("Self node %s is proposing", p.nodeID)
		p.log.Infof("%s", err.Error())
		return err
	}
	defer func() {
		p.isProposing = 0
	}()

	p.syncPPMPropList.RLock()
	if p.ppmPropList.Len() == 0 {
		err := fmt.Errorf("Current ppm prop list size 0")
		p.log.Infof("%s", err.Error())
		p.syncPPMPropList.RUnlock()
		return err
	}
	p.syncPPMPropList.RUnlock()

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

	canPropose, vrfProof, maxPri, err := p.canProposeBlock(csStateRN, latestBlock)
	if !canPropose {
		err = fmt.Errorf("Can't propose block at the epoch : epoch=%d, height=%d, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, err)
		p.log.Infof("%s", err.Error())
		return err
	}

	stateRoot, err := csStateRN.StateRoot()
	if err != nil {
		p.log.Errorf("Can't get state root: %v", err)
		return err
	}
	proposeBlock, err := p.produceProposeBlock(csStateRN.ChainID(), latestEpoch, latestBlock, vrfProof, maxPri, stateRoot)
	if err != nil {
		p.log.Errorf("Produce propose block error: epoch=%d, height=%d, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, err)
		return err
	}

	if can := p.validator.canProcessForwardProposeMsg(ctx, maxPri, proposeBlock); !can {
		err = fmt.Errorf("Can't delive propose message: epoch=%d, height=%d", latestBlock.Head.Epoch, latestBlock.Head.Height)
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
		err = errors.New("Finally nil dkgBls and can't deliver propose message")
		p.log.Errorf("%v", err)
		return err
	}

	p.log.Infof("Message deliver ready, state version %d height %d self node %s", proposeBlock.StateVersion, latestBlock.Head.Height+1, p.nodeID)

	if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
		p.log.Errorf("Deliver propose message err: epoch =%d, height=%d, err=%v", latestBlock.Head.Epoch, latestBlock.Head.Height, err)
		return err
	}

	p.log.Infof("Propose block sucessfully, state version %d height %d self node %s", proposeBlock.StateVersion, latestBlock.Head.Height+1, p.nodeID)

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

func (p *consensusProposer) createBlockHead(chainID types.ChainID, epoch uint64, vrfProof []byte, maxPri []byte, frontPPMProp *PreparePackedMessageProp, latestBlock *types.Block, stateRoot []byte) (*types.BlockHead, uint64, error) {
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

	return &types.BlockHead{
		ChainID:         []byte(chainID),
		Version:         types.BLOCK_VER,
		Height:          latestBlock.Head.Height + 1,
		Epoch:           epoch,
		Round:           latestBlock.Head.Height,
		ParentBlockHash: blockHashBytes,
		Launcher:        frontPPMProp.Launcher,
		Proposer:        []byte(p.nodeID),
		Proof:           csProofBytes,
		VRFProof:        vrfProof,
		MaxPri:          maxPri,
		TxCount:         uint32(len(frontPPMProp.TxHashs)),
		TxRoot:          frontPPMProp.TxRoot,
		TxResultRoot:    frontPPMProp.TxResultRoot,
		StateRoot:       stateRoot,
		TimeStamp:       uint64(time.Now().UnixNano()),
	}, frontPPMProp.StateVersion, nil
}

func (p *consensusProposer) produceProposeBlock(chainID types.ChainID,
	latestEpoch *tpcmm.EpochInfo,
	latestBlock *types.Block,
	vrfProof []byte,
	maxPri []byte,
	stateRoot []byte) (*ProposeMessage, error) {
	p.syncPPMPropList.RLock()
	frontPPMProp := p.ppmPropList.Front().Value.(*PreparePackedMessageProp)
	if frontPPMProp.StateVersion != latestBlock.Head.Height+1 {
		err := fmt.Errorf("Invalid prepare packed message prop: expected front state version %d, actual %d", latestBlock.Head.Height+1, frontPPMProp.StateVersion)
		p.syncPPMPropList.RUnlock()
		return nil, err
	}
	p.syncPPMPropList.RUnlock()

	newBlockHead, stateVersion, err := p.createBlockHead(chainID, latestEpoch.Epoch, vrfProof, maxPri, frontPPMProp, latestBlock, stateRoot)
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
		TxHashs:       frontPPMProp.TxHashs,
		TxResultHashs: frontPPMProp.TxResultHashs,
		BlockHead:     newBlockHeadBytes,
	}, nil
}
