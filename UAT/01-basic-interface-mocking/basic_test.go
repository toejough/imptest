// Package basic_test demonstrates the core philosophy of imptest:
// Zero mock code. Full interactive control.
package basic_test

import (
	"testing"

	basic "github.com/toejough/imptest/UAT/01-basic-interface-mocking"
)

// imptest identifies whether the target is an interface or a function.
// By default, it generates a name like <Interface>Imp (OpsImp in this case).
// You can also use --name to specify a custom name (CustomOpsImp here).
//go:generate go run ../../impgen/main.go basic.Ops
//go:generate go run ../../impgen/main.go basic.Ops --name CustomOpsImp

// TestBasicMocking demonstrates the "Interactive Control" pattern using a
// custom-named mock implementation.
func TestBasicMocking(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock implementation using its default name.
	imp := NewOpsImp(t)

	// Run the code under test in a goroutine so the test can interact with it synchronously.
	go basic.PerformOps(imp.Mock)

	// Interactive Control Pattern: Expect -> Inject -> Resolve

	// 1. Intercept 'Add' and provide a return value via InjectResult.
	imp.ExpectCallIs.Add().ExpectArgsAre(1, 2).InjectResult(3)

	// 2. Intercept 'Store' and provide multiple return values via InjectResults.
	imp.ExpectCallIs.Store().ExpectArgsAre("foo", "bar").InjectResults(100, nil)

	// 3. Intercept 'Log' (void method) and signal completion via Resolve.
	imp.ExpectCallIs.Log().ExpectArgsAre("action performed").Resolve()

	// 4. Intercept 'Notify' (variadic) and provide a return value.
	// Note: Variadic arguments are passed normally to ExpectArgsAre.
	imp.ExpectCallIs.Notify().ExpectArgsAre("alert", 1, 2, 3).InjectResult(true)

	// 5. Intercept 'Finish' (no args) and provide a return value.
	imp.ExpectCallIs.Finish().InjectResult(true)
}

// TestCustomNaming demonstrates that the --name flag can be used to generate a custom imp.
func TestCustomNaming(t *testing.T) {
	t.Parallel()

	_ = NewCustomOpsImp()(t)
}
