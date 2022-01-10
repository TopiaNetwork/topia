package consensus

import (
	"context"

	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
)

type DeliverStrategy byte

const (
	DeliverStrategy_Unknown = iota
	DeliverStrategy_All
	DeliverStrategy_Specifically
)

type messageDeliverI interface {
	deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error
	deliverVoteMessage(ctx context.Context, msg *VoteMessage) error
	deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error
	deliverDKGDealMessage(ctx context.Context, pubKey string, msg *DKGDealMessage) error
	deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error
}

type messageDeliver struct {
	log       tplog.Logger
	strategy  DeliverStrategy
	network   network.Network
	marshaler codec.Marshaler
}

func newMessageDeliver(log tplog.Logger, strategy DeliverStrategy, network network.Network, marshaler codec.Marshaler) *messageDeliver {
	return &messageDeliver{
		log:       log,
		strategy:  strategy,
		network:   network,
		marshaler: marshaler,
	}
}

func (md *messageDeliver) getAllConsensusNodes() ([]string, error) {
	return nil, nil
}

func (md *messageDeliver) deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error {
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := md.getAllConsensusNodes()
		if err != nil {
			md.log.Errorf("Can't get all consensus nodes: err=%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDs)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.AsyncSendProtocolID, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) getNextLeader() (string, error) {
	return "", nil
}

func (md *messageDeliver) deliverVoteMessage(ctx context.Context, msg *VoteMessage) error {
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}

	peerID, err := md.getNextLeader()
	if err != nil {
		md.log.Errorf("Can't get the next leader: err=%v", err)
		return err
	}
	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerID)
	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.AsyncSendProtocolID, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send vote message to network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error {
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGPartPubKeyMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := md.getAllConsensusNodes()
		if err != nil {
			md.log.Errorf("Can't get all consensus nodes: err=%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDs)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.AsyncSendProtocolID, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG deal message to network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) getPeerByDKGPubKey(dkgPubKey string) (string, error) {
	return "", nil
}

func (md *messageDeliver) deliverDKGDealMessage(ctx context.Context, pubKey string, msg *DKGDealMessage) error {
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGDealMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerID, err := md.getPeerByDKGPubKey(pubKey)
		if err != nil {
			md.log.Errorf("Can't get all consensus nodes: err=%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerID)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.AsyncSendProtocolID, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG deal message to network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error {
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGDealRespMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := md.getAllConsensusNodes()
		if err != nil {
			md.log.Errorf("Can't get all consensus nodes: err=%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDs)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.AsyncSendProtocolID, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG deal response message to network failed: err=%v", err)
	}

	return err
}
