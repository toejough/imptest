package imptest

import (
	"fmt"
	"reflect"
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

// Imp is the central coordinator for v2 API.
// It wraps the v1 Controller to provide conversational testing.
type Imp struct {
	*Controller[*GenericCall]

	t TestReporter
}

// NewImp creates a new Imp coordinator.
func NewImp(testReporter TestReporter) *Imp {
	// Wrap the test reporter to satisfy the Tester interface
	tester := &testerAdapter{t: testReporter}

	return &Imp{
		Controller: NewController[*GenericCall](tester),
		t:          testReporter,
	}
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
func (i *Imp) GetCallOrdered(timeout time.Duration, methodName string, validator func([]any) error) *GenericCall {
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

// TestReporter is the minimal interface imptest needs from test frameworks.
// testing.T, testing.B, and *Imp all implement this interface.
type TestReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

// testerAdapter adapts TestReporter to Tester interface.
type testerAdapter struct {
	t TestReporter
}

func (a *testerAdapter) Fatalf(format string, args ...any) {
	a.t.Fatalf(format, args...)
}

func (a *testerAdapter) Helper() {
	a.t.Helper()
}

// valuesEqual checks if two values are equal using reflect.DeepEqual.
func valuesEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
