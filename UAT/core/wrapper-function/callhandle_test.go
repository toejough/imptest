package callable_test

import (
	"testing"
	"time"

	"github.com/toejough/imptest"
	"github.com/toejough/imptest/match"
	callable "github.com/toejough/imptest/UAT/core/wrapper-function"
)

//go:generate impgen callable.PanicWithMessage --target
//go:generate impgen callable.SlowAddFunc --target
//go:generate impgen callable.SlowFuncFunc --target
//go:generate impgen callable.ProcessFunc --target
//go:generate impgen callable.MultiplyFunc --target
//go:generate impgen callable.SlowMultiplyFunc --target
//go:generate impgen callable.DivideFunc --target
//go:generate impgen callable.PanicIntFunc --target
//go:generate impgen callable.ComputeFunc --target
//go:generate impgen callable.SideEffectFunc --target
//go:generate impgen callable.ConditionalFunc --target
//go:generate impgen callable.PanicFunc --target
//go:generate impgen callable.AddFunc --target

// TestCallHandle_ConcurrentCalls verifies multiple goroutines can be managed independently.
//
// REQUIREMENT: Multiple concurrent calls work independently with separate handles.
func TestCallHandle_ConcurrentCalls(t *testing.T) {
	t.Parallel()

	// Function that sleeps to ensure concurrent execution
	slowAdd := func(a, b int, delay time.Duration) int {
		time.Sleep(delay)
		return a + b
	}

	// Start 3 concurrent calls with different delays
	// call3 finishes first (10ms), call2 second (20ms), call1 last (30ms)
	call1 := StartSlowAddFunc(t, slowAdd, 1, 2, 30*time.Millisecond)
	call2 := StartSlowAddFunc(t, slowAdd, 10, 20, 20*time.Millisecond)
	call3 := StartSlowAddFunc(t, slowAdd, 100, 200, 10*time.Millisecond)

	// Despite finish order (3, 2, 1), each handle should track its own result
	call1.ReturnsEqual(3)
	call2.ReturnsEqual(30)
	call3.ReturnsEqual(300)
}

// TestCallHandle_EventuallyExpectPanic verifies async Eventually with panic expectations.
func TestCallHandle_EventuallyExpectPanic(t *testing.T) {
	t.Parallel()

	call := StartPanicWithMessage(t, callable.PanicWithMessage, "expected panic")

	// Register panic expectation (NON-BLOCKING)
	call.Eventually.PanicEquals("expected panic")

	// Wait for expectation to be satisfied
	imptest.Wait(t)
}

// TestCallHandle_EventuallyExpectReturns verifies async Eventually() pattern on target wrappers.
//
// REQUIREMENT: Call handles support Eventually() for non-blocking expectation registration.
// This enables registering multiple expectations before any call completes,
// then using Controller.Wait() to block until all are satisfied.
func TestCallHandle_EventuallyExpectReturns(t *testing.T) {
	t.Parallel()

	// Function with delay to ensure calls don't complete immediately
	slowAdd := func(a, b int, delay time.Duration) int {
		time.Sleep(delay)
		return a + b
	}

	// Start multiple calls
	call1 := StartSlowAddFunc(t, slowAdd, 1, 2, 30*time.Millisecond)
	call2 := StartSlowAddFunc(t, slowAdd, 10, 20, 20*time.Millisecond)
	call3 := StartSlowAddFunc(t, slowAdd, 100, 200, 10*time.Millisecond)

	// Register expectations (NON-BLOCKING) - this is the key difference from current API
	// With regular ReturnsEqual, we'd block on call1 before moving to call2
	call1.Eventually.ReturnsEqual(3)
	call2.Eventually.ReturnsEqual(30)
	call3.Eventually.ReturnsEqual(300)

	// Wait for all expectations to be satisfied
	imptest.Wait(t)
}

// TestCallHandle_ExpectCallsWaitForResponse verifies Expect* methods internally call WaitForResponse.
//
// REQUIREMENT: WaitForResponse() is called internally by Expect methods.
// User doesn't need to call it manually when using Expect* methods.
func TestCallHandle_ExpectCallsWaitForResponse(t *testing.T) {
	t.Parallel()

	// Function with artificial delay
	slowFunc := func() int {
		time.Sleep(50 * time.Millisecond)
		return 42
	}

	call := StartSlowFuncFunc(t, slowFunc)

	// ReturnsEqual should wait internally - no need to call WaitForResponse first
	// If this fails (times out or gets wrong value), the Expect method didn't wait properly
	call.ReturnsEqual(42)
}

// TestCallHandle_ExpectReturnMatch verifies ExpectReturnMatch method works.
func TestCallHandle_ExpectReturnMatch(t *testing.T) {
	t.Parallel()

	process := func(x int) (string, error) {
		if x < 0 {
			return "", nil // simplified - just return empty string
		}

		return "processed", nil
	}

	call := StartProcessFunc(t, process, 10)

	// Should be able to use matchers (using BeAny matcher)
	call.ReturnsShould(
		match.BeAny,
		match.BeAny,
	)
}

// TestCallHandle_HasExpectMethods verifies call handles have proper Expect* methods.
//
// REQUIREMENT: Call handles must have these methods:
// - ReturnsEqual(...)
// - ReturnsShould(...)
// - PanicEquals(...)
// - PanicShould(...)
func TestCallHandle_HasExpectMethods(t *testing.T) {
	t.Parallel()

	multiply := func(a, b int) int { return a * b }

	call := StartMultiplyFunc(t, multiply, 5, 7)

	// ReturnsEqual should work
	call.ReturnsEqual(35)
}

