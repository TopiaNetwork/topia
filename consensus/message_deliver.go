package consensus

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/TopiaNetwork/topia/chain"
	"math/big"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
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

	deliverResultValidateReqMessage(ctx context.Context, msg *ExeResultValidateReqMessage) (*ExeResultValidateRespMessage, error)

	deliverResultValidateRespMessage(actorCtx actor.Context, msg *ExeResultValidateRespMessage) error

	deliverVoteMessage(ctx context.Context, msg *VoteMessage, proposer string) error

	deliverCommitMessage(ctx context.Context, msg *CommitMessage) error

	deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error

	deliverDKGDealMessage(ctx context.Context, nodeID string, msg *DKGDealMessage) error

	deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error

	updateDKGBls(dkgBls DKGBls)
}

type messageDeliver struct {
	log          tplog.Logger
	nodeID       string
	priKey       tpcrtypes.PrivateKey
	strategy     DeliverStrategy
	network      network.Network
	ledger       ledger.Ledger
	marshaler    codec.Marshaler
	cryptService tpcrt.CryptService
	dkgBls       DKGBls
}

func newMessageDeliver(log tplog.Logger, nodeID string, priKey tpcrtypes.PrivateKey, strategy DeliverStrategy, network network.Network, marshaler codec.Marshaler, cryptService tpcrt.CryptService, ledger ledger.Ledger) *messageDeliver {
	return &messageDeliver{
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

func (md *messageDeliver) deliverNetwork() network.Network {
	return md.network
}

func (md *messageDeliver) deliverSendCommon(ctx context.Context, protocolID string, moduleName string, msgType ConsensusMessage_Type, dataBytes []byte) error {
	csMsg := &ConsensusMessage{
		MsgType: msgType,
		Data:    dataBytes,
	}

	csMsgBytes, err := md.marshaler.Marshal(csMsg)
	if err != nil {
		md.log.Errorf("ConsensusMessage marshal err: type=%d, err=%v", msgType.String(), err)
		return err
	}

	return md.network.Send(ctx, protocolID, moduleName, csMsgBytes)
}

func (md *messageDeliver) deliverSendWithRespCommon(ctx context.Context, protocolID string, moduleName string, msgType ConsensusMessage_Type, dataBytes []byte) ([][]byte, error) {
	csMsg := &ConsensusMessage{
		MsgType: msgType,
		Data:    dataBytes,
	}

	csMsgBytes, err := md.marshaler.Marshal(csMsg)
	if err != nil {
		md.log.Errorf("ConsensusMessage marshal err: type=%d, err=%v", msgType.String(), err)
		return nil, err
	}

	return md.network.SendWithResponse(ctx, protocolID, moduleName, csMsgBytes)
}

func (md *messageDeliver) deliverPreparePackagedMessageExe(ctx context.Context, msg *PreparePackedMessageExe) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	sigData, err := md.cryptService.Sign(md.priKey, msg.TxsData())
	if err != nil {
		md.log.Errorf("Can't sign when deliver PreparePackedMessageExe err: %v", err)
		return err
	}
	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't convert to pub key when deliverPreparePackedMessageExe err: %v", err)
		return err
	}

	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("Deliver PreparePackedMessageExe marshal err: %v", err)
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
	err = md.deliverSendCommon(ctx, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, ConsensusMessage_PrepareExe, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to execute network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverPreparePackagedMessageProp(ctx context.Context, msg *PreparePackedMessageProp) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	sigData, err := md.cryptService.Sign(md.priKey, msg.TxHashsData())
	if err != nil {
		md.log.Errorf("Can't sign PreparePackedMessageProp when deliverPreparePackagedMessageProp err: %v", err)
		return err
	}
	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't convert to pub key when deliverPreparePackagedMessageProp err: %v", err)
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
	err = md.deliverSendCommon(ctx, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_PrepareProp, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to propose network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)

	ctxProposer := ctx
	ctxValidator := ctx
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerActiveProposerIDs, err := csStateRN.GetActiveProposerIDs()
		if err != nil {
			md.log.Errorf("Can't get all active proposer nodes: err=%v", err)
			return err
		}
		peerActiveProposerIDs = tpcmm.RemoveIfExistString(md.nodeID, peerActiveProposerIDs)

		ctxProposer = context.WithValue(ctxProposer, tpnetcmn.NetContextKey_PeerList, peerActiveProposerIDs)

		peerActiveValidatorIDs, err := csStateRN.GetActiveValidatorIDs()
		if err != nil {
			md.log.Errorf("Can't get all active validator nodes: err=%v", err)
			return err
		}
		ctxValidator = context.WithValue(ctxValidator, tpnetcmn.NetContextKey_PeerList, peerActiveValidatorIDs)
	}

	sigData, err := md.cryptService.Sign(md.priKey, msg.BlockHead)
	if err != nil {
		md.log.Errorf("Sign err for propose msg: %v", err)
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
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}
	err = md.deliverSendCommon(ctxProposer, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_Propose, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to proposer network failed: err=%v", err)
		return nil
	}

	sigData, pubKey, err = md.dkgBls.Sign(msg.BlockHead)
	if err != nil {
		md.log.Errorf("DKG sign propose msg err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey
	msgBytes, err = md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}
	err = md.deliverSendCommon(ctxValidator, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_Propose, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to validator network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverResultValidateReqMessage(ctx context.Context, msg *ExeResultValidateReqMessage) (*ExeResultValidateRespMessage, error) {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)

	var randExecutorID string
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := csStateRN.GetActiveExecutorIDs()
		if err != nil {
			md.log.Errorf("Can't get all active executor nodes: err=%v", err)
			return nil, err
		}

		maxIndex := big.NewInt(int64(len(peerIDs) - 1))
		randIndex, err := rand.Int(rand.Reader, maxIndex)
		if err != nil {
			md.log.Errorf("Can't get rand active executor nodes index: err=%v", err)
			return nil, err
		}
		randExecutorID = peerIDs[randIndex.Uint64()]
		md.log.Debugf("Rand active executor nodes: %d, ", randIndex.Uint64(), peerIDs[randIndex.Uint64()])

		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, []string{randExecutorID})
	}

	sigData, err := md.cryptService.Sign(md.priKey, msg.TxAndResultHashsData())
	if err != nil {
		md.log.Errorf("Sign err for execution result validate request msg: %v", err)
		return nil, err
	}

	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't get public key from private key: %v", err)
		return nil, err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ExeResultValidateReqMessage marshal err: %v", err)
		return nil, err
	}
	resp, err := md.deliverSendWithRespCommon(ctx, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, ConsensusMessage_ExeRSValidateReq, msgBytes)
	if err != nil {
		md.log.Errorf("Send execution result validate request message to executor network failed: err=%v", err)
		return nil, err
	}

	if len(resp) <= 0 {
		err = fmt.Errorf("Received execution result validate request resp %d from executor %s", len(resp), randExecutorID)
		return nil, err
	}

	var validateResp ExeResultValidateRespMessage
	err = md.marshaler.Unmarshal(resp[0], &validateResp)
	if err != nil {
		err = fmt.Errorf("Can't unmarshal received execution result validate request from executor %s: %v", randExecutorID, err)
		return nil, err
	}

	return &validateResp, err
}

