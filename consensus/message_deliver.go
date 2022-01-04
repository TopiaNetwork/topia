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

func (md *messageDeliver) deliverProposeMesage(ctx context.Context, msg *ProposeMessage) error {
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

func (md *messageDeliver) deliverVoteMesage(ctx context.Context, msg *VoteMessage) error {
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
