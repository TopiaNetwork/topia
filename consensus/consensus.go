package consensus

import (
	"context"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/eventhub"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
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

	Start(*actor.ActorSystem) error

	Stop()
}

type consensus struct {
	log          tplog.Logger
	level        tplogcmm.LogLevel
	handler      ConsensusHandler
	marshaler    codec.Marshaler
	network      tpnet.Network
	executor     *consensusExecutor
	proposer     *consensusProposer
	validator    *consensusValidator
	dkgEx        *dkgExchange
	epochService *epochService
}

func NewConsensus(nodeID string,
	priKey tpcrtypes.PrivateKey,
	level tplogcmm.LogLevel,
	log tplog.Logger,
	codecType codec.CodecType,
	network tpnet.Network,
	txPool txpool.TransactionPool,
	ledger ledger.Ledger,
	config *tpconfig.ConsensusConfiguration) Consensus {
	consLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	marshaler := codec.CreateMarshaler(codecType)
	roundCh := make(chan *RoundInfo)
	preprePackedMsgExeChan := make(chan *PreparePackedMessageExe)
	preprePackedMsgPropChan := make(chan *PreparePackedMessageProp)
	proposeMsgChan := make(chan *ProposeMessage)
	voteMsgChan := make(chan *VoteMessage)
	commitMsgChan := make(chan *CommitMessage)
	blockAddedCh := make(chan *tpchaintypes.Block)
	partPubKey := make(chan *DKGPartPubKeyMessage, PartPubKeyChannel_Size)
	dealMsgCh := make(chan *DKGDealMessage, DealMSGChannel_Size)
	dealRespMsgCh := make(chan *DKGDealRespMessage, DealRespMsgChannel_Size)

	var cryptS tpcrt.CryptService
	if config.CrptyType == tpcrtypes.CryptType_Ed25519 {
		cryptS = &CryptServiceMock{}
	} else {
		cryptS = tpcrt.CreateCryptService(log, config.CrptyType)
	}

	deliver := newMessageDeliver(log, nodeID, priKey, DeliverStrategy_All, network, marshaler, cryptS, ledger)

	exeScheduler := execution.NewExecutionScheduler(log)

	executor := newConsensusExecutor(log, nodeID, priKey, txPool, marshaler, ledger, exeScheduler, deliver, preprePackedMsgExeChan, commitMsgChan, config.ExecutionPrepareInterval)
	validator := newConsensusValidator(log, nodeID, proposeMsgChan, ledger, deliver)
	proposer := newConsensusProposer(log, nodeID, priKey, roundCh, preprePackedMsgPropChan, voteMsgChan, cryptS, deliver, ledger, marshaler, validator)
	dkgEx := newDKGExchange(log, partPubKey, dealMsgCh, dealRespMsgCh, config.InitDKGPrivKey, config.InitDKGPartPubKeys, deliver, ledger)

	epService := newEpochService(log, blockAddedCh, config.EpochInterval, config.DKGStartBeforeEpoch, exeScheduler, ledger, dkgEx)
	csHandler := NewConsensusHandler(log, blockAddedCh, preprePackedMsgExeChan, preprePackedMsgPropChan, proposeMsgChan, voteMsgChan, partPubKey, dealMsgCh, dealRespMsgCh, ledger, marshaler, deliver, exeScheduler)

	dkgEx.addDKGBLSUpdater(deliver)
	dkgEx.addDKGBLSUpdater(proposer)

	return &consensus{
		log:          consLog,
		level:        level,
		handler:      csHandler,
		marshaler:    codec.CreateMarshaler(codecType),
		network:      network,
		executor:     executor,
		proposer:     proposer,
		validator:    validator,
		dkgEx:        dkgEx,
		epochService: epService,
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

func (cons *consensus) Start(sysActor *actor.ActorSystem) error {
	actorPID, err := CreateConsensusActor(cons.level, cons.log, sysActor, cons)
	if err != nil {
		cons.log.Panicf("CreateConsensusActor error: %v", err)
		return err
	}

	cons.network.RegisterModule(MOD_NAME, actorPID, cons.marshaler)

	ctx := context.Background()

	eventhub.GetEventHub().Observe(ctx, eventhub.EventName_BlockAdded, cons.ProcessSubEvent)

	cons.epochService.start(ctx)
	cons.executor.start(ctx)
	cons.proposer.start(ctx)
	cons.validator.start(ctx)
	cons.dkgEx.startLoop(ctx)

	return nil
}

func (cons *consensus) dispatch(actorCtx actor.Context, data []byte) {
	var consMsg ConsensusMessage
	err := cons.marshaler.Unmarshal(data, &consMsg)
	if err != nil {
		cons.log.Errorf("Consensus receive invalid data %v", data)
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
