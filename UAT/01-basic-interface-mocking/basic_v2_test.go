package basic_test

// This file demonstrates v2 API for dependency mocking.

import (
	"testing"

	basic "github.com/toejough/imptest/UAT/01-basic-interface-mocking"
	"github.com/toejough/imptest/imptest"
)

// TestV2_BasicMocking demonstrates the v2 API conversational pattern.
func TestV2_BasicMocking(t *testing.T) {
	t.Parallel()

	// Create the coordinator
	imp := imptest.NewImp(t)

	// Create mock for the dependency interface
	ops := MockOps(imp)

	// Start execution in goroutine
	go basic.PerformOps(ops.Interface())

	// Interactive Control Pattern (v2): ExpectCalled -> InjectReturnValues

	// 1. Intercept 'Add' and provide a return value
	ops.Add.ExpectCalledWithExactly(1, 2).InjectReturnValues(3)

	// 2. Intercept 'Store' and provide multiple return values
	ops.Store.ExpectCalledWithExactly("foo", "bar").InjectReturnValues(100, nil)

	// 3. Intercept 'Log' (void method) and signal completion
	ops.Log.ExpectCalledWithExactly("action performed").InjectReturnValues()

	// 4. Intercept 'Notify' (variadic) and provide a return value
	// Note: Variadic arguments are passed normally to ExpectCalledWithExactly
	ops.Notify.ExpectCalledWithExactly("alert", 1, 2, 3).InjectReturnValues(true)

	// 5. Intercept 'Finish' (no args) and provide a return value
	ops.Finish.ExpectCalledWithExactly().InjectReturnValues(true)
}
