package chain

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/chain/types"

	tplog "github.com/TopiaNetwork/topia/log"
)

type NodeActor struct {
	log   tplog.Logger
	pid   *actor.PID
	chain *chain
}

func CreateChainActor(log tplog.Logger, sysActor *actor.ActorSystem, chain *chain) (*actor.PID, error) {
	nActor := &NodeActor{
		log:   log,
		chain: chain,
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return nActor
	})
	pid, err := sysActor.Root.SpawnNamed(props, types.MOD_ACTOR_NAME)

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
		na.log.Error("Chain actor receive invalid msg")
	}
}
