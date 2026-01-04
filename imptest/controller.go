package imptest

import (
	"sync"
	"time"
)

// Call represents a single call to a mock or callable.
type Call interface {
	Name() string
	Done() bool
}

// CallableController manages the state of a single function execution.
type CallableController[T any] struct {
	T          TestReporter
	ReturnChan chan T
	PanicChan  chan any
	Returned   *T
	Panicked   any
}

// NewCallableController creates a new callable controller.
func NewCallableController[T any](t TestReporter) *CallableController[T] {
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

// Controller manages the call queue and synchronization for a mock or callable.
type Controller[T Call] struct {
	T        TestReporter
	Timer    Timer
	CallChan chan T

	mu        sync.Mutex   // Protects callQueue and waiters
	callQueue []T          // Unclaimed calls waiting for future waiters
	waiters   []*waiter[T] // Goroutines waiting for matching calls
}

// NewController creates a new controller with the default real timer.
func NewController[T Call](t TestReporter) *Controller[T] {
	return NewControllerWithTimer[T](t, realTimer{})
}

// NewControllerWithTimer creates a new controller with a custom timer for testing.
func NewControllerWithTimer[T Call](t TestReporter, timer Timer) *Controller[T] {
	ctrl := &Controller[T]{
		T:        t,
		Timer:    timer,
		CallChan: make(chan T, 1),
	}
	go ctrl.dispatchLoop()

	return ctrl
}

// GetCall waits for a call that matches the given validator. The validator returns
// nil for a match, or an error describing why the call didn't match.
func (c *Controller[T]) GetCall(timeout time.Duration, validator func(T) error) T {
	c.T.Helper()

	c.mu.Lock()

	// Check queue first (while holding lock) - scans entire queue
	for i, call := range c.callQueue {
		if validator(call) == nil {
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

// GetCallEventually waits indefinitely for a call that matches the given validator,
// scanning the entire queue before waiting. The validator returns nil for a match,
// or an error describing why the call didn't match.
func (c *Controller[T]) GetCallEventually(validator func(T) error) T {
	c.T.Helper()

	c.mu.Lock()

	// Eventually mode: scan ENTIRE queue for a match
	for i, call := range c.callQueue {
		if validator(call) == nil {
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

	// Wait indefinitely for a matching call
	return <-myWaiter.result
}

// GetCallOrdered waits for a call that matches the given validator, but fails
// fast if a non-matching call arrives first. The validator returns nil for a match,
// or an error describing why the call didn't match.
//
//nolint:funlen // sequential logic for ordered call matching cannot be easily extracted
func (c *Controller[T]) GetCallOrdered(
	timeout time.Duration,
	validator func(T) error,
) T {
	c.T.Helper()

	c.mu.Lock()

	// Check queue first (while holding lock) - ordered mode checks FIRST call only
	if len(c.callQueue) > 0 {
		firstCall := c.callQueue[0]

		err := validator(firstCall)
		if err == nil {
			// First queued call matches - return it
			c.callQueue = c.callQueue[1:]
			c.mu.Unlock()

			return firstCall
		}

		// First queued call doesn't match - fail fast in ordered mode
		c.callQueue = c.callQueue[1:]
		c.mu.Unlock()

		c.T.Fatalf("ordered mode fail-fast: %v", err)

		var zero T

		return zero
	}

	// Register as ordered waiter (fail-fast mode)
	myWaiter := &waiter[T]{
		validator:      validator,
		result:         make(chan T, 1),
		failOnMismatch: true,
	}
	c.waiters = append(c.waiters, myWaiter)
	c.mu.Unlock()

	// Wait for result with timeout (caller handles mismatch channel)
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

		// Check first waiter for fail-fast mode BEFORE trying other waiters
		matched := false

		if len(c.waiters) > 0 {
			firstWaiter := c.waiters[0]

			if firstWaiter.failOnMismatch {
				err := firstWaiter.validator(call)
				if err != nil {
					// First waiter is ordered and call doesn't match - fail immediately
					c.waiters = c.waiters[1:] // Remove failed waiter
					c.mu.Unlock()

					c.T.Fatalf("ordered mode fail-fast: %v", err)

					return
				}
			}
		}

		// Try to match with waiters
		if !matched {
			for i, w := range c.waiters {
				if w.validator(call) == nil {
					w.result <- call

					c.waiters = append(c.waiters[:i], c.waiters[i+1:]...)
					matched = true

					break
				}
			}
		}

		// If still no match, queue for future waiters
		if !matched {
			c.callQueue = append(c.callQueue, call)
		}

		c.mu.Unlock()
	}
}

// TestReporter is the minimal interface imptest needs from test frameworks.
// testing.T, testing.B, and *Imp all implement this interface.
type TestReporter interface {
	Helper()
	Fatalf(format string, args ...any)
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

// waiter represents a goroutine waiting for a matching call.
type waiter[T any] struct {
	validator      func(T) error // Returns nil for match, error for mismatch
	result         chan T
	failOnMismatch bool // If true, fail immediately on mismatch instead of queuing
}
