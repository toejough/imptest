package core

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// GenericCall represents a call to any mocked method.
// It contains the response channel that the mock blocks on.
type GenericCall struct {
	MethodName   string
	Args         []any
	ResponseChan chan GenericResponse
	done         bool
}

// Done returns whether the call has been responded to.
func (c *GenericCall) Done() bool {
	return c.done
}

// MarkDone marks the call as done (called when response is injected).
func (c *GenericCall) MarkDone() {
	c.done = true
}

// Name returns the method name for the Call interface.
func (c *GenericCall) Name() string {
	return c.MethodName
}

// GenericResponse holds the response to send back to a mocked method.
type GenericResponse struct {
	Type         string // "return", "panic"
	ReturnValues []any
	PanicValue   any
}

// Imp is the central coordinator for dependency mocking.
// It wraps the Controller to provide conversational testing.
type Imp struct {
	*Controller[*GenericCall]

	t TestReporter

	pendingMu           sync.Mutex
	pendingExpectations []*PendingExpectation
}

// NewImp creates a new Imp coordinator.
func NewImp(testReporter TestReporter) *Imp {
	imp := &Imp{
		Controller: NewController[*GenericCall](testReporter),
		t:          testReporter,
	}

	// Set up pending matcher to intercept calls for async Eventually()
	imp.PendingMatcher = imp.matchPendingExpectation

	return imp
}

// Fatalf fails the test with a formatted message.
// Implements TestReporter interface.
func (i *Imp) Fatalf(format string, args ...any) {
	i.t.Fatalf(format, args...)
}

// GetCallEventually waits indefinitely for a call matching both the method name and
// argument validator, scanning the entire queue first. The validator returns nil for
// matching args, or an error describing why they didn't match.
func (i *Imp) GetCallEventually(methodName string, validator func([]any) error) *GenericCall {
	combinedValidator := func(call *GenericCall) error {
		if call.MethodName != methodName {
			//nolint:err113 // validation error with dynamic context
			return fmt.Errorf("expected method %q, got %q", methodName, call.MethodName)
		}

		err := validator(call.Args)
		if err != nil {
			return fmt.Errorf("method %q: %w", methodName, err)
		}

		return nil
	}

	return i.Controller.GetCallEventually(combinedValidator)
}

// GetCallOrdered waits for a call matching both the method name and argument validator,
// but fails fast if a non-matching call arrives first. The validator returns nil for
// matching args, or an error describing why they didn't match.
func (i *Imp) GetCallOrdered(
	timeout time.Duration,
	methodName string,
	validator func([]any) error,
) *GenericCall {
	combinedValidator := func(call *GenericCall) error {
		if call.MethodName != methodName {
			//nolint:err113 // validation error with dynamic context
			return fmt.Errorf("expected method %q, got %q", methodName, call.MethodName)
		}

		err := validator(call.Args)
		if err != nil {
			return fmt.Errorf("method %q: %w", methodName, err)
		}

		return nil
	}

	return i.Controller.GetCallOrdered(timeout, combinedValidator)
}

// Helper marks the calling function as a test helper.
// Implements TestReporter interface.
func (i *Imp) Helper() {
	i.t.Helper()
}

// RegisterPendingExpectation registers a new pending expectation.
// Returns the expectation for chaining Return/Panic.
// Also scans the queue for an existing match (in case the call arrived before
// the expectation was registered).
func (i *Imp) RegisterPendingExpectation(
	methodName string,
	validator func([]any) error,
) *PendingExpectation {
	pending := &PendingExpectation{
		MethodName:  methodName,
		Validator:   validator,
		done:        make(chan struct{}),
		matchedChan: make(chan struct{}),
	}

	i.pendingMu.Lock()
	i.pendingExpectations = append(i.pendingExpectations, pending)
	i.pendingMu.Unlock()

	// Check the queue for an existing match (call arrived before expectation)
	i.mu.Lock()

	for idx, call := range i.callQueue {
		if call.MethodName == methodName && validator(call.Args) == nil {
			// Found a match - remove from queue and match the expectation
			i.callQueue = append(
				i.callQueue[:idx],
				i.callQueue[idx+1:]...,
			)
			i.mu.Unlock()
			pending.setMatched(call.ResponseChan, call.Args)

			return pending
		}
	}

	i.mu.Unlock()

	return pending
}

// SetTimeout configures the timeout for all blocking operations.
// A duration of 0 means no timeout (block forever).
func (i *Imp) SetTimeout(d time.Duration) {
	// TODO: implement timeout tracking
	_ = d
}

// Wait blocks until all pending expectations are satisfied.
// Call this after registering expectations with Eventually().
func (i *Imp) Wait() {
	i.pendingMu.Lock()
	expectations := make([]*PendingExpectation, len(i.pendingExpectations))
	copy(expectations, i.pendingExpectations)
	i.pendingMu.Unlock()

	// Wait for each pending expectation to complete
	for _, pe := range expectations {
		<-pe.done
	}
}

// matchPendingExpectation checks if a call matches any pending expectation.
// Returns true if matched (call was handled), false otherwise.
func (i *Imp) matchPendingExpectation(call *GenericCall) bool {
	i.pendingMu.Lock()
	defer i.pendingMu.Unlock()

	for _, pending := range i.pendingExpectations {
		// Skip already matched expectations
		pending.mu.Lock()
		alreadyMatched := pending.Matched
		pending.mu.Unlock()

		if alreadyMatched {
			continue
		}

		// Check method name first
		if pending.MethodName != call.MethodName {
			continue
		}

		// Check validator
		if pending.Validator(call.Args) != nil {
			continue
		}

		// Match found - set the response channel and args
		pending.setMatched(call.ResponseChan, call.Args)

		return true
	}

	return false
}

// valuesEqual checks if two values are equal using reflect.DeepEqual.
func valuesEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
