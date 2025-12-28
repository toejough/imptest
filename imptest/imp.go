package imptest

import (
	"fmt"
	"reflect"
	"sync"
)

// TestReporter is the minimal interface imptest needs from test frameworks.
// testing.T, testing.B, and *Imp all implement this interface.
type TestReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Expectation represents an expected call to a dependency.
type Expectation struct {
	name         string
	expectedArgs []any
	argMatchers  []Matcher
	useMatchers  bool
	returnValues []any
	panicValue   any
	shouldPanic  bool
	ordered      bool
	actualArgs   []any // stored when the expectation is matched
	called       bool
}

// Imp is the central coordinator for all test interactions.
// It manages the flow of calls, returns, and panics between targets and dependencies.
type Imp struct {
	t            TestReporter
	expectations []*Expectation
	nextIndex    int // for ordered mode - index of next expectation to match
	mu           sync.Mutex
}

// NewImp creates a new Imp coordinator for a test.
func NewImp(t TestReporter) *Imp {
	return &Imp{
		t:            t,
		expectations: make([]*Expectation, 0),
		nextIndex:    0,
	}
}

// Helper marks the calling function as a test helper.
// Implements TestReporter interface.
func (i *Imp) Helper() {
	i.t.Helper()
}

// Fatalf fails the test with a formatted message.
// Implements TestReporter interface.
func (i *Imp) Fatalf(format string, args ...any) {
	i.t.Fatalf(format, args...)
}

// AddExpectation adds a new expectation to the queue.
// Returns the expectation so it can be configured further.
func (i *Imp) AddExpectation(name string, args []any, matchers []Matcher, ordered bool) *Expectation {
	i.mu.Lock()
	defer i.mu.Unlock()

	exp := &Expectation{
		name:         name,
		expectedArgs: args,
		argMatchers:  matchers,
		useMatchers:  len(matchers) > 0,
		ordered:      ordered,
		called:       false,
	}

	i.expectations = append(i.expectations, exp)
	return exp
}

// MatchCall finds and matches an expectation for the given function call.
// In ordered mode: matches the next expectation in the queue
// In unordered mode (Eventually): searches for any matching expectation
func (i *Imp) MatchCall(name string, args []any) *Expectation {
	i.mu.Lock()
	defer i.mu.Unlock()

	// For now, only implement ordered mode
	// Unordered mode (Eventually) will be added in a later iteration
	if i.nextIndex >= len(i.expectations) {
		i.t.Helper()
		i.t.Fatalf("unexpected call to %s with args %v: no more expectations", name, args)
		return nil
	}

	exp := i.expectations[i.nextIndex]
	i.nextIndex++

	// Verify the call matches the expectation
	if exp.name != name {
		i.t.Helper()
		i.t.Fatalf("expected call to %s, got call to %s", exp.name, name)
		return nil
	}

	// Verify arguments
	if !i.argsMatch(exp, args) {
		i.t.Helper()
		if exp.useMatchers {
			i.t.Fatalf("call to %s: arguments %v don't match matchers", name, args)
		} else {
			i.t.Fatalf("call to %s: expected args %v, got %v", name, exp.expectedArgs, args)
		}
		return nil
	}

	// Store actual args and mark as called
	exp.actualArgs = args
	exp.called = true

	return exp
}

// argsMatch checks if actual args match the expectation.
func (i *Imp) argsMatch(exp *Expectation, actualArgs []any) bool {
	if exp.useMatchers {
		if len(actualArgs) != len(exp.argMatchers) {
			return false
		}
		for idx, matcher := range exp.argMatchers {
			success, err := matcher.Match(actualArgs[idx])
			if err != nil || !success {
				return false
			}
		}
		return true
	}

	// Exact matching
	if len(actualArgs) != len(exp.expectedArgs) {
		return false
	}
	for idx, expected := range exp.expectedArgs {
		if !reflect.DeepEqual(actualArgs[idx], expected) {
			return false
		}
	}
	return true
}

// FormatValue formats a value for display in error messages.
func FormatValue(v any) string {
	return fmt.Sprintf("%#v", v)
}
