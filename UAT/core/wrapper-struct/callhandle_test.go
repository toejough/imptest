package calculator_test

import (
	"testing"

	"github.com/toejough/imptest"
	calculator "github.com/toejough/imptest/UAT/core/wrapper-struct"
)

// Generate wrapper for Counter struct
//go:generate impgen calculator.Counter --target

// TestStructWrapper_CallHandleHasExpectMethods verifies Expect* methods exist.
func TestStructWrapper_CallHandleHasExpectMethods(t *testing.T) {
	t.Parallel()

	counter := calculator.NewCounter(10)
	wrapper := WrapCounter(t, counter)

	call := wrapper.Method.AddAmount.Start(5)

	// ExpectReturnsEqual should work
	call.ExpectReturnsEqual(15)
}

// TestStructWrapper_CallHandleHasReturnedField verifies Returned field access.
func TestStructWrapper_CallHandleHasReturnedField(t *testing.T) {
	t.Parallel()

	counter := calculator.NewCounter(0)
	wrapper := WrapCounter(t, counter)

	call := wrapper.Method.Increment.Start()

	// WaitForResponse should be available
	call.WaitForResponse()

	// After waiting, should be able to access Returned field directly
	if call.Returned == nil {
		t.Fatalf("expected Returned to be set after WaitForResponse()")
	}

	result := call.Returned.Result0
	if result != 1 {
		t.Errorf("expected result 1, got %d", result)
	}
}

// TestStructWrapper_ConcurrentCalls verifies multiple goroutines work independently.
func TestStructWrapper_ConcurrentCalls(t *testing.T) {
	t.Parallel()

	// Use separate counters for concurrent calls to avoid state sharing
	counter1 := calculator.NewCounter(0)
	counter2 := calculator.NewCounter(100)
	counter3 := calculator.NewCounter(1000)
	wrapper1 := WrapCounter(t, counter1)
	wrapper2 := WrapCounter(t, counter2)
	wrapper3 := WrapCounter(t, counter3)

	// Start 3 concurrent calls on different counters
	call1 := wrapper1.Method.AddAmount.Start(1)
	call2 := wrapper2.Method.AddAmount.Start(10)
	call3 := wrapper3.Method.AddAmount.Start(100)

	// Each handle should independently verify its own results
	call1.ExpectReturnsEqual(1)
	call2.ExpectReturnsEqual(110)
	call3.ExpectReturnsEqual(1100)
}

// TestStructWrapper_ExpectReturnsMatch verifies matcher support.
func TestStructWrapper_ExpectReturnsMatch(t *testing.T) {
	t.Parallel()

	counter := calculator.NewCounter(5)
	wrapper := WrapCounter(t, counter)

	call := wrapper.Method.AddAmount.Start(10)

	// Should be able to use matchers
	call.ExpectReturnsMatch(imptest.Any())
}

// TestStructWrapper_NoGetCallsMethod verifies GetCalls() doesn't exist.
//
// REQUIREMENT: Struct method wrappers must NOT have GetCalls() method.
func TestStructWrapper_NoGetCallsMethod(t *testing.T) {
	t.Parallel()

	counter := calculator.NewCounter(0)
	wrapper := WrapCounter(t, counter)

	// Type assertion to check if GetCalls exists
	type getCaller interface {
		GetCalls() any
	}

	if _, hasGetCalls := any(wrapper.Method.Increment).(getCaller); hasGetCalls {
		t.Fatalf(
			"wrapper.Method.Increment has GetCalls() method - this should not exist with call handle pattern",
		)
	}
}

// TestStructWrapper_StartReturnsUniqueCallHandles verifies each Start() returns NEW handle.
//
// REQUIREMENT: Struct method wrappers must return unique *CallHandle from each Start() call,
// just like function wrappers and interface wrappers.
func TestStructWrapper_StartReturnsUniqueCallHandles(t *testing.T) {
	t.Parallel()

	counter := calculator.NewCounter(0)
	wrapper := WrapCounter(t, counter)

	// Start first call and wait for it to complete
	call1 := wrapper.Method.Increment.Start()
	call1.WaitForResponse()

	// Now start second call
	call2 := wrapper.Method.Increment.Start()

	// Verify they are different objects
	if call1 == call2 {
		t.Fatalf("Start() returned same handle twice - expected unique handles for each call")
	}

	// Each handle should independently verify its own results
	// call1 executed first (waited), so it gets 1
	// call2 executes second, so it gets 2
	call1.ExpectReturnsEqual(1)
	call2.ExpectReturnsEqual(2)
}

// TestUnifiedPattern_FunctionInterfaceStructSameAPI verifies all wrapper types have identical API.
//
// REQUIREMENT: Function, interface, and struct wrappers must all:
// - Have Start() returning *CallHandle
// - NO GetCalls() method
// - Call handles have Returned, Panicked fields
// - Call handles have ExpectReturnsEqual, ExpectReturnsMatch, ExpectPanicEquals, ExpectPanicMatches, WaitForResponse
//
// This test uses struct wrappers to verify they match the API pattern.
func TestUnifiedPattern_FunctionInterfaceStructSameAPI(t *testing.T) {
	t.Parallel()

	// This test documents that the API is the SAME across all wrapper types
	// We use Counter and Calculator to verify struct wrapper API

	// Struct wrapper (Counter)
	counter := calculator.NewCounter(3)
	structWrapper := WrapCounter(t, counter)
	structCall := structWrapper.Method.GetValue.Start()

	// All should have identical API
	// 1. Return call handles with ExpectReturnsEqual
	structCall.ExpectReturnsEqual(3)

	// 2. Have WaitForResponse
	structCall2 := structWrapper.Method.Increment.Start()
	structCall2.WaitForResponse()

	// 3. Have Returned field access
	if structCall2.Returned == nil {
		t.Fatalf("expected call handle to have Returned field")
	}

	// 4. Support ExpectReturnsMatch
	structCall3 := structWrapper.Method.AddAmount.Start(5)
	structCall3.ExpectReturnsMatch(imptest.Any())

	// Success: Struct wrappers have identical API to function and interface wrappers
}
