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
