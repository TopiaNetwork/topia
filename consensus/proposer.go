package consensus

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
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
	lastRoundNum            uint64
	roundCh                 chan *RoundInfo
	preprePackedMsgPropChan chan *PreparePackedMessageProp
	proposeMsgChan          chan *ProposeMessage
	deliver                 messageDeliverI
	marshaler               codec.Marshaler
	ledger                  ledger.Ledger
	cryptService            tpcrt.CryptService
	syncPPMPropList         sync.RWMutex
	syncPropMsgCached       sync.RWMutex
	ppmPropList             *list.List
	propMsgCached           *ProposeMessage
}

func newConsensusProposer(log tplog.Logger, nodeID string, priKey tpcrtypes.PrivateKey, roundCh chan *RoundInfo, preprePackedMsgPropChan chan *PreparePackedMessageProp, proposeMsgChan chan *ProposeMessage, crypt tpcrt.CryptService, deliver messageDeliverI, ledger ledger.Ledger, marshaler codec.Marshaler) *consensusProposer {
	return &consensusProposer{
		log:                     log,
		nodeID:                  nodeID,
		priKey:                  priKey,
		roundCh:                 roundCh,
		preprePackedMsgPropChan: preprePackedMsgPropChan,
		proposeMsgChan:          proposeMsgChan,
		deliver:                 deliver,
		marshaler:               marshaler,
		ledger:                  ledger,
		cryptService:            crypt,
		ppmPropList:             list.New(),
	}
}

