package consensus

import (
	"context"
	"fmt"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
)

type dkgMessageDeliverI interface {
	updateCandNodeIDs(propCandNodeIDs []string, valCandNodeIDs []string)

	deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error

	deliverDKGDealMessage(ctx context.Context, nodeID string, msg *DKGDealMessage) error

	deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error

	deliverDKGFinishedMessage(ctx context.Context, msg *DKGFinishedMessage) error
}

type dkgMessageDeliver struct {
	log             tplog.Logger
	nodeID          string
	priKey          tpcrtypes.PrivateKey
	strategy        DeliverStrategy
	network         tpnet.Network
	ledger          ledger.Ledger
	marshaler       codec.Marshaler
	cryptService    tpcrt.CryptService
	propCandNodeIDs []string
	valCandNodeIDs  []string
}

func NewDkgMessageDeliver(log tplog.Logger, nodeID string, priKey tpcrtypes.PrivateKey, strategy DeliverStrategy, network tpnet.Network, marshaler codec.Marshaler, cryptService tpcrt.CryptService, ledger ledger.Ledger) dkgMessageDeliverI {
	return &dkgMessageDeliver{
		log:          log,
		nodeID:       nodeID,
		priKey:       priKey,
		strategy:     strategy,
		network:      network,
		ledger:       ledger,
		marshaler:    marshaler,
		cryptService: cryptService,
	}
}

func (md *dkgMessageDeliver) updateCandNodeIDs(propCandNodeIDs []string, valCandNodeIDs []string) {
	md.propCandNodeIDs = propCandNodeIDs
	md.valCandNodeIDs = valCandNodeIDs
}

func (md *dkgMessageDeliver) deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error {
	sigData, err := md.cryptService.Sign(md.priKey, msg.PartPubKey)
	if err != nil {
		md.log.Errorf("Sign err for commit msg: %v", err)
		return err
	}

	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't get public key from private key: %v", err)
		return err
	}

	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGPartPubKeyMessage marshal err: %v", err)
		return err
	}

	propCtx := ctx
	ValCtx := ctx
	switch md.strategy {
	case DeliverStrategy_Specifically:
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_PeerList, md.propCandNodeIDs)
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_PeerList, md.valCandNodeIDs)
	}

	propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(propCtx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_PartPubKey, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG part pub key message to propose network failed: err=%v", err)
	}

	ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(propCtx, md.log, md.marshaler, md.network, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_PartPubKey, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG part pub key message to validate network failed: err=%v", err)
	}

	return err
}

func (md *dkgMessageDeliver) deliverDKGDealMessage(ctx context.Context, nodeID string, msg *DKGDealMessage) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	nodeInfo, err := csStateRN.GetNode(nodeID)
	if err != nil {
		md.log.Errorf("Can't get node info: %v", err)
		return err
	}

	forwardProtocol := ""
	if nodeInfo.Role&tpcmm.NodeRole_Proposer == tpcmm.NodeRole_Proposer {
		forwardProtocol = tpnetprotoc.ForwardPropose_Msg
	} else if nodeInfo.Role&tpcmm.NodeRole_Validator == tpcmm.NodeRole_Validator {
		forwardProtocol = tpnetprotoc.FrowardValidate_Msg
	} else {
		err = fmt.Errorf("Invalid deal dest nodeID %s, role=%d", nodeID, nodeInfo.Role)
		md.log.Error(err.Error())
		return err
	}

	sigData, err := md.cryptService.Sign(md.priKey, msg.DealData)
	if err != nil {
		md.log.Errorf("Sign err for commit msg: %v", err)
		return err
	}

	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't get public key from private key: %v", err)
		return err
	}

	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGDealMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, []string{nodeID})
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(ctx, md.log, md.marshaler, md.network, forwardProtocol, MOD_NAME, ConsensusMessage_DKGDeal, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG deal message to network failed: err=%v", err)
	}

	return err
}

func (md *dkgMessageDeliver) deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error {
	sigData, err := md.cryptService.Sign(md.priKey, msg.RespData)
	if err != nil {
		md.log.Errorf("Sign err for commit msg: %v", err)
		return err
	}

	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't get public key from private key: %v", err)
		return err
	}

	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGDealRespMessage marshal err: %v", err)
		return err
	}

	propCtx := ctx
	ValCtx := ctx
	switch md.strategy {
	case DeliverStrategy_Specifically:
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_PeerList, md.propCandNodeIDs)
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_PeerList, md.valCandNodeIDs)
	}

	if propCtx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		err = deliverSendCommon(propCtx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_DKGDealResp, msgBytes)
		if err != nil {
			md.log.Errorf("Send deal resp message to propose network failed: err=%v", err)
		}
	}

	if ValCtx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		err = deliverSendCommon(ValCtx, md.log, md.marshaler, md.network, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_DKGDealResp, msgBytes)
		if err != nil {
			md.log.Errorf("Send deal resp message to validate network failed: err=%v", err)
		}
	}

	return err
}

func (md *dkgMessageDeliver) deliverDKGFinishedMessage(ctx context.Context, msg *DKGFinishedMessage) error {
	sigData, err := md.cryptService.Sign(md.priKey, msg.PubPolyCommit)
	if err != nil {
		md.log.Errorf("Sign err for commit msg: %v", err)
		return err
	}

	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't get public key from private key: %v", err)
		return err
	}

	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("DKGFinishedMessage marshal err: %v", err)
		return err
	}

	propCtx := ctx
	ValCtx := ctx
	switch md.strategy {
	case DeliverStrategy_Specifically:
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_PeerList, md.propCandNodeIDs)
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_PeerList, md.valCandNodeIDs)
	}

	if propCtx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		err = deliverSendCommon(propCtx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_DKGFinished, msgBytes)
		if err != nil {
			md.log.Errorf("Send dkg finished message to propose network failed: err=%v", err)
		}
	}

	if ValCtx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		err = deliverSendCommon(ValCtx, md.log, md.marshaler, md.network, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_DKGFinished, msgBytes)
		if err != nil {
			md.log.Errorf("Send dkg finished message to validate network failed: err=%v", err)
		}
	}

	return err
}
