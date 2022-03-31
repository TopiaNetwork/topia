package sync

import (
	"github.com/AsynkronIT/protoactor-go/actor"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type SyncActor struct {
	log    tplog.Logger
	pid    *actor.PID
	syncer *syncer
}

func CreateSyncActor(level tplogcmm.LogLevel, parentLog tplog.Logger, sysActor *actor.ActorSystem, syncer *syncer) (*actor.PID, error) {
	log := tplog.CreateModuleLogger(level, MOD_ACTOR_NAME, parentLog)
	sActor := &SyncActor{
		log:    log,
		syncer: syncer,
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return sActor
	})
	pid, err := sysActor.Root.SpawnNamed(props, MOD_ACTOR_NAME)

	sActor.pid = pid

	return pid, err
}

func (sa *SyncActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		sa.log.Info("Starting, initialize actor here")
	case *actor.Stopping:
		sa.log.Info("Stopping, actor is about to shut down")
	case *actor.Stopped:
		sa.log.Info("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		sa.log.Info("Restarting, actor is about to restart")
	case []byte:
		sa.syncer.dispatch(context, msg)
	default:
		sa.log.Error("Sync actor receive invalid msg")
	}
}
