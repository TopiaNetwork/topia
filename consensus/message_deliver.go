package consensus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"go.uber.org/atomic"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/consensus/vrf"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	tpnetmsg "github.com/TopiaNetwork/topia/network/message"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
)

type DeliverStrategy byte

const (
	DeliverStrategy_Unknown = iota
	DeliverStrategy_All
	DeliverStrategy_Specifically
)

const (
	TxsResultValidity_MaxExecutor = 2
)

type messageDeliverI interface {
	deliverNetwork() tpnet.Network

	deliverConsensusDomainSelectedMessageExe(ctx context.Context, msg *ConsensusDomainSelectedMessage) error

	deliverPreparePackagedMessageExe(ctx context.Context, msg *PreparePackedMessageExe) error

	deliverPreparePackagedMessageExeIndication(ctx context.Context, launcherID string, msg *PreparePackedMessageExeIndication) error

	deliverPreparePackagedMessageProp(ctx context.Context, msg *PreparePackedMessageProp) error

	deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error

	deliverResultValidateReqMessage(ctx context.Context, msg *ExeResultValidateReqMessage) (*ExeResultValidateRespMessage, error)

	deliverResultValidateRespMessage(actorCtx actor.Context, msg *ExeResultValidateRespMessage, err error) error

	deliverBestProposeMessage(ctx context.Context, msg *BestProposeMessage) error

	deliverVoteMessage(ctx context.Context, msg *VoteMessage, proposer string) error

	deliverCommitMessage(ctx context.Context, msg *CommitMessage) error

	setEpochService(epService EpochService)

	updateDKGBls(dkgBls DKGBls)

	isReady() bool
}

type messageDeliver struct {
	log          tplog.Logger
	nodeID       string
	ready        atomic.Bool
	priKey       tpcrtypes.PrivateKey
	strategy     DeliverStrategy
	network      tpnet.Network
	ledger       ledger.Ledger
	epochService EpochService
	marshaler    codec.Marshaler
	cryptService tpcrt.CryptService
	dkgBls       DKGBls
}

func newMessageDeliver(log tplog.Logger, nodeID string, priKey tpcrtypes.PrivateKey, strategy DeliverStrategy, network tpnet.Network, marshaler codec.Marshaler, cryptService tpcrt.CryptService, ledger ledger.Ledger) *messageDeliver {
	msgDeliver := &messageDeliver{
		log:          log,
		nodeID:       nodeID,
		priKey:       priKey,
		strategy:     strategy,
		network:      network,
		ledger:       ledger,
		marshaler:    marshaler,
		cryptService: cryptService,
	}

	msgDeliver.ready.Store(false)

	return msgDeliver
}

func (md *messageDeliver) deliverNetwork() tpnet.Network {
	return md.network
}

func deliverSendCommon(ctx context.Context, log tplog.Logger, marshaler codec.Marshaler, network tpnet.Network, protocolID string, moduleName string, msgType ConsensusMessage_Type, dataBytes []byte) error {
	csMsg := &ConsensusMessage{
		MsgType: msgType,
		Data:    dataBytes,
	}

	csMsgBytes, err := marshaler.Marshal(csMsg)
	if err != nil {
		log.Errorf("ConsensusMessage marshal err: type=%d, err=%v", msgType.String(), err)
		return err
	}

	return network.Send(ctx, protocolID, moduleName, csMsgBytes)
}

func deliverSendWithRespCommon(ctx context.Context, selfNodeID string, log tplog.Logger, marshaler codec.Marshaler, network tpnet.Network, protocolID string, moduleName string, msgType ConsensusMessage_Type, dataBytes []byte) ([][]byte, error) {
	csMsg := &ConsensusMessage{
		MsgType: msgType,
		Data:    dataBytes,
	}

	csMsgBytes, err := marshaler.Marshal(csMsg)
	if err != nil {
		log.Errorf("ConsensusMessage marshal err: type=%d, err=%v", msgType.String(), err)
		return nil, err
	}

	var respBytes [][]byte
	var respErrs []string

	sendCycleMaxCount := 1
	for sendCycleMaxCount > 0 {
		sendCycleMaxCount--
		resps, err := network.SendWithResponse(ctx, protocolID, moduleName, csMsgBytes)
		if err != nil {
			return nil, err
		}
		_, respBytes, respErrs = tpnetmsg.ParseSendResp(resps)
		rtnBytes := [][]byte(nil)
		for i, respErr := range respErrs {
			if respErr == "" {
				rtnBytes = append(rtnBytes, respBytes[i])
			}
		}

		if len(rtnBytes) > 0 {
			return rtnBytes, nil
		}
	}

	return nil, fmt.Errorf("Network exception and can't get the final response: errs %v, protocolID %s, consensusMsg %s, self node %s", respErrs, protocolID, msgType.String(), selfNodeID)
}

