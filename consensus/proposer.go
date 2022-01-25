package consensus

import (
	"context"
	"encoding/binary"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tptypes "github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type consensusProposer struct {
	log          tplog.Logger
	nodeID       string
	prikey       tpcrtypes.PrivateKey
	lastRoundNum uint64
	roundCh      chan *RoundInfo
	deliver      messageDeliverI
	csState      consensusStore
	marshaler    codec.Marshaler
	hasher       tpcmm.Hasher
	selector     ProposerSelector
}

func newConsensusProposer(nodeID string, log tplog.Logger, roundCh chan *RoundInfo, crypt tpcrt.CryptService, deliver messageDeliverI, csState consensusStore, marshaler codec.Marshaler) *consensusProposer {
	selector := NewProposerSelector(ProposerSelectionType_Poiss, crypt, tpcmm.NewBlake2bHasher(0))
	return &consensusProposer{
		log:       log,
		nodeID:    nodeID,
		roundCh:   roundCh,
		deliver:   deliver,
		csState:   csState,
		marshaler: marshaler,
		hasher:    tpcmm.NewBlake2bHasher(0),
		selector:  selector,
	}
}

func getVrfInputData(roundInfo *RoundInfo, hasher tpcmm.Hasher) ([]byte, error) {
	hasher.Reset()

	if err := binary.Write(hasher.Writer(), binary.BigEndian, roundInfo.Epoch); err != nil {
		return nil, err
	}
	if err := binary.Write(hasher.Writer(), binary.BigEndian, roundInfo.CurRoundNum); err != nil {
		return nil, err
	}

	csProofBytes, err := roundInfo.Proof.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err = hasher.Writer().Write(csProofBytes); err != nil {
		return nil, err
	}

	return hasher.Bytes(), nil
}

func canProposeBlock(nodeID string, csState consensusStore, vrfInputData []byte, priKey tpcrtypes.PrivateKey, selector ProposerSelector) (bool, []byte, error) {
	totalWeight, err := csState.GetChainTotalWeight()
	if err != nil {
		return false, nil, err
	}
	nodeWeight, err := csState.GetNodeWeight(nodeID)
	if err != nil {
		return false, nil, err
	}

	vrfProof, err := selector.ComputeVRF(priKey, vrfInputData)
	if err != nil {
		return false, nil, err
	}

	pCount := selector.SelectProposer(vrfProof, nodeWeight, totalWeight)
	if pCount >= 1 {
		return true, vrfProof, nil
	}

	return false, nil, nil
}

func (p *consensusProposer) start(ctx context.Context) {
	go func() {
		for {
			select {
			case roundInfo := <-p.roundCh:
				vrfInputData, err := getVrfInputData(roundInfo, p.hasher)
				if err != nil {
					p.log.Errorf("Can't get vrf inputing data: epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
					continue
				}

				canPropose, vrfProof, err := canProposeBlock(p.nodeID, p.csState, vrfInputData, p.prikey, p.selector)
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
				proposeBlock.Proof = vrfProof
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
