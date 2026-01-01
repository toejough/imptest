package imptest_test

// This file contains regression tests for data races in the mockTester pattern.
//
// BACKGROUND:
// Several tests use a mockTester pattern where local test variables are captured
// in closures passed to mockTester.Fatalf. The dispatcher goroutine calls Fatalf
// asynchronously, writing to these variables, while the test goroutine reads them
// after a sleep. This creates a classic data race: concurrent reads and writes
// to shared variables without synchronization.
//
// THE RACES:
// 1. TestDispatchLoop_OrderedFailsOnDispatcherMismatch (controller_test.go:554-600)
//    - Write: dispatcher goroutine in Fatalf closure (lines 563-564)
//    - Read: test goroutine (lines 592, 597)
//
// 2. TestGetCallOrdered_FailsOnMismatch (controller_test.go:200-241)
//    - Write: dispatcher goroutine in Fatalf closure (lines 209-210)
//    - Read: test goroutine (lines 233, 238)
//
// 3. TestImpGetCallOrdered_WrongMethod (imp_test.go:149-189)
//    - Write: dispatcher goroutine in Fatalf closure (lines 156-157)
//    - Read: test goroutine (lines 183, 186)
//
// WHY THESE REGRESSION TESTS:
// These tests are designed to reliably reproduce the race conditions using
// Go's race detector. They demonstrate the exact failure mode: unsynchronized
// access to closure variables shared between goroutines. Each test runs the
// racy code and MUST be run with -race to detect the issue.
//
// RUN WITH: go test -race ./imptest
//
// The tests will PASS functionally but the race detector will report:
// "WARNING: DATA RACE" showing the concurrent read/write conflict.

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toejough/imptest/imptest"
)

// TestProperSynchronization_AtomicBased demonstrates another CORRECT pattern
// using atomic operations and sync.WaitGroup.
//
// SOLUTION: Use atomic operations for the boolean flag and a sync.WaitGroup
// or mutex for the string message.
//
//nolint:varnamelen // Standard Go test parameter name
func TestProperSynchronization_AtomicBased(t *testing.T) {
	t.Parallel()

	// CORRECT: Use atomic for boolean, mutex for string
	var fatalfCalled atomic.Bool

	var msgMu sync.Mutex

	var fatalfMsg string

	mockTester := &mockTester{
		helper: func() {},
		fatalf: func(format string, args ...any) {
			fatalfCalled.Store(true) // Atomic write

			msgMu.Lock()

			fatalfMsg = fmt.Sprintf(format, args...)

			msgMu.Unlock()
		},
	}

	ctrl := imptest.NewController[*testCall](mockTester)
	callB := &testCall{name: "CallB"}

	go func() {
		ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallA" {
				return fmt.Errorf("expected CallA, got %q", call.name)
			}

			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)

	ctrl.CallChan <- callB

	time.Sleep(100 * time.Millisecond)

	// CORRECT: Atomic read
	if !fatalfCalled.Load() {
		t.Error("expected Fatalf to be called")
	}

	// CORRECT: Mutex-protected read
	msgMu.Lock()

	msg := fatalfMsg

	msgMu.Unlock()

	if !contains(msg, "CallA") || !contains(msg, "CallB") {
		t.Errorf("expected message to mention both calls, got: %s", msg)
	}
}

