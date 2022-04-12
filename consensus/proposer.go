package consensus

import (
	"container/list"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/TopiaNetwork/topia/chain"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
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
	blockAddedCh            chan *tpchaintypes.Block
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
	blockAddedCh chan *tpchaintypes.Block,
	preprePackedMsgPropChan chan *PreparePackedMessageProp,
	voteMsgChan chan *VoteMessage,
	crypt tpcrt.CryptService,
	proposeMaxInterval time.Duration,
	deliver messageDeliverI,
	ledger ledger.Ledger,
	marshaler codec.Marshaler,
	validator *consensusValidator) *consensusProposer {
	csProposer := &consensusProposer{
		log:                     log,
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

func (p *consensusProposer) getVrfInputData(block *tpchaintypes.Block) ([]byte, error) {
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

func (p *consensusProposer) canProposeBlock(block *tpchaintypes.Block) (bool, []byte, []byte, error) {
	csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
	defer csStateRN.Stop()

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
				csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
				defer csStateRN.Stop()

				p.syncPPMPropList.Lock()
				defer p.syncPPMPropList.Unlock()

				latestBlock, err := csStateRN.GetLatestBlock()
				if err != nil {
					p.log.Errorf("Can't get the latest bock when receiving prepare packed msg prop: %v", err)
					continue
				}
				if ppmProp.StateVersion <= latestBlock.Head.Height {
					p.log.Errorf("Received outdated prepare packed msg prop: %v", err)
					continue
				}

				if p.ppmPropList.Len() > 0 {
					latestPPMProp := p.ppmPropList.Back().Value.(*PreparePackedMessageProp)
					if ppmProp.StateVersion != latestPPMProp.StateVersion+1 {
						p.log.Errorf("Received invalid prepare packed msg prop: expected state version %d, actual %d", latestPPMProp.StateVersion+1, ppmProp.StateVersion)
						continue
					}
				}

				p.ppmPropList.PushBack(ppmProp)

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

	newBlockHeadBytes, err := p.marshaler.Marshal(blockHead)
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
				aggSign, err := p.voteCollector.tryAggregateSignAndAddVote(voteMsg)
				if err != nil {
					p.log.Errorf("Try to aggregate sign and add vote faild: err=%v", err)
					continue
				}

				if aggSign == nil {
					p.log.Debug("Haven't reached threshold at present")
					continue
				}

				commitMsg, _ := p.produceCommitMsg(voteMsg, aggSign)
				err = p.deliver.deliverCommitMessage(context.Background(), commitMsg)
				if err != nil {
					p.log.Errorf("Can't deliver commit message: %v", err)
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

func (p *consensusProposer) proposeBlockSpecification(ctx context.Context, addedBlock *tpchaintypes.Block) error {
	if atomic.CompareAndSwapUint32(&p.isProposing, 0, 1) {
		err := fmt.Errorf("Self node %s is proposing", p.nodeID)
		p.log.Infof("%s", err.Error())
		return err
	}
	defer func() {
		p.isProposing = 0
	}()

	if p.ppmPropList.Len() == 0 {
		err := fmt.Errorf("Current ppm prop list size 0")
		p.log.Infof("%s", err.Error())
		return err
	}

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

	canPropose, vrfProof, maxPri, err := p.canProposeBlock(latestBlock)
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
		p.log.Errorf("Produce propose block error: epoch=%d, height=%d, err=%v", addedBlock.Head.Epoch, addedBlock.Head.Height, err)
		return err
	}

	if can := p.validator.canProcessForwardProposeMsg(ctx, maxPri, proposeBlock); !can {
		err = fmt.Errorf("Can't delive propose message: epoch=%d, height=%d", addedBlock.Head.Epoch, addedBlock.Head.Height)
		p.log.Infof("%s", err.Error())
		return err
	}

	if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
		p.log.Errorf("Consensus deliver propose message err: epoch =%d, height=%d, err=%v", addedBlock.Head.Epoch, addedBlock.Head.Height, err)
		return err
	}

	return nil
}

func (p *consensusProposer) proposeBlockStart(ctx context.Context) {
	go func() {
		for {
			timer := time.NewTimer(p.proposeMaxInterval)
			defer timer.Stop()
			select {
			case addedBlock := <-p.blockAddedCh:
				if err := p.proposeBlockSpecification(ctx, addedBlock); err != nil {
					continue
				}
			case <-timer.C:
				if err := p.proposeBlockSpecification(ctx, nil); err != nil {
					continue
				}
				timer.Reset(p.proposeMaxInterval)
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

func (p *consensusProposer) createBlockHead(chainID chain.ChainID, epoch uint64, vrfProof []byte, maxPri []byte, frontPPMProp *PreparePackedMessageProp, latestBlock *tpchaintypes.Block, stateRoot []byte) (*tpchaintypes.BlockHead, uint64, error) {
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
		MaxPri:          maxPri,
		TxCount:         uint32(len(frontPPMProp.TxHashs)),
		TxRoot:          frontPPMProp.TxRoot,
		TxResultRoot:    frontPPMProp.TxResultRoot,
		StateRoot:       stateRoot,
		TimeStamp:       uint64(time.Now().UnixNano()),
	}, frontPPMProp.StateVersion, nil
}

func (p *consensusProposer) produceProposeBlock(chainID chain.ChainID,
	latestEpoch *chain.EpochInfo,
	latestBlock *tpchaintypes.Block,
	vrfProof []byte,
	maxPri []byte,
	stateRoot []byte) (*ProposeMessage, error) {
	p.syncPPMPropList.Lock()
	frontPPMProp := p.ppmPropList.Front().Value.(*PreparePackedMessageProp)
	if frontPPMProp.StateVersion != latestBlock.Head.Height+1 {
		err := fmt.Errorf("Invalid prepare packed message prop: expected front state version %d, actual %d", latestBlock.Head.Height+1, frontPPMProp.StateVersion)
		defer p.syncPPMPropList.Unlock()
		return nil, err
	}
	p.syncPPMPropList.Unlock()

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
