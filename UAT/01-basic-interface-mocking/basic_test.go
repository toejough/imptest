package basic_test

import (
	"testing"

	basic "github.com/toejough/imptest/UAT/01-basic-interface-mocking"
)

//go:generate go run ../../impgen/main.go basic.BasicOps --name BasicOpsImp

func TestBasicMocking(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock.
	mock := NewBasicOpsImp(t)

	// Run the code under test in a goroutine so the test can interact with it synchronously.
	go basic.PerformOps(mock.Mock)

	// 1. Intercept 'Add' and provide a return value via InjectResult.
	mock.ExpectCallIs.Add().ExpectArgsAre(1, 2).InjectResult(3)

	// 2. Intercept 'Store' and provide multiple return values via InjectResults.
	mock.ExpectCallIs.Store().ExpectArgsAre("foo", "bar").InjectResults(100, nil)

	// 3. Intercept 'Log' (void method) and signal completion via Resolve.
	mock.ExpectCallIs.Log().ExpectArgsAre("action performed").Resolve()

	// 4. Intercept 'Notify' (variadic) and provide a return value.
	// Note: Variadic arguments are passed normally to ExpectArgsAre.
	mock.ExpectCallIs.Notify().ExpectArgsAre("alert", 1, 2, 3).InjectResult(true)
}
