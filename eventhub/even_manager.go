package eventhub

import (
	"context"
	"fmt"
	"sync"

	tplog "github.com/TopiaNetwork/topia/log"
)

type eventManager struct {
	sync     sync.RWMutex
	eventMap map[string]*Event
}

func newEventManager() *eventManager {
	return &eventManager{
		eventMap: make(map[string]*Event),
	}
}

func (evc *eventManager) registerEvent(name string, dataType string) error {
	evc.sync.Lock()
	defer evc.sync.Unlock()

	if _, ok := evc.eventMap[name]; ok {
		return fmt.Errorf("Duplicated event name: %s", name)
	}

	ev := &Event{
		Name:        name,
		DataType:    dataType,
		handlerList: make(map[string]EventHandler),
	}

	evc.eventMap[name] = ev

	return nil
}

func (evc *eventManager) removeEvent(name string) error {
	evc.sync.Lock()
	defer evc.sync.Unlock()

	delete(evc.eventMap, name)

	return nil
}

func (evc *eventManager) addEvObserver(obsID string, evName string, evHandler EventHandler) error {
	evc.sync.RLock()
	defer evc.sync.RUnlock()

	if ev, ok := evc.eventMap[evName]; ok {
		return ev.addObserver(obsID, evHandler)
	}

	return fmt.Errorf("Unsupported event: %s, so can't add the responding observer", evName)
}

func (evc *eventManager) removeEvObserver(obsID string, evName string) error {
	evc.sync.RLock()
	defer evc.sync.RUnlock()

	if ev, ok := evc.eventMap[evName]; ok {
		return ev.removeObserver(obsID)
	}

	return fmt.Errorf("Unsupported event: %s, so can't remove the responding observer", evName)
}

func (evc *eventManager) disptach(ctx context.Context, log tplog.Logger, evMsg *EventMsg) error {
	evc.sync.RLock()
	defer evc.sync.RUnlock()

	if ev, ok := evc.eventMap[evMsg.Name]; ok {
		return ev.process(log, ctx, evMsg.Data)
	}

	return fmt.Errorf("Unsupported event %s", evMsg.Name)
}
