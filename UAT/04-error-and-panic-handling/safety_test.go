package safety_test

import (
	"testing"

	safety "github.com/toejough/imptest/UAT/04-error-and-panic-handling"
)

//go:generate go run ../../impgen/main.go safety.CriticalDependency --name CriticalDependencyImp
//go:generate go run ../../impgen/main.go safety.SafeRunner --name SafeRunnerImp --call
//go:generate go run ../../impgen/main.go safety.UnsafeRunner --name UnsafeRunnerImp --call

func TestRecoverFromPanic(t *testing.T) {
	t.Parallel()

	depImp := NewCriticalDependencyImp(t)
	runnerImp := NewSafeRunnerImp(t, safety.SafeRunner)

	// Start SafeRunner.
	runnerImp.Start(depImp.Mock)

	// Inject a panic into the dependency call.
	depImp.ExpectCallIs.DoWork().InjectPanic("boom")

	// Verify that SafeRunner recovered and returned false.
	runnerImp.ExpectReturnedValuesAre(false)
}

func TestPropagatePanic(t *testing.T) {
	t.Parallel()

	depImp := NewCriticalDependencyImp(t)
	runnerImp := NewUnsafeRunnerImp(t, safety.UnsafeRunner)

	// Start UnsafeRunner.
	runnerImp.Start(depImp.Mock)

	// Inject a panic into the dependency call.
	depImp.ExpectCallIs.DoWork().InjectPanic("fatal error")

	// Verify that the panic was propagated through the runner.
	runnerImp.ExpectPanicWith("fatal error")
}