func (md *messageDeliver) deliverConsensusDomainSelectedMessageExe(ctx context.Context, msg *ConsensusDomainSelectedMessage) error {
	sigData, err := md.cryptService.Sign(md.priKey, msg.DataBytes())
	if err != nil {
		md.log.Errorf("Can't sign when deliver PreparePackedMessageExe err: %v", err)
		return err
	}
	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't convert to pub key when deliver PreparePackedMessageExe err: %v", err)
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
		peerIDsExecutor = md.epochService.GetActiveExecutorIDs()
		if len(peerIDsExecutor) == 0 {
			err := fmt.Errorf("Zero active executor node")
			md.log.Errorf("%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDsExecutor)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(ctx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, ConsensusMessage_CSDomainSel, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to execute network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverPreparePackagedMessageExe(ctx context.Context, msg *PreparePackedMessageExe) error {
	sigData, err := md.cryptService.Sign(md.priKey, msg.TxsData())
	if err != nil {
		md.log.Errorf("Can't sign when deliver PreparePackedMessageExe err: %v", err)
		return err
	}
	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't convert to pub key when deliver PreparePackedMessageExe err: %v", err)
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
		peerIDsExecutor = md.epochService.GetActiveExecutorIDs()
		if len(peerIDsExecutor) == 0 {
			err := fmt.Errorf("Zero active executor node")
			md.log.Errorf("%v", err)
			return err
		}
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDsExecutor)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(ctx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, ConsensusMessage_PrepareExe, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to execute network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverPreparePackagedMessageExeIndication(ctx context.Context, launcherID string, msg *PreparePackedMessageExeIndication) error {
	sigData, err := md.cryptService.Sign(md.priKey, msg.DataBytes())
	if err != nil {
		md.log.Errorf("Can't sign when deliver PreparePackedMessageExeIndication err: %v", err)
		return err
	}
	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't convert to pub key when deliver PreparePackedMessageExeIndication err: %v", err)
		return err
	}

	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("Deliver PreparePackedMessageExeIndication marshal err: %v", err)
		return err
	}

	switch md.strategy {
	case DeliverStrategy_Specifically:
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, []string{launcherID})
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(ctx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, ConsensusMessage_PrepareExeIndic, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message indication to execute network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverPreparePackagedMessageProp(ctx context.Context, msg *PreparePackedMessageProp) error {
	sigData, err := md.cryptService.Sign(md.priKey, msg.TxHashsData())
	if err != nil {
		md.log.Errorf("Can't sign PreparePackedMessageProp when deliver PreparePackagedMessageProp err: %v", err)
		return err
	}
	pubKey, err := md.cryptService.ConvertToPublic(md.priKey)
	if err != nil {
		md.log.Errorf("Can't convert to pub key when deliver PreparePackagedMessageProp err: %v", err)
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
		peerIDsProposer = md.epochService.GetActiveProposerIDs()
		for len(peerIDsProposer) == 0 {
			time.Sleep(50 * time.Millisecond)
			peerIDsProposer = md.epochService.GetActiveProposerIDs()
		}
		md.log.Infof("Active proposer node: %v, state version %d", peerIDsProposer, msg.StateVersion)

		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDsProposer)
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(ctx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_PrepareProp, msgBytes)
	if err != nil {
		md.log.Errorf("Send prepare packed message to propose network failed: err=%v", err)
		return err
	}

	return nil
}

func (md *messageDeliver) deliverProposeMessage(ctx context.Context, msg *ProposeMessage) error {
	if msg == nil {
		return errors.New("Nil ProposeMessage for delivering")
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)

	ctxProposer := ctx
	ctxValidator := ctx
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerActiveProposerIDs := md.epochService.GetActiveProposerIDs()
		peerActiveProposerIDs = tpcmm.RemoveIfExistString(md.nodeID, peerActiveProposerIDs)
		ctxProposer = context.WithValue(ctxProposer, tpnetcmn.NetContextKey_PeerList, peerActiveProposerIDs)

		peerActiveValidatorIDs := md.epochService.GetActiveValidatorIDs()
		ctxValidator = context.WithValue(ctxValidator, tpnetcmn.NetContextKey_PeerList, peerActiveValidatorIDs)
	}

	if md.dkgBls == nil {
		err := errors.New("Nil dkg for delivering propose message")
		return err
	}

	sigData, pubKey, err := md.dkgBls.Sign(msg.BlockHead)
	if err != nil {
		md.log.Errorf("DKG sign propose msg err: %v", err)
		return err
	}
	msg.Signature = sigData
	msg.PubKey = pubKey
	msgBytes, err := md.marshaler.Marshal(msg)
	if err != nil {
		md.log.Errorf("ProposeMessage marshal err: %v", err)
		return err
	}
	err = deliverSendCommon(ctxProposer, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_Propose, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to proposer network failed: err=%v", err)
		return nil
	}

	err = deliverSendCommon(ctxValidator, md.log, md.marshaler, md.network, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_Propose, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to validator network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverResultValidateReqMessage(ctx context.Context, msg *ExeResultValidateReqMessage) (*ExeResultValidateRespMessage, error) {
	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RespThreshold, float32(1.0))

	randExecutorID := make([]string, TxsResultValidity_MaxExecutor)
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs := md.epochService.GetActiveExecutorIDs()
		if len(peerIDs) == 0 {
			md.log.Warnf("Active executor nodes count 0: self node %s", md.nodeID)
		}

		randIndexs := tpcmm.GenerateRandomNumber(0, len(peerIDs), TxsResultValidity_MaxExecutor)
		for i := 0; i < len(randExecutorID); i++ {
			randExecutorID[i] = peerIDs[randIndexs[i]]
		}

		if len(randExecutorID) == 0 {
			md.log.Warnf("Randomly selected executor nodes count 0: self node %s", md.nodeID)
		}

		md.log.Debugf("Rand active executor nodes: %v", randExecutorID)

		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, randExecutorID)
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
	resp, err := deliverSendWithRespCommon(ctx, md.nodeID, md.log, md.marshaler, md.network, tpnetprotoc.ForwardExecute_SyncMsg, MOD_NAME, ConsensusMessage_ExeRSValidateReq, msgBytes)
	if err != nil {
		md.log.Errorf("Send execution result validate request message to executor network failed: err=%v", err)
		return nil, err
	}

	if len(resp) <= 0 {
		err = fmt.Errorf("Received execution result validate request resp %d from executor %v", len(resp), randExecutorID)
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

func (md *messageDeliver) deliverResultValidateRespMessage(actorCtx actor.Context, msg *ExeResultValidateRespMessage, err error) error {
	errRtn := err
	msgBytes := []byte(nil)
	defer func() {
		respData := &tpnetmsg.ResponseData{
			Data: msgBytes,
		}
		if errRtn != nil {
			respData.ErrMsg = errRtn.Error()
		}

		respDataBytes, _ := json.Marshal(&respData)
		actorCtx.Respond(respDataBytes)
		md.log.Infof("Successfully deliver result validate resp message: state version %d, executor %s, self node %s", msg.StateVersion, string(msg.Executor), md.nodeID)
	}()

	if errRtn != nil {
		return errRtn
	}

	msg.Executor = []byte(md.nodeID)

	sigData, errRtn := md.cryptService.Sign(md.priKey, msg.TxAndResultProofsData())
	if errRtn != nil {
		md.log.Errorf("Sign err for execution result validate response msg: %v", errRtn)
		return errRtn
	}

	pubKey, errRtn := md.cryptService.ConvertToPublic(md.priKey)
	if errRtn != nil {
		md.log.Errorf("Can't get public key from private key: %v", errRtn)
		return errRtn
	}
	msg.Signature = sigData
	msg.PubKey = pubKey

	msgBytes, errRtn = md.marshaler.Marshal(msg)
	if errRtn != nil {
		md.log.Errorf("ExeResultValidateRespMessage marshal err: %v", errRtn)
		return errRtn
	}

	return nil
}

func (md *messageDeliver) deliverBestProposeMessage(ctx context.Context, msg *BestProposeMessage) error {
	if msg == nil {
		return errors.New("Nil ProposeMessage for delivering")
	}

	ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)

	ctxProposer := ctx
	ctxValidator := ctx
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerActiveProposerIDs := md.epochService.GetActiveProposerIDs()
		peerActiveProposerIDs = tpcmm.RemoveIfExistString(md.nodeID, peerActiveProposerIDs)
		ctxProposer = context.WithValue(ctxProposer, tpnetcmn.NetContextKey_PeerList, peerActiveProposerIDs)

		peerActiveValidatorIDs := md.epochService.GetActiveValidatorIDs()
		ctxValidator = context.WithValue(ctxValidator, tpnetcmn.NetContextKey_PeerList, peerActiveValidatorIDs)
	}

	sigData, err := md.cryptService.Sign(md.priKey, msg.PropMsgData)
	if err != nil {
		md.log.Errorf("Sign the best propose msg err: %v", err)
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
	err = deliverSendCommon(ctxProposer, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_BestPropose, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to proposer network failed: err=%v", err)
		return nil
	}

	err = deliverSendCommon(ctxValidator, md.log, md.marshaler, md.network, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_BestPropose, msgBytes)
	if err != nil {
		md.log.Errorf("Send propose message to validator network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) getVoterCollector(voterRound uint64) (string, []byte, error) {
	lastBlock, err := state.GetLatestBlock(md.ledger)
	if err != nil {
		md.log.Errorf("Can't get the latest block: %v", err)
		return "", nil, err
	}

	if lastBlock.Head.Round != voterRound-1 {
		err := fmt.Errorf("Stale vote epoch: %d", voterRound)
		md.log.Errorf(err.Error())
		return "", nil, err
	}

	selVoteColectors, vrfProof, err := vrf.NewLeaderSelectorVRF(md.log, md.nodeID, md.cryptService).Select(vrf.RoleSelector_VoteCollector, 0, md.priKey, lastBlock, md.epochService, 1)
	if len(selVoteColectors) != 1 {
		err := fmt.Errorf("Expect vote collector count 1, got %d", len(selVoteColectors))
		md.log.Errorf("%v", err)
		return "", nil, err
	}

	return selVoteColectors[0], vrfProof, err
}

func (md *messageDeliver) deliverVoteMessage(ctx context.Context, msg *VoteMessage, proposer string) error {
	if msg == nil {
		return errors.New("Nil VoteMessage for delivering")
	}

	if md.dkgBls == nil {
		err := errors.New("Nil dkg for delivering vote message")
		return err
	}

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
	err = deliverSendCommon(ctx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_Vote, msgBytes)
	if err != nil {
		md.log.Errorf("Send vote message to proposer network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) deliverCommitMessage(ctx context.Context, msg *CommitMessage) error {
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

	exeCtx := ctx
	propCtx := ctx
	ValCtx := ctx
	switch md.strategy {
	case DeliverStrategy_Specifically:
		peerIDs := md.epochService.GetActiveExecutorIDs()
		exeCtx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, peerIDs)

		peerProposerIDs := md.epochService.GetActiveProposerIDs()
		peerProposerIDs = tpcmm.RemoveIfExistString(md.nodeID, peerProposerIDs)
		propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_PeerList, peerProposerIDs)

		peerValidatorIDs := md.epochService.GetActiveValidatorIDs()
		ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_PeerList, peerValidatorIDs)
	}

	exeCtx = context.WithValue(exeCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(exeCtx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardExecute_Msg, MOD_NAME, ConsensusMessage_Commit, msgBytes)
	if err != nil {
		md.log.Errorf("Send commit message to executor network failed: err=%v", err)
	}

	propCtx = context.WithValue(propCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(propCtx, md.log, md.marshaler, md.network, tpnetprotoc.ForwardPropose_Msg, MOD_NAME, ConsensusMessage_Commit, msgBytes)
	if err != nil {
		md.log.Errorf("Send commit message to propose network failed: err=%v", err)
	}

	ValCtx = context.WithValue(ValCtx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
	err = deliverSendCommon(ValCtx, md.log, md.marshaler, md.network, tpnetprotoc.FrowardValidate_Msg, MOD_NAME, ConsensusMessage_Commit, msgBytes)
	if err != nil {
		md.log.Errorf("Send commit message to validate network failed: err=%v", err)
	}

	return err
}

func (md *messageDeliver) setEpochService(epService EpochService) {
	md.epochService = epService
}

func (md *messageDeliver) updateDKGBls(dkgBls DKGBls) {
	md.dkgBls = dkgBls
	md.ready.Swap(true)
}

func (md *messageDeliver) isReady() bool {
	return md.ready.Load() && md.dkgBls.Finished()
}
