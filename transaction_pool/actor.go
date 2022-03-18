package transactionpool

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)



type TransactionPoolActor struct {
	log    tplog.Logger
	pid    *actor.PID
	txPool *transactionPool
}

func CreateTransactionPoolActor(level tplogcmm.LogLevel, log tplog.Logger, sysActor *actor.ActorSystem, txPool *transactionPool) (*actor.PID, error) {
	logTxPoolActor := tplog.CreateModuleLogger(level, "TransactionPoolActor", log)
	tpActor := &TransactionPoolActor{
		log: logTxPoolActor,
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return tpActor
	})
	pid, err := sysActor.Root.SpawnNamed(props, "transaction_pool-actor")

	tpActor.pid = pid

	return pid, err
}

func (ta *TransactionPoolActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		ta.log.Info("Starting, initialize actor here")
	case *actor.Stopping:
		ta.log.Info("Stopping, actor is about to shut down")
	case *actor.Stopped:
		ta.log.Info("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		ta.log.Info("Restarting, actor is about to restart")
	case []byte:
		ta.txPool.Dispatch(context, msg)
	default:
		ta.log.Error("TransactionPool actor receive invalid msg")
	}
}
