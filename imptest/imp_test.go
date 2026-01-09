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
		call := mockReporter.Method.Fatalf.ExpectCalledWithMatches(imptest.Any())
		call.InjectReturnValues()
	}()

	imp := imptest.NewImp(mockReporter.Mock)
	imp.Fatalf("test message")
}

// TestImpGetCallEventually_QueuesOtherMethods tests that Imp.GetCallEventually
// queues calls with different method names while waiting for the matching method.
func TestImpGetCallEventually_QueuesOtherMethods(t *testing.T) {
	t.Parallel()

	mockReporter := MockTestReporter(t)

	// Handle Helper calls (one per GetCallEventually call)
	go func() {
		call := mockReporter.Method.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = mockReporter.Method.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	imp := imptest.NewImp(mockReporter.Mock)

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
