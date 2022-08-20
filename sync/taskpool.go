package sync

//
//import (
//	"errors"
//	"log"
//	"sync"
//	"sync/atomic"
//	"time"
//)
//
//// errors
//var (
//	ErrPoolCap    = errors.New("invalid task pool cap")
//	ErrPoolClosed = errors.New("task pool already closed")
//)
//
//const DefaultWorkerSize = uint64(50)
//
//// task pool running status
//const (
//	poolRunning = 1
//	poolStopped = 0
//)
//
//type JobType uint
//
//const (
//	msgBlock JobType = iota
//	msgEpoch
//	msgChain
//	msgNode
//	msgAccount
//)
//
//// Task task to-do
//type Task struct {
//	job     func(JobType, interface{})
//	jobType JobType
//	data    interface{}
//}
//
//type TaskPool struct {
//	capacity       uint64
//	runningWorkers uint64
//	status         int64
//	chanTask       chan *Task
//	panicHandler   func(interface{})
//	sync.Mutex
//}
//
//func NewTaskPool(capacity uint64) (*TaskPool, error) {
//	if capacity <= 0 {
//		return nil, ErrPoolCap
//	}
//	p := &TaskPool{
//		capacity: capacity,
//		status:   poolRunning,
//		chanTask: make(chan *Task, capacity),
//	}
//
//	return p, nil
//}
//
//func (p *TaskPool) checkWorker() {
//	p.Lock()
//	defer p.Unlock()
//
//	if p.runningWorkers == 0 && len(p.chanTask) > 0 {
//		p.run()
//	}
//}
//
//func (p *TaskPool) GetCap() uint64 {
//	return p.capacity
//}
//
//func (p *TaskPool) GetRunningWorkers() uint64 {
//	return atomic.LoadUint64(&p.runningWorkers)
//}
//
//func (p *TaskPool) incRunningWorker() {
//	atomic.AddUint64(&p.runningWorkers, 1)
//}
//
//func (p *TaskPool) decRunningWorker() {
//	if atomic.LoadUint64(&p.runningWorkers) > 0 {
//		atomic.AddUint64(&p.runningWorkers, ^uint64(0))
//	}
//}
//
//// Put a task to pool
//func (p *TaskPool) Put(task *Task) error {
//	p.Lock()
//	defer p.Unlock()
//
//	if p.status == poolStopped {
//		return ErrPoolClosed
//	}
//
//	// run worker
//	if p.GetRunningWorkers() < p.GetCap() {
//		p.run()
//	}
//
//	// send task
//	if p.status == poolRunning {
//		p.chanTask <- task
//	}
//
//	return nil
//}
//
//func (p *TaskPool) run() {
//	p.incRunningWorker()
//
//	go func() {
//		defer func() {
//			p.decRunningWorker()
//			if r := recover(); r != nil {
//				if p.panicHandler != nil {
//					p.panicHandler(r)
//				} else {
//					log.Printf("Worker panic: %s\n", r)
//				}
//			}
//			p.checkWorker() // check worker avoid no worker running
//		}()
//
//		for {
//			select {
//			case task, ok := <-p.chanTask:
//				if !ok {
//					return
//				}
//				task.job(task.jobType, task.data)
//			}
//		}
//	}()
//}
//
//func (p *TaskPool) setStatus(status int64) bool {
//	p.Lock()
//	defer p.Unlock()
//
//	if p.status == status {
//		return false
//	}
//
//	p.status = status
//
//	return true
//}
//
//// Close pool graceful
//func (p *TaskPool) Close() {
//
//	if !p.setStatus(poolStopped) { // stop put task
//		return
//	}
//
//	for len(p.chanTask) > 0 { // wait all task be consumed
//		time.Sleep(1e6) // reduce CPU load
//	}
//
//	close(p.chanTask)
//}
