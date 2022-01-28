package consensus

import (
	"context"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tptypes "github.com/TopiaNetwork/topia/common/types"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const defaultLeaderCount = int(3)

type consensusProposer struct {
	log          tplog.Logger
	nodeID       string
	priKey       tpcrtypes.PrivateKey
	lastRoundNum uint64
	roundCh      chan *RoundInfo
	deliver      messageDeliverI
	csState      consensusStore
	marshaler    codec.Marshaler
	selector     *roleSelectorVRF
}

func newConsensusProposer(nodeID string, priKey tpcrtypes.PrivateKey, log tplog.Logger, roundCh chan *RoundInfo, crypt tpcrt.CryptService, deliver messageDeliverI, csState consensusStore, marshaler codec.Marshaler) *consensusProposer {
	return &consensusProposer{
		log:       log,
		nodeID:    nodeID,
		priKey:    priKey,
		roundCh:   roundCh,
		deliver:   deliver,
		csState:   csState,
		marshaler: marshaler,
		selector:  newLeaderSelectorVRF(log, crypt, csState),
	}
}

func (p *consensusProposer) canProposeBlock(roundInfo *RoundInfo) (bool, []byte, error) {
	selProposers, vrfProof, err := p.selector.Select(RoleSelector_Proposer, roundInfo, p.priKey, defaultLeaderCount)
	if err != nil {
		return false, nil, err
	}
	if len(selProposers) < defaultLeaderCount {
		p.log.Errorf("expected proposer count %d, got %d", defaultLeaderCount, len(selProposers))
		return false, vrfProof, nil
	}
	for _, leader := range selProposers {
		if leader.nodeID == p.nodeID {
			return true, vrfProof, nil
		}
	}

	return false, nil, nil
}

func (p *consensusProposer) start(ctx context.Context) {
	go func() {
		for {
			select {
			case roundInfo := <-p.roundCh:
				canPropose, vrfProof, err := p.canProposeBlock(roundInfo)
				if err != nil {
					p.log.Errorf("Error happens when judge proposing block : epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
					continue
				}
				if !canPropose {
					p.log.Infof("Can't  propose block at the round : epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
					continue
				}

				p.lastRoundNum = roundInfo.LastRoundNum
				proposeBlock, err := p.produceProposeBlock(roundInfo)
				if err != nil {
					p.log.Errorf("Produce propose block error: epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
					continue
				}
				proposeBlock.ProposerProof = vrfProof
				if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
					p.log.Errorf("Consensus deliver propose message err: epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
				}
			case <-ctx.Done():
				p.log.Info("Consensus proposer exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) createBlock(roundInfo *RoundInfo) (*tptypes.Block, error) {
	latestBlock, err := p.csState.GetLatestBlock()
	if err != nil {
		p.log.Errorf("can't get the latest block: %v", err)
	}

	blockHashBytes, err := latestBlock.HashBytes(tpcmm.NewBlake2bHasher(0), p.marshaler)
	if err != nil {
		p.log.Errorf("can't get the hash bytes of block height %d: %v", latestBlock.Head.Height, err)
		return nil, err
	}

	return &tptypes.Block{
		Head: &tptypes.BlockHead{
			ChainID:         []byte(p.csState.ChainID()),
			Version:         tptypes.BLOCK_VER,
			Height:          latestBlock.Head.Height + 1,
			Epoch:           roundInfo.Epoch,
			Round:           roundInfo.CurRoundNum,
			ParentBlockHash: blockHashBytes,
		},
	}, nil
}

func (p *consensusProposer) produceProposeBlock(roundInfo *RoundInfo) (*ProposeMessage, error) {
	newBlock, err := p.createBlock(roundInfo)
	if err != nil {
		p.log.Errorf("Create block failed: %v", err)
		return nil, err
	}
	newBlockBytes, err := p.marshaler.Marshal(newBlock)
	if err != nil {
		p.log.Errorf("Marshal block failed: %v", err)
		return nil, err
	}

	csProofBytes, err := p.marshaler.Marshal(roundInfo.Proof)
	if err != nil {
		p.log.Errorf("Marshal consensus proof failed: %v", err)
		return nil, err
	}

	return &ProposeMessage{
		ChainID: []byte(p.csState.ChainID()),
		Version: CONSENSUS_VER,
		Epoch:   roundInfo.Epoch,
		Round:   roundInfo.CurRoundNum,
		Proof:   csProofBytes,
		Block:   newBlockBytes,
	}, nil
}
