package consensus

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	"sync"
)

type consensusVoteCollector struct {
	sync.Mutex
	log       tplog.Logger
	threshold uint64
	votes     []*VoteMessage
}

func newConsensusVoteCollector(log tplog.Logger) *consensusVoteCollector {
	return &consensusVoteCollector{
		log: log,
	}
}

func (vc *consensusVoteCollector) setThreshold(threshold uint64) {
	vc.threshold = threshold
}

func (vc *consensusVoteCollector) tryAggregateSignAndAddVote(vote *VoteMessage) (tpcrtypes.Signature, error) {
	vc.Lock()
	defer vc.Unlock()

	vc.votes = append(vc.votes, vote)
	if uint64(len(vc.votes)) >= vc.threshold {
		return vc.produceAggSign()
	} else {
		vc.log.Debugf("Received vote count=%d, required=%d", len(vc.votes), vc.threshold)
	}

	return nil, nil
}

func (vc *consensusVoteCollector) produceAggSign() (tpcrtypes.Signature, error) {
	return nil, nil
}

func (vc *consensusVoteCollector) reset() {
	vc.votes = vc.votes[:0]
}
