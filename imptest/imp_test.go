package imptest_test

//go:generate ../bin/impgen TestReporter --dependency

import (
	"fmt"
	"testing"
	"time"

	"github.com/toejough/imptest/imptest"
)

// TestGenericCallDone tests GenericCall.Done.
func TestGenericCallDone(t *testing.T) {
	t.Parallel()

	call := &imptest.GenericCall{}
	if call.Done() {
		t.Error("expected Done() to return false initially")
	}

	call.MarkDone()

	if !call.Done() {
		t.Error("expected Done() to return true after MarkDone()")
	}
}

// TestGenericCallName tests GenericCall.Name.
func TestGenericCallName(t *testing.T) {
	t.Parallel()

	call := &imptest.GenericCall{MethodName: "TestMethod"}
	if call.Name() != "TestMethod" {
		t.Errorf("expected Name() to return 'TestMethod', got %q", call.Name())
	}
}

// TestImpFatalf tests that Imp.Fatalf delegates to the underlying test reporter.
func TestImpFatalf(t *testing.T) {
	t.Parallel()

	mockReporter := MockTestReporter(t)

	// Handle the Fatalf call
	go func() {
		call := mockReporter.Fatalf.ExpectCalledWithMatches(imptest.Any())
		call.InjectReturnValues()
	}()

	imp := imptest.NewImp(mockReporter.Interface())
	imp.Fatalf("test message")
}

// TestImpGetCallEventually_QueuesOtherMethods tests that Imp.GetCallEventually
// queues calls with different method names while waiting for the matching method.
func TestImpGetCallEventually_QueuesOtherMethods(t *testing.T) {
	t.Parallel()

	mockReporter := MockTestReporter(t)

	// Handle Helper calls (one per GetCallEventually call)
	go func() {
		call := mockReporter.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = mockReporter.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	imp := imptest.NewImp(mockReporter.Interface())

	// Validator that accepts any arguments
	validator := func(_ []any) error {
		return nil // Accept any args
	}

	// Start waiter for "Add" method
	resultChan := make(chan *imptest.GenericCall, 1)

	go func() {
		call := imp.GetCallEventually("Add", validator)
		resultChan <- call
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send "Multiply" call first (should be queued)
	multiplyCall := &imptest.GenericCall{
		MethodName:   "Multiply",
		Args:         []any{5, 6},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	imp.CallChan <- multiplyCall

	// Send "Add" call second (should match the waiter)
	addCall := &imptest.GenericCall{
		MethodName:   "Add",
		Args:         []any{2, 3},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	imp.CallChan <- addCall

	// Verify we received the "Add" call
	select {
	case received := <-resultChan:
		if received != addCall {
			t.Errorf("expected to receive addCall, got different call")
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for GetCallEventually to return")
	}

	// Verify "Multiply" call is still queued
	// We use GetCall (which checks queue first) with a validator for "Multiply"
	queuedCall := imp.GetCall(500*time.Millisecond, func(call *imptest.GenericCall) error {
		if call.MethodName != "Multiply" {
			return fmt.Errorf("expected method 'Multiply', got %q", call.MethodName)
		}

		return nil
	})

	if queuedCall != multiplyCall {
		t.Errorf("expected multiplyCall to be queued, got different call")
	}
}

// TestImpGetCallOrdered_MatchingCall tests that Imp.GetCallOrdered returns
// a call when it arrives with the expected method name.
func TestImpGetCallOrdered_MatchingCall(t *testing.T) {
	t.Parallel()

	mockReporter := MockTestReporter(t)

	// Handle Helper call
	go func() {
		call := mockReporter.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	imp := imptest.NewImp(mockReporter.Interface())
	mismatchChan := make(chan *imptest.GenericCall, 1)

	// Validator that accepts calls with Args matching [2, 3]
	validator := func(args []any) error {
		if len(args) != 2 {
			return fmt.Errorf("expected 2 args, got %d", len(args))
		}

		if args[0] != 2 || args[1] != 3 {
			return fmt.Errorf("expected args [2, 3], got %v", args)
		}

		return nil
	}

	// Start waiter in background
	resultChan := make(chan *imptest.GenericCall, 1)

	go func() {
		call := imp.GetCallOrdered(1*time.Second, "Add", validator)
		resultChan <- call
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send matching call
	expectedCall := &imptest.GenericCall{
		MethodName:   "Add",
		Args:         []any{2, 3},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	imp.CallChan <- expectedCall

	// Verify we received the call
	select {
	case received := <-resultChan:
		if received != expectedCall {
			t.Errorf("expected to receive the sent call, got different call")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for GetCallOrdered to return")
	}

	// Verify no mismatch
	select {
	case <-mismatchChan:
		t.Error("unexpected mismatch signal")
	default:
		// Expected - no mismatch
	}
}

// TestImpGetCallOrdered_WrongMethod tests that Imp.GetCallOrdered sends a
// mismatch signal when a call with a different method name arrives first.
func TestImpGetCallOrdered_WrongMethod(t *testing.T) {
	t.Parallel()

	mockReporter := MockTestReporter(t)

	// Handle Helper call
	go func() {
		call := mockReporter.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	imp := imptest.NewImp(mockReporter.Interface())

	// Validator that accepts "Add" calls with any args
	validator := func(_ []any) error {
		return nil // Accept any args
	}

	// Send call with wrong method name BEFORE registering waiter
	wrongMethodCall := &imptest.GenericCall{
		MethodName:   "Multiply",
		Args:         []any{5, 6},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	imp.CallChan <- wrongMethodCall

	// Give dispatcher time to queue the call
	time.Sleep(10 * time.Millisecond)

	// Handle expected Fatalf call
	go func() {
		call := mockReporter.Fatalf.ExpectCalledWithMatches(imptest.Any(), imptest.Any())

		// Verify error message mentions both expected and actual methods
		raw := call.RawArgs()

		format, ok := raw[0].(string)
		if !ok {
			t.Errorf("expected format string, got %T", raw[0])

			return
		}

		var fullMsg string
		if len(raw) > 1 {
			fullMsg = fmt.Sprintf(format, raw[1:]...)
		} else {
			fullMsg = format
		}

		if !contains(fullMsg, "Add") || !contains(fullMsg, "Multiply") {
			t.Errorf("expected error message to mention both 'Add' and 'Multiply', got: %s", fullMsg)
		}

		call.InjectReturnValues()
	}()

	// Now call GetCallOrdered - it should fail-fast on queued mismatch with good error message
	imp.GetCallOrdered(1*time.Second, "Add", validator)
}
