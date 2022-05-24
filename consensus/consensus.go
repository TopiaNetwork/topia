package consensus

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/state"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

const (
	MOD_NAME      = "consensus"
	CONSENSUS_VER = uint32(1)
)

type RoundInfo struct {
	Epoch        uint64
	LastRoundNum uint64
	CurRoundNum  uint64
	Proof        *ConsensusProof
}

type Consensus interface {
	VerifyBlock(*tpchaintypes.Block) error

	ProcessPropose(*ProposeMessage) error

	ProcessVote(*VoteMessage) error

	UpdateHandler(handler ConsensusHandler)

	Start(*actor.ActorSystem, uint64, uint64, uint64) error

	TriggerDKG(epoch uint64)

	Stop()
}

type consensus struct {
	nodeID       string
	log          tplog.Logger
	level        tplogcmm.LogLevel
	handler      ConsensusHandler
	marshaler    codec.Marshaler
	network      tpnet.Network
	ledger       ledger.Ledger
	executor     *consensusExecutor
	proposer     *consensusProposer
	validator    *consensusValidator
	dkgEx        *dkgExchange
	epochService *epochService
	config       *tpconfig.ConsensusConfiguration
}

func NewConsensus(chainID tpchaintypes.ChainID,
	nodeID string,
	priKey tpcrtypes.PrivateKey,
	level tplogcmm.LogLevel,
	log tplog.Logger,
	codecType codec.CodecType,
	network tpnet.Network,
	txPool txpooli.TransactionPool,
	ledger ledger.Ledger,
	exeScheduler execution.ExecutionScheduler,
	config *tpconfig.Configuration) Consensus {
	consLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	marshaler := codec.CreateMarshaler(codecType)
	epochNewCh := make(chan *tpcmm.EpochInfo)
	preprePackedMsgExeChan := make(chan *PreparePackedMessageExe, PreparePackedExeChannel_Size)
	preprePackedMsgExeIndicChan := make(chan *PreparePackedMessageExeIndication, PreparePackedExeIndicChannel_Size)
	preprePackedMsgPropChan := make(chan *PreparePackedMessageProp, PreparePackedPropChannel_Size)
	proposeMsgChan := make(chan *ProposeMessage, ProposeChannel_Size)
	voteMsgChan := make(chan *VoteMessage)
	commitMsgChan := make(chan *CommitMessage)
	blockAddedEpochCh := make(chan *tpchaintypes.Block)
	blockAddedProposerCh := make(chan *tpchaintypes.Block)
	partPubKey := make(chan *DKGPartPubKeyMessage, PartPubKeyChannel_Size)
	dealMsgCh := make(chan *DKGDealMessage, DealMSGChannel_Size)
	dealRespMsgCh := make(chan *DKGDealRespMessage, DealRespMsgChannel_Size)

	csConfig := config.CSConfig

	var cryptS tpcrt.CryptService
	if csConfig.CrptyType == tpcrtypes.CryptType_Ed25519 {
		cryptS = &CryptServiceMock{}
	} else {
		cryptS = tpcrt.CreateCryptService(consLog, csConfig.CrptyType)
	}

	deliver := newMessageDeliver(consLog, nodeID, priKey, DeliverStrategy_Specifically, network, marshaler, cryptS, ledger)

	executor := newConsensusExecutor(consLog, nodeID, priKey, txPool, marshaler, ledger, exeScheduler, deliver, preprePackedMsgExeChan, preprePackedMsgExeIndicChan, commitMsgChan, cryptS, csConfig.ExecutionPrepareInterval)
	validator := newConsensusValidator(consLog, nodeID, proposeMsgChan, ledger, deliver)
	proposer := newConsensusProposer(consLog, nodeID, priKey, preprePackedMsgPropChan, voteMsgChan, blockAddedProposerCh, cryptS, csConfig.ProposerBlockMaxInterval, csConfig.BlockMaxCyclePeriod, deliver, ledger, marshaler, validator)
	dkgEx := newDKGExchange(consLog, chainID, nodeID, partPubKey, dealMsgCh, dealRespMsgCh, csConfig.InitDKGPrivKey, deliver, ledger)

	epService := newEpochService(consLog, nodeID, blockAddedEpochCh, epochNewCh, csConfig.EpochInterval, csConfig.DKGStartBeforeEpoch, exeScheduler, ledger, dkgEx)
	csHandler := NewConsensusHandler(consLog, epochNewCh, preprePackedMsgExeChan, preprePackedMsgExeIndicChan, preprePackedMsgPropChan, proposeMsgChan, voteMsgChan, commitMsgChan, blockAddedEpochCh, blockAddedProposerCh, partPubKey, dealMsgCh, dealRespMsgCh, ledger, marshaler, deliver, exeScheduler)

	dkgEx.addDKGBLSUpdater(deliver)
	dkgEx.addDKGBLSUpdater(proposer)

	return &consensus{
		nodeID:       nodeID,
		log:          consLog,
		level:        level,
		handler:      csHandler,
		marshaler:    codec.CreateMarshaler(codecType),
		network:      network,
		ledger:       ledger,
		executor:     executor,
		proposer:     proposer,
		validator:    validator,
		dkgEx:        dkgEx,
		epochService: epService,
		config:       csConfig,
	}
}

