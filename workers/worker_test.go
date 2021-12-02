package workers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setup() {
	// reset the collection
	activeWorkers = map[*RunLoop]*RunLoop{}
	lock = sync.Mutex{}
}

func TestWorkers(t *testing.T) {
	setup()
	t.Run("test creates run loop", testRunLoopCreatesWorkerAndAddsToLoop)
	t.Run("test can be cancelled", testRunLoopTimeoutCancelsWork)
}

func testRunLoopCreatesWorkerAndAddsToLoop(t *testing.T) {
	done := false

	DoWork(func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		done = true
		return nil
	})

	assert.Eventually(t, func() bool { return ActiveWorkers() == 1 }, 100*time.Millisecond, 1*time.Millisecond)
	assert.Eventually(t, func() bool { return done }, 100*time.Millisecond, 1*time.Millisecond)
	assert.Eventually(t, func() bool { return ActiveWorkers() == 0 }, 100*time.Millisecond, 1*time.Millisecond)
}

func testRunLoopTimeoutCancelsWork(t *testing.T) {
	done := false
	timeout = 5 * time.Millisecond // set the timeout to a value that forces a timeout

	t.Cleanup(func() {
		timeout = 30 * time.Second
	})

	DoWork(func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		// expect the context to timeout
		if ctx.Err() != nil {
			done = true
			return nil
		}

		return nil
	})

	assert.Eventually(t, func() bool { return ActiveWorkers() == 1 }, 100*time.Millisecond, 1*time.Millisecond)
	assert.Eventually(t, func() bool { return done }, 100*time.Millisecond, 1*time.Millisecond)
	assert.Eventually(t, func() bool { return ActiveWorkers() == 0 }, 100*time.Millisecond, 1*time.Millisecond)
}
