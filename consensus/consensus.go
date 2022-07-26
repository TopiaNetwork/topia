package consensus

import (
	"context"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/consensus/vrf"

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
	Proof        *tpchaintypes.ConsensusProof
}

type Consensus interface {
	VerifyBlock(*tpchaintypes.Block) error

	ProcessPropose(*ProposeMessage) error

	ProcessVote(*VoteMessage) error

	UpdateHandler(handler ConsensusHandler)

	Start(*actor.ActorSystem, uint64, uint64, uint64) error

	TriggerDKG(newBlock *tpchaintypes.Block)

	Stop()
}

type consensus struct {
	nodeID          string
	nodeRole        tpcmm.NodeRole
	log             tplog.Logger
	level           tplogcmm.LogLevel
	handler         ConsensusHandler
	marshaler       codec.Marshaler
	network         tpnet.Network
	ledger          ledger.Ledger
	executor        *consensusExecutor
	proposer        *consensusProposer
	validator       *consensusValidator
	dkgEx           *dkgExchange
	epochService    EpochService
	csDomainService *domainConsensusService
	config          *tpconfig.ConsensusConfiguration
}

func NewConsensus(chainID tpchaintypes.ChainID,
	nodeID string,
	stateBuilderType state.CompStateBuilderType,
	priKey tpcrtypes.PrivateKey,
	level tplogcmm.LogLevel,
	log tplog.Logger,
	codecType codec.CodecType,
	network tpnet.Network,
	txPool txpooli.TransactionPool,
	ledger ledger.Ledger,
	exeScheduler execution.ExecutionScheduler,
	config *tpconfig.Configuration) Consensus {

	compStateRN := state.CreateCompositionStateReadonly(log, ledger)
	defer compStateRN.Stop()

	currentEpoch, err := compStateRN.GetLatestEpoch()
	if err != nil {
		log.Panicf("Can't get the latest epoch: err=%v", err)
	}

	consLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	marshaler := codec.CreateMarshaler(codecType)
	preparePackedMsgExeChan := make(chan *PreparePackedMessageExe, PreparePackedExeChannel_Size)
	preparePackedMsgExeIndicChan := make(chan *PreparePackedMessageExeIndication, PreparePackedExeIndicChannel_Size)
	preparePackedMsgPropChan := make(chan *PreparePackedMessageProp, PreparePackedPropChannel_Size)
	proposeMsgChan := make(chan *ProposeMessage, ProposeChannel_Size)
	bestProposeMsgChan := make(chan *BestProposeMessage, BestProposeChannel_Size)
	voteMsgChan := make(chan *VoteMessage)
	commitMsgChan := make(chan *CommitMessage)
	blockAddedCSDomain := make(chan *tpchaintypes.Block)
	blockAddedProposerCh := make(chan *tpchaintypes.Block)
	partPubKey := make(chan *DKGPartPubKeyMessage, PartPubKeyChannel_Size)
	dealMsgCh := make(chan *DKGDealMessage, DealMSGChannel_Size)
	dealRespMsgCh := make(chan *DKGDealRespMessage, DealRespMsgChannel_Size)
	finishedMsgCh := make(chan *DKGFinishedMessage, FinishedMsgChannel_Size)
	csDomainSelectedMsgCh := make(chan *ConsensusDomainSelectedMessage, FinishedMsgChannel_Size)

	csConfig := config.CSConfig

	var cryptS tpcrt.CryptService
	if csConfig.CrptyType == tpcrtypes.CryptType_Ed25519 {
		cryptS = &CryptServiceMock{}
	} else {
		cryptS = tpcrt.CreateCryptService(consLog, csConfig.CrptyType)
	}

	roleSelector := vrf.NewLeaderSelectorVRF(log, nodeID, cryptS)

	deliver := newMessageDeliver(consLog, nodeID, priKey, DeliverStrategy_Specifically, network, marshaler, cryptS, ledger)

	exeActiveNodeIDs, err := compStateRN.GetActiveExecutorIDs()
	if err != nil {
		log.Panicf("Can't get the all active executor node ids: err=%v", err)
	}

	exeActiveNodes, err := compStateRN.GetAllActiveExecutors()
	if err != nil {
		log.Panicf("Can't get the all active executor node: err=%v", err)
	}

	dkgDeliver := NewDkgMessageDeliver(consLog, nodeID, priKey, DeliverStrategy_Specifically, network, marshaler, cryptS, ledger, exeActiveNodeIDs)

	dkgEx := newDKGExchange(consLog, chainID, nodeID, partPubKey, dealMsgCh, dealRespMsgCh, finishedMsgCh, csConfig.InitDKGPrivKey, dkgDeliver, ledger)

	epService := NewEpochService(consLog, nodeID, stateBuilderType, csConfig.EpochInterval, currentEpoch, csConfig.DKGStartBeforeEpoch, exeScheduler, ledger, deliver, dkgEx, exeActiveNodes)

	csDomainService := NewDomainConsensusService(nodeID, stateBuilderType, log, ledger, blockAddedCSDomain, roleSelector, csConfig, dkgEx, exeScheduler)

	executor := newConsensusExecutor(consLog, nodeID, priKey, txPool, marshaler, ledger, exeScheduler, epService, deliver, csDomainSelectedMsgCh, preparePackedMsgExeChan, preparePackedMsgExeIndicChan, commitMsgChan, cryptS, csConfig.ExecutionPrepareInterval)
	validator := newConsensusValidator(consLog, nodeID, proposeMsgChan, bestProposeMsgChan, commitMsgChan, ledger, deliver, marshaler, epService)
	proposer := newConsensusProposer(consLog, nodeID, priKey, preparePackedMsgPropChan, voteMsgChan, blockAddedProposerCh, cryptS, epService, csConfig.ProposerBlockMaxInterval, csConfig.BlockMaxCyclePeriod, deliver, ledger, marshaler, validator)

	deliver.setEpochService(epService)
	csHandler := NewConsensusHandler(consLog, preparePackedMsgExeChan, preparePackedMsgExeIndicChan, preparePackedMsgPropChan, proposeMsgChan, bestProposeMsgChan, voteMsgChan, commitMsgChan, blockAddedCSDomain, blockAddedProposerCh, partPubKey, dealMsgCh, dealRespMsgCh, finishedMsgCh, csDomainSelectedMsgCh, ledger, marshaler, deliver, exeScheduler, epService)

	epService.AddDKGBLSUpdater(deliver)
	epService.AddDKGBLSUpdater(proposer)

	return &consensus{
		nodeID:          nodeID,
		log:             consLog,
		level:           level,
		handler:         csHandler,
		marshaler:       codec.CreateMarshaler(codecType),
		network:         network,
		ledger:          ledger,
		executor:        executor,
		proposer:        proposer,
		validator:       validator,
		dkgEx:           dkgEx,
		epochService:    epService,
		csDomainService: csDomainService,
		config:          csConfig,
	}
}