func (md *messageDeliver) deliverResultValidateRespMessage(actorCtx actor.Context, msg *ExeResultValidateRespMessage) error {
	msg.Executor = []byte(md.nodeID)

	sigData, err := md.cryptService.Sign(md.priKey, msg.TxAndResultProofsData())
	if err != nil {
		md.log.Errorf("Sign err for execution result validate response msg: %v", err)
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
		md.log.Errorf("ExeResultValidateRespMessage marshal err: %v", err)
		return err
	}

	actorCtx.Respond(msgBytes)

	return nil
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
		err := fmt.Errorf("Stale vote epoch: %d", voterRound)
		md.log.Errorf(err.Error())
		return "", nil, err
	}

	selVoteColectors, vrfProof, err := newLeaderSelectorVRF(md.log, md.cryptService).Select(RoleSelector_VoteCollector, 0, md.priKey, csStateRN, 1)
	if len(selVoteColectors) != 1 {
		err := fmt.Errorf("Expect vote collector count 1, got %d", len(selVoteColectors))
		md.log.Errorf("%v", err)
		return "", nil, err
	}

	return selVoteColectors[0].nodeID, vrfProof, err
}

func (md *messageDeliver) deliverVoteMessage(ctx context.Context, msg *VoteMessage, proposer string) error {
	sigData, pubKey, err := md.dkgBls.Sign(msg.BlockHead)
	if err != nil {
		md.log.Errorf("DKG sign VoteMessage err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("VoteMessage marshal err: %v", err)
		return err
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, []string{proposer})
	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.deliverSendCommon(ctx, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_Vote, msgBytes)
	if err != nil {
		md.log.Errorf("Send vote message to proposer network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverCommitMessage(ctx context.Context, msg *CommitMessage) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	sigData, err := md.cryptService.Sign(md.priKey, msg.BlockHead)
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
		md.log.Errorf("CommitMessage marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs, err := csStateRN.GetActiveExecutorIDs()
		if err != nil {
			md.log.Errorf("Can't get all active executor nodes: err=%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDs)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.deliverSendCommon(ctx, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, ConsensusMessage_Commit, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to executor network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverDKGPartPubKeyMessage(ctx context.Context, msg *DKGPartPubKeyMessage) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

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
		peerProposerIDs, err := csStateRN.GetActiveProposerIDs()
		if err != nil {
			md.log.Errorf("Can't get all active proposer nodes: err=%v", err)
			return err
		}
		peerProposerIDs = tpcmm.RemoveIfExistString(md.nodeID, peerProposerIDs)
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_PeerList, peerProposerIDs)

		peerValidatorIDs, err := csStateRN.GetActiveValidatorIDs()
		if err != nil {
			md.log.Errorf("Can't get all active validator nodes: err=%v", err)
			return err
		}
		peerValidatorIDs = tpcmm.RemoveIfExistString(md.nodeID, peerValidatorIDs)
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_PeerList, peerValidatorIDs)
	}

	propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.deliverSendCommon(propCtx, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_PartPubKey, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG part pub key message to propose network failed: err=%v", err)
	}

	ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = md.deliverSendCommon(propCtx, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_PartPubKey, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG part pub key message to validate network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverDKGDealMessage(ctx context.Context, nodeID string, msg *DKGDealMessage) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

	nodeInfo, err := csStateRN.GetNode(nodeID)
	if err != nil {
		md.log.Errorf("Can't get node info: %v", err)
		return err
	}

	forwardProtocol := ""
	if nodeInfo.Role&chain.NodeRole_Proposer == chain.NodeRole_Proposer {
		forwardProtocol = tpnetprotoc.ForwardPropose_Msg
	} else if nodeInfo.Role&chain.NodeRole_Validator == chain.NodeRole_Validator {
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
	err = md.deliverSendCommon(ctx, forwardProtocol, MOD_NAME, ConsensusMessage_DKGDeal, msgBytes)
	if err != nil {
		md.log.Errorf("Send DKG deal message to network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverDKGDealRespMessage(ctx context.Context, msg *DKGDealRespMessage) error {
	csStateRN := state.CreateCompositionStateReadonly(md.log, md.ledger)
	defer csStateRN.Stop()

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
		peerProposerIDs, err := csStateRN.GetActiveProposerIDs()
		if err != nil {
			md.log.Errorf("Can't get all active proposer nodes: err=%v", err)
			return err
		}
		peerProposerIDs = tpcmm.RemoveIfExistString(md.nodeID, peerProposerIDs)
		if len(peerProposerIDs) > 0 {
			propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_PeerList, peerProposerIDs)
		}

		peerValidatorIDs, err := csStateRN.GetActiveValidatorIDs()
		if err != nil {
			md.log.Errorf("Can't get all active validator nodes: err=%v", err)
			return err
		}
		peerValidatorIDs = tpcmm.RemoveIfExistString(md.nodeID, peerValidatorIDs)
		if len(peerValidatorIDs) > 0 {
			ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_PeerList, peerValidatorIDs)
		}
	}

	if propCtx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		err = md.deliverSendCommon(propCtx, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_DKGDealResp, msgBytes)
		if err != nil {
			md.log.Errorf("Send deal resp message to propose network failed: err=%v", err)
		}
	}

	if ValCtx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		err = md.deliverSendCommon(ValCtx, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_DKGDealResp, msgBytes)
		if err != nil {
			md.log.Errorf("Send deal resp message to validate network failed: err=%v", err)
		}
	}

	return err
}

func (md *messageDeliver) updateDKGBls(dkgBls DKGBls) {
	md.dkgBls = dkgBls
}
