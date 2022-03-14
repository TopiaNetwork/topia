package consensus

import (
	"context"
	"fmt"

	"github.com/TopiaNetwork/topia/codec"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
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

	deliverCommitMessage(ctx context.Context, msg *CommitMessage) error

	deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error

	deliverDKGDealMessage(ctx context.Context, pubKey string, msg *DKGDealMessage) error

	deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error

	updateDKGBls(dkgBls DKGBls)
}

type messageDeliver struct {
	log       tplog.Logger
	priKey    tpcrtypes.PrivateKey
	strategy  DeliverStrategy
	network   network.Network
	marshaler codec.Marshaler
	csState   consensusStore
	selector  *roleSelectorVRF
	dkgBls    DKGBls
}

func newMessageDeliver(log tplog.Logger, priKey tpcrtypes.PrivateKey, strategy DeliverStrategy, network network.Network, marshaler codec.Marshaler, crypt tpcrt.CryptService, csState consensusStore) *messageDeliver {
	return &messageDeliver{
		log:       log,
		strategy:  strategy,
		network:   network,
		marshaler: marshaler,
		selector:  newLeaderSelectorVRF(log, crypt, csState),
	}
}

func (md *messageDeliver) deliverPreparePackagedMessage(ctx context.Context, msg *PreparePackedMessage) error {
	sigData, pubKey, err := md.dkgBls.Sign(msg.TxRoot)
	if err != nil {
		md.log.Errorf("DKG sign PreparePackedMessage err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("PreparePackedMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := md.csState.GetActiveExecutorIDs()
		if err != nil {
			md.log.Errorf("Can't get all active executor nodes: err=%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDs)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error {
	sigData, pubKey, err := md.dkgBls.Sign(msg.Block)
	if err != nil {
		md.log.Errorf("DKG sign ProposeMessage err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := md.csState.GetAllConsensusNodes()
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

func (md *messageDeliver) getVoterCollector(voterRound uint64) (string, []byte, error) {
	lastBlock, err := md.csState.GetLatestBlock()
	if err != nil {
		md.log.Errorf("Can't get the latest block: %v", err)
		return "", nil, err
	}

	if lastBlock.Head.Round != voterRound-1 {
		err := fmt.Errorf("Stale vote round: %d", voterRound)
		md.log.Errorf(err.Error())
		return "", nil, err
	}
	roundInfo := &RoundInfo{
		Epoch:        lastBlock.Head.Epoch,
		LastRoundNum: lastBlock.Head.Round,
		CurRoundNum:  voterRound,
		Proof: &ConsensusProof{
			ParentBlockHash: lastBlock.Head.ParentBlockHash,
			Height:          lastBlock.Head.Height,
			AggSign:         lastBlock.Head.VoteAggSignature,
		},
	}

	selVoteColectors, vrfProof, err := md.selector.Select(RoleSelector_VoteCollector, roundInfo, md.priKey, 1)
	if len(selVoteColectors) != 1 {
		err := fmt.Errorf("Expect vote collector count 1, got %d", len(selVoteColectors))
		md.log.Errorf(err.Error())
		return "", nil, err
	}

	return selVoteColectors[0].nodeID, vrfProof, err
}

func (md *messageDeliver) deliverVoteMessage(ctx context.Context, msg *VoteMessage) error {
	sigData, pubKey, err := md.dkgBls.Sign(msg.Block)
	if err != nil {
		md.log.Errorf("DKG sign VoteMessage err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}

	peerID, vrfProof, err := md.getVoterCollector(msg.Round)
	if err != nil {
		md.log.Errorf("Can't get the next leader: err=%v", err)
		return err
	}

	msg.VoterProof = vrfProof

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerID)
	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.AsyncSendProtocolID, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send vote message to network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverCommitMessage(ctx context.Context, msg *CommitMessage) error {
	sigData, pubKey, err := md.dkgBls.Sign(msg.Block)
	if err != nil {
		md.log.Errorf("DKG sign CommitMessage err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := md.csState.GetAllConsensusNodes()
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

func (md *messageDeliver) deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error {
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGPartPubKeyMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := md.csState.GetAllConsensusNodes()
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
		peerIDs, err := md.csState.GetAllConsensusNodes()
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

func (md *messageDeliver) updateDKGBls(dkgBls DKGBls) {
	md.dkgBls = dkgBls
}
