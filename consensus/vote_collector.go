package consensus

import (
	"bytes"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"sync"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type consensusVoteCollector struct {
	sync.Mutex
	log          tplog.Logger
	latestHeight uint64
	threshold    uint64
	votes        []*VoteMessage
	dkgBls       DKGBls
}

func newConsensusVoteCollector(log tplog.Logger, latestHeight uint64, dkgBls DKGBls) *consensusVoteCollector {
	return &consensusVoteCollector{
		log:          log,
		latestHeight: latestHeight,
		dkgBls:       dkgBls,
	}
}

func (vc *consensusVoteCollector) setThreshold(threshold uint64) {
	vc.threshold = threshold
}

func (vc *consensusVoteCollector) tryAggregateSignAndAddVote(nodeID string, vote *VoteMessage) (tpcrtypes.Signature, error) {
	vc.Lock()
	defer vc.Unlock()

	if len(vc.votes) > 0 {
		if !bytes.Equal(vc.votes[0].BlockHead, vote.BlockHead) {
			vc.log.Infof("Received not same voted block and will be discarded: %v, expected %v, self node %s", vote.BlockHead, vc.votes[0].BlockHead, nodeID)
			return nil, nil
		} else {
			vc.votes = append(vc.votes, vote)
		}
	} else {
		vc.votes = append(vc.votes, vote)
	}

	if len(vc.votes) >= vc.dkgBls.Threshold() {
		signArr := make([][]byte, 0)
		for _, voteT := range vc.votes {
			sign := tpcmm.BytesCopy(voteT.Signature)
			signArr = append(signArr, sign)
		}
		msg := tpcmm.BytesCopy(vc.votes[0].BlockHead)
		return vc.produceAggSign(msg, signArr)
	} else {
		vc.log.Infof("Received vote count=%d, required=%d, self node %s", len(vc.votes), vc.dkgBls.Threshold(), nodeID)
	}

	return nil, nil
}

func (vc *consensusVoteCollector) produceAggSign(msg []byte, signArr [][]byte) (tpcrtypes.Signature, error) {
	return vc.dkgBls.RecoverSig(msg, signArr)
}

func (vc *consensusVoteCollector) updateDKGBls(dkgBls DKGBls) {
	vc.dkgBls = dkgBls
}

func (vc *consensusVoteCollector) reset() {
	vc.Lock()
	defer vc.Unlock()

	vc.votes = vc.votes[:0]
}
