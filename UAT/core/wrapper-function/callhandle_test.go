package callable_test

import (
	"testing"
	"time"

	"github.com/toejough/imptest"
	callable "github.com/toejough/imptest/UAT/core/wrapper-function"
)

//go:generate impgen callable.PanicWithMessage --target

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
	wrapper := WrapSlowAdd(t, slowAdd)

	// Start 3 concurrent calls with different delays
	// call3 finishes first (10ms), call2 second (20ms), call1 last (30ms)
	call1 := wrapper.Method.Start(1, 2, 30*time.Millisecond)
	call2 := wrapper.Method.Start(10, 20, 20*time.Millisecond)
	call3 := wrapper.Method.Start(100, 200, 10*time.Millisecond)

	// Despite finish order (3, 2, 1), each handle should track its own result
	call1.ExpectReturnsEqual(3)
	call2.ExpectReturnsEqual(30)
	call3.ExpectReturnsEqual(300)
}

// TestCallHandle_EventuallyExpectPanic verifies async Eventually with panic expectations.
func TestCallHandle_EventuallyExpectPanic(t *testing.T) {
	t.Parallel()

	wrapper := WrapPanicWithMessage(t, callable.PanicWithMessage)

	call := wrapper.Method.Start("expected panic")

	// Register panic expectation (NON-BLOCKING)
	call.Eventually.ExpectPanicEquals("expected panic")

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
	wrapper := WrapSlowAdd(t, slowAdd)

	// Start multiple calls
	call1 := wrapper.Method.Start(1, 2, 30*time.Millisecond)
	call2 := wrapper.Method.Start(10, 20, 20*time.Millisecond)
	call3 := wrapper.Method.Start(100, 200, 10*time.Millisecond)

	// Register expectations (NON-BLOCKING) - this is the key difference from current API
	// With regular ExpectReturnsEqual, we'd block on call1 before moving to call2
	call1.Eventually.ExpectReturnsEqual(3)
	call2.Eventually.ExpectReturnsEqual(30)
	call3.Eventually.ExpectReturnsEqual(300)

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
	wrapper := WrapSlowFunc(t, slowFunc)

	call := wrapper.Method.Start()

	// ExpectReturnsEqual should wait internally - no need to call WaitForResponse first
	// If this fails (times out or gets wrong value), the Expect method didn't wait properly
	call.ExpectReturnsEqual(42)
}

// TestCallHandle_ExpectReturnsMatch verifies ExpectReturnsMatch method works.
func TestCallHandle_ExpectReturnsMatch(t *testing.T) {
	t.Parallel()

	process := func(x int) (string, error) {
		if x < 0 {
			return "", nil // simplified - just return empty string
		}

		return "processed", nil
	}
	wrapper := WrapProcess(t, process)

	call := wrapper.Method.Start(10)

	// Should be able to use matchers (using Any() matcher)
	call.ExpectReturnsMatch(
		imptest.Any,
		imptest.Any,
	)
}

// TestCallHandle_HasExpectMethods verifies call handles have proper Expect* methods.
//
// REQUIREMENT: Call handles must have these methods:
// - ExpectReturnsEqual(...)
// - ExpectReturnsMatch(...)
// - ExpectPanicEquals(...)
// - ExpectPanicMatches(...)
func TestCallHandle_HasExpectMethods(t *testing.T) {
	t.Parallel()

	multiply := func(a, b int) int { return a * b }
	wrapper := WrapMultiply(t, multiply)

	call := wrapper.Method.Start(5, 7)

	// ExpectReturnsEqual should work
	call.ExpectReturnsEqual(35)
}

// TestCallHandle_InterleavedStarts verifies starting calls in sequence doesn't break independence.
func TestCallHandle_InterleavedStarts(t *testing.T) {
	t.Parallel()

	// Function that takes time
	slowMultiply := func(a int, delay time.Duration) int {
		time.Sleep(delay)
		return a * 2
	}
	wrapper := WrapSlowMultiply(t, slowMultiply)

	// Start call1, then call2, but call2 finishes first
	call1 := wrapper.Method.Start(10, 100*time.Millisecond)
	call2 := wrapper.Method.Start(20, 10*time.Millisecond)

	// Verify in order started (not finish order)
	call1.ExpectReturnsEqual(20)
	call2.ExpectReturnsEqual(40)
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
	wrapper := WrapDivide(t, divide)

	call := wrapper.Method.Start(10, 2)

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
	wrapper := WrapPanicInt(t, panicFunc)

	call := wrapper.Method.Start()
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
	wrapper := WrapCompute(t, compute)

	call := wrapper.Method.Start(21)
	call.ExpectReturnsEqual(42, "computed", true)
}

// TestCallHandle_NoReturns verifies handles work with functions that have no returns.
func TestCallHandle_NoReturns(t *testing.T) {
	t.Parallel()

	// Function with no return value - just runs and completes
	sideEffect := func(x int) {
		_ = x // do something
	}
	wrapper := WrapSideEffect(t, sideEffect)

	call := wrapper.Method.Start(100)

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
	wrapper := WrapConditional(t, conditional)

	callPanic := wrapper.Method.Start(-5)
	callReturn := wrapper.Method.Start(5)

	// One should panic, one should return - independent verification
	callPanic.ExpectPanicEquals("negative value")
	callReturn.ExpectReturnsEqual(50)
}

// TestCallHandle_PanicCapture verifies panic handling with call handles.
//
// REQUIREMENT: Call handles must support ExpectPanicEquals() and ExpectPanicMatches().
func TestCallHandle_PanicCapture(t *testing.T) {
	t.Parallel()

	panicFunc := func() { panic("boom") }
	wrapper := WrapPanic(t, panicFunc)

	call := wrapper.Method.Start()
	call.ExpectPanicEquals("boom")
}

// TestCallHandle_PanicMatches verifies panic matching with matchers.
func TestCallHandle_PanicMatches(t *testing.T) {
	t.Parallel()

	wrapper := WrapPanicWithMessage(t, callable.PanicWithMessage)

	call := wrapper.Method.Start("critical error")
	// Use Any() matcher to accept any panic value
	call.ExpectPanicMatches(imptest.Any)
}

// TestCallHandle_UniqueHandles verifies that each Start() call returns a unique call handle.
//
// REQUIREMENT: Each Start() call must return a NEW unique call handle, not the wrapper itself.
// This enables managing multiple concurrent goroutines independently.
func TestCallHandle_UniqueHandles(t *testing.T) {
	t.Parallel()

	// Simple addition function for testing
	add := func(a, b int) int { return a + b }
	wrapper := WrapAdd(t, add)

	// Start two calls - each should return a different handle
	call1 := wrapper.Method.Start(10, 20)
	call2 := wrapper.Method.Start(30, 40)

	// Verify they are different objects (not same pointer)
	if call1 == call2 {
		t.Fatalf("Start() returned same handle twice - expected unique handles for each call")
	}

	// Each handle should independently verify its own results
	call1.ExpectReturnsEqual(30)
	call2.ExpectReturnsEqual(70)
}
