package core_test

import (
	"testing"

	"github.com/toejough/imptest"
)

// TestDependencyCall_GetArgs verifies GetArgs returns correct argument values.
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