// TestProperSynchronization_ChannelBased demonstrates the CORRECT pattern
// for synchronizing between the Fatalf call and the test goroutine.
//
// SOLUTION: Use a channel to signal when Fatalf is called, and capture the
// message safely.
//
//nolint:varnamelen // Standard Go test parameter name
func TestProperSynchronization_ChannelBased(t *testing.T) {
	t.Parallel()

	// CORRECT: Use a channel for synchronization
	fatalfChan := make(chan string, 1)

	mockTester := &mockTester{
		helper: func() {},
		fatalf: func(format string, args ...any) {
			msg := fmt.Sprintf(format, args...)
			fatalfChan <- msg // Send to channel - provides synchronization
		},
	}

	ctrl := imptest.NewController[*testCall](mockTester)
	callB := &testCall{name: "CallB"}

	go func() {
		ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallA" {
				return fmt.Errorf("expected CallA, got %q", call.name)
			}

			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)

	ctrl.CallChan <- callB

	// CORRECT: Receive from channel - happens-before relationship established
	select {
	case msg := <-fatalfChan:
		if !contains(msg, "CallA") || !contains(msg, "CallB") {
			t.Errorf("expected message to mention both calls, got: %s", msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for Fatalf to be called")
	}
}

// TestRaceRegression_DispatcherFatalfClosure reproduces the race condition in
// TestDispatchLoop_OrderedFailsOnDispatcherMismatch where the dispatcher
// goroutine writes to test-local variables via a Fatalf closure while the
// test goroutine reads them.
//
// RACE CONDITION:
//   - Write: dispatcher goroutine calls mockTester.Fatalf() which writes to
//     fatalfCalled and fatalfMsg via closure
//   - Read: test goroutine reads fatalfCalled and fatalfMsg after sleep
//   - NO SYNCHRONIZATION between the write and read
//
// This test MUST be run with -race to detect the issue.
//
//nolint:varnamelen // Standard Go test parameter name
func TestRaceRegression_DispatcherFatalfClosure(t *testing.T) {
	t.Parallel()

	// These variables are accessed by both test goroutine and dispatcher goroutine
	// WITHOUT SYNCHRONIZATION - this is the race!
	fatalfCalled := false // RACE: written by dispatcher, read by test

	var fatalfMsg string // RACE: written by dispatcher, read by test

	mockTester := &mockTester{
		helper: func() {},
		fatalf: func(format string, args ...any) {
			// WRITE: This runs in the dispatcher goroutine
			fatalfCalled = true                      // RACE: concurrent write
			fatalfMsg = fmt.Sprintf(format, args...) // RACE: concurrent write
		},
	}

	ctrl := imptest.NewController[*testCall](mockTester)
	callB := &testCall{name: "CallB"}

	// Register ordered waiter in a goroutine
	go func() {
		ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallA" {
				return fmt.Errorf("expected CallA, got %q", call.name)
			}

			return nil
		})
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send non-matching call - triggers Fatalf in dispatcher goroutine
	ctrl.CallChan <- callB

	// Wait for Fatalf to be called
	time.Sleep(100 * time.Millisecond)

	// READ: This is in the test goroutine - RACE!
	if !fatalfCalled { // RACE: concurrent read
		t.Error("expected Fatalf to be called")
	}

	// READ: Another racy read
	if !contains(fatalfMsg, "CallA") { // RACE: concurrent read
		t.Errorf("expected error message to mention CallA, got: %s", fatalfMsg)
	}
}

