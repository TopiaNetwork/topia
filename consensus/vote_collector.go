package consensus

import (
	"sync"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type consensusVoteCollector struct {
	sync.Mutex
	log       tplog.Logger
	threshold uint64
	votes     []*VoteMessage
	dkgBls    DKGBls
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
	if len(vc.votes) >= vc.dkgBls.Threshold() {
		return vc.produceAggSign()
	} else {
		vc.log.Infof("Received vote count=%d, required=%d", len(vc.votes), vc.dkgBls.Threshold())
	}

	return nil, nil
}

func (vc *consensusVoteCollector) produceAggSign() (tpcrtypes.Signature, error) {
	signArr := make([][]byte, 0)
	for _, vote := range vc.votes {
		signArr = append(signArr, vote.Signature)
	}

	return vc.dkgBls.RecoverSig(vc.votes[0].BlockHead, signArr)
}

func (vc *consensusVoteCollector) updateDKGBls(dkgBls DKGBls) {
	vc.dkgBls = dkgBls
}

func (vc *consensusVoteCollector) reset() {
	vc.votes = vc.votes[:0]
}
