package consensus

import (
	"context"
	"github.com/TopiaNetwork/topia/codec"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tptypes "github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type consensusProposer struct {
	log          tplog.Logger
	lastRoundNum uint64
	roundCh      chan *RoundInfo
	deliver      *messageDeliver
	csState      ConsensusStore
	marshaler    codec.Marshaler
	hasher       tpcmm.Hasher
}

func NewConsensusProposer(log tplog.Logger, roundCh chan *RoundInfo, deliver *messageDeliver, csState ConsensusStore, marshaler codec.Marshaler) *consensusProposer {
	return &consensusProposer{
		log:       log,
		roundCh:   roundCh,
		deliver:   deliver,
		csState:   csState,
		marshaler: marshaler,
		hasher:    tpcmm.NewBlake2bHasher(0),
	}
}

func (p *consensusProposer) start(ctx context.Context) {
	go func() {
		for {
			select {
			case roundInfo := <-p.roundCh:
				p.lastRoundNum = roundInfo.LastRoundNum
				proposeBlock, err := p.produceProposeBlock(roundInfo)
				if err != nil {
					p.log.Errorf("Produce propose block error: new round=%d, err=%v", roundInfo.CurRoundNum, err)
					continue
				}
				if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
					p.log.Errorf("Consensus deliver propose message err: %v", err)
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

	blockHashBytes, err := latestBlock.HashBytes(p.hasher, p.marshaler)
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