func (cons *consensus) UpdateHandler(handler ConsensusHandler) {
	cons.handler = handler
}

func (cons *consensus) VerifyBlock(block *tpchaintypes.Block) error {
	return cons.handler.VerifyBlock(block)
}

func (cons *consensus) ProcessPreparePackedExe(msg *PreparePackedMessageExe) error {
	return cons.handler.ProcessPreparePackedMsgExe(msg)
}

func (cons *consensus) ProcessPreparePackedExeIndication(msg *PreparePackedMessageExeIndication) error {
	return cons.handler.ProcessPreparePackedMsgExeIndication(msg)
}

func (cons *consensus) ProcessPreparePackedProp(msg *PreparePackedMessageProp) error {
	return cons.handler.ProcessPreparePackedMsgProp(msg)
}

func (cons *consensus) ProcessPropose(msg *ProposeMessage) error {
	return cons.handler.ProcessPropose(msg)
}

func (cons *consensus) ProcesExeResultValidateReq(actorCtx actor.Context, msg *ExeResultValidateReqMessage) error {
	return cons.handler.ProcesExeResultValidateReq(actorCtx, msg)
}

func (cons *consensus) ProcessVote(msg *VoteMessage) error {
	return cons.handler.ProcessVote(msg)
}

func (cons *consensus) ProcessCommit(msg *CommitMessage) error {
	return cons.handler.ProcessCommit(msg)
}

func (cons *consensus) ProcessSubEvent(ctx context.Context, data interface{}) error {
	if block, ok := data.(*tpchaintypes.Block); ok {
		return cons.handler.ProcessBlockAdded(block)
	}

	if epoch, ok := data.(*tpcmm.EpochInfo); ok {
		return cons.handler.ProcessEpochNew(epoch)
	}

	panic("Unknown sub event data")
}

func (cons *consensus) ProcessDKGPartPubKey(msg *DKGPartPubKeyMessage) error {
	return cons.handler.ProcessDKGPartPubKey(msg)
}

func (cons *consensus) ProcessDKGDeal(msg *DKGDealMessage) error {
	return cons.handler.ProcessDKGDeal(msg)
}

func (cons *consensus) ProcessDKGDealResp(msg *DKGDealRespMessage) error {
	return cons.handler.ProcessDKGDealResp(msg)
}

