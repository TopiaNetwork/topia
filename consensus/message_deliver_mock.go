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
	finishedMsgSync  sync.RWMutex
	partPubKeyChMap  map[string]chan *DKGPartPubKeyMessage
	dealMsgChMap     map[string]chan *DKGDealMessage
	dealRespMsgChMap map[string]chan *DKGDealRespMessage
	finishedMsgChMap map[string]chan *DKGFinishedMessage
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

func (md *messageDeliverMock) deliverPreparePackagedMessageExeIndication(ctx context.Context, launcherID string, msg *PreparePackedMessageExeIndication) error {
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

func (md *messageDeliverMock) deliverResultValidateRespMessage(actorCtx actor.Context, msg *ExeResultValidateRespMessage, err error) error {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) deliverBestProposeMessage(ctx context.Context, msg *BestProposeMessage) error {
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
	defer md.dealRespMsgSync.Unlock()

	for nodeIndex, dealRespMsgCh := range md.dealRespMsgChMap {
		if nodeIndex != md.nodeID {
			md.log.Infof("DKG deal response message %s deliver to %s", md.nodeID, nodeIndex)
			dealRespMsgCh <- msg
		}
	}

	return nil
}

func (md *messageDeliverMock) deliverDKGFinishedMessage(ctx context.Context, msg *DKGFinishedMessage) error {
	md.finishedMsgSync.Lock()
	defer md.finishedMsgSync.Unlock()

	for nodeIndex, finishedMsgCh := range md.finishedMsgChMap {
		if nodeIndex != md.nodeID {
			md.log.Infof("DKG finished message %s deliver to %s", md.nodeID, nodeIndex)
			finishedMsgCh <- msg
		}
	}

	return nil
}

func (md *messageDeliverMock) updateCandNodeIDs(propCandNodeIDs []string, valCandNodeIDs []string) {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) setEpochService(epService EpochService) {
	//TODO implement me
	panic("implement me")
}

func (md *messageDeliverMock) updateDKGBls(dkgBls DKGBls) {
	md.dkgBls = dkgBls
}

func (md *messageDeliverMock) isReady() bool {
	return true
}
