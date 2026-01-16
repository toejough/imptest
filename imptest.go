// Package imptest provides test mocking infrastructure for Go.
// It generates type-safe mocks from interfaces with interactive test control.
//
// # User API
//
// These are meant to be used directly in test code:
//
//   - [Any] - matcher that accepts any value
//   - [Satisfies] - matcher using a custom predicate function
//   - [TestReporter] - interface for test frameworks (usually *testing.T)
//   - [GetOrCreateImp] - get/create shared coordinator for a test (used by generated code)
//   - [Wait] - block until all async expectations for a test are satisfied
//
// # Generated Code API
//
// These are used by code generated via `impgen`. Users interact with them
// indirectly through the generated type-safe wrappers:
//
//   - [Imp] - coordinator for dependency mocks
//   - [Controller] - manages call queue and synchronization
//   - [DependencyMethod], [DependencyCall], [DependencyArgs] - mock internals
//   - [CallableController], [TargetController] - wrapper internals
//   - [GenericCall], [GenericResponse] - low-level call/response types
//   - [PendingExpectation], [PendingCompletion] - async expectation internals
//   - [Matcher], [Timer], [Call] - supporting interfaces and types
package imptest

import (
	"github.com/toejough/imptest/internal/core"
)

// Call represents a single call to a mock or callable.
type Call = core.Call

// CallableController manages the state of a single function execution.
type CallableController[T any] = core.CallableController[T]

// NewCallableController creates a new callable controller.
func NewCallableController[T any](t TestReporter) *CallableController[T] {
	return core.NewCallableController[T](t)
}

// Controller manages the call queue and synchronization for a mock or callable.
type Controller[T Call] = core.Controller[T]

// NewController creates a new controller with the default real timer.

// DependencyArgs provides access to the actual arguments that were passed to the dependency.
type DependencyArgs = core.DependencyArgs

// DependencyCall represents an expected call to a dependency.
type DependencyCall = core.DependencyCall

// DependencyMethod represents a method on a mocked interface.
type DependencyMethod = core.DependencyMethod

// NewDependencyMethod creates a new DependencyMethod.
func NewDependencyMethod(imp *Imp, methodName string) *DependencyMethod {
	return core.NewDependencyMethod(imp, methodName)
}

// Types re-exported from internal/core.

// GenericCall represents a call to any mocked method.
type GenericCall = core.GenericCall

// GenericResponse holds the response to send back to a mocked method.
type GenericResponse = core.GenericResponse

// Imp is the central coordinator for dependency mocking.
type Imp = core.Imp

// Matcher defines the interface for flexible value matching.
type Matcher = core.Matcher

// PendingCompletion represents an expectation on a target wrapper call
// that hasn't been satisfied yet.
type PendingCompletion = core.PendingCompletion

// PendingExpectation represents an expectation registered with Eventually()
// that hasn't been matched yet.
type PendingExpectation = core.PendingExpectation

// TargetController manages pending completions for target wrappers.
type TargetController = core.TargetController

// NewControllerWithTimer creates a new controller with a custom timer for testing.

// NewTargetController creates a new target controller.
func NewTargetController(t TestReporter) *TargetController {
	return core.NewTargetController(t)
}

// TestReporter is the minimal interface imptest needs from test frameworks.
type TestReporter = core.TestReporter

// Timer abstracts time-based operations for testability.
type Timer = core.Timer

// Any returns a matcher that matches any value.
func Any() Matcher {
	return core.Any()
}

// MatchValue checks if actual matches expected.
func MatchValue(actual, expected any) (bool, string) {
	return core.MatchValue(actual, expected)
}

// Satisfies returns a matcher that uses a predicate function to check for a match.
func Satisfies[T any](predicate func(T) error) Matcher {
	return core.Satisfies(predicate)
}