func (cons *consensus) Start(sysActor *actor.ActorSystem, epoch uint64, epochStartHeight uint64, height uint64) error {
	actorPID, err := CreateConsensusActor(cons.level, cons.log, sysActor, cons)
	if err != nil {
		cons.log.Panicf("CreateConsensusActor error: %v", err)
		return err
	}

	cons.network.RegisterModule(MOD_NAME, actorPID, cons.marshaler)

	ctx := context.Background()

	eventhub.GetEventHubManager().GetEventHub(cons.nodeID).Observe(ctx, eventhub.EventName_BlockAdded, cons.ProcessSubEvent)

	csStateRN := state.CreateCompositionStateReadonly(cons.log, cons.ledger)
	defer csStateRN.Stop()
	nodeInfo, err := csStateRN.GetNode(cons.nodeID)
	if err != nil {
		cons.log.Panicf("Get node error: %v", err)
		return err
	}

	cons.log.Infof("Self Node id=%s, role=%d, state=%d", nodeInfo.NodeID, nodeInfo.Role, nodeInfo.State)

	cons.epochService.start(ctx)

	if nodeInfo.Role&tpcmm.NodeRole_Executor == tpcmm.NodeRole_Executor {
		cons.executor.start(ctx)
	}

	if nodeInfo.Role&tpcmm.NodeRole_Proposer == tpcmm.NodeRole_Proposer || nodeInfo.Role&tpcmm.NodeRole_Validator == tpcmm.NodeRole_Validator {
		if nodeInfo.Role&tpcmm.NodeRole_Proposer == tpcmm.NodeRole_Proposer {
			cons.proposer.start(ctx)
		}

		cons.validator.start(ctx)
		cons.dkgEx.startLoop(ctx)

		err = cons.dkgEx.updateDKGPartPubKeys(csStateRN)
		if err != nil {
			cons.log.Panicf("Update DKG exchange part pub keys err: %v", err)
			return err
		}
		cons.dkgEx.initWhenStart(epoch)
		for i, verfer := range cons.dkgEx.dkgCrypt.dkGenerator.Verifiers() {
			longterm, pub := verfer.Key()
			cons.log.Infof("After init, dkgInitPrivKey=%s, Verifier i %d: longterm=%s, pub=%s, index=%d", cons.dkgEx.dkgExData.initDKGPrivKey.Load().(string), i, longterm.String(), pub.String(), verfer.Index())
		}
	}

	return nil
}

func (cons *consensus) TriggerDKG(epoch uint64) {
	cons.dkgEx.start(epoch)
}

func (cons *consensus) dispatch(actorCtx actor.Context, data []byte) {
	var consMsg ConsensusMessage
	err := cons.marshaler.Unmarshal(data, &consMsg)
	if err != nil {
		cons.log.Errorf("Consensus receive invalid data %v", err)
		return
	}

	switch consMsg.MsgType {
	case ConsensusMessage_PrepareExe:
		var msg PreparePackedMessageExe
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessPreparePackedExe(&msg)
	case ConsensusMessage_PrepareExeIndic:
		var msg PreparePackedMessageExeIndication
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessPreparePackedExeIndication(&msg)
	case ConsensusMessage_PrepareProp:
		var msg PreparePackedMessageProp
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessPreparePackedProp(&msg)
	case ConsensusMessage_Propose:
		var msg ProposeMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessPropose(&msg)
	case ConsensusMessage_ExeRSValidateReq:
		var msg ExeResultValidateReqMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcesExeResultValidateReq(actorCtx, &msg)
	case ConsensusMessage_Vote:
		var msg VoteMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessVote(&msg)
	case ConsensusMessage_Commit:
		var msg CommitMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessCommit(&msg)
	case ConsensusMessage_DKGDeal:
		var msg DKGDealMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessDKGDeal(&msg)
	case ConsensusMessage_DKGDealResp:
		var msg DKGDealRespMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
			return
		}
		cons.ProcessDKGDealResp(&msg)
	default:
		cons.log.Errorf("Consensus receive invalid msg %d", consMsg.MsgType)
		return
	}
}

func (cons *consensus) Stop() {
	cons.dkgEx.stop()
	cons.log.Info("Consensus exit")
}
