package consensus

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/state"

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
	deliverNetwork() network.Network

	deliverPreparePackagedMessageExe(ctx context.Context, msg *PreparePackedMessageExe) error

	deliverPreparePackagedMessageProp(ctx context.Context, msg *PreparePackedMessageProp) error

	deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error

	deliverVoteMessage(ctx context.Context, msg *VoteMessage) error

	deliverCommitMessage(ctx context.Context, msg *CommitMessage) error

	deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error

	deliverDKGDealMessage(ctx context.Context, pubKey string, msg *DKGDealMessage) error

	deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error

	updateDKGBls(dkgBls DKGBls)
}

type messageDeliver struct {
	log          tplog.Logger
	priKey       tpcrtypes.PrivateKey
	strategy     DeliverStrategy
	network      network.Network
	ledger       ledger.Ledger
	marshaler    codec.Marshaler
	cryptService tpcrt.CryptService
	dkgBls       DKGBls
}

func newMessageDeliver(log tplog.Logger, priKey tpcrtypes.PrivateKey, strategy DeliverStrategy, network network.Network, marshaler codec.Marshaler, crypt tpcrt.CryptService, ledger ledger.Ledger) *messageDeliver {
	return &messageDeliver{
		log:       log,
		priKey:    priKey,
		strategy:  strategy,
		network:   network,
		ledger:    ledger,
		marshaler: marshaler,
	}
}

func (md *messageDeliver) deliverNetwork() network.Network {
	return md.network
}

func (md *messageDeliver) deliverPreparePackagedMessageExe(ctx context.Context, msg *PreparePackedMessageExe) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	sigData, pubKey, err := md.dkgBls.Sign(msg.TxsData())
	if err != nil {
		md.log.Errorf("DKG sign PreparePackedMessageExe err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("PreparePackedMessageExe marshal err: %v", err)
		return err
	}

	var peerIDsExecutor []string
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDsExecutor, err = csStateRN.GetActiveExecutorIDs()
		if err != nil {
			md.log.Errorf("Can't get all active executor nodes: err=%v", err)
			return err
		}
		if len(peerIDsExecutor) == 0 {
			err := fmt.Errorf("Zero active executor node")
			md.log.Errorf("%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDsExecutor)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to execute network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverPreparePackagedMessageProp(ctx context.Context, msg *PreparePackedMessageProp) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	sigData, pubKey, err := md.dkgBls.Sign(msg.TxHashsData())
	if err != nil {
		md.log.Errorf("DKG sign PreparePackedMessageProp err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("PreparePackedMessageProp marshal err: %v", err)
		return err
	}

	var peerIDsProposer []string
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDsProposer, err = csStateRN.GetActiveProposerIDs()
		if err != nil {
			md.log.Errorf("Can't get all active proposer nodes: err=%v", err)
			return err
		}
		if len(peerIDsProposer) == 0 {
			err := fmt.Errorf("Zero active proposer node")
			md.log.Errorf("%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDsProposer)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.network.Send(ctx, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to propose network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

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
		peerIDs, err := csStateRN.GetAllConsensusNodes()
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
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	lastBlock, err := csStateRN.GetLatestBlock()
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

	selVoteColectors, vrfProof, err := newLeaderSelectorVRF(md.log, md.cryptService, csStateRN).Select(RoleSelector_VoteCollector, roundInfo, md.priKey, 1)
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
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

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
		peerIDs, err := csStateRN.GetAllConsensusNodes()
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
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGPartPubKeyMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := csStateRN.GetAllConsensusNodes()
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
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGDealRespMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := csStateRN.GetAllConsensusNodes()
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