func (cons *consensus) UpdateHandler(handler ConsensusHandler) {
	cons.handler = handler
}

func (cons *consensus) VerifyBlock(block *tpchaintypes.Block) error {
	return cons.handler.VerifyBlock(block)
}

func (cons *consensus) ProcessCSDomainSelectedMsg(msg *ConsensusDomainSelectedMessage) error {
	return cons.handler.ProcessCSDomainSelectedMsg(msg)
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

func (cons *consensus) ProcessExeResultValidateReq(actorCtx actor.Context, msg *ExeResultValidateReqMessage) error {
	return cons.handler.ProcessExeResultValidateReq(actorCtx, msg)
}

func (cons *consensus) ProcessBestPropose(msg *BestProposeMessage) error {
	return cons.handler.ProcessBestPropose(msg)
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

func (cons *consensus) ProcessDKGFinishedMsg(msg *DKGFinishedMessage) error {
	return cons.handler.ProcessDKGFinishedMsg(msg)
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

	latestBlock, err := csStateRN.GetLatestBlock()
	if err != nil {
		cons.log.Panicf("Get latest block error: %v", err)
		return err
	}

	nodeInfo, err := csStateRN.GetNode(cons.nodeID)
	if err != nil {
		cons.log.Panicf("Get node error: %v", err)
		return err
	}
	cons.nodeRole = nodeInfo.Role
	cons.log.Infof("Self Node id=%s, role=%s, state=%d", nodeInfo.NodeID, nodeInfo.Role.String(), nodeInfo.State)

	if nodeInfo.Role&tpcmm.NodeRole_Executor == tpcmm.NodeRole_Executor {
		cons.executor.start(ctx)
	}

	if nodeInfo.Role&tpcmm.NodeRole_Proposer == tpcmm.NodeRole_Proposer || nodeInfo.Role&tpcmm.NodeRole_Validator == tpcmm.NodeRole_Validator {
		if nodeInfo.Role&tpcmm.NodeRole_Proposer == tpcmm.NodeRole_Proposer {
			cons.proposer.validator.updateLogger(cons.proposer.log)
			cons.proposer.start(ctx)
		}

		cons.validator.start(ctx)
		cons.dkgEx.startLoop(ctx)

		cons.csDomainService.Start(ctx)
		cons.epochService.Start(ctx, latestBlock.Head.Height)
	}

	return nil
}

func (cons *consensus) TriggerDKG(newBlock *tpchaintypes.Block) {
	cons.csDomainService.Trigger(newBlock)
}

func (cons *consensus) dispatch(ctx context.Context, actorCtx actor.Context, data []byte) {
	var consMsg ConsensusMessage
	err := cons.marshaler.Unmarshal(data, &consMsg)
	if err != nil {
		cons.log.Errorf("Consensus receive invalid data %v", err)
		return
	}

	finishedCh := make(chan bool)
	go func() {
		switch consMsg.MsgType {
		case ConsensusMessage_CSDomainSel:
			var msg ConsensusDomainSelectedMessage
			err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
			if err != nil {
				cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
				return
			}
			cons.ProcessCSDomainSelectedMsg(&msg)
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
			cons.log.Infof("Received execute result validate msg: state version %d, validator %s, self node %s", msg.StateVersion, string(msg.Validator), cons.nodeID)
			cons.ProcessExeResultValidateReq(actorCtx, &msg)
			cons.log.Infof("Finished result validate msg: state version %d, validator %s, self node %s", msg.StateVersion, string(msg.Validator), cons.nodeID)
		case ConsensusMessage_BestPropose:
			var msg BestProposeMessage
			err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
			if err != nil {
				cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
				return
			}
			cons.ProcessBestPropose(&msg)
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
		case ConsensusMessage_DKGFinished:
			var msg DKGFinishedMessage
			err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
			if err != nil {
				cons.log.Errorf("Consensus unmarshal msg %s err %v", consMsg.MsgType.String(), err)
				return
			}
			cons.ProcessDKGFinishedMsg(&msg)
		default:
			cons.log.Errorf("Consensus receive invalid msg %d", consMsg.MsgType.String())
			return
		}

		finishedCh <- true
	}()

	select {
	case <-finishedCh:
		return
	case <-ctx.Done():
		cons.log.Errorf("Msg handler timeout: %s, self node %s", consMsg.MsgType.String(), cons.nodeID)
		return
	}
}

func (cons *consensus) Stop() {
	cons.dkgEx.stop()
	cons.log.Info("Consensus exit")
}