// TestRaceRegression_ImpWrongMethodClosure reproduces the race condition in
// TestImpGetCallOrdered_WrongMethod in imp_test.go.
//
// RACE CONDITION:
//   - Write: Fatalf closure writes to fatalfCalled and fatalfMsg
//   - Read: test goroutine reads these variables
//   - NO SYNCHRONIZATION
//
// This test MUST be run with -race to detect the issue.
//
//nolint:varnamelen // Standard Go test parameter name
func TestRaceRegression_ImpWrongMethodClosure(t *testing.T) {
	t.Parallel()

	// These variables are accessed without synchronization
	fatalfCalled := false // RACE: written by closure, read by test

	var fatalfMsg string // RACE: written by closure, read by test

	mockReporter := &mockTestReporter{
		fatal: func(format string, args ...any) {
			// WRITE: Can run in dispatcher goroutine
			fatalfCalled = true                      // RACE: concurrent write
			fatalfMsg = fmt.Sprintf(format, args...) // RACE: concurrent write
		},
	}

	imp := imptest.NewImp(mockReporter)

	// Validator for "Add" calls
	validator := func(_ []any) error {
		return nil // Accept any args
	}

	// Send wrong method call before registering waiter
	wrongMethodCall := &imptest.GenericCall{
		MethodName:   "Multiply",
		Args:         []any{5, 6},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	imp.CallChan <- wrongMethodCall

	// Give dispatcher time to queue
	time.Sleep(10 * time.Millisecond)

	// Call GetCallOrdered - should fail-fast on queued mismatch
	imp.GetCallOrdered(1*time.Second, "Add", validator)

	// READ: Racy reads
	if !fatalfCalled { // RACE: concurrent read
		t.Error("expected Fatalf to be called")
	}

	if !contains(fatalfMsg, "Add") { // RACE: concurrent read
		t.Errorf("expected error message to mention Add, got: %s", fatalfMsg)
	}
}

// TestRaceRegression_QueuedCallFatalfClosure reproduces the race condition in
// TestGetCallOrdered_FailsOnMismatch where GetCallOrdered checks a queued call
// and calls Fatalf in the test goroutine, but the closure still captures
// variables that could be racy in similar patterns.
//
// This test demonstrates the same pattern but with the call already queued.
//
// RACE CONDITION (same as above):
//   - Write: when Fatalf is called (can be in different goroutine in real usage)
//   - Read: test goroutine reads the variables
//   - NO SYNCHRONIZATION
//
// This test MUST be run with -race to detect the issue.
//
//nolint:varnamelen // Standard Go test parameter name
func TestRaceRegression_QueuedCallFatalfClosure(t *testing.T) {
	t.Parallel()

	// These variables are the source of the race
	fatalfCalled := false // RACE: written by closure, read by test

	var fatalfMsg string // RACE: written by closure, read by test

	mockTester := &mockTester{
		helper: func() {},
		fatalf: func(format string, args ...any) {
			// WRITE: Can run in different goroutine
			fatalfCalled = true                      // RACE: concurrent write
			fatalfMsg = fmt.Sprintf(format, args...) // RACE: concurrent write
		},
	}

	ctrl := imptest.NewController[*testCall](mockTester)
	callB := &testCall{name: "callB"}

	// Queue callB BEFORE creating waiter
	ctrl.CallChan <- callB

	// Give dispatcher time to queue it
	time.Sleep(50 * time.Millisecond)

	// Call GetCallOrdered - fails fast on queued mismatch
	// In this case, Fatalf is called synchronously in the calling goroutine,
	// but the pattern is the same: variables are shared without sync
	ctrl.GetCallOrdered(1*time.Second, func(call *testCall) error {
		if call.name != "callA" {
			return fmt.Errorf("expected callA, got %q", call.name)
		}

		return nil
	})

	// READ: Racy reads (even though in this specific case it might be synchronous,
	// the pattern is racy and could fail with different timing)
	if !fatalfCalled { // RACE: concurrent read
		t.Error("expected Fatalf to be called")
	}

	if !contains(fatalfMsg, "callA") { // RACE: concurrent read
		t.Errorf("expected error message to mention callA, got: %s", fatalfMsg)
	}
}

// TestRaceRegression_StressTest is an aggressive test that repeatedly triggers
// the race condition to make it more likely to be detected. This test creates
// multiple racing accesses in quick succession.
//
// RACE CONDITION:
//   - Write: dispatcher goroutine continuously writes to variables
//   - Read: test goroutine continuously reads the same variables
//   - NO SYNCHRONIZATION - maximum race exposure
//
// This test MUST be run with -race to detect the issue.
//
//nolint:varnamelen // Standard Go test parameter name
func TestRaceRegression_StressTest(t *testing.T) {
	t.Parallel()

	// Racy variables - accessed from multiple goroutines
	fatalfCalled := false // RACE: written/read concurrently

	var fatalfMsg string // RACE: written/read concurrently

	mockTester := &mockTester{
		helper: func() {},
		fatalf: func(format string, args ...any) {
			// WRITE: Repeatedly write to create race opportunity
			for range 10 {
				fatalfCalled = true                      // RACE: concurrent write
				fatalfMsg = fmt.Sprintf(format, args...) // RACE: concurrent write
			}
		},
	}

	ctrl := imptest.NewController[*testCall](mockTester)
	callB := &testCall{name: "CallB"}

	// Register waiter
	go func() {
		ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallA" {
				return fmt.Errorf("expected CallA, got %q", call.name)
			}

			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)

	ctrl.CallChan <- callB

	// READ: Repeatedly read to create race opportunity
	for range 10 {
		time.Sleep(10 * time.Millisecond)

		_ = fatalfCalled // RACE: concurrent read
		_ = fatalfMsg    // RACE: concurrent read
	}

	// Final check
	if !fatalfCalled {
		t.Error("expected Fatalf to be called")
	}

	if !contains(fatalfMsg, "CallA") {
		t.Errorf("expected message to mention CallA, got: %s", fatalfMsg)
	}
}

// mockTestReporter is a simple mock for testing.TB interface
type mockTestReporter struct {
	fatal func(string, ...any)
}

func (m *mockTestReporter) Fatalf(format string, args ...any) {
	if m.fatal != nil {
		m.fatal(format, args...)
	}
}

func (m *mockTestReporter) Helper() {}

// mockTester is a simple mock for demonstrating race conditions in these regression tests
type mockTester struct {
	helper func()
	fatalf func(string, ...any)
}

func (m *mockTester) Fatalf(format string, args ...any) {
	if m.fatalf != nil {
		m.fatalf(format, args...)
	}
}

func (m *mockTester) Helper() {
	if m.helper != nil {
		m.helper()
	}
}
