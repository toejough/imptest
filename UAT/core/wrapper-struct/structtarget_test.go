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
// - `StartCalculator(t, calc)` returns a wrapper with method interceptors
// - Each method on the struct gets its own interceptor (Add, Multiply, Divide, Process)
// - Calls are intercepted and recorded while still delegating to the original struct
package calculator_test

import (
	"testing"

	calculator "github.com/toejough/imptest/UAT/core/wrapper-struct"
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
	wrapper := StartCalculator(t, calc)

	// Call Add through the wrapped struct
	// Expected: wrapper.Add should exist and intercept the call
	wrapper.Add.Start(5, 3).ExpectReturn(18)
}

// TestWrapCalculator_ErrorHandling demonstrates wrapping methods that return errors.
// This test verifies that struct wrapping handles error return values correctly.
func TestWrapCalculator_ErrorHandling(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(2, 0)
	wrapper := StartCalculator(t, calc)

	// Test error path with negative input - Process returns error
	call1 := wrapper.Process.Start(-5)
	call1.WaitForResponse()

	result1 := call1.Returned.Result0

	err1 := call1.Returned.Result1
	if err1 == nil {
		t.Fatal("expected error from Process with negative input")
	}

	if result1 != "" {
		t.Errorf("expected empty result string, got %q", result1)
	}

	// Test success path - Process should: Multiply(10) = 20, Add(20, 5) = 25 (no offset)
	wrapper.Process.Start(10).ExpectReturn("Result: 25", nil)
}

// TestWrapCalculator_MethodInteraction demonstrates wrapping a method that calls other methods.
// This test verifies that struct wrapping intercepts both the high-level method AND
// the methods it calls internally (if called through the wrapper).
func TestWrapCalculator_MethodInteraction(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(3, 10)
	wrapper := StartCalculator(t, calc)

	// Call Process, which internally calls Multiply and Add
	// Process(5): Multiply(5) = 15, Add(15, 5) = 30
	wrapper.Process.Start(5).ExpectReturn("Result: 30", nil)

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
	wrapper := StartCalculator(t, calc)

	// Call multiple different methods through the wrapped struct
	// All should be intercepted independently

	// Test Add: 10 + 20 + offset=5 = 35
	wrapper.Add.Start(10, 20).ExpectReturn(35)

	// Test Multiply: 7 * multiplier=3 = 21
	wrapper.Multiply.Start(7).ExpectReturn(21)

	// Test Divide: 100 / 4 = 25, true
	wrapper.Divide.Start(100, 4).ExpectReturn(25, true)
}

// TestWrapCalculator_MultipleReturnValues demonstrates wrapping methods with multiple return values.
// This test verifies that struct wrapping handles methods with multiple returns (like Divide).
func TestWrapCalculator_MultipleReturnValues(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(1, 0)
	wrapper := StartCalculator(t, calc)

	// Test successful division
	wrapper.Divide.Start(50, 5).ExpectReturn(10, true)

	// Test division by zero
	wrapper.Divide.Start(50, 0).ExpectReturn(0, false)
}

// TestWrapCalculator_RepeatedCalls demonstrates that multiple calls to the same method are recorded.
// This test verifies that struct wrapping maintains a history of all calls to each method.
func TestWrapCalculator_RepeatedCalls(t *testing.T) {
	t.Parallel()

	calc := calculator.NewCalculator(1, 0)
	wrapper := StartCalculator(t, calc)

	// Make multiple calls to the same method - each returns unique handle
	wrapper.Add.Start(1, 2).ExpectReturn(3)  // 1 + 2 + 0 = 3
	wrapper.Add.Start(3, 4).ExpectReturn(7)  // 3 + 4 + 0 = 7
	wrapper.Add.Start(5, 6).ExpectReturn(11) // 5 + 6 + 0 = 11
}

// TestWrapCalculator_StatePreservation demonstrates that the wrapped struct maintains state.
// This test verifies that wrapping doesn't interfere with the struct's internal state.
func TestWrapCalculator_StatePreservation(t *testing.T) {
	t.Parallel()

	// Create calculator with specific multiplier and offset
	calc := calculator.NewCalculator(5, 100)
	wrapper := StartCalculator(t, calc)

	// Test that methods use the correct state values
	// Multiply: 10 * multiplier=5 = 50
	wrapper.Multiply.Start(10).ExpectReturn(50)

	// Add: 10 + 20 + offset=100 = 130
	wrapper.Add.Start(10, 20).ExpectReturn(130)
}
