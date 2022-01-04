package consensus

import (
	"context"
	"github.com/TopiaNetwork/topia/ledger"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	tptypes "github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
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
	VerifyBlock(*tptypes.Block) error

	ProcessPropose(*ProposeMessage) error

	ProcessVote(*VoteMessage) error

	UpdateHandler(handler ConsensusHandler)

	Start(*actor.ActorSystem) error

	Stop()
}

type consensus struct {
	log       tplog.Logger
	level     tplogcmm.LogLevel
	handler   ConsensusHandler
	marshaler codec.Marshaler
	network   tpnet.Network
	proposer  *consensusProposer
	voter     *consensusVoter
}

func NewConsensus(level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType, network tpnet.Network, ledger ledger.Ledger) Consensus {
	consLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	marshaler := codec.CreateMarshaler(codecType)
	roundCh := make(chan *RoundInfo)
	proposeMsgChan := make(chan *ProposeMessage)
	deliver := newMessageDeliver(log, DeliverStrategy_All, network, marshaler)

	proposer := NewConsensusProposer(log, roundCh, deliver, ledger, marshaler)
	voter := newConsensusVoter(log, proposeMsgChan, deliver)

	return &consensus{
		log:       consLog,
		level:     level,
		handler:   NewConsensusHandler(log, roundCh, proposeMsgChan, 2, ledger, marshaler),
		marshaler: codec.CreateMarshaler(codecType),
		network:   network,
		proposer:  proposer,
		voter:     voter,
	}
}

func (cons *consensus) UpdateHandler(handler ConsensusHandler) {
	cons.handler = handler
}

func (cons *consensus) VerifyBlock(block *tptypes.Block) error {
	return cons.handler.VerifyBlock(block)
}

func (cons *consensus) ProcessPropose(msg *ProposeMessage) error {
	return cons.handler.ProcessPropose(msg)
}

func (cons *consensus) ProcessVote(msg *VoteMessage) error {
	return cons.handler.ProcessVote(msg)
}

func (cons *consensus) Start(sysActor *actor.ActorSystem) error {
	actorPID, err := CreateConsensusActor(cons.level, cons.log, sysActor, cons)
	if err != nil {
		cons.log.Panicf("CreateConsensusActor error: %v", err)
		return err
	}

	cons.network.RegisterModule(MOD_NAME, actorPID, cons.marshaler)

	ctx := context.Background()

	cons.proposer.start(ctx)
	cons.voter.start(ctx)

	return nil
}

func (cons *consensus) dispatch(context actor.Context, data []byte) {
	var consMsg ConsensusMessage
	err := cons.marshaler.Unmarshal(data, &consMsg)
	if err != nil {
		cons.log.Errorf("Consensus receive invalid data %v", data)
		return
	}

	switch consMsg.MsgType {
	case ConsensusMessage_Propose:
		var msg ProposeMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %d err %v", consMsg.MsgType, err)
			return
		}
		cons.ProcessPropose(&msg)
	case ConsensusMessage_Vote:
		var msg VoteMessage
		err := cons.marshaler.Unmarshal(consMsg.Data, &msg)
		if err != nil {
			cons.log.Errorf("Consensus unmarshal msg %d err %v", consMsg.MsgType, err)
			return
		}
		cons.ProcessVote(&msg)
	default:
		cons.log.Errorf("Consensus receive invalid msg %d", consMsg.MsgType)
		return
	}
}

func (cons *consensus) Stop() {
	panic("implement me")
}
