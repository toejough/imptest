package safety_test

import (
	"testing"

	safety "github.com/toejough/imptest/UAT/04-error-and-panic-handling"
)

// TestPropagatePanic demonstrates verifying that a panic is correctly
// propagated when not explicitly handled.
//
// Key Requirements Met:
//  1. Panic Propagation: Verify that panics triggered in dependencies
//     actually reach the caller using ExpectPanicEquals.
func TestPropagatePanic(t *testing.T) {
	t.Parallel()

	depMock := MockCriticalDependency(t)
	wrapper := WrapUnsafeRunner(t, safety.UnsafeRunner)

	// Start UnsafeRunner.
	wrapper.Start(depMock.Interface())

	// Inject a panic into the dependency call.
	depMock.DoWork.ExpectCalledWithExactly().InjectPanicValue("fatal error")

	// Requirement: Verify that the panic was propagated through the runner.
	wrapper.ExpectPanicEquals("fatal error")
}

//go:generate impgen safety.CriticalDependency --dependency
//go:generate impgen safety.SafeRunner --target
//go:generate impgen safety.UnsafeRunner --target

// TestRecoverFromPanic demonstrates injecting a panic into a dependency call
// to verify the system-under-test's recovery logic.
//
// Key Requirements Met:
//  1. Panic Injection: Intentionally trigger a panic when a dependency is
//     called to simulate catastrophic failures.
//  2. Recovery Verification: Ensure that the calling code correctly recovers
//     from the panic and returns an appropriate value.
func TestRecoverFromPanic(t *testing.T) {
	t.Parallel()

	depMock := MockCriticalDependency(t)
	wrapper := WrapSafeRunner(t, safety.SafeRunner)

	// Start SafeRunner.
	wrapper.Start(depMock.Interface())

	// Requirement: Inject a panic into the dependency call.
	depMock.DoWork.ExpectCalledWithExactly().InjectPanicValue("boom")

	// Requirement: Verify that SafeRunner recovered and returned false.
	wrapper.ExpectReturnsEqual(false)
}
