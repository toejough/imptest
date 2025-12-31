package imptest_test

import (
	"testing"

	"github.com/toejough/imptest/imptest"
)

// TestDependencyCall_GetArgs verifies GetArgs returns correct argument values.
//
//nolint:varnamelen // Standard test parameter name
func TestDependencyCall_GetArgs(t *testing.T) {
	t.Parallel()

	// Create an Imp and a mock dependency method
	imp := imptest.NewImp(t)
	method := imptest.NewDependencyMethod(imp, "TestMethod")

	// Simulate a call with 3 arguments in a goroutine
	go func() {
		call := &imptest.GenericCall{
			MethodName:   "TestMethod",
			Args:         []any{42, "hello", true},
			ResponseChan: make(chan imptest.GenericResponse, 1),
		}
		imp.CallChan <- call
	}()

	// Get the call and verify arguments
	depCall := method.ExpectCalledWithExactly(42, "hello", true)
	args := depCall.GetArgs()

	if args.A1 != 42 {
		t.Errorf("expected A1=42, got %v", args.A1)
	}

	if args.A2 != "hello" {
		t.Errorf("expected A2='hello', got %v", args.A2)
	}

	if args.A3 != true {
		t.Errorf("expected A3=true, got %v", args.A3)
	}

	// Inject response to unblock the mock
	depCall.InjectReturnValues()
}

// TestDependencyCall_InjectPanicValue verifies panic injection works correctly.
//
//nolint:varnamelen // Standard test parameter name
func TestDependencyCall_InjectPanicValue(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	method := imptest.NewDependencyMethod(imp, "PanicMethod")

	// Start a goroutine that will panic when we inject
	done := make(chan bool)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if r != "test panic" {
					t.Errorf("expected panic 'test panic', got %v", r)
				}

				done <- true
			}
		}()

		// Send a call
		responseChan := make(chan imptest.GenericResponse, 1)

		call := &imptest.GenericCall{
			MethodName:   "PanicMethod",
			Args:         []any{},
			ResponseChan: responseChan,
		}
		imp.CallChan <- call

		// Wait for response and panic if instructed
		resp := <-responseChan
		if resp.Type == "panic" {
			panic(resp.PanicValue)
		}
	}()

	// Inject panic value
	depCall := method.ExpectCalledWithExactly()
	depCall.InjectPanicValue("test panic")

	// Wait for goroutine to complete
	<-done
}

// TestDependencyMethod_ExpectCalledWithMatches verifies matcher-based expectations work.
//
//nolint:varnamelen // Standard test parameter name
func TestDependencyMethod_ExpectCalledWithMatches(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	method := imptest.NewDependencyMethod(imp, "MatcherMethod")

	// Start goroutine to send call
	go func() {
		call := &imptest.GenericCall{
			MethodName:   "MatcherMethod",
			Args:         []any{42},
			ResponseChan: make(chan imptest.GenericResponse, 1),
		}
		imp.CallChan <- call
	}()

	// Expect call with matcher
	depCall := method.ExpectCalledWithMatches(imptest.Any())
	args := depCall.GetArgs()

	if args.A1 != 42 {
		t.Errorf("expected A1=42, got %v", args.A1)
	}

	depCall.InjectReturnValues()
}

// TestDependencyMethodDefaultOrdered verifies default DependencyMethod uses ordered mode.
// Default behavior should fail fast on mismatched calls (GetCallOrdered).
//
//nolint:varnamelen // Standard test parameter name
func TestDependencyMethodDefaultOrdered(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	method := imptest.NewDependencyMethod(imp, "TestMethod")

	// Start goroutine to send matching call
	go func() {
		call := &imptest.GenericCall{
			MethodName:   "TestMethod",
			Args:         []any{42},
			ResponseChan: make(chan imptest.GenericResponse, 1),
		}
		imp.CallChan <- call
	}()

	// Call ExpectCalledWithExactly - should use GetCallOrdered (fail-fast mode)
	depCall := method.ExpectCalledWithExactly(42)

	// Verify we got the call
	args := depCall.GetArgs()
	if args.A1 != 42 {
		t.Errorf("expected A1=42, got %v", args.A1)
	}

	depCall.InjectReturnValues()
}

// TestDependencyMethodEventuallyMode verifies Eventually() switches to eventually mode.
// Eventually mode should queue mismatches (GetCallEventually).
//
//nolint:varnamelen // Standard test parameter name
func TestDependencyMethodEventuallyMode(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	method := imptest.NewDependencyMethod(imp, "TestMethod")

	// Switch to eventually mode (no timeout parameter)
	eventualMethod := method.Eventually()

	// Start goroutine to send matching call
	go func() {
		call := &imptest.GenericCall{
			MethodName:   "TestMethod",
			Args:         []any{42},
			ResponseChan: make(chan imptest.GenericResponse, 1),
		}
		imp.CallChan <- call
	}()

	// Call ExpectCalledWithExactly - should use GetCallEventually (queue mode)
	depCall := eventualMethod.ExpectCalledWithExactly(42)

	// Verify we got the call
	args := depCall.GetArgs()
	if args.A1 != 42 {
		t.Errorf("expected A1=42, got %v", args.A1)
	}

	depCall.InjectReturnValues()
}

// TestDependencyMethodEventuallyReturnsNew verifies Eventually() returns a new instance.
// The original DependencyMethod should remain in ordered mode.
//
//nolint:varnamelen // Standard test parameter name
func TestDependencyMethodEventuallyReturnsNew(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	dm1 := imptest.NewDependencyMethod(imp, "TestMethod")

	// Call Eventually() to get new instance
	dm2 := dm1.Eventually()

	// Verify dm1 is still in ordered mode (eventually = false)
	// We'll test this by verifying both instances work independently
	go func() {
		// Send call for dm1 (ordered mode)
		call := &imptest.GenericCall{
			MethodName:   "TestMethod",
			Args:         []any{1},
			ResponseChan: make(chan imptest.GenericResponse, 1),
		}
		imp.CallChan <- call
	}()

	depCall1 := dm1.ExpectCalledWithExactly(1)
	args1 := depCall1.GetArgs()
	if args1.A1 != 1 {
		t.Errorf("dm1: expected A1=1, got %v", args1.A1)
	}
	depCall1.InjectReturnValues()

	go func() {
		// Send call for dm2 (eventually mode)
		call := &imptest.GenericCall{
			MethodName:   "TestMethod",
			Args:         []any{2},
			ResponseChan: make(chan imptest.GenericResponse, 1),
		}
		imp.CallChan <- call
	}()

	depCall2 := dm2.ExpectCalledWithExactly(2)
	args2 := depCall2.GetArgs()
	if args2.A1 != 2 {
		t.Errorf("dm2: expected A1=2, got %v", args2.A1)
	}
	depCall2.InjectReturnValues()
}
