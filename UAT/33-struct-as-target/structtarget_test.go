// Package calculator_test demonstrates that impgen can wrap struct types with --target flag.
//
// STRUCT AS TARGET vs METHOD AS TARGET:
// - Method as target (UAT-02): Wraps individual methods one at a time (Calculator.Add, Calculator.Multiply, etc.)
// - Struct as target (UAT-33): Wraps the entire struct type, creating wrappers for ALL methods at once
//
// This UAT tests whether struct types can be used with --target flag for wrapping.
// The Capability Matrix shows "?" for "Struct type as Target", so this test will
// determine if this capability is supported or should be marked as unsupported.
//
// Expected behavior (similar to interface wrapping in UAT-32):
// - `impgen Calculator --target` should generate `WrapCalculator` function
// - `WrapCalculator(t, calc)` returns a wrapper with method interceptors
// - Each method on the struct gets its own interceptor (Add, Multiply, Divide, Process)
// - Calls are intercepted and recorded while still delegating to the original struct
package calculator_test

import (
	"testing"

	calculator "github.com/toejough/imptest/UAT/33-struct-as-target"
)

// Generate target wrapper for Calculator struct
// This is the key directive: attempting to wrap a struct type with --target
// Expected: Should generate WrapCalculator function that wraps ALL methods
//
//go:generate impgen calculator.Calculator --target

// TestWrapCalculator_BasicWrapping demonstrates basic struct wrapping with --target.
// This test verifies that we can wrap a Calculator struct to intercept all method calls.
func TestWrapCalculator_BasicWrapping(t *testing.T) {
	t.Parallel()

	// Create a calculator instance
	calc := calculator.NewCalculator(2, 10)

	// Wrap the entire Calculator struct to intercept all method calls
	// This is the key test: Can impgen wrap a struct with --target?
	wrapper := WrapCalculator(t, calc)

	// Call Add through the wrapped struct
	// Expected: wrapper.Add should exist and intercept the call
	result := wrapper.Add.Start(5, 3).WaitForResponse()

	// Verify the result is correct (5 + 3 + offset=10 = 18)
	if result != 18 {
		t.Errorf("expected 18, got %d", result)
	}

	// Verify the call was intercepted and recorded
	calls := wrapper.Add.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 Add call, got %d", len(calls))
	}

	if calls[0].Params.A != 5 || calls[0].Params.B != 3 {
		t.Errorf("expected params (5, 3), got (%d, %d)", calls[0].Params.A, calls[0].Params.B)
	}

	if calls[0].Returns.Result0 != 18 {
		t.Errorf("expected return value 18, got %d", calls[0].Returns.Result0)
	}
}

// TestWrapCalculator_ErrorHandling demonstrates wrapping methods that return errors.
// This test verifies that struct wrapping handles error return values correctly.
func TestWrapCalculator_ErrorHandling(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(2, 0)
	wrapper := WrapCalculator(t, calc)

	// Test error path with negative input
	result, err := wrapper.Process.Start(-5).WaitForResponse()

	// Verify error was returned
	if err == nil {
		t.Fatal("expected error from Process with negative input")
	}

	if result != "" {
		t.Errorf("expected empty result string, got %q", result)
	}

	// Test success path
	result, err = wrapper.Process.Start(10).WaitForResponse()
	// Verify success
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Process should: Multiply(10) = 20, Add(20, 5) = 25 (no offset), so "Result: 25"
	expectedResult := "Result: 25"
	if result != expectedResult {
		t.Errorf("expected %q, got %q", expectedResult, result)
	}

	// Verify both calls were recorded
	calls := wrapper.Process.GetCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 Process calls, got %d", len(calls))
	}
}

// TestWrapCalculator_MethodInteraction demonstrates wrapping a method that calls other methods.
// This test verifies that struct wrapping intercepts both the high-level method AND
// the methods it calls internally (if called through the wrapper).
func TestWrapCalculator_MethodInteraction(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(3, 10)
	wrapper := WrapCalculator(t, calc)

	// Call Process, which internally calls Multiply and Add
	// Process(5): Multiply(5) = 15, Add(15, 5) = 30
	result, err := wrapper.Process.Start(5).WaitForResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedResult := "Result: 30"
	if result != expectedResult {
		t.Errorf("expected %q, got %q", expectedResult, result)
	}

	// Verify Process call was recorded
	processCalls := wrapper.Process.GetCalls()
	if len(processCalls) != 1 {
		t.Fatalf("expected 1 Process call, got %d", len(processCalls))
	}

	if processCalls[0].Params.Input != 5 {
		t.Errorf("Process call: expected input 5, got %d", processCalls[0].Params.Input)
	}

	// Note: The internal calls to Multiply and Add from within Process
	// will NOT be intercepted because they're called on the original struct (c),
	// not through the wrapper. This is expected behavior - struct wrapping
	// only intercepts calls made explicitly through the wrapper.
	//
	// To intercept ALL calls including internal ones, callers would need to
	// use the wrapped methods exclusively.
}

