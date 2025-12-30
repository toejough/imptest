package imptest_test

//go:generate ../bin/impgen Tester --dependency

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toejough/imptest/imptest"
)

// TestGetCall_ConcurrentWaiters verifies that the waiter registration pattern
// correctly handles concurrent goroutines waiting for different calls.
//
// This test verifies that:
// 1. Multiple goroutines can wait concurrently
// 2. Each goroutine receives the correct call matching its validator
// 3. No calls are lost or delivered to the wrong waiter
// 4. Helper() is called but Fatalf() is NOT called (no timeout).
//
//nolint:varnamelen // Standard Go test parameter name
func TestGetCall_ConcurrentWaiters(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callA := &testCall{name: "callA"}
	callB := &testCall{name: "callB"}

	var (
		waitGroup            sync.WaitGroup
		receivedA, receivedB atomic.Bool
	)

	// Goroutine waiting for callA
	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		result := ctrl.GetCall(1*time.Second, func(call *testCall) bool {
			return call.name == "callA"
		})

		if result == callA {
			receivedA.Store(true)
		}
	}()

	// Goroutine waiting for callB
	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		result := ctrl.GetCall(1*time.Second, func(call *testCall) bool {
			return call.name == "callB"
		})

		if result == callB {
			receivedB.Store(true)
		}
	}()

	// Handle the two Helper() calls (one per GetCall)
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Send both calls (dispatcher receives them immediately)
	ctrl.CallChan <- callA

	ctrl.CallChan <- callB

	waitGroup.Wait()

	// If we reach here, the test succeeded - both GetCall()s completed
	// without timing out (which would have called Fatalf and blocked).

	if !receivedA.Load() {
		t.Error("Goroutine A did not receive callA")
	}

	if !receivedB.Load() {
		t.Error("Goroutine B did not receive callB")
	}
}

// TestGetCall_QueuedCallsMatchLaterWaiters verifies that calls queued before
// a waiter arrives are correctly matched when the waiter calls GetCall.
// Also verifies that Helper() is called but Fatalf() is NOT called (no timeout).
//
//nolint:varnamelen // Standard Go test parameter name
func TestGetCall_QueuedCallsMatchLaterWaiters(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	call1 := &testCall{name: "call1"}
	call2 := &testCall{name: "call2"}

	// Handle the two Helper() calls (one per GetCall)
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Send calls (dispatcher receives and queues them immediately)
	ctrl.CallChan <- call1

	ctrl.CallChan <- call2

	// Now wait for call2 (skipping call1)
	result := ctrl.GetCall(1*time.Second, func(call *testCall) bool {
		return call.name == "call2"
	})

	if result != call2 {
		t.Errorf("Expected to receive call2, got %v", result)
	}

	// call1 should still be in the queue
	result = ctrl.GetCall(1*time.Second, func(call *testCall) bool {
		return call.name == "call1"
	})

	if result != call1 {
		t.Errorf("Expected to receive call1, got %v", result)
	}
}

// testCall implements the Call interface for testing.
type testCall struct {
	name string
	done atomic.Bool
}

func (c *testCall) Done() bool {
	return c.done.Load()
}

func (c *testCall) MarkDone() {
	c.done.Store(true)
}

func (c *testCall) Name() string {
	return c.name
}
