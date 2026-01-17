package safety_test

import (
	"testing"

	safety "github.com/toejough/imptest/UAT/variations/behavior/panic-handling"
)

// TestPropagatePanic demonstrates verifying that a panic is correctly
// propagated when not explicitly handled.
//
// Key Requirements Met:
//  1. Panic Propagation: Verify that panics triggered in dependencies
//     actually reach the caller using ExpectPanicEquals.
func TestPropagatePanic(t *testing.T) {
	t.Parallel()

	depMock, depImp := MockCriticalDependency(t)
	wrapper := WrapUnsafeRunner(t, safety.UnsafeRunner)

	// Start UnsafeRunner.
	call := wrapper.Method.Start(depMock)

	// Inject a panic into the dependency call.
	depImp.DoWork.Expect().Panic("fatal error")

	// Requirement: Verify that the panic was propagated through the runner.
	call.ExpectPanicEquals("fatal error")
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

	depMock, depImp := MockCriticalDependency(t)
	wrapper := WrapSafeRunner(t, safety.SafeRunner)

	// Start SafeRunner.
	call := wrapper.Method.Start(depMock)

	// Requirement: Inject a panic into the dependency call.
	depImp.DoWork.Expect().Panic("boom")

	// Requirement: Verify that SafeRunner recovered and returned false.
	call.ExpectReturnsEqual(false)
}