// TestWrapCalculator_MultipleMethodCalls demonstrates intercepting multiple different methods.
// This test verifies that struct wrapping creates interceptors for ALL methods on the struct.
func TestWrapCalculator_MultipleMethodCalls(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(3, 5)
	wrapper := WrapCalculator(t, calc)

	// Call multiple different methods through the wrapped struct
	// All should be intercepted independently

	// Test Add: 10 + 20 + offset=5 = 35
	addResult := wrapper.Add.Start(10, 20).WaitForResponse()
	if addResult != 35 {
		t.Errorf("Add: expected 35, got %d", addResult)
	}

	// Test Multiply: 7 * multiplier=3 = 21
	mulResult := wrapper.Multiply.Start(7).WaitForResponse()
	if mulResult != 21 {
		t.Errorf("Multiply: expected 21, got %d", mulResult)
	}

	// Test Divide: 100 / 4 = 25, true
	divResult, divOk := wrapper.Divide.Start(100, 4).WaitForResponse()
	if divResult != 25 || !divOk {
		t.Errorf("Divide: expected (25, true), got (%d, %v)", divResult, divOk)
	}

	// Verify each method's calls were intercepted independently
	addCalls := wrapper.Add.GetCalls()
	if len(addCalls) != 1 {
		t.Errorf("expected 1 Add call, got %d", len(addCalls))
	}

	mulCalls := wrapper.Multiply.GetCalls()
	if len(mulCalls) != 1 {
		t.Errorf("expected 1 Multiply call, got %d", len(mulCalls))
	}

	divCalls := wrapper.Divide.GetCalls()
	if len(divCalls) != 1 {
		t.Errorf("expected 1 Divide call, got %d", len(divCalls))
	}
}

// TestWrapCalculator_MultipleReturnValues demonstrates wrapping methods with multiple return values.
// This test verifies that struct wrapping handles methods with multiple returns (like Divide).
func TestWrapCalculator_MultipleReturnValues(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(1, 0)
	wrapper := WrapCalculator(t, calc)

	// Test successful division
	quotient, divideOk := wrapper.Divide.Start(50, 5).WaitForResponse()
	if quotient != 10 || !divideOk {
		t.Errorf("expected (10, true), got (%d, %v)", quotient, divideOk)
	}

	// Test division by zero
	quotient, divideOk = wrapper.Divide.Start(50, 0).WaitForResponse()
	if quotient != 0 || divideOk {
		t.Errorf("expected (0, false), got (%d, %v)", quotient, divideOk)
	}

	// Verify both calls were recorded
	calls := wrapper.Divide.GetCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 Divide calls, got %d", len(calls))
	}

	// Verify first call parameters and returns
	verifyDivideCall(t, calls[0], 50, 5, 10, true)

	// Verify second call parameters and returns
	verifyDivideCall(t, calls[1], 50, 0, 0, false)
}

// TestWrapCalculator_RepeatedCalls demonstrates that multiple calls to the same method are recorded.
// This test verifies that struct wrapping maintains a history of all calls to each method.
func TestWrapCalculator_RepeatedCalls(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(1, 0)
	wrapper := WrapCalculator(t, calc)

	// Make multiple calls to the same method
	wrapper.Add.Start(1, 2).WaitForResponse()
	wrapper.Add.Start(3, 4).WaitForResponse()
	wrapper.Add.Start(5, 6).WaitForResponse()

	// Verify all calls were recorded
	calls := wrapper.Add.GetCalls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 Add calls, got %d", len(calls))
	}

	// Verify call parameters
	expectedParams := [][2]int{
		{1, 2},
		{3, 4},
		{5, 6},
	}

	for i, call := range calls {
		if call.Params.A != expectedParams[i][0] || call.Params.B != expectedParams[i][1] {
			t.Errorf("call %d: expected params (%d, %d), got (%d, %d)",
				i, expectedParams[i][0], expectedParams[i][1],
				call.Params.A, call.Params.B)
		}
	}
}

// TestWrapCalculator_StatePreservation demonstrates that the wrapped struct maintains state.
// This test verifies that wrapping doesn't interfere with the struct's internal state.
func TestWrapCalculator_StatePreservation(t *testing.T) {
	t.Parallel()

	// Create calculator with specific multiplier and offset
	calc := calculator.NewCalculator(5, 100)
	wrapper := WrapCalculator(t, calc)

	// Test that methods use the correct state values
	// Multiply: 10 * multiplier=5 = 50
	mulResult := wrapper.Multiply.Start(10).WaitForResponse()
	if mulResult != 50 {
		t.Errorf("Multiply: expected 50 (10*5), got %d", mulResult)
	}

	// Add: 10 + 20 + offset=100 = 130
	addResult := wrapper.Add.Start(10, 20).WaitForResponse()
	if addResult != 130 {
		t.Errorf("Add: expected 130 (10+20+100), got %d", addResult)
	}
}

// verifyDivideCall checks that a recorded Divide call has the expected parameters and return values.
// Reduces cyclomatic complexity by consolidating repeated verification logic.
func verifyDivideCall(
	t *testing.T,
	call WrapCalculatorWrapperDivideWrapperCallRecord,
	expNum, expDenom, expResult int,
	expOk bool,
) {
	t.Helper()

	if call.Params.Numerator != expNum || call.Params.Denominator != expDenom {
		t.Errorf("expected params (%d, %d), got (%d, %d)",
			expNum, expDenom, call.Params.Numerator, call.Params.Denominator)
	}

	if call.Returns.Result0 != expResult || call.Returns.Result1 != expOk {
		t.Errorf("expected returns (%d, %v), got (%d, %v)",
			expResult, expOk, call.Returns.Result0, call.Returns.Result1)
	}
}
