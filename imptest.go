// Package imptest provides test mocking infrastructure for Go.
// It generates type-safe mocks from interfaces with interactive test control.
//
// # User API
//
// These are meant to be used directly in test code:
//
//   - [TestReporter] - interface for test frameworks (usually *testing.T)
//   - [GetOrCreateImp] - get/create shared coordinator for a test (used by generated code)
//   - [Wait] - block until all async expectations for a test are satisfied
//   - [SetTimeout] - configure timeout for blocking operations
//
// For matchers (BeAny, Satisfy), import the match package:
//
//	import . "github.com/toejough/imptest/match"
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
	"time"

	"github.com/toejough/imptest/internal/core"
)

type Call = core.Call

type CallableController[T any] = core.CallableController[T]

type Controller[T Call] = core.Controller[T]

type DependencyArgs = core.DependencyArgs

type DependencyCall = core.DependencyCall

type DependencyMethod = core.DependencyMethod

// NewDependencyMethod creates a new DependencyMethod.
func NewDependencyMethod(imp *Imp, methodName string) *DependencyMethod {
	return core.NewDependencyMethod(imp, methodName)
}

type GenericCall = core.GenericCall

type GenericResponse = core.GenericResponse

type Imp = core.Imp

type Matcher = core.Matcher

type PendingCompletion = core.PendingCompletion

type PendingExpectation = core.PendingExpectation

type TargetController = core.TargetController

// NewTargetController creates a new target controller.
func NewTargetController(t TestReporter) *TargetController {
	return core.NewTargetController(t)
}

type TestReporter = core.TestReporter

type Timer = core.Timer

// GetOrCreateImp returns the Imp for the given test, creating one if needed.
// Multiple calls with the same TestReporter return the same Imp instance.
// This enables coordination between mocks and wrappers in the same test.
//
// If the TestReporter supports Cleanup (like *testing.T), the Imp is
// automatically removed from the registry when the test completes.
func GetOrCreateImp(t TestReporter) *Imp {
	return core.GetOrCreateImp(t)
}

// MatchValue checks if actual matches expected.
func MatchValue(actual, expected any) (bool, string) {
	return core.MatchValue(actual, expected)
}

// NewCallableController creates a new callable controller.
func NewCallableController[T any](t TestReporter) *CallableController[T] {
	return core.NewCallableController[T](t)
}

// SetTimeout configures the timeout for all blocking operations in the test.
// A duration of 0 means no timeout (block forever).
//
// If no Imp has been created for t yet, one is created.
func SetTimeout(t TestReporter, d time.Duration) {
	core.SetTimeout(t, d)
}

// Wait blocks until all async expectations registered under t are satisfied.
// This is the package-level wait that coordinates across all mocks/wrappers
// sharing the same TestReporter.
//
// If no Imp has been created for t yet, Wait returns immediately.
func Wait(t TestReporter) {
	core.Wait(t)
}
