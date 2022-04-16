package chain

import (
	"github.com/AsynkronIT/protoactor-go/actor"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type NodeActor struct {
	log   tplog.Logger
	pid   *actor.PID
	chain *chain
}

func CreateChainActor(level tplogcmm.LogLevel, parentLog tplog.Logger, sysActor *actor.ActorSystem, chain *chain) (*actor.PID, error) {
	log := tplog.CreateModuleLogger(level, MOD_NAME, parentLog)
	nActor := &NodeActor{
		log:   log,
		chain: chain,
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return nActor
	})
	pid, err := sysActor.Root.SpawnNamed(props, MOD_ACTOR_NAME)

	nActor.pid = pid

	return pid, err
}

func (na *NodeActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		na.log.Info("Starting, initialize actor here")
	case *actor.Stopping:
		na.log.Info("Stopping, actor is about to shut down")
	case *actor.Stopped:
		na.log.Info("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		na.log.Info("Restarting, actor is about to restart")
	case []byte:
		na.chain.dispatch(context, msg)
	default:
		na.log.Error("Sync actor receive invalid msg")
	}
}
