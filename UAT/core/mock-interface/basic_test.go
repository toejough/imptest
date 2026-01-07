// Package basic_test demonstrates the core philosophy of imptest:
// Zero mock code. Full interactive control.
package basic_test

import (
	"testing"

	basic "github.com/toejough/imptest/UAT/core/mock-interface"
)

// imptest identifies whether the target is an interface or a function.
// V2 API uses --dependency flag to generate MockXxx instead of XxxImp.
//go:generate impgen basic.Ops --dependency

// TestBasicMocking demonstrates the "Interactive Control" pattern using the v2 API.
func TestBasicMocking(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock using the v2 API.
	mock := MockOps(t)

	// Run the code under test in a goroutine so the test can interact with it synchronously.
	go basic.PerformOps(mock.Interface())

	// Interactive Control Pattern: Expect -> Inject -> Resolve

	// 1. Intercept 'Add' and provide a return value via InjectReturnValues.
	mock.Add.ExpectCalledWithExactly(1, 2).InjectReturnValues(3)

	// 2. Intercept 'Store' and provide multiple return values via InjectReturnValues.
	mock.Store.ExpectCalledWithExactly("foo", "bar").InjectReturnValues(100, nil)

	// 3. Intercept 'Log' (void method) and signal completion via InjectReturnValues with no args.
	mock.Log.ExpectCalledWithExactly("action performed").InjectReturnValues()

	// 4. Intercept 'Notify' (variadic) and provide a return value.
	// Note: Variadic arguments are passed normally to ExpectCalledWithExactly.
	mock.Notify.ExpectCalledWithExactly("alert", 1, 2, 3).InjectReturnValues(true)

	// 5. Intercept 'Finish' (no args) and provide a return value.
	mock.Finish.ExpectCalledWithExactly().InjectReturnValues(true)
}

// You can also use --name to specify a custom base name (CustomOps here, which becomes MockCustomOps).
//go:generate impgen basic.Ops --name MockCustomOps --dependency

// TestCustomNaming demonstrates that the --name flag can be used to generate a custom mock.
func TestCustomNaming(t *testing.T) {
	t.Parallel()

	_ = MockCustomOps(t)
}
