package callhandle_test

import (
	"testing"
	"time"

	callhandle "github.com/toejough/imptest/UAT/42-call-handle-pattern"
	"github.com/toejough/imptest/imptest"
)

// TestInterfaceWrapper_CallHandleHasExpectMethods verifies Expect* methods exist.
//
// REQUIREMENT: Call handles from interface method wrappers must have:
// - ExpectReturnsEqual(...)
// - ExpectReturnsMatch(...)
// - ExpectPanicEquals(...)
// - ExpectPanicMatches(...)
func TestInterfaceWrapper_CallHandleHasExpectMethods(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(2)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.Add.Start(5, 7)

	// ExpectReturnsEqual should work
	call.ExpectReturnsEqual(14) // 2 + 5 + 7
}

// TestInterfaceWrapper_CallHandleHasReturnedField verifies Returned field access.
//
// REQUIREMENT: Call handles must have Returned field (struct with Result0, Result1, etc.)
// and WaitForResponse() method for manual waiting.
func TestInterfaceWrapper_CallHandleHasReturnedField(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(1)
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

	svc := callhandle.NewSlowService(10 * time.Millisecond)
	wrapper := WrapSlowService(t, svc)

	// Start 3 concurrent calls
	call1 := wrapper.Process.Start(1)
	call2 := wrapper.Process.Start(10)
	call3 := wrapper.Process.Start(100)

	// Each handle should independently verify its own results
	call1.ExpectReturnsEqual(2)
	call2.ExpectReturnsEqual(20)
	call3.ExpectReturnsEqual(200)
}

// TestInterfaceWrapper_ExecutesInGoroutine verifies async execution.
//
// REQUIREMENT: Interface method wrappers must execute in goroutine (async),
// just like function wrappers. Start() returns immediately, caller uses
// WaitForResponse() or Expect* methods to get results.
func TestInterfaceWrapper_ExecutesInGoroutine(t *testing.T) {
	t.Parallel()

	svc := callhandle.NewSlowService(50 * time.Millisecond)
	wrapper := WrapSlowService(t, svc)

	// Start should return immediately (not block for 50ms)
	start := time.Now()
	call := wrapper.Process.Start(100)
	elapsed := time.Since(start)

	// If Start() blocked, this would take ~50ms
	// Since it runs in goroutine, should be nearly instant (<10ms)
	if elapsed > 10*time.Millisecond {
		t.Fatalf("Start() took %v - expected immediate return (goroutine execution)", elapsed)
	}

	// Now wait for actual result
	call.ExpectReturnsEqual(200)
}

// TestInterfaceWrapper_ExpectReturnsMatch verifies matcher support.
func TestInterfaceWrapper_ExpectReturnsMatch(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(2)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.Divide.Start(10, 2)

	// Should be able to use matchers
	call.ExpectReturnsMatch(
		imptest.Any(),
		imptest.Any(),
	)
}

// TestInterfaceWrapper_MultipleReturns verifies handles work with multiple return values.
func TestInterfaceWrapper_MultipleReturns(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(1)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.Divide.Start(20, 4)
	call.ExpectReturnsEqual(5, true)
}

// TestInterfaceWrapper_MultiplyMethod verifies Multiply method coverage.
//
// REQUIREMENT: All Calculator methods should be tested for coverage.
func TestInterfaceWrapper_MultiplyMethod(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(3)
	wrapper := WrapCalculator(t, calc)

	// Call Multiply directly to ensure coverage
	wrapper.Multiply.Start(7).ExpectReturnsEqual(21)
}

// TestInterfaceWrapper_NoGetCallsMethod verifies GetCalls() doesn't exist.
//
// REQUIREMENT: Interface method wrappers must NOT have GetCalls() method.
// Call handle pattern eliminates need for call history tracking.
// This test verifies we removed legacy call tracking.
func TestInterfaceWrapper_NoGetCallsMethod(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(10)
	wrapper := WrapCalculator(t, calc)

	// Compile-time check: wrapper.Add should not have GetCalls() method
	// If this compiles, the test fails (GetCalls exists when it shouldn't)
	// This will fail to compile when GetCalls is removed (which is correct)

	// Type assertion to check if GetCalls exists
	type getCaller interface {
		GetCalls() any
	}

	if _, hasGetCalls := any(wrapper.Add).(getCaller); hasGetCalls {
		t.Fatalf("wrapper.Add has GetCalls() method - this should not exist with call handle pattern")
	}
}

