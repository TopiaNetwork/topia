package eventhub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"sync"

	"github.com/AsynkronIT/protoactor-go/actor"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type EventHub interface {
	Start(sysActor *actor.ActorSystem) error
	Stop()
	Trig(ctx context.Context, name string, data interface{}) error
	Observe(ctx context.Context, evName string, evHandler EventHandler) (string, error) //return observation id
	UnObserve(ctx context.Context, obsID string, evName string) error
}

var once sync.Once

var eventHubG EventHub

func GetEventHub(params ...interface{}) EventHub {
	once.Do(func() {
		if len(params) != 2 {
			panic("Invalid params size: " + fmt.Sprintf("expected 2, actual %d", len(params)))
		}

		var level tplogcmm.LogLevel
		var log tplog.Logger
		ok := true
		if level, ok = params[0].(tplogcmm.LogLevel); !ok {
			panic("Invalid params 0 type: " + fmt.Sprintf("expected LogLevel, actual %s", reflect.TypeOf(params[0]).Name()))
		}
		if log, ok = params[1].(tplog.Logger); !ok {
			panic("Invalid params 1 type: " + fmt.Sprintf("expected LogLevel, actual %s", reflect.TypeOf(params[1]).Name()))
		}

		eventHubG = NewEventHub(level, log)
	})

	return eventHubG
}

type eventHub struct {
	log       tplog.Logger
	sysActor  *actor.ActorSystem
	evPID     *actor.PID
	evManager *eventManager
}

func NewEventHub(level tplogcmm.LogLevel, log tplog.Logger) EventHub {
	logEVActor := tplog.CreateModuleLogger(level, "EventHub", log)

	evManager := newEventManager()

	evManager.registerEvent(EventName_TxReceived, reflect.TypeOf(&txbasic.Transaction{}).String())
	evManager.registerEvent(EventName_TxPrepared, reflect.TypeOf(&txbasic.Transaction{}).String())
	evManager.registerEvent(EventName_TxRollbacked, reflect.TypeOf(&txbasic.Transaction{}).String())
	evManager.registerEvent(EventName_TxCommited, reflect.TypeOf(&txbasic.Transaction{}).String())
	evManager.registerEvent(EventName_BlockCreated, reflect.TypeOf(&tpchaintypes.Block{}).String())
	evManager.registerEvent(EventName_BlockCommited, reflect.TypeOf(&tpchaintypes.Block{}).String())
	evManager.registerEvent(EventName_BlockVerified, reflect.TypeOf(&tpchaintypes.Block{}).String())
	evManager.registerEvent(EventName_BlockConfirmed, reflect.TypeOf(&tpchaintypes.Block{}).String())
	evManager.registerEvent(EventName_BlockAdded, reflect.TypeOf(&tpchaintypes.Block{}).String())

	return &eventHub{
		log:       logEVActor,
		evManager: evManager,
	}
}

func (hub *eventHub) Start(sysActor *actor.ActorSystem) error {
	evPID, err := createEventActor(hub.log, sysActor, hub.evManager)
	if err != nil {
		hub.log.Errorf("create event actor error: %v", err)
		return err
	}

	hub.sysActor = sysActor
	hub.evPID = evPID
	return nil
}

func (hub *eventHub) Trig(ctx context.Context, name string, data interface{}) error {
	hub.sysActor.Root.Send(hub.evPID, &EventMsg{name, data})

	return nil
}

func (hub *eventHub) generateObsID() (string, error) {
	r := make([]byte, 10)
	_, err := rand.Read(r)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(r), nil
}

func (hub *eventHub) Observe(ctx context.Context, evName string, evHandler EventHandler) (string, error) {
	obsID, err := hub.generateObsID()
	if err != nil {
		hub.log.Errorf("Can't generate observation id: %v", err)
		return "", err
	}

	return obsID, hub.evManager.addEvObserver(obsID, evName, evHandler)
}

func (hub *eventHub) UnObserve(ctx context.Context, obsID string, evName string) error {
	return nil
}

func (hub *eventHub) Stop() {
	hub.sysActor.Root.Poison(hub.evPID)
}
