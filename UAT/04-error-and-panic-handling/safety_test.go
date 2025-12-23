package safety_test

import (
	"testing"

	safety "github.com/toejough/imptest/UAT/04-error-and-panic-handling"
)

//go:generate impgen safety.CriticalDependency --name CriticalDependencyImp
//go:generate impgen safety.SafeRunner --name SafeRunnerImp
//go:generate impgen safety.UnsafeRunner --name UnsafeRunnerImp

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

	depImp := NewCriticalDependencyImp(t)
	runnerImp := NewSafeRunnerImp(t, safety.SafeRunner)

	// Start SafeRunner.
	runnerImp.Start(depImp.Mock)

	// Requirement: Inject a panic into the dependency call.
	depImp.ExpectCallIs.DoWork().InjectPanic("boom")

	// Requirement: Verify that SafeRunner recovered and returned false.
	runnerImp.ExpectReturnedValuesAre(false)
}

// TestPropagatePanic demonstrates verifying that a panic is correctly
// propagated when not explicitly handled.
//
// Key Requirements Met:
//  1. Panic Propagation: Verify that panics triggered in dependencies
//     actually reach the caller using ExpectPanicWith.
func TestPropagatePanic(t *testing.T) {
	t.Parallel()

	depImp := NewCriticalDependencyImp(t)
	runnerImp := NewUnsafeRunnerImp(t, safety.UnsafeRunner)

	// Start UnsafeRunner.
	runnerImp.Start(depImp.Mock)

	// Inject a panic into the dependency call.
	depImp.ExpectCallIs.DoWork().InjectPanic("fatal error")

	// Requirement: Verify that the panic was propagated through the runner.
	runnerImp.ExpectPanicWith("fatal error")
}
