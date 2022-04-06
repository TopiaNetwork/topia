package eventhub

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	EventName_TxReceived     = "TxReceived"
	EventName_TxPrepared     = "TxPrepared"
	EventName_TxRollbacked   = "TxRollbacked"
	EventName_TxCommited     = "TxCommited"
	EventName_BlockCreated   = "BlockCreated"
	EventName_BlockCommited  = "BlockCommited"
	EventName_BlockVerified  = "BlockVerified"
	EventName_BlockConfirmed = "BlockConfirmed"
	EventName_BlockAdded     = "BlockAdded"
)

type EventTrigger interface {
	Trig(ctx context.Context, name string, data interface{}) error
}

type EventHandler func(ctx context.Context, data interface{}) error

type EventObserver interface {
	Observe(ctx context.Context, evName string, evHandler EventHandler) (string, error) //return observation id
	UnObserve(ctx context.Context, obsID string, evName string) error
}

type EventMsg struct {
	Name string
	Data interface{}
}

type Event struct {
	Name        string
	DataType    string
	sync        sync.RWMutex
	handlerList map[string]EventHandler //observation id -> EventHandler
}

func (ev *Event) addObserver(obsID string, evHandler EventHandler) error {
	ev.sync.Lock()
	defer ev.sync.Unlock()

	if _, ok := ev.handlerList[obsID]; ok {
		return fmt.Errorf("Duplicated observation id: %s", obsID)
	}

	ev.handlerList[obsID] = evHandler

	return nil
}

func (ev *Event) removeObserver(obsID string) error {
	ev.sync.Lock()
	defer ev.sync.Unlock()

	delete(ev.handlerList, obsID)

	return nil
}

func (ev *Event) process(log tplog.Logger, ctx context.Context, data interface{}) error {
	if reflect.TypeOf(data).String() != ev.DataType {
		err := fmt.Errorf("Invalid event data type: expected %s, actual %s", ev.DataType, reflect.TypeOf(data).String())
		log.Errorf("%v", err)
		return err
	}

	ev.sync.Lock()
	defer ev.sync.Unlock()

	for _, evHandler := range ev.handlerList {
		go evHandler(ctx, data)
	}

	return nil
}
