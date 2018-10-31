package network

import (
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/SevenPlusPlus/gomesh/pkg/types"
	"github.com/SevenPlusPlus/gomesh/pkg/utils"
)

type workerFunc func()

type WorkerPool struct {
	workers []*worker
	closeChan chan struct{}
}

type worker struct {
	idx int
	cbFuncChan chan workerFunc
	closeChan chan struct{}
}

func newWorker(idx int, cbChanSize int, closeChan chan struct{}) *worker {
	w := &worker{
		idx: idx,
		cbFuncChan: make(chan workerFunc, cbChanSize),
		closeChan: closeChan,
	}
	go w.start()
	return w
}

func (w *worker) start() {
	for{
		select {
		case <- w.closeChan:
			log.DefaultLogger.Infof("worker-%d is stopping\n", w.idx)
			return
		case cb := <- w.cbFuncChan:
			cb()
		}
	}
}

func (w *worker) put(cb workerFunc) error {
	select {
	case w.cbFuncChan <- cb:
		return nil
	default:
		return types.ErrWouldBlock
	}
}

func newWorkerPool(size int) *WorkerPool {
	return newWorkerPoolWithOpts(size, 1024)
}

func newWorkerPoolWithOpts(size int, workerChanSize int) *WorkerPool {
	pool := &WorkerPool{
		workers: make([]*worker, size),
		closeChan: make(chan struct{}),
	}
	for i := range pool.workers {
		pool.workers[i] = newWorker(i, workerChanSize, pool.closeChan)
	}
	return pool
}

func (wp *WorkerPool) Put(k interface{}, cb func()) error {
	code := utils.HashCode(k)
	return wp.workers[code & uint32(len(wp.workers)-1)].put(workerFunc(cb))
}

func (wp *WorkerPool) Close() {
	close(wp.closeChan)
}

func (wp *WorkerPool) Size() int{
	return len(wp.workers)
}