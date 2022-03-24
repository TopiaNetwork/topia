package consensus

import (
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
	deliver                 messageDeliverI
	marshaler               codec.Marshaler
	ledger                  ledger.Ledger
	cryptService            tpcrt.CryptService
	syncPPPMPropList        sync.RWMutex
	ppmPropList             *list.List
}

func newConsensusProposer(log tplog.Logger, nodeID string, priKey tpcrtypes.PrivateKey, roundCh chan *RoundInfo, preprePackedMsgPropChan chan *PreparePackedMessageProp, crypt tpcrt.CryptService, deliver messageDeliverI, ledger ledger.Ledger, marshaler codec.Marshaler) *consensusProposer {
	return &consensusProposer{
		log:                     log,
		nodeID:                  nodeID,
		priKey:                  priKey,
		roundCh:                 roundCh,
		preprePackedMsgPropChan: preprePackedMsgPropChan,
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

		return true, vrfProof, maxPri, nil
	}

	return false, nil, nil, fmt.Errorf("Can't propose block at the round: winCount=%d", winCount)
}

func (p *consensusProposer) receivePreparePackedMessagePropLoop(ctx context.Context) {
	go func() {
		for {
			select {
			case ppmProp := <-p.preprePackedMsgPropChan:
				csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
				defer csStateRN.Stop()

				p.syncPPPMPropList.Lock()
				defer p.syncPPPMPropList.Unlock()

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

func (p *consensusProposer) start(ctx context.Context) {
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
				proposeBlock.ProposerProof = vrfProof
				if err = p.deliver.deliverProposeMessage(ctx, proposeBlock); err != nil {
					p.log.Errorf("Consensus deliver propose message err: epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
				}
			case <-ctx.Done():
				p.log.Info("Consensus proposer round exit")
				return
			}
		}
	}()

	p.receivePreparePackedMessagePropLoop(ctx)
}

func (p *consensusProposer) createBlockHead(roundInfo *RoundInfo, frontPPMProp *PreparePackedMessageProp, latestBlock *tpchaintypes.Block, csStateRN state.CompositionStateReadonly) (*tpchaintypes.BlockHead, uint64, error) {
	blockHashBytes, err := latestBlock.HashBytes(tpcmm.NewBlake2bHasher(0), p.marshaler)
	if err != nil {
		p.log.Errorf("Can't get the hash bytes of block height %d: %v", latestBlock.Head.Height, err)
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
		TxCount:         uint32(len(frontPPMProp.TxHashs)),
		TxRoot:          frontPPMProp.TxRoot,
		TxResultRoot:    frontPPMProp.TxResultRoot,
		StateRoot:       stateRoot,
		TimeStamp:       uint64(time.Now().UnixNano()),
	}, frontPPMProp.StateVersion, nil
}

func (p *consensusProposer) produceProposeBlock(roundInfo *RoundInfo, vrfRoot []byte, maxPri []byte) (*ProposeMessage, error) {
	csStateRN := state.CreateCompositionStateReadonly(p.log, p.ledger)
	defer csStateRN.Stop()

	latestBlock, err := csStateRN.GetLatestBlock()
	if err != nil {
		p.log.Errorf("Can't get the latest block: %v", err)
		return nil, err
	}

	p.syncPPPMPropList.Lock()
	frontPPMProp := p.ppmPropList.Front().Value.(*PreparePackedMessageProp)
	if frontPPMProp.StateVersion != latestBlock.Head.Height+1 {
		err = fmt.Errorf("Invalid prepare packed message prop: expected front state version %d, actual %d", latestBlock.Head.Height+1, frontPPMProp.StateVersion)
		defer p.syncPPPMPropList.Unlock()
		return nil, err
	}
	p.syncPPPMPropList.Unlock()

	newBlockHead, stateVersion, err := p.createBlockHead(roundInfo, frontPPMProp, latestBlock, csStateRN)
	if err != nil {
		p.log.Errorf("Create block failed: %v", err)
		return nil, err
	}
	newBlockHeadBytes, err := p.marshaler.Marshal(newBlockHead)
	if err != nil {
		p.log.Errorf("Marshal block head failed: %v", err)
		return nil, err
	}

	csProofBytes, err := p.marshaler.Marshal(roundInfo.Proof)
	if err != nil {
		p.log.Errorf("Marshal consensus proof failed: %v", err)
		return nil, err
	}

	signData, err := p.cryptService.Sign(p.priKey, newBlockHeadBytes)
	if err != nil {
		p.log.Errorf("Sign err for propose msg: %v", err)
		return nil, err
	}

	pubKey, err := p.cryptService.ConvertToPublic(p.priKey)
	if err != nil {
		p.log.Errorf("Can't get public key from private key: %v", err)
		return nil, err
	}

	return &ProposeMessage{
		ChainID:       []byte(csStateRN.ChainID()),
		Version:       CONSENSUS_VER,
		Epoch:         roundInfo.Epoch,
		Round:         roundInfo.CurRoundNum,
		Proposer:      []byte(p.nodeID),
		Proof:         csProofBytes,
		ProposerProof: vrfRoot,
		MaxPri:        maxPri,
		Signature:     signData,
		PubKey:        pubKey,
		StateVersion:  stateVersion,
		TxHashs:       frontPPMProp.TxHashs,
		TxResultHashs: frontPPMProp.TxResultHashs,
		BlockHead:     newBlockHeadBytes,
	}, nil
}
