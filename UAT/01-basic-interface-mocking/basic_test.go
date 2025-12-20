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
	// TODO: rename mock to imp, everywhere. The returns from New...Imp should always be imp or xxxImp, to really
	// differentiate this from a normal mock, especially because the mock itself is actually in imp.Mock. Calling the
	// return value 'imp' makes it clearer that this is the generated implementation, rather than just a mock object.

	// Initialize the generated mock.
	mock := NewOpsImp(t)

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
