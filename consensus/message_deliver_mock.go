package consensus

import (
	"context"
	"github.com/AsynkronIT/protoactor-go/actor"
	"sync"

	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/network"
)

type messageDeliverMock struct {
	log              tplog.Logger
	index            int
	initPubKeys      []string
	partPKSync       sync.RWMutex
	dealMsgsync      sync.RWMutex
	dealRespMsgSync  sync.RWMutex
	partPubKeyChMap  map[int]chan *DKGPartPubKeyMessage
	dealMsgChMap     map[int]chan *DKGDealMessage
	dealRespMsgChMap map[int]chan *DKGDealRespMessage
	dkgBls           DKGBls
}

func (md *messageDeliverMock) deliverNetwork() network.Network {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverPreparePackagedMessageExe(ctx context.Context, msg *PreparePackedMessageExe) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverPreparePackagedMessageProp(ctx context.Context, msg *PreparePackedMessageProp) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverResultValidateReqMessage(ctx context.Context, msg *ExeResultValidateReqMessage) (*ExeResultValidateRespMessage, error) {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverResultValidateRespMessage(actorCtx actor.Context, msg *ExeResultValidateRespMessage) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverVoteMessage(ctx context.Context, msg *VoteMessage, proposer string) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverCommitMessage(ctx context.Context, msg *CommitMessage) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverCommitMessage(ctx context.Context, msg *CommitMessage) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error {
	md.partPKSync.Lock()
	defer md.partPKSync.Unlock()

	for index := 0; index < len(md.partPubKeyChMap); index++ {
		if index != md.index {
			md.log.Infof("DKG part pub key message %d deliver to %d", md.index, index)
			md.partPubKeyChMap[index] <- msg
		}
	}

	return nil
}

func (md *messageDeliverMock) deliverDKGDealMessage(ctx context.Context, pubKey string, msg *DKGDealMessage) error {
	md.dealMsgsync.Lock()
	defer md.dealMsgsync.Unlock()

	for index, pk := range md.initPubKeys {
		if pk == pubKey {
			md.log.Infof("DKG deal message from %d deliver to %d", md.index, index)
			md.dealMsgChMap[index] <- msg
		}
	}

	return nil
}

func (md *messageDeliverMock) deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error {
	md.dealRespMsgSync.Lock()
	md.dealRespMsgSync.Unlock()

	for index := 0; index < len(md.dealRespMsgChMap); index++ {
		if index != md.index {
			md.log.Infof("DKG deal response message %d deliver to %d", md.index, index)
			md.dealRespMsgChMap[index] <- msg
		}
	}

	return nil
}

func (md *messageDeliverMock) updateDKGBls(dkgBls DKGBls) {
	md.dkgBls = dkgBls
}
