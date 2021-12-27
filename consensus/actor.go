package consensus

import (
	"github.com/AsynkronIT/protoactor-go/actor"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type ConsensusActor struct {
	log  tplog.Logger
	pid  *actor.PID
	cons *consensus
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
	pid, err := sysActor.Root.SpawnNamed(props, "consensus-actor")

	consActor.pid = pid

	return pid, err
}

func (ca *ConsensusActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		ca.log.Info("Starting, initialize actor here")
	case *actor.Stopping:
		ca.log.Info("Stopping, actor is about to shut down")
	case *actor.Stopped:
		ca.log.Info("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		ca.log.Info("Restarting, actor is about to restart")
	case []byte:
		ca.log.Infof("Received consensus message data=%v", msg)
		ca.cons.dispatch(context, msg)
	default:
		ca.log.Error("Consensus actor receive invalid msg")
	}
}
