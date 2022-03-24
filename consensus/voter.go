package consensus

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
)

type consensusVoter struct {
	log            tplog.Logger
	proposeMsgChan chan *ProposeMessage
	deliver        *messageDeliver
}

func newConsensusVoter(log tplog.Logger, proposeMsgChan chan *ProposeMessage, deliver *messageDeliver) *consensusVoter {
	return &consensusVoter{
		log:            log,
		proposeMsgChan: proposeMsgChan,
		deliver:        deliver,
	}
}

func (v *consensusVoter) start(ctx context.Context) {
	go func() {
		for {
			select {
			case propMsg := <-v.proposeMsgChan:
				voteMsg, err := v.produceVoteMsg(propMsg)
				if err != nil {
					v.log.Errorf("Can't produce vote msg: err=%v", err)
					continue
				}
				if err = v.deliver.deliverVoteMessage(ctx, voteMsg); err != nil {
					v.log.Errorf("Consensus deliver vote message err: %v", err)
				}
			case <-ctx.Done():
				v.log.Info("Consensus voter exit")
				return
			}
		}
	}()
}

func (v *consensusVoter) produceVoteMsg(msg *ProposeMessage) (*VoteMessage, error) {
	return &VoteMessage{
		ChainID: msg.ChainID,
		Version: CONSENSUS_VER,
		Epoch:   msg.Epoch,
		Round:   msg.Round,
		Proof:   msg.Proof,
		Block:   msg.BlockHead,
	}, nil
}
