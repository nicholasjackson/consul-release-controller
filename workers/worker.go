package workers

import (
	"context"
	"sync"
	"time"
)

var activeWorkers map[*RunLoop]*RunLoop
var lock sync.Mutex
var timeout = 30 * time.Second

func init() {
	activeWorkers = map[*RunLoop]*RunLoop{}
	lock = sync.Mutex{}
}

// DoWork creates a background work process with a default timeout
func DoWork(work func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	rl := &RunLoop{ctx, cancel, work}
	rl.Run()
}

func ActiveWorkers() int {
	return len(activeWorkers)
}

// RunLoop allows work to run in the background, and remain cancellable
type RunLoop struct {
	ctx    context.Context
	cancel context.CancelFunc
	work   func(ctx context.Context) error
}

// Run is a blocking function that monitors deployments
func (r *RunLoop) Run() {
	// register ourself
	lock.Lock()
	activeWorkers[r] = r
	lock.Unlock()

	go func() {
		// clean up resources if we finish before timeout
		defer r.cancel()

		r.work(r.ctx)

		// remove from running tasks
		lock.Lock()
		delete(activeWorkers, r)
		lock.Unlock()
	}()
}

// IsRunning returns true when the run loop is running
func (r *RunLoop) IsRunning() bool {
	return r.ctx.Err() == nil
}
