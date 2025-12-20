package basic_test

import (
	"testing"

	basic "github.com/toejough/imptest/UAT/01-basic-interface-mocking"
)

// TODO: demonstrate without the --name option. and then below a copy of the test, but this time, with the custom
// naming, to demonstrate the difference.

//go:generate go run ../../impgen/main.go basic.Ops --name OpsImp

func TestBasicMocking(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock implementation.
	imp := NewOpsImp(t)

	// Run the code under test in a goroutine so the test can interact with it synchronously.
	go basic.PerformOps(imp.Mock)

	// 1. Intercept 'Add' and provide a return value via InjectResult.
	imp.ExpectCallIs.Add().ExpectArgsAre(1, 2).InjectResult(3)

	// 2. Intercept 'Store' and provide multiple return values via InjectResults.
	imp.ExpectCallIs.Store().ExpectArgsAre("foo", "bar").InjectResults(100, nil)

	// 3. Intercept 'Log' (void method) and signal completion via Resolve.
	imp.ExpectCallIs.Log().ExpectArgsAre("action performed").Resolve()

	// 4. Intercept 'Notify' (variadic) and provide a return value.
	// Note: Variadic arguments are passed normally to ExpectArgsAre.
	imp.ExpectCallIs.Notify().ExpectArgsAre("alert", 1, 2, 3).InjectResult(true)
}
