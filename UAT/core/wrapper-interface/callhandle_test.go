package handlers_test

import (
	"testing"

	"github.com/toejough/imptest"
	handlers "github.com/toejough/imptest/UAT/core/wrapper-interface"
)

// TestInterfaceWrapper_CallHandleHasExpectMethods verifies Expect* methods exist.
//
// REQUIREMENT: Call handles from interface method wrappers must have:
// - ExpectReturn(...)
// - ExpectReturnMatch(...)
// - ExpectPanic(...)
// - ExpectPanicMatch(...)
func TestInterfaceWrapper_CallHandleHasExpectMethods(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(2)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.Add.Start(5, 7)

	// ExpectReturn should work
	call.ExpectReturn(14) // 2 + 5 + 7
}

// TestInterfaceWrapper_CallHandleHasReturnedField verifies Returned field access.
//
// REQUIREMENT: Call handles must have Returned field (struct with Result0, Result1, etc.)
// and WaitForResponse() method for manual waiting.
func TestInterfaceWrapper_CallHandleHasReturnedField(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(1)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.Divide.Start(10, 2)

	// WaitForResponse should be available for manual waiting
	call.WaitForResponse()

	// After waiting, should be able to access Returned field directly
	if call.Returned == nil {
		t.Fatalf("expected Returned to be set after WaitForResponse()")
	}

	result := call.Returned.Result0
	success := call.Returned.Result1

	if result != 5 {
		t.Errorf("expected result 5, got %d", result)
	}

	if !success {
		t.Errorf("expected success=true, got false")
	}
}

// TestInterfaceWrapper_ConcurrentCalls verifies multiple goroutines work independently.
//
// REQUIREMENT: Multiple concurrent calls work independently with separate handles.
func TestInterfaceWrapper_ConcurrentCalls(t *testing.T) {
	t.Parallel()

	// Use Calculator for concurrent calls test
	calc := handlers.NewCalculatorImpl(0)
	wrapper := WrapCalculator(t, calc)

	// Start 3 concurrent calls
	call1 := wrapper.Add.Start(1, 1)
	call2 := wrapper.Add.Start(10, 10)
	call3 := wrapper.Add.Start(100, 100)

	// Each handle should independently verify its own results
	call1.ExpectReturn(2)
	call2.ExpectReturn(20)
	call3.ExpectReturn(200)
}

// TestInterfaceWrapper_ExpectReturnMatch verifies matcher support.
func TestInterfaceWrapper_ExpectReturnMatch(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(2)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.Divide.Start(10, 2)

	// Should be able to use matchers
	call.ExpectReturnMatch(
		imptest.Any,
		imptest.Any,
	)
}

// TestInterfaceWrapper_MultipleReturns verifies handles work with multiple return values.
func TestInterfaceWrapper_MultipleReturns(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(1)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.Divide.Start(20, 4)
	call.ExpectReturn(5, true)
}

// TestInterfaceWrapper_MultiplyMethod verifies Multiply method coverage.
//
// REQUIREMENT: All Calculator methods should be tested for coverage.
func TestInterfaceWrapper_MultiplyMethod(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(3)
	wrapper := WrapCalculator(t, calc)

	// Call Multiply directly to ensure coverage
	wrapper.Multiply.Start(7).ExpectReturn(21)
}

// TestInterfaceWrapper_NoGetCallsMethod verifies GetCalls() doesn't exist.
//
// REQUIREMENT: Interface method wrappers must NOT have GetCalls() method.
// Call handle pattern eliminates need for call history tracking.
// This test verifies we removed legacy call tracking.
func TestInterfaceWrapper_NoGetCallsMethod(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(10)
	wrapper := WrapCalculator(t, calc)

	// Type assertion to check if GetCalls exists
	type getCaller interface {
		GetCalls() any
	}

	if _, hasGetCalls := any(wrapper.Add).(getCaller); hasGetCalls {
		t.Fatalf(
			"wrapper.Add has GetCalls() method - this should not exist with call handle pattern",
		)
	}
}

// TestInterfaceWrapper_PanicCapture verifies panic handling with call handles.
//
// REQUIREMENT: Call handles must support ExpectPanic() and ExpectPanicMatch().
func TestInterfaceWrapper_PanicCapture(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(5)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.ProcessValue.Start(-1)
	call.ExpectPanic("negative values not supported")
}

// TestInterfaceWrapper_PanicFieldAccess verifies manual Panicked field access.
//
// REQUIREMENT: Call handles must have Panicked field accessible after WaitForResponse().
func TestInterfaceWrapper_PanicFieldAccess(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(5)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.ProcessValue.Start(-1)
	call.WaitForResponse()

	// After panic, Panicked field should be set
	if call.Panicked == nil {
		t.Fatalf("expected Panicked to be set after panic")
	}

	panicValue, ok := call.Panicked.(string)
	if !ok {
		t.Fatalf("expected panic value to be string, got %T", call.Panicked)
	}

	if panicValue != "negative values not supported" {
		t.Errorf("expected panic value 'negative values not supported', got %q", panicValue)
	}
}

// Generate wrappers for interface types
//go:generate impgen handlers.Calculator --target

// TestInterfaceWrapper_StartReturnsUniqueCallHandles verifies each Start() returns NEW handle.
//
// REQUIREMENT: Interface method wrappers must return unique *CallHandle from each Start() call,
// just like function wrappers do. This enables managing multiple concurrent goroutines independently.
func TestInterfaceWrapper_StartReturnsUniqueCallHandles(t *testing.T) {
	t.Parallel()

	calc := handlers.NewCalculatorImpl(10)
	wrapper := WrapCalculator(t, calc)

	// Start two calls to the same method - each should return different handle
	call1 := wrapper.Add.Start(5, 3)
	call2 := wrapper.Add.Start(10, 20)

	// Verify they are different objects (not same pointer)
	if call1 == call2 {
		t.Fatalf("Start() returned same handle twice - expected unique handles for each call")
	}

	// Each handle should independently verify its own results
	call1.ExpectReturn(18) // 10 + 5 + 3
	call2.ExpectReturn(40) // 10 + 10 + 20
}
