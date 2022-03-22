package consensus

import (
	"context"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/state"
	"time"

	tplog "github.com/TopiaNetwork/topia/log"
)

type epochService struct {
	log           tplog.Logger
	currentEpoch  uint64
	currentRound  uint64
	roundCh       chan *RoundInfo
	roundDuration time.Duration //unit: ms
	epochInterval uint64        //the round number between two epochs
	ledger        ledger.Ledger
	dkgExchange   *dkgExchange
}

func newEpochService(log tplog.Logger,
	roundCh chan *RoundInfo,
	roundDuration time.Duration,
	epochInterval uint64,
	ledger ledger.Ledger,
	dkgExchange *dkgExchange) *epochService {
	if ledger == nil || dkgExchange == nil {
		panic("Invalid input parameter and can't create epoch service!")
	}

	csStateRN := state.CreateCompositionStateReadonly(log, ledger)
	defer csStateRN.Stop()

	currentEpoch := csStateRN.GetCurrentEpoch()
	currentRound := csStateRN.GetCurrentRound()

	return &epochService{
		log:           log,
		currentEpoch:  currentEpoch,
		currentRound:  currentRound,
		roundCh:       roundCh,
		roundDuration: roundDuration,
		epochInterval: epochInterval,
		ledger:        ledger,
		dkgExchange:   dkgExchange,
	}
}

func (es *epochService) start(ctx context.Context) {
	startRound := uint64(0)

	csStateRN := state.CreateCompositionStateReadonly(es.log, es.ledger)
	defer csStateRN.Stop()

	es.currentEpoch = csStateRN.GetCurrentEpoch()
	es.currentRound = csStateRN.GetCurrentRound()

	if es.currentEpoch != 0 {
		startRound = es.currentRound
	}

	nextEpochDuration := int64(es.epochInterval-startRound) * es.roundDuration.Nanoseconds()

	nextEpochTimer := time.After(time.Duration(nextEpochDuration))

	roundTimer := time.After(es.roundDuration)

	for {
		select {
		case <-nextEpochTimer:
			csStateEpoch := state.CreateCompositionState(es.log, es.ledger)
			defer csStateEpoch.Commit()

			es.currentEpoch = es.currentEpoch + 1
			csStateEpoch.SetCurrentEpoch(es.currentEpoch)
			es.dkgExchange.start(es.currentEpoch)
		case <-roundTimer:
			csStateRound := state.CreateCompositionState(es.log, es.ledger)
			defer csStateRound.Commit()

			es.currentRound = es.currentRound + 1
			csStateRound.SetCurrentRound(es.currentRound)
			if !es.dkgExchange.dkgCrypt.finished() || es.dkgExchange.dkgCrypt.epoch < es.currentEpoch {
				es.log.Warnf("Current epoch %d DKG unfinished and ignore the round %d", es.currentEpoch, es.currentRound)
				continue
			}
			latestBlock, err := csStateRN.GetLatestBlock()
			if err != nil {
				es.log.Errorf("Can't get the latest block: err=%v", err)
				continue
			}

			csProof := &ConsensusProof{
				ParentBlockHash: latestBlock.Head.ParentBlockHash,
				Height:          latestBlock.Head.Height,
				AggSign:         latestBlock.Head.VoteAggSignature,
			}

			roundInfo := &RoundInfo{
				Epoch:        es.currentEpoch,
				LastRoundNum: es.currentRound - 1,
				CurRoundNum:  es.currentRound,
				Proof:        csProof,
			}

			es.roundCh <- roundInfo
		case <-ctx.Done():
			es.log.Info("Exit epoch cycle")
			return
		}
	}

}