// TestInterfaceWrapper_PanicCapture verifies panic handling with call handles.
//
// REQUIREMENT: Call handles must support ExpectPanicEquals() and ExpectPanicMatches().
func TestInterfaceWrapper_PanicCapture(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(5)
	wrapper := WrapCalculator(t, calc)

	call := wrapper.ProcessValue.Start(-1)
	call.ExpectPanicEquals("negative values not supported")
}

// TestInterfaceWrapper_PanicFieldAccess verifies manual Panicked field access.
//
// REQUIREMENT: Call handles must have Panicked field accessible after WaitForResponse().
func TestInterfaceWrapper_PanicFieldAccess(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(5)
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

// Generate wrappers for interface and struct types
//go:generate impgen callhandle.Calculator --target
//go:generate impgen callhandle.Counter --target
//go:generate impgen callhandle.SlowService --target

// ================================================================================
// INTERFACE WRAPPER TESTS - Call Handle Pattern
// ================================================================================

// TestInterfaceWrapper_StartReturnsUniqueCallHandles verifies each Start() returns NEW handle.
//
// REQUIREMENT: Interface method wrappers must return unique *CallHandle from each Start() call,
// just like function wrappers do. This enables managing multiple concurrent goroutines independently.
func TestInterfaceWrapper_StartReturnsUniqueCallHandles(t *testing.T) {
	t.Parallel()

	calc := callhandle.NewCalculatorImpl(10)
	wrapper := WrapCalculator(t, calc)

	// Start two calls to the same method - each should return different handle
	call1 := wrapper.Add.Start(5, 3)
	call2 := wrapper.Add.Start(10, 20)

	// Verify they are different objects (not same pointer)
	if call1 == call2 {
		t.Fatalf("Start() returned same handle twice - expected unique handles for each call")
	}

	// Each handle should independently verify its own results
	call1.ExpectReturnsEqual(18) // 10 + 5 + 3
	call2.ExpectReturnsEqual(40) // 10 + 10 + 20
}

// TestStructWrapper_CallHandleHasExpectMethods verifies Expect* methods exist.
func TestStructWrapper_CallHandleHasExpectMethods(t *testing.T) {
	t.Parallel()

	counter := callhandle.NewCounter(10)
	wrapper := WrapCounter(t, counter)

	call := wrapper.AddAmount.Start(5)

	// ExpectReturnsEqual should work
	call.ExpectReturnsEqual(15)
}

// TestStructWrapper_CallHandleHasReturnedField verifies Returned field access.
func TestStructWrapper_CallHandleHasReturnedField(t *testing.T) {
	t.Parallel()

	counter := callhandle.NewCounter(0)
	wrapper := WrapCounter(t, counter)

	call := wrapper.Increment.Start()

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

	svc := callhandle.NewSlowService(10 * time.Millisecond)
	wrapper := WrapSlowService(t, svc)

	// Start 3 concurrent calls
	call1 := wrapper.Process.Start(1)
	call2 := wrapper.Process.Start(10)
	call3 := wrapper.Process.Start(100)

	// Each handle should independently verify its own results
	call1.ExpectReturnsEqual(2)
	call2.ExpectReturnsEqual(20)
	call3.ExpectReturnsEqual(200)
}

// TestStructWrapper_ExecutesInGoroutine verifies async execution.
//
// REQUIREMENT: Struct method wrappers must execute in goroutine (async).
func TestStructWrapper_ExecutesInGoroutine(t *testing.T) {
	t.Parallel()

	svc := callhandle.NewSlowService(50 * time.Millisecond)
	wrapper := WrapSlowService(t, svc)

	// Start should return immediately
	start := time.Now()
	call := wrapper.Process.Start(100)
	elapsed := time.Since(start)

	// Should be nearly instant (<10ms), not 50ms
	if elapsed > 10*time.Millisecond {
		t.Fatalf("Start() took %v - expected immediate return (goroutine execution)", elapsed)
	}

	// Now wait for actual result
	call.ExpectReturnsEqual(200)
}

// TestStructWrapper_ExpectReturnsMatch verifies matcher support.
func TestStructWrapper_ExpectReturnsMatch(t *testing.T) {
	t.Parallel()

	counter := callhandle.NewCounter(5)
	wrapper := WrapCounter(t, counter)

	call := wrapper.AddAmount.Start(10)

	// Should be able to use matchers
	call.ExpectReturnsMatch(imptest.Any())
}

// TestStructWrapper_NoGetCallsMethod verifies GetCalls() doesn't exist.
//
// REQUIREMENT: Struct method wrappers must NOT have GetCalls() method.
func TestStructWrapper_NoGetCallsMethod(t *testing.T) {
	t.Parallel()

	counter := callhandle.NewCounter(0)
	wrapper := WrapCounter(t, counter)

	// Type assertion to check if GetCalls exists
	type getCaller interface {
		GetCalls() any
	}

	if _, hasGetCalls := any(wrapper.Increment).(getCaller); hasGetCalls {
		t.Fatalf("wrapper.Increment has GetCalls() method - this should not exist with call handle pattern")
	}
}

// ================================================================================
// STRUCT WRAPPER TESTS - Call Handle Pattern
// ================================================================================

// TestStructWrapper_StartReturnsUniqueCallHandles verifies each Start() returns NEW handle.
//
// REQUIREMENT: Struct method wrappers must return unique *CallHandle from each Start() call,
// just like function wrappers and interface wrappers.
func TestStructWrapper_StartReturnsUniqueCallHandles(t *testing.T) {
	t.Parallel()

	counter := callhandle.NewCounter(0)
	wrapper := WrapCounter(t, counter)

	// Start first call and wait for it to complete
	call1 := wrapper.Increment.Start()
	call1.WaitForResponse()

	// Now start second call
	call2 := wrapper.Increment.Start()

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

// ================================================================================
// CROSS-TYPE CONSISTENCY TESTS
// ================================================================================

// TestUnifiedPattern_FunctionInterfaceStructSameAPI verifies all wrapper types have identical API.
//
// REQUIREMENT: Function, interface, and struct wrappers must all:
// - Have Start() returning *CallHandle
// - NO GetCalls() method
// - Call handles have Returned, Panicked fields
// - Call handles have ExpectReturnsEqual, ExpectReturnsMatch, ExpectPanicEquals, ExpectPanicMatches, WaitForResponse
// - Execute in goroutines (async)
func TestUnifiedPattern_FunctionInterfaceStructSameAPI(t *testing.T) {
	t.Parallel()

	// This test documents that the API is the SAME across all wrapper types
	// Individual tests above verify each aspect - this is the integration test

	// Function wrapper (from callhandle_test.go)
	add := func(a, b int) int { return a + b }
	funcWrapper := WrapAdd(t, add)
	funcCall := funcWrapper.Start(1, 2)

	// Interface wrapper
	calc := callhandle.NewCalculatorImpl(0)
	ifaceWrapper := WrapCalculator(t, calc)
	ifaceCall := ifaceWrapper.Add.Start(1, 2)

	// Struct wrapper
	counter := callhandle.NewCounter(3)
	structWrapper := WrapCounter(t, counter)
	structCall := structWrapper.GetValue.Start()

	// All three should have identical API
	// 1. All return call handles with ExpectReturnsEqual
	funcCall.ExpectReturnsEqual(3)
	ifaceCall.ExpectReturnsEqual(3)
	structCall.ExpectReturnsEqual(3)

	// 2. All have WaitForResponse
	funcCall2 := funcWrapper.Start(5, 6)
	ifaceCall2 := ifaceWrapper.Add.Start(5, 6)
	structCall2 := structWrapper.Increment.Start()

	funcCall2.WaitForResponse()
	ifaceCall2.WaitForResponse()
	structCall2.WaitForResponse()

	// 3. All have Returned field access
	if funcCall2.Returned == nil || ifaceCall2.Returned == nil || structCall2.Returned == nil {
		t.Fatalf("expected all call handles to have Returned field")
	}

	// 4. All support ExpectReturnsMatch
	funcCall3 := funcWrapper.Start(10, 20)
	ifaceCall3 := ifaceWrapper.Add.Start(10, 20)
	structCall3 := structWrapper.AddAmount.Start(5)

	funcCall3.ExpectReturnsMatch(imptest.Any())
	ifaceCall3.ExpectReturnsMatch(imptest.Any())
	structCall3.ExpectReturnsMatch(imptest.Any())

	// Success: All three wrapper types have identical API
}
