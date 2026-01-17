// Package core provides the internal implementation of imptest's mock and
// target controller infrastructure.
package core

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

type Call interface {
	Name() string
	Done() bool
}

type CallableController[T any] struct {
	T          TestReporter
	ReturnChan chan T
	PanicChan  chan any
	Returned   *T
	Panicked   any
}

// WaitForResponse blocks until the wrapped function returns or panics.
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

type Controller[T Call] struct {
	T        TestReporter
	Timer    Timer
	CallChan chan T

	mu        sync.Mutex   // Protects callQueue and waiters
	callQueue []T          // Unclaimed calls waiting for future waiters
	waiters   []*waiter[T] // Goroutines waiting for matching calls

	// PendingMatcher is called for each incoming call before checking waiters.
	// If it returns true, the call was handled by a pending expectation.
	// This allows Imp to intercept calls for async Eventually() handling.
	PendingMatcher func(T) bool
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

// checkFailFast checks the first waiter for fail-fast mode.
// Returns true if fail-fast triggered (caller should return), false otherwise.
// Must be called with c.mu held.
func (c *Controller[T]) checkFailFast(call T) bool {
	if len(c.waiters) == 0 {
		return false
	}

	firstWaiter := c.waiters[0]
	if !firstWaiter.failOnMismatch {
		return false
	}

	err := firstWaiter.validator(call)
	if err == nil {
		return false
	}

	// First waiter is ordered and call doesn't match - fail immediately
	c.waiters = c.waiters[1:] // Remove failed waiter
	c.mu.Unlock()

	c.T.Fatalf("ordered mode fail-fast: %v", err)

	return true
}

// dispatchLoop receives calls and either matches them to waiters or queues them.
func (c *Controller[T]) dispatchLoop() {
	for call := range c.CallChan {
		// Check pending expectations FIRST (before taking lock)
		// This allows async Eventually() to intercept calls
		if c.PendingMatcher != nil && c.PendingMatcher(call) {
			continue // Call was handled by pending expectation
		}

		c.mu.Lock()

		// Check first waiter for fail-fast mode BEFORE trying other waiters
		if c.checkFailFast(call) {
			return
		}

		// Try to match with waiters
		matched := false

		for i, w := range c.waiters {
			if w.validator(call) == nil {
				w.result <- call

				c.waiters = append(c.waiters[:i], c.waiters[i+1:]...)
				matched = true

				break
			}
		}

		// If still no match, queue for future waiters
		if !matched {
			c.callQueue = append(c.callQueue, call)
		}

		c.mu.Unlock()
	}
}

type PendingCompletion struct {
	t    TestReporter
	mu   sync.Mutex
	done chan struct{}

	// Expected values (set by ExpectReturnsEqual/ExpectPanicEquals)
	expectReturn       bool
	expectedReturnVals []any
	expectPanic        bool
	expectedPanicVal   any
	useMatchers        bool // true for ExpectReturnsMatch/ExpectPanicMatches

	// Actual values (set when call completes)
	completed   bool
	returnedVal any // The Returned struct
	panickedVal any
}

// ExpectReturnMatch registers an expectation that the call returns values matching the matchers.

// If already completed, check now

// ExpectPanic registers an expectation that the call panics with the given value.
func (pc *PendingCompletion) ExpectPanic(value any) {
	pc.mu.Lock()
	pc.expectPanic = true
	pc.expectedPanicVal = value
	pc.useMatchers = false
	completed := pc.completed
	returnedVal := pc.returnedVal
	panickedVal := pc.panickedVal
	pc.mu.Unlock()

	// If already completed, check now
	if completed {
		pc.checkExpectation(returnedVal, panickedVal)
	}
}

// ExpectReturn registers an expectation that the call returns the given values.
func (pc *PendingCompletion) ExpectReturn(values ...any) {
	pc.mu.Lock()
	pc.expectReturn = true
	pc.expectedReturnVals = values
	pc.useMatchers = false
	completed := pc.completed
	returnedVal := pc.returnedVal
	panickedVal := pc.panickedVal
	pc.mu.Unlock()

	// If already completed, check now
	if completed {
		pc.checkExpectation(returnedVal, panickedVal)
	}
}

// ExpectPanicMatch registers an expectation that the call panics with a value matching the matcher.

// If already completed, check now

// SetCompleted is called when the call completes with a return value or panic.
// If an expectation is already registered, it checks the expectation.
func (pc *PendingCompletion) SetCompleted(returnedVal, panickedVal any) {
	pc.mu.Lock()
	pc.completed = true
	pc.returnedVal = returnedVal
	pc.panickedVal = panickedVal
	hasExpectation := pc.expectReturn || pc.expectPanic
	pc.mu.Unlock()

	// If expectation already registered, check now
	if hasExpectation {
		pc.checkExpectation(returnedVal, panickedVal)
	}
}

// checkExpectation verifies the actual values against the expected values.
func (pc *PendingCompletion) checkExpectation(returnedVal, panickedVal any) {
	pc.t.Helper()

	pc.mu.Lock()
	expectReturn := pc.expectReturn
	expectPanic := pc.expectPanic
	expectedReturnVals := pc.expectedReturnVals
	expectedPanicVal := pc.expectedPanicVal
	useMatchers := pc.useMatchers
	pc.mu.Unlock()

	if expectReturn {
		if panickedVal != nil {
			pc.t.Fatalf("expected function to return, but it panicked with: %v", panickedVal)
		}

		pc.checkReturnValues(returnedVal, expectedReturnVals, useMatchers)
	} else if expectPanic {
		if panickedVal == nil {
			pc.t.Fatalf("expected function to panic, but it returned")
		}

		ok, msg := MatchValue(panickedVal, expectedPanicVal)
		if !ok {
			pc.t.Fatalf("panic value: %s", msg)
		}
	}

	close(pc.done)
}

// checkReturnValues uses reflection to compare return values.
// returnedVal is a struct with Result0, Result1, etc. fields.
func (pc *PendingCompletion) checkReturnValues(
	returnedVal any,
	expectedVals []any,
	useMatchers bool,
) {
	pc.t.Helper()

	if returnedVal == nil {
		if len(expectedVals) > 0 {
			pc.t.Fatalf("expected return values but got nil")
		}

		return
	}

	// Use reflection to get struct fields
	val := reflect.ValueOf(returnedVal)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		pc.t.Fatalf("expected return value to be struct, got %T", returnedVal)

		return
	}

	for index, expected := range expectedVals {
		fieldName := fmt.Sprintf("Result%d", index)
		field := val.FieldByName(fieldName)

		if !field.IsValid() {
			pc.t.Fatalf("return value struct missing field %s", fieldName)

			return
		}

		actual := field.Interface()

		if useMatchers {
			ok, msg := MatchValue(actual, expected)
			if !ok {
				pc.t.Fatalf("return value %d: %s", index, msg)
			}
		} else if !reflect.DeepEqual(actual, expected) {
			pc.t.Fatalf("expected return value %d to be %v, got %v", index, expected, actual)
		}
	}
}

