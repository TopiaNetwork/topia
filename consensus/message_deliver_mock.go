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
	nodeID           string
	initPubKeys      []string
	partPKSync       sync.RWMutex
	dealMsgsync      sync.RWMutex
	dealRespMsgSync  sync.RWMutex
	partPubKeyChMap  map[string]chan *DKGPartPubKeyMessage
	dealMsgChMap     map[string]chan *DKGDealMessage
	dealRespMsgChMap map[string]chan *DKGDealRespMessage
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

func (md *messageDeliverMock) deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error {
	md.partPKSync.Lock()
	defer md.partPKSync.Unlock()

	for nodeIndex, partPubKeyCh := range md.partPubKeyChMap {
		if nodeIndex != md.nodeID {
			md.log.Infof("DKG part pub key message %s deliver to %s", md.nodeID, nodeIndex)
			partPubKeyCh <- msg
		}
	}

	return nil
}

func (md *messageDeliverMock) deliverDKGDealMessage(ctx context.Context, nodeID string, msg *DKGDealMessage) error {
	md.dealMsgsync.Lock()
	defer md.dealMsgsync.Unlock()

	if dealMsgCh, ok := md.dealMsgChMap[nodeID]; ok {
		md.log.Infof("DKG deal message from %s deliver to %s", md.nodeID, nodeID)
		dealMsgCh <- msg
	}

	return nil
}

func (md *messageDeliverMock) deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error {
	md.dealRespMsgSync.Lock()
	md.dealRespMsgSync.Unlock()

	for nodeIndex, dealRespMsgCh := range md.dealRespMsgChMap {
		if nodeIndex != md.nodeID {
			md.log.Infof("DKG deal response message %s deliver to %s", md.nodeID, nodeIndex)
			dealRespMsgCh <- msg
		}
	}

	return nil
}

func (md *messageDeliverMock) updateDKGBls(dkgBls DKGBls) {
	md.dkgBls = dkgBls
}
