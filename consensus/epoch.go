package consensus

import (
	"context"
	"time"

	tptypes "github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type EpochState interface {
	GetCurrentRound() uint64
	SetCurrentRound(round uint64)
	GetCurrentEpoch() uint64
	SetCurrentEpoch(epoch uint64)
	GetLatestBlock() (*tptypes.Block, error)
}

type epochService struct {
	log           tplog.Logger
	currentEpoch  uint64
	currentRound  uint64
	roundCh       chan *RoundInfo
	roundDuration time.Duration //unit: ms
	epochInterval uint64        //the round number between two epochs
	state         EpochState
	dkgExchange   *dkgExchange
}

func newEpochService(log tplog.Logger,
	roundCh chan *RoundInfo,
	roundDuration time.Duration,
	epochInterval uint64,
	state EpochState,
	dkgExchange *dkgExchange) *epochService {
	if state == nil || dkgExchange == nil {
		panic("Invalid input parameter and can't create epoch service!")
	}
	currentEpoch := state.GetCurrentEpoch()
	currentRound := state.GetCurrentRound()

	return &epochService{
		log:           log,
		currentEpoch:  currentEpoch,
		currentRound:  currentRound,
		roundCh:       roundCh,
		roundDuration: roundDuration,
		epochInterval: epochInterval,
		state:         state,
		dkgExchange:   dkgExchange,
	}
}

func (es *epochService) start(ctx context.Context) {
	startRound := uint64(0)

	es.currentEpoch = es.state.GetCurrentEpoch()
	es.currentRound = es.state.GetCurrentRound()

	if es.currentEpoch != 0 {
		startRound = es.currentRound
	}

	nextEpochDuration := int64(es.epochInterval-startRound) * es.roundDuration.Nanoseconds()

	nextEpochTimer := time.After(time.Duration(nextEpochDuration))

	roundTimer := time.After(es.roundDuration)

	for {
		select {
		case <-nextEpochTimer:
			es.currentEpoch = es.currentEpoch + 1
			es.state.SetCurrentEpoch(es.currentEpoch)
			es.dkgExchange.start(es.currentEpoch)
		case <-roundTimer:
			es.currentRound = es.currentRound + 1
			es.state.SetCurrentRound(es.currentRound)
			latestBlock, err := es.state.GetLatestBlock()
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
