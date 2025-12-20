package imptest

import (
	"sync"
	"time"
)

// Tester is a subset of *testing.T.
type Tester interface {
	Fatalf(format string, args ...any)
	Helper()
	Parallel()
}

// Call represents a single call to a mock or callable.
type Call interface {
	Name() string
	Done() bool
}

// Controller manages the call queue and synchronization for a mock or callable.
type Controller[T Call] struct {
	T         Tester
	CallChan  chan T
	callQueue []T
	queueLock sync.Mutex
}

// NewController creates a new controller.
func NewController[T Call](t Tester) *Controller[T] {
	return &Controller[T]{
		T:        t,
		CallChan: make(chan T, 1),
	}
}

// GetCall waits for a call that matches the given validator.

func (c *Controller[T]) GetCall(timeout time.Duration, validator func(T) bool) T {
	c.T.Helper()

	c.queueLock.Lock()

	// Check queue first

	for index, call := range c.callQueue {
		if validator(call) {
			c.callQueue = append(c.callQueue[:index], c.callQueue[index+1:]...)

			c.queueLock.Unlock()

			return call
		}
	}

	c.queueLock.Unlock()

	var timeoutChan <-chan time.Time

	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	for {
		select {
		case call := <-c.CallChan:
			if validator(call) {
				return call
			}

			c.queueLock.Lock()
			c.callQueue = append(c.callQueue, call)
			c.queueLock.Unlock()
		case <-timeoutChan:
			c.T.Fatalf("timeout waiting for call matching validator")

			var zero T

			return zero
		}
	}
}
