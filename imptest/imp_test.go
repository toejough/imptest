package imptest

import (
	"testing"
	"time"
)

// TestGenericCallDone tests GenericCall.Done.
func TestGenericCallDone(t *testing.T) {
	t.Parallel()

	call := &GenericCall{}
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

	call := &GenericCall{MethodName: "TestMethod"}
	if call.Name() != "TestMethod" {
		t.Errorf("expected Name() to return 'TestMethod', got %q", call.Name())
	}
}

// TestImpFatalf tests that Imp.Fatalf delegates to the underlying test reporter.
func TestImpFatalf(t *testing.T) {
	t.Parallel()

	called := false
	mockReporter := &mockTestReporter{
		fatal: func(_ string, _ ...any) {
			called = true
		},
	}

	imp := &Imp{t: mockReporter}
	imp.Fatalf("test message")

	if !called {
		t.Error("expected Fatalf to be called on underlying reporter")
	}
}

// TestTesterAdapterFatalf tests that testerAdapter.Fatalf delegates correctly.
func TestTesterAdapterFatalf(t *testing.T) {
	t.Parallel()

	called := false
	mockReporter := &mockTestReporter{
		fatal: func(_ string, _ ...any) {
			called = true
		},
	}

	adapter := &testerAdapter{t: mockReporter}
	adapter.Fatalf("test")

	if !called {
		t.Error("expected Fatalf to be called on underlying reporter")
	}
}

type mockTestReporter struct {
	fatal func(string, ...any)
}

func (m *mockTestReporter) Fatalf(format string, args ...any) {
	if m.fatal != nil {
		m.fatal(format, args...)
	}
}

func (m *mockTestReporter) Helper() {}

// TestImpGetCallOrdered_MatchingCall tests that Imp.GetCallOrdered returns
// a call when it arrives with the expected method name.
func TestImpGetCallOrdered_MatchingCall(t *testing.T) {
	t.Parallel()

	mockReporter := &mockTestReporter{
		fatal: func(_ string, _ ...any) {
			t.Fatal("unexpected call to Fatalf")
		},
	}

	imp := NewImp(mockReporter)
	mismatchChan := make(chan *GenericCall, 1)

	// Validator that accepts calls with Args matching [2, 3]
	validator := func(args []any) bool {
		if len(args) != 2 {
			return false
		}
		return args[0] == 2 && args[1] == 3
	}

	// Start waiter in background
	resultChan := make(chan *GenericCall, 1)
	go func() {
		call := imp.GetCallOrdered(1*time.Second, "Add", validator, mismatchChan)
		resultChan <- call
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send matching call
	expectedCall := &GenericCall{
		MethodName:   "Add",
		Args:         []any{2, 3},
		ResponseChan: make(chan GenericResponse, 1),
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

	mockReporter := &mockTestReporter{
		fatal: func(_ string, _ ...any) {
			t.Fatal("unexpected call to Fatalf")
		},
	}

	imp := NewImp(mockReporter)
	mismatchChan := make(chan *GenericCall, 1)

	// Validator that accepts any arguments
	validator := func(_ []any) bool {
		return true
	}

	// Start waiter for "Add" method
	go func() {
		imp.GetCallOrdered(2*time.Second, "Add", validator, mismatchChan)
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send call with wrong method name
	wrongMethodCall := &GenericCall{
		MethodName:   "Multiply",
		Args:         []any{5, 6},
		ResponseChan: make(chan GenericResponse, 1),
	}
	imp.CallChan <- wrongMethodCall

	// Verify mismatch signal received
	select {
	case mismatched := <-mismatchChan:
		if mismatched != wrongMethodCall {
			t.Errorf("expected wrongMethodCall on mismatch channel, got different call")
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for mismatch signal")
	}
}

// TestImpGetCallEventually_QueuesOtherMethods tests that Imp.GetCallEventually
// queues calls with different method names while waiting for the matching method.
func TestImpGetCallEventually_QueuesOtherMethods(t *testing.T) {
	t.Parallel()

	mockReporter := &mockTestReporter{
		fatal: func(_ string, _ ...any) {
			t.Fatal("unexpected call to Fatalf")
		},
	}

	imp := NewImp(mockReporter)

	// Validator that accepts any arguments
	validator := func(_ []any) bool {
		return true
	}

	// Start waiter for "Add" method
	resultChan := make(chan *GenericCall, 1)
	go func() {
		call := imp.GetCallEventually(2*time.Second, "Add", validator)
		resultChan <- call
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send "Multiply" call first (should be queued)
	multiplyCall := &GenericCall{
		MethodName:   "Multiply",
		Args:         []any{5, 6},
		ResponseChan: make(chan GenericResponse, 1),
	}
	imp.CallChan <- multiplyCall

	// Send "Add" call second (should match the waiter)
	addCall := &GenericCall{
		MethodName:   "Add",
		Args:         []any{2, 3},
		ResponseChan: make(chan GenericResponse, 1),
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
	queuedCall := imp.GetCall(500*time.Millisecond, func(call *GenericCall) bool {
		return call.MethodName == "Multiply"
	})

	if queuedCall != multiplyCall {
		t.Errorf("expected multiplyCall to be queued, got different call")
	}
}
