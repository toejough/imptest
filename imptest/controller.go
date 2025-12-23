package imptest

import (
	"sync"
	"time"
)

// Tester is a subset of testing.TB.
type Tester interface {
	Fatalf(format string, args ...any)
	Helper()
}

// Call represents a single call to a mock or callable.
type Call interface {
	Name() string
	Done() bool
}

// Controller manages the call queue and synchronization for a mock or callable.
type Controller[T Call] struct {
	T               Tester
	CallChan        chan T
	callQueue       []T
	queueLock       sync.Mutex
	queueUpdated    chan struct{} // Closed when queue is updated to notify waiters
	queueUpdateLock sync.Mutex    // Protects queueUpdated channel
}

// NewController creates a new controller.
func NewController[T Call](t Tester) *Controller[T] {
	return &Controller[T]{
		T:            t,
		CallChan:     make(chan T, 1),
		queueUpdated: make(chan struct{}),
	}
}

// checkQueue looks for a matching call in the queue and removes it if found.
// Returns the call and true if found, zero value and false otherwise.
func (c *Controller[T]) checkQueue(validator func(T) bool) (T, bool) {
	c.queueLock.Lock()
	defer c.queueLock.Unlock()

	for index, call := range c.callQueue {
		if validator(call) {
			c.callQueue = append(c.callQueue[:index], c.callQueue[index+1:]...)

			return call, true
		}
	}

	var zero T

	return zero, false
}

// GetCall waits for a call that matches the given validator.

func (c *Controller[T]) GetCall(timeout time.Duration, validator func(T) bool) T {
	c.T.Helper()

	// Check queue first
	if call, found := c.checkQueue(validator); found {
		return call
	}

	var timeoutChan <-chan time.Time

	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	for {
		// Get current queue-update notification channel
		c.queueUpdateLock.Lock()
		updateChan := c.queueUpdated
		c.queueUpdateLock.Unlock()

		select {
		case call := <-c.CallChan:
			if validator(call) {
				return call
			}

			c.queueLock.Lock()
			c.callQueue = append(c.callQueue, call)
			c.queueLock.Unlock()

			// Notify all waiting goroutines that queue was updated
			c.queueUpdateLock.Lock()
			close(c.queueUpdated)
			c.queueUpdated = make(chan struct{}) // New channel for next update
			c.queueUpdateLock.Unlock()

			// Re-check queue ourselves (another goroutine might have queued what we want)
			if queuedCall, found := c.checkQueue(validator); found {
				return queuedCall
			}

		case <-updateChan:
			// Queue was updated by another goroutine, re-check it
			if call, found := c.checkQueue(validator); found {
				return call
			}
			// Didn't find a match, loop back to wait again

		case <-timeoutChan:
			c.T.Fatalf("timeout waiting for call matching validator")

			var zero T

			return zero
		}
	}
}

// CallableController manages the state of a single function execution.
type CallableController[T any] struct {
	T          Tester
	ReturnChan chan T
	PanicChan  chan any
	Returned   *T
	Panicked   any
}

// NewCallableController creates a new callable controller.
func NewCallableController[T any](t Tester) *CallableController[T] {
	return &CallableController[T]{
		T:          t,
		ReturnChan: make(chan T, 1),
		PanicChan:  make(chan any, 1),
	}
}

func (c *CallableController[T]) WaitForResponse() {
	if c.Returned != nil || c.Panicked != nil {
		return
	}

	select {
	case ret := <-c.ReturnChan:
		c.Returned = &ret
	case p := <-c.PanicChan:
		c.Panicked = p
	}
}
