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

// Timer abstracts time-based operations for testability.
type Timer interface {
	After(d time.Duration) <-chan time.Time
}

// realTimer is the default timer implementation that uses the standard time package.
type realTimer struct{}

func (realTimer) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Call represents a single call to a mock or callable.
type Call interface {
	Name() string
	Done() bool
}

// waiter represents a goroutine waiting for a matching call.
type waiter[T any] struct {
	validator func(T) bool
	result    chan T
}

// Controller manages the call queue and synchronization for a mock or callable.
type Controller[T Call] struct {
	T        Tester
	Timer    Timer
	CallChan chan T

	mu        sync.Mutex   // Protects callQueue and waiters
	callQueue []T          // Unclaimed calls waiting for future waiters
	waiters   []*waiter[T] // Goroutines waiting for matching calls
}

// NewController creates a new controller with the default real timer.
func NewController[T Call](t Tester) *Controller[T] {
	return NewControllerWithTimer[T](t, realTimer{})
}

// NewControllerWithTimer creates a new controller with a custom timer for testing.
func NewControllerWithTimer[T Call](t Tester, timer Timer) *Controller[T] {
	ctrl := &Controller[T]{
		T:        t,
		Timer:    timer,
		CallChan: make(chan T, 1),
	}
	go ctrl.dispatchLoop()

	return ctrl
}

// GetCall waits for a call that matches the given validator.
func (c *Controller[T]) GetCall(timeout time.Duration, validator func(T) bool) T {
	c.T.Helper()

	c.mu.Lock()

	// Check queue first (while holding lock)
	for i, call := range c.callQueue {
		if validator(call) {
			c.callQueue = append(c.callQueue[:i], c.callQueue[i+1:]...)
			c.mu.Unlock()

			return call
		}
	}

	// Register as waiter BEFORE unlocking (this prevents race conditions)
	myWaiter := &waiter[T]{
		validator: validator,
		result:    make(chan T, 1),
	}
	c.waiters = append(c.waiters, myWaiter)
	c.mu.Unlock()

	// Wait for result with timeout
	var timeoutChan <-chan time.Time

	if timeout > 0 {
		timeoutChan = c.Timer.After(timeout)
	}

	select {
	case call := <-myWaiter.result:
		return call
	case <-timeoutChan:
		// Remove self from waiters list
		c.mu.Lock()

		for i, waiter := range c.waiters {
			if waiter == myWaiter {
				c.waiters = append(c.waiters[:i], c.waiters[i+1:]...)

				break
			}
		}

		c.mu.Unlock()

		c.T.Fatalf("timeout waiting for call matching validator")

		var zero T

		return zero
	}
}

// dispatchLoop receives calls and either matches them to waiters or queues them.
func (c *Controller[T]) dispatchLoop() {
	for call := range c.CallChan {
		c.mu.Lock()

		// Try to match with waiting goroutines
		matched := false

		for i, w := range c.waiters {
			if w.validator(call) {
				w.result <- call

				c.waiters = append(c.waiters[:i], c.waiters[i+1:]...)
				matched = true

				break
			}
		}

		// No matching waiter - queue for future
		if !matched {
			c.callQueue = append(c.callQueue, call)
		}

		c.mu.Unlock()
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