type PendingExpectation struct {
	mu           sync.Mutex
	MethodName   string
	Validator    func([]any) error
	ReturnValues []any                  // nil until Return called
	PanicValue   any                    // non-nil if Panic called
	IsPanic      bool                   // true if this should panic instead of return
	Matched      bool                   // true when a call matched the validator
	Injected     bool                   // true when Return/Panic was called
	responseChan chan<- GenericResponse // set when validator matches
	done         chan struct{}          // signals when BOTH matched AND injected
	matchedArgs  []any                  // args from the call that matched
	matchedChan  chan struct{}          // closed when a call is matched
}

// GetMatchedArgs returns the args from the matched call.
// Blocks until a call is matched if not yet matched.
func (pe *PendingExpectation) GetMatchedArgs() []any {
	pe.WaitForMatch()

	pe.mu.Lock()
	args := pe.matchedArgs
	pe.mu.Unlock()

	return args
}

// Panic specifies the value the mock should panic with.
// Can be called before or after the call is matched.
func (pe *PendingExpectation) Panic(value any) {
	pe.mu.Lock()
	pe.PanicValue = value
	pe.IsPanic = true
	pe.Injected = true
	matched := pe.Matched
	responseChan := pe.responseChan
	pe.mu.Unlock()

	// If already matched, send response now
	if matched && responseChan != nil {
		responseChan <- GenericResponse{
			Type:       "panic",
			PanicValue: value,
		}

		close(pe.done)
	}
}

// Return specifies the values the mock should return.
// Can be called before or after the call is matched.
func (pe *PendingExpectation) Return(values ...any) {
	pe.mu.Lock()
	pe.ReturnValues = values
	pe.Injected = true
	matched := pe.Matched
	responseChan := pe.responseChan
	pe.mu.Unlock()

	// If already matched, send response now
	if matched && responseChan != nil {
		responseChan <- GenericResponse{
			Type:         "return",
			ReturnValues: values,
		}

		close(pe.done)
	}
}

// WaitForMatch blocks until a call matches this expectation.
func (pe *PendingExpectation) WaitForMatch() {
	pe.mu.Lock()
	matchedChan := pe.matchedChan
	pe.mu.Unlock()

	if matchedChan != nil {
		<-matchedChan
	}
}

// setMatched is called when a call matches this expectation.
// If already injected, sends response immediately.
func (pe *PendingExpectation) setMatched(responseChan chan<- GenericResponse, args []any) {
	pe.mu.Lock()
	pe.Matched = true
	pe.responseChan = responseChan
	pe.matchedArgs = args
	injected := pe.Injected
	isPanic := pe.IsPanic
	returnValues := pe.ReturnValues
	panicValue := pe.PanicValue
	matchedChan := pe.matchedChan
	pe.mu.Unlock()

	// Signal that a call was matched
	if matchedChan != nil {
		close(matchedChan)
	}

	// If already injected, send response now
	if injected {
		if isPanic {
			responseChan <- GenericResponse{
				Type:       "panic",
				PanicValue: panicValue,
			}
		} else {
			responseChan <- GenericResponse{
				Type:         "return",
				ReturnValues: returnValues,
			}
		}

		close(pe.done)
	}
}

type TargetController struct {
	t                  TestReporter
	mu                 sync.Mutex
	pendingCompletions []*PendingCompletion
}

// NewTargetController creates a new target controller.
func NewTargetController(t TestReporter) *TargetController {
	return &TargetController{t: t}
}

// RegisterPendingCompletion registers a new pending completion.
func (tc *TargetController) RegisterPendingCompletion() *PendingCompletion {
	completion := &PendingCompletion{
		t:    tc.t,
		done: make(chan struct{}),
	}

	tc.mu.Lock()
	tc.pendingCompletions = append(tc.pendingCompletions, completion)
	tc.mu.Unlock()

	return completion
}

type TestReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

type Timer interface {
	After(d time.Duration) <-chan time.Time
}

// NewCallableController creates a new callable controller.
func NewCallableController[T any](t TestReporter) *CallableController[T] {
	return &CallableController[T]{
		T:          t,
		ReturnChan: make(chan T, 1),
		PanicChan:  make(chan any, 1),
	}
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

type realTimer struct{}

func (realTimer) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type waiter[T any] struct {
	validator      func(T) error // Returns nil for match, error for mismatch
	result         chan T
	failOnMismatch bool // If true, fail immediately on mismatch instead of queuing
}
