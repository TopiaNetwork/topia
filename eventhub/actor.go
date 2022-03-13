package eventhub

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"

	tplog "github.com/TopiaNetwork/topia/log"
)

type EventActor struct {
	log       tplog.Logger
	pid       *actor.PID
	evManager *eventManager
}

func createEventActor(log tplog.Logger, sysActor *actor.ActorSystem, evManager *eventManager) (*actor.PID, error) {
	evActor := &EventActor{
		log:       log,
		evManager: evManager,
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return evActor
	})
	pid, err := sysActor.Root.SpawnNamed(props, "event-actor")

	evActor.pid = pid

	return pid, err
}

func (ea *EventActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		ea.log.Info("Starting, initialize actor here")
	case *actor.Stopping:
		ea.log.Info("Stopping, actor is about to shut down")
	case *actor.Stopped:
		ea.log.Info("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		ea.log.Info("Restarting, actor is about to restart")
	case *EventMsg:
		ea.log.Infof("Received event message data=%v", msg)
		ctx := context.WithValue(context.Background(), "actorID", actorCtx.Self().Id)
		ctx = context.WithValue(ctx, "actorAddr", actorCtx.Self().Address)
		ea.evManager.disptach(ctx, ea.log, msg)
	}
}