// TestCallHandle_InterleavedStarts verifies starting calls in sequence doesn't break independence.
func TestCallHandle_InterleavedStarts(t *testing.T) {
	t.Parallel()

	// Function that takes time
	slowMultiply := func(a int, delay time.Duration) int {
		time.Sleep(delay)
		return a * 2
	}

	// Start call1, then call2, but call2 finishes first
	call1 := StartSlowMultiplyFunc(t, slowMultiply, 10, 100*time.Millisecond)
	call2 := StartSlowMultiplyFunc(t, slowMultiply, 20, 10*time.Millisecond)

	// Verify in order started (not finish order)
	call1.ReturnsEqual(20)
	call2.ReturnsEqual(40)
}

// TestCallHandle_ManualFieldAccess verifies manual access to Returned/Panic fields.
//
// REQUIREMENT: Call handles must have these fields:
// - Returned (struct with Result0, Result1, etc.)
// - Panic (any)
// And WaitForResponse() must be callable to wait before accessing fields.
func TestCallHandle_ManualFieldAccess(t *testing.T) {
	t.Parallel()

	divide := func(a, b int) (int, bool) {
		if b == 0 {
			return 0, false
		}

		return a / b, true
	}

	call := StartDivideFunc(t, divide, 10, 2)

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
		t.Errorf("expected ok=true, got false")
	}
}

// TestCallHandle_ManualPanicFieldAccess verifies accessing Panic field after panic.
func TestCallHandle_ManualPanicFieldAccess(t *testing.T) {
	t.Parallel()

	panicFunc := func() int { panic("error") }

	call := StartPanicIntFunc(t, panicFunc)
	call.WaitForResponse()

	// After panic, Panicked field should be set
	if call.Panicked == nil {
		t.Fatalf("expected Panicked to be set after panic")
	}

	panicValue, ok := call.Panicked.(string)
	if !ok {
		t.Fatalf("expected panic value to be string, got %T", call.Panicked)
	}

	if panicValue != "error" {
		t.Errorf("expected panic value 'error', got %q", panicValue)
	}
}

// TestCallHandle_MultipleReturns verifies handles work with multiple return values.
func TestCallHandle_MultipleReturns(t *testing.T) {
	t.Parallel()

	compute := func(x int) (int, string, bool) {
		return x * 2, "computed", true
	}

	call := StartComputeFunc(t, compute, 21)
	call.ReturnsEqual(42, "computed", true)
}

// TestCallHandle_NoReturns verifies handles work with functions that have no returns.
func TestCallHandle_NoReturns(t *testing.T) {
	t.Parallel()

	// Function with no return value - just runs and completes
	sideEffect := func(x int) {
		_ = x // do something
	}

	call := StartSideEffectFunc(t, sideEffect, 100)

	// Should have WaitForResponse even if no return values
	call.WaitForResponse()

	// Calling WaitForResponse again should be safe (already complete)
	call.WaitForResponse()
}

// TestCallHandle_PanicAndReturnDifferentCalls verifies panic and return are per-call.
func TestCallHandle_PanicAndReturnDifferentCalls(t *testing.T) {
	t.Parallel()

	// Function that panics on negative, returns on positive
	conditional := func(x int) int {
		if x < 0 {
			panic("negative value")
		}

		return x * 10
	}

	callPanic := StartConditionalFunc(t, conditional, -5)
	callReturn := StartConditionalFunc(t, conditional, 5)

	// One should panic, one should return - independent verification
	callPanic.PanicEquals("negative value")
	callReturn.ReturnsEqual(50)
}

// TestCallHandle_PanicCapture verifies panic handling with call handles.
//
// REQUIREMENT: Call handles must support PanicEquals() and PanicShould().
func TestCallHandle_PanicCapture(t *testing.T) {
	t.Parallel()

	panicFunc := func() { panic("boom") }

	call := StartPanicFunc(t, panicFunc)
	call.PanicEquals("boom")
}

// TestCallHandle_PanicMatches verifies panic matching with matchers.
func TestCallHandle_PanicMatches(t *testing.T) {
	t.Parallel()

	call := StartPanicWithMessage(t, callable.PanicWithMessage, "critical error")
	// Use BeAny matcher to accept any panic value
	call.PanicShould(match.BeAny)
}

// TestCallHandle_UniqueHandles verifies that each Start() call returns a unique call handle.
//
// REQUIREMENT: Each Start() call must return a NEW unique call handle, not the wrapper itself.
// This enables managing multiple concurrent goroutines independently.
func TestCallHandle_UniqueHandles(t *testing.T) {
	t.Parallel()

	// Simple addition function for testing
	add := func(a, b int) int { return a + b }

	// Start two calls - each should return a different handle
	call1 := StartAddFunc(t, add, 10, 20)
	call2 := StartAddFunc(t, add, 30, 40)

	// Verify they are different objects (not same pointer)
	if call1 == call2 {
		t.Fatalf("Start() returned same handle twice - expected unique handles for each call")
	}

	// Each handle should independently verify its own results
	call1.ReturnsEqual(30)
	call2.ReturnsEqual(70)
}
