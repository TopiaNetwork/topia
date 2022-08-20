package consensus

import (
	"context"
	"encoding/hex"
	"github.com/AsynkronIT/protoactor-go/actor"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"time"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type ConsensusActor struct {
	log  tplog.Logger
	pid  *actor.PID
	cons *consensus
}

type Statistics interface {
	MailboxStarted()
	MessagePosted(message interface{})
	MessageReceived(message interface{})
	MailboxEmpty()
}

//CSActorMsgStatistics is only for monitor
type CSActorMsgStatistics struct {
	nodeID string
	log    tplog.Logger
}

func (ms *CSActorMsgStatistics) MailboxStarted() {
	ms.log.Infof("Consensus actor mail box started: self node %s", ms.nodeID)
}

func (ms *CSActorMsgStatistics) MessagePosted(message interface{}) {
	switch msg := message.(type) {
	case []byte:
		dataHash := tpcmm.NewBlake2bHasher(0)
		hashString := hex.EncodeToString(dataHash.Compute(string(msg)))
		ms.log.Infof("Message posted: msg hash %s, self node %s", hashString, ms.nodeID)
	case *actor.MessageEnvelope:
		dataHash := tpcmm.NewBlake2bHasher(0)
		hashString := hex.EncodeToString(dataHash.Compute(string(msg.Message.([]byte))))
		ms.log.Infof("Message posted: msg hash %s, self node %s", hashString, ms.nodeID)
	default:
	}
}

func (ms *CSActorMsgStatistics) MessageReceived(message interface{}) {
	switch msg := message.(type) {
	case []byte:
		dataHash := tpcmm.NewBlake2bHasher(0)
		hashString := hex.EncodeToString(dataHash.Compute(string(msg)))
		ms.log.Infof("Message received: msg hash %s, self node %s", hashString, ms.nodeID)
	case *actor.MessageEnvelope:
		dataHash := tpcmm.NewBlake2bHasher(0)
		hashString := hex.EncodeToString(dataHash.Compute(string(msg.Message.([]byte))))
		ms.log.Infof("Message received: msg hash %s, self node %s", hashString, ms.nodeID)
	default:
	}
}

func (CSActorMsgStatistics) MailboxEmpty() {

}

func CreateConsensusActor(level tplogcmm.LogLevel, log tplog.Logger, sysActor *actor.ActorSystem, cons *consensus) (*actor.PID, error) {
	logConsActor := tplog.CreateModuleLogger(level, "ConsensusActor", log)
	consActor := &ConsensusActor{
		log:  logConsActor,
		cons: cons,
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return consActor
	})
	//props = props.WithMailbox(mailbox.Unbounded(&CSActorMsgStatistics{nodeID: cons.nodeID, log: logConsActor}))
	pid, err := sysActor.Root.SpawnNamed(props, "consensus-actor")

	consActor.pid = pid

	return pid, err
}

func (ca *ConsensusActor) Receive(actCtx actor.Context) {
	switch msg := actCtx.Message().(type) {
	case *actor.Started:
		ca.log.Info("Starting, initialize actor here")
	case *actor.Stopping:
		ca.log.Info("Stopping, actor is about to shut down")
	case *actor.Stopped:
		ca.log.Info("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		ca.log.Info("Restarting, actor is about to restart")
	case []byte:
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		ca.cons.dispatch(ctx, actCtx, msg)
		cancel()
	default:
		ca.log.Error("Consensus actor receive invalid msg")
	}
}
