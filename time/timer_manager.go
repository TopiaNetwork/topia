package time

import (
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	tplog "github.com/TopiaNetwork/topia/log"
)

type TimerFunc func() bool

type TimerType byte

const (
	TimerType_Unknown TimerType = iota
	TimerType_OneTime
	TimerType_Periodic
)

type TimerStatus int32

const (
	TimerStatus_Unknown TimerStatus = iota
	TimerStatus_Stopped
	TimerStatus_Running
)

type TimerManager interface {
	RegisterPeriodicTimer(name string, routine TimerFunc, interval uint32)

	RegisterOneTimeRoutine(name string, routine TimerFunc, delay uint32)

	RemoveTimer(name string)

	ClearTimers()

	StartTimer(name string, triggerNextTicker bool)

	StartAndTriggerTimer(name string)

	StopTimer(name string)
}

type timerRoutine struct {
	id              string
	handler         TimerFunc
	interval        uint32
	lastTicker      uint64
	status          TimerStatus
	triggerNextTick int32
	rType           TimerType
}

type timerManager struct {
	log       tplog.Logger
	beginTime time.Time
	timer     *time.Ticker
	ticker    uint64
	id        string
	timers    sync.Map // key: string, value: *timerRoutine
}

func NewTimerManager(id string) TimerManager {
	ticker := &timerManager{
		id:        id,
		beginTime: time.Now(),
	}

	go ticker.routine()

	return ticker
}

func (m *timerManager) addTimer(name string, tr *timerRoutine) {
	m.timers.Store(name, tr)
}

func (m *timerManager) getTimer(name string) *timerRoutine {
	if v, ok := m.timers.Load(name); ok {
		return v.(*timerRoutine)
	}
	return nil
}

func (m *timerManager) trigger(routine *timerRoutine) bool {
	defer func() {
		if routine.rType == TimerType_OneTime {
			m.RemoveTimer(routine.id)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			m.log.Errorf("error：%v", r)
			s := debug.Stack()
			m.log.Error(string(s))
		}
	}()

	t := m.ticker
	lastTicker := atomic.LoadUint64(&routine.lastTicker)

	if TimerStatus(atomic.LoadInt32((*int32)(&routine.status))) != TimerStatus_Running {
		return false
	}

	b := false
	if lastTicker < t && atomic.CompareAndSwapUint64(&routine.lastTicker, lastTicker, t) {
		b = routine.handler()
	}
	return b
}

func (m *timerManager) routine() {
	m.timer = time.NewTicker(1 * time.Second)
	for range m.timer.C {
		m.ticker++
		m.timers.Range(func(key, value interface{}) bool {
			rt := value.(*timerRoutine)
			if (TimerStatus(atomic.LoadInt32((*int32)(&rt.status))) == TimerStatus_Running && m.ticker-rt.lastTicker >= uint64(rt.interval)) || atomic.LoadInt32(&rt.triggerNextTick) == 1 {
				atomic.CompareAndSwapInt32(&rt.triggerNextTick, 1, 0)
				go m.trigger(rt)
			}
			return true
		})
	}
}

func (m *timerManager) RegisterPeriodicTimer(name string, routine TimerFunc, interval uint32) {
	if rt := m.getTimer(name); rt != nil {
		return
	}
	r := &timerRoutine{
		rType:           TimerType_Periodic,
		interval:        interval,
		handler:         routine,
		lastTicker:      m.ticker,
		id:              name,
		status:          TimerStatus_Stopped,
		triggerNextTick: 0,
	}
	m.addTimer(name, r)
}

func (m *timerManager) RegisterOneTimeRoutine(name string, routine TimerFunc, delay uint32) {
	if rt := m.getTimer(name); rt != nil {
		rt.lastTicker = m.ticker
		return
	}

	r := &timerRoutine{
		rType:           TimerType_OneTime,
		interval:        delay,
		handler:         routine,
		lastTicker:      m.ticker,
		id:              name,
		status:          TimerStatus_Running,
		triggerNextTick: 0,
	}
	m.addTimer(name, r)
}

func (m *timerManager) RemoveTimer(name string) {
	m.timers.Delete(name)
}

func (m *timerManager) ClearTimers() {
	m.timers.Range(func(key, value interface{}) bool {
		m.timers.Delete(key)
		return true
	})
}

func (m *timerManager) StartTimer(name string, triggerNextTicker bool) {
	routine := m.getTimer(name)
	if routine == nil {
		return
	}
	atomic.CompareAndSwapInt32(&routine.triggerNextTick, 0, 1)
	atomic.CompareAndSwapInt32((*int32)(&routine.status), int32(TimerStatus_Stopped), int32(TimerStatus_Running))
}

func (m *timerManager) StartAndTriggerTimer(name string) {
	routine := m.getTimer(name)
	if routine == nil {
		return
	}

	atomic.CompareAndSwapInt32((*int32)(&routine.status), int32(TimerStatus_Stopped), int32(TimerStatus_Running))
}

func (m *timerManager) StopTimer(name string) {
	routine := m.getTimer(name)
	if routine == nil {
		return
	}

	atomic.CompareAndSwapInt32((*int32)(&routine.status), int32(TimerStatus_Running), int32(TimerStatus_Stopped))
}