func (p *consensusProposer) canProposeBlock(roundInfo *RoundInfo) (bool, []byte, []byte, error) {
	csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
	defer csStateRN.Stop()

	proposerSel := NewProposerSelector(ProposerSelectionType_Poiss, p.cryptService)

	vrfData, err := json.Marshal(roundInfo)
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

		p.syncPropMsgCached.RLock()
		if p.propMsgCached != nil {
			bhCached, err := p.propMsgCached.BlockHeadInfo()
			if err != nil {
				p.log.Errorf("Can't get cached propose msg bock head info: %v", err)
			}
			cachedMaxPri := bhCached.MaxPri

			if new(big.Int).SetBytes(maxPri).Cmp(new(big.Int).SetBytes(cachedMaxPri)) <= 0 {
				err = fmt.Errorf("Cached propose msg bock max pri bigger")
				p.log.Errorf("%v", err)
				return false, nil, nil, err
			}

		}
		p.syncPropMsgCached.Unlock()

		return true, vrfProof, maxPri, nil
	}

	return false, nil, nil, fmt.Errorf("Can't propose block at the round: winCount=%d", winCount)
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

				latestPPMProp := p.ppmPropList.Back().Value.(*PreparePackedMessageProp)
				if ppmProp.StateVersion != latestPPMProp.StateVersion+1 {
					p.log.Errorf("Received invalid prepare packed msg prop: expected state version %d, actual %d", latestPPMProp.StateVersion+1, ppmProp.StateVersion)
					continue
				}

				p.ppmPropList.PushBack(ppmProp)
			case <-ctx.Done():
				p.log.Info("Consensus proposer receiveing prepare packed msg prop exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) receiveProposeMessageStart(ctx context.Context) {
	go func() {
		for {
			select {
			case proposeMsg := <-p.proposeMsgChan:
				csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
				defer csStateRN.Stop()

				p.syncPropMsgCached.Lock()
				defer p.syncPropMsgCached.Unlock()

				bh, err := proposeMsg.BlockHeadInfo()
				if err != nil {
					p.log.Errorf("Can't get propose block head info: %v", err)
					continue
				}

				if p.propMsgCached != nil {
					if bytes.Compare(proposeMsg.ChainID, p.propMsgCached.ChainID) != 0 ||
						proposeMsg.Version != p.propMsgCached.Version ||
						proposeMsg.Epoch != p.propMsgCached.Epoch ||
						proposeMsg.Round != p.propMsgCached.Round {
						p.log.Errorf("Difference propose basic info between received and cached, received: chainID=%d, version, epoch=%d, round=%d; cached: chainID=%d, version, epoch=%d, round=%d",
							string(proposeMsg.ChainID), proposeMsg.Version, proposeMsg.Epoch, proposeMsg.Round,
							string(p.propMsgCached.ChainID), p.propMsgCached.Version, p.propMsgCached.Epoch, p.propMsgCached.Round)
						continue
					}

					propMsgMaxPri := new(big.Int).SetBytes(bh.MaxPri)

					bhCached, err := p.propMsgCached.BlockHeadInfo()
					if err != nil {
						p.log.Errorf("Can't get cached propose block head info: %v", err)
						continue
					}

					propMsgFMaxPri := new(big.Int).SetBytes(bhCached.MaxPri)

					if propMsgMaxPri.Cmp(propMsgFMaxPri) > 0 {
						p.propMsgCached = proposeMsg
					}
				} else {
					p.propMsgCached = proposeMsg
				}

			case <-ctx.Done():
				p.log.Info("Consensus proposer receiveing prepare packed msg prop exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) proposeBlockStart(ctx context.Context) {
	go func() {
		for {
			select {
			case roundInfo := <-p.roundCh:
				if p.ppmPropList.Len() == 0 {
					p.log.Debug("Current ppm prop list size 0")
					continue
				}

				canPropose, vrfProof, maxPri, err := p.canProposeBlock(roundInfo)
				if !canPropose {
					p.log.Infof("Can't propose block at the round : epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
					continue
				}

				p.lastRoundNum = roundInfo.LastRoundNum
				proposeBlock, err := p.produceProposeBlock(roundInfo, vrfProof, maxPri)
				if err != nil {
					p.log.Errorf("Produce propose block error: epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
					continue
				}
				if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
					p.log.Errorf("Consensus deliver propose message err: epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
				}
			case <-ctx.Done():
				p.log.Info("Consensus proposer round exit")
				return
			}
		}
	}()
}

func (p *consensusProposer) start(ctx context.Context) {
	p.proposeBlockStart(ctx)

	p.receiveProposeMessageStart(ctx)

	p.receivePreparePackedMessagePropStart(ctx)
}

func (p *consensusProposer) createBlockHead(roundInfo *RoundInfo, vrfProof []byte, maxPri []byte, frontPPMProp *PreparePackedMessageProp, latestBlock *tpchaintypes.Block, csStateRN state.CompositionStateReadonly) (*tpchaintypes.BlockHead, uint64, error) {
	blockHashBytes, err := latestBlock.HashBytes(tpcmm.NewBlake2bHasher(0), p.marshaler)
	if err != nil {
		p.log.Errorf("Can't get the hash bytes of block height %d: %v", latestBlock.Head.Height, err)
		return nil, 0, err
	}

	csProofBytes, err := p.marshaler.Marshal(roundInfo.Proof)
	if err != nil {
		p.log.Errorf("Marshal consensus proof failed: %v", err)
		return nil, 0, err
	}

	stateRoot, err := csStateRN.StateRoot()
	if err != nil {
		p.log.Errorf("Can't get state root: %v", err)
		return nil, 0, err
	}

	return &tpchaintypes.BlockHead{
		ChainID:         []byte(csStateRN.ChainID()),
		Version:         tpchaintypes.BLOCK_VER,
		Height:          latestBlock.Head.Height + 1,
		Epoch:           roundInfo.Epoch,
		Round:           roundInfo.CurRoundNum,
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

func (p *consensusProposer) produceProposeBlock(roundInfo *RoundInfo, vrfProof []byte, maxPri []byte) (*ProposeMessage, error) {
	csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
	defer csStateRN.Stop()

	latestBlock, err := csStateRN.GetLatestBlock()
	if err != nil {
		p.log.Errorf("Can't get the latest block: %v", err)
		return nil, err
	}

	p.syncPPMPropList.Lock()
	frontPPMProp := p.ppmPropList.Front().Value.(*PreparePackedMessageProp)
	if frontPPMProp.StateVersion != latestBlock.Head.Height+1 {
		err = fmt.Errorf("Invalid prepare packed message prop: expected front state version %d, actual %d", latestBlock.Head.Height+1, frontPPMProp.StateVersion)
		defer p.syncPPMPropList.Unlock()
		return nil, err
	}
	p.syncPPMPropList.Unlock()

	newBlockHead, stateVersion, err := p.createBlockHead(roundInfo, vrfProof, maxPri, frontPPMProp, latestBlock, csStateRN)
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
		ChainID:       []byte(csStateRN.ChainID()),
		Version:       CONSENSUS_VER,
		Epoch:         roundInfo.Epoch,
		Round:         roundInfo.CurRoundNum,
		StateVersion:  stateVersion,
		TxHashs:       frontPPMProp.TxHashs,
		TxResultHashs: frontPPMProp.TxResultHashs,
		BlockHead:     newBlockHeadBytes,
	}, nil
}
