package cuba

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
)

type Task[I any] func(*Handle[I])

var PoolAbortedErr = errors.New("pool has been aborted")

const (
	PoolRun = iota
	PoolFinish
	PoolAbort
)

type Pool[I any] struct {
	mutex      *sync.Mutex
	bucket     Bucket[I]
	cond       *sync.Cond
	numWorkers int32
	maxWorkers int32
	state      int
	task       Task[I]
	wg         *sync.WaitGroup
}

// New constructs a new Cuba thread pool.
//
// The worker callback will be called by multiple goroutines in parallel, so is
// expected to be thread safe.
//
// Bucket affects the order that items will be processed in. cuba.NewQueue()
// provides FIFO ordering, while cuba.NewStack() provides LIFO ordered work.
func New[I any](task Task[I], bucket Bucket[I]) *Pool[I] {
	m := &sync.Mutex{}
	return &Pool[I]{
		mutex:      m,
		bucket:     bucket,
		cond:       sync.NewCond(m),
		task:       task,
		maxWorkers: int32(runtime.NumCPU()),
		wg:         &sync.WaitGroup{},
		state:      PoolRun,
	}
}

// SetMaxWorkers sets the maximum number of worker goroutines.
//
// Default: runtime.NumCPU() (i.e. the number of CPU cores available)
func (pool *Pool[I]) SetMaxWorkers(n int32) {
	pool.maxWorkers = n
}

// Push an item into the worker pool. This will be scheduled to run on a worker
// immediately.
func (pool *Pool[I]) Push(item I) error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.state == PoolAbort {
		return PoolAbortedErr
	}

	// The ideal might be to have a fixed pool of worker goroutines which all
	// close down when the work is done.
	// However, since the bucket can drain down to 0 and appear done before the
	// final worker queues more items it's a little complicated.
	// Having a floating pool means we can restart workers as we discover more
	// work to be done, which solves this problem at the cost of a little
	// inefficiency.
	if atomic.LoadInt32(&pool.numWorkers) < pool.maxWorkers {
		atomic.AddInt32(&pool.numWorkers, 1)
		pool.wg.Add(1)
		go pool.runWorker()
	}

	pool.bucket.Push(item)
	pool.cond.Signal()

	return nil
}

// PushAll pushes multiple items into the worker pool.
//
// Compared to Push() this only acquires the lock once, so may reduce lock
// contention.
func (pool *Pool[I]) PushAll(items []I) error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.state == PoolAbort {
		return PoolAbortedErr
	}

	for i := 0; i < len(items); i++ {
		if atomic.LoadInt32(&pool.numWorkers) >= pool.maxWorkers {
			break
		}
		atomic.AddInt32(&pool.numWorkers, 1)
		pool.wg.Add(1)
		go pool.runWorker()
	}

	pool.bucket.PushAll(items)
	pool.cond.Broadcast()

	return nil
}

// Finish waits for all work to complete, and allows goroutines to shut
// down.
func (pool *Pool[I]) Finish() {
	pool.mutex.Lock()

	pool.state = PoolFinish
	pool.cond.Broadcast()

	pool.mutex.Unlock()
	pool.wg.Wait()
}

func (pool *Pool[I]) Abort() {
	pool.mutex.Lock()

	pool.state = PoolAbort
	pool.cond.Broadcast()

	pool.mutex.Unlock()
	pool.wg.Wait()
}

func (pool *Pool[I]) next() (interface{}, bool) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	for pool.bucket.IsEmpty() {
		if pool.state == PoolFinish || pool.state == PoolAbort {
			return nil, false
		}
		pool.cond.Wait()
	}

	item := pool.bucket.Pop()

	return item, true
}

func (pool *Pool[I]) runWorker() {
	handle := Handle[I]{
		pool: pool,
	}
	for {
		item, ok := pool.next()
		if !ok {
			break
		}
		handle.item = item

		pool.task(&handle)
		handle.Sync()
	}
	atomic.AddInt32(&pool.numWorkers, -1)

	pool.wg.Done()
}
