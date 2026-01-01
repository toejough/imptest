package imptest_test

//go:generate ../bin/impgen Tester --dependency

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toejough/imptest/imptest"
)

// TestDispatchLoop_FIFOPriority verifies that the dispatcher follows FIFO order
// when multiple waiters are registered, prioritizing the first waiter.
//
// This test verifies that:
// 1. First waiter (ordered for "CallA") is registered
// 2. Second waiter (eventually for "CallB") is registered after
// 3. When "CallA" arrives, the first waiter receives it (FIFO)
// 4. When "CallB" arrives, the second waiter receives it
//
//nolint:varnamelen,funlen // Standard Go test parameter name; test requires setup
func TestDispatchLoop_FIFOPriority(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callA := &testCall{name: "CallA"}
	callB := &testCall{name: "CallB"}

	var (
		orderedReceivedA    atomic.Bool
		eventuallyReceivedB atomic.Bool
	)

	// Handle Helper() calls (one for ordered, one for eventually)
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Register ordered waiter FIRST for CallA
	go func() {
		result := ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallA" {
				return fmt.Errorf("expected CallA, got %q", call.name)
			}

			return nil
		})
		if result == callA {
			orderedReceivedA.Store(true)
		}
	}()

	// Give first waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Register eventually waiter SECOND for CallB
	go func() {
		result := ctrl.GetCallEventually(func(call *testCall) error {
			if call.name != "CallB" {
				return fmt.Errorf("expected CallB, got %q", call.name)
			}

			return nil
		})
		if result == callB {
			eventuallyReceivedB.Store(true)
		}
	}()

	// Give second waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send CallA - should go to FIRST waiter (ordered) due to FIFO
	ctrl.CallChan <- callA

	// Send CallB - should go to SECOND waiter (eventually)
	ctrl.CallChan <- callB

	// Wait for both waiters to receive
	time.Sleep(100 * time.Millisecond)

	// Verify FIFO: ordered waiter (first registered) received CallA
	if !orderedReceivedA.Load() {
		t.Error("Ordered waiter (first registered) did not receive CallA - FIFO violation")
	}

	// Verify eventually waiter received CallB
	if !eventuallyReceivedB.Load() {
		t.Error("Eventually waiter should have received CallB")
	}
}

// TestDispatchLoop_MultipleOrderedSequence verifies the dispatcher's behavior
// with multiple ordered waiters in sequence.
//
// This test verifies that:
// 1. Multiple ordered waiters can be registered in sequence
// 2. When calls arrive in the expected order, each waiter receives its call
// 3. When calls arrive in wrong order, first waiter gets mismatch signal
// 4. FIFO ordering is maintained for ordered waiters
//
//nolint:varnamelen,funlen // Standard Go test parameter name; test requires setup
func TestDispatchLoop_MultipleOrderedSequence(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callA := &testCall{name: "CallA"}
	callB := &testCall{name: "CallB"}

	var (
		firstReceivedA  atomic.Bool
		secondReceivedB atomic.Bool
	)

	// Handle Helper() calls (two for successful sequence)
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Register first ordered waiter for "CallA"
	go func() {
		result := ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallA" {
				return fmt.Errorf("expected CallA, got %q", call.name)
			}

			return nil
		})
		if result == callA {
			firstReceivedA.Store(true)
		}
	}()

	// Give first waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Register second ordered waiter for "CallB"
	go func() {
		result := ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallB" {
				return fmt.Errorf("expected CallB, got %q", call.name)
			}

			return nil
		})
		if result == callB {
			secondReceivedB.Store(true)
		}
	}()

	// Give second waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send CallA, then CallB (correct order)
	ctrl.CallChan <- callA

	ctrl.CallChan <- callB

	// Wait for both waiters to receive
	time.Sleep(100 * time.Millisecond)

	// Verify both received their expected calls
	if !firstReceivedA.Load() {
		t.Error("First ordered waiter did not receive CallA")
	}

	if !secondReceivedB.Load() {
		t.Error("Second ordered waiter did not receive CallB")
	}

	// Note: In the new design, ordered waiters fail-fast on mismatch.
	// If you need out-of-order calls, use Eventually mode instead.
	// The original test had a second part testing wrong order, but that's
	// removed because ordered mode now correctly fails the test on mismatch.
}

// TestDispatchLoop_OrderedFailsOnDispatcherMismatch verifies that an ordered
// waiter fails fast when a non-matching call arrives at the dispatcher.
//
// This test verifies that:
// 1. Ordered waiter for "CallA" is registered
// 2. When "CallB" arrives at dispatcher, ordered waiter fails with Fatalf
// 3. Error message includes both expected and actual call names
//
//nolint:varnamelen // Standard Go test parameter name
func TestDispatchLoop_OrderedFailsOnDispatcherMismatch(t *testing.T) {
	t.Parallel()

	testerMock := NewTesterImp(t)

	// Handle Helper() call
	go func() {
		testerMock.ExpectCallIs.Helper().Resolve()
	}()

	ctrl := imptest.NewController[*testCall](testerMock.Mock)

	callB := &testCall{name: "CallB"}

	// Register ordered waiter for "CallA"
	go func() {
		ctrl.GetCallOrdered(2*time.Second, func(call *testCall) error {
			if call.name != "CallA" {
				return fmt.Errorf("expected CallA, got %q", call.name)
			}

			return nil
		})
	}()

	// Give ordered waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Handle expected Fatalf call
	go func() {
		fatalfCall := testerMock.ExpectCallIs.Fatalf().ExpectArgsShould(imptest.Any(), imptest.Any())

		// Verify error message mentions both expected and actual
		fullMsg := fmt.Sprintf(fatalfCall.format, fatalfCall.args...)
		if !contains(fullMsg, "CallA") || !contains(fullMsg, "CallB") {
			t.Errorf("expected error message to mention both 'CallA' and 'CallB', got: %s", fullMsg)
		}

		fatalfCall.Resolve()
	}()

	// Send CallB - should trigger fail-fast for ordered waiter
	ctrl.CallChan <- callB
}

// TestGetCallEventually_ChecksQueueFirst verifies that GetCallEventually
// checks the queue before registering as a waiter.
//
// This test verifies that:
// 1. A call (callA) is sent before any waiter exists (gets queued)
// 2. GetCallEventually is called for "callA"
// 3. GetCallEventually immediately receives callA from the queue
// 4. Helper() is called but Fatalf() is NOT called (no timeout on GetCallEventually)
//
//nolint:varnamelen // Standard Go test parameter name
func TestGetCallEventually_ChecksQueueFirst(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callA := &testCall{name: "callA"}

	// Handle the Helper() call for GetCallEventually
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Send callA BEFORE any waiter exists (will be queued)
	ctrl.CallChan <- callA

	// Give the dispatcher time to queue the call
	time.Sleep(50 * time.Millisecond)

	// Now call GetCallEventually - should immediately get callA from queue
	result := ctrl.GetCallEventually(func(call *testCall) error {
		if call.name != "callA" {
			return fmt.Errorf("expected callA, got %q", call.name)
		}

		return nil
	})

	if result != callA {
		t.Errorf("Expected to receive callA from queue, got %v", result)
	}
}

// TestGetCallEventually_QueuesOnMismatch verifies that an eventually waiter
// allows non-matching calls to queue while waiting for the matching call.
//
// This test verifies that:
// 1. An eventually waiter (normal mode) is created correctly
// 2. When non-matching calls arrive first, they are queued
// 3. When the matching call arrives, the waiter receives it
// 4. Earlier non-matching calls remain queued for later retrieval
// 5. Helper() is called but Fatalf() is NOT called (no timeout)
//
//nolint:varnamelen // Standard Go test parameter name
func TestGetCallEventually_QueuesOnMismatch(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callA := &testCall{name: "callA"}
	callB := &testCall{name: "callB"}

	var receivedA atomic.Bool

	// Handle the two Helper() calls
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Create an eventually waiter for "callA"
	go func() {
		result := ctrl.GetCall(2*time.Second, func(call *testCall) error {
			if call.name != "callA" {
				return fmt.Errorf("expected callA, got %q", call.name)
			}

			return nil
		})
		if result == callA {
			receivedA.Store(true)
		}
	}()

	// Give the waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send callB first (should be queued, not fail the waiter)
	ctrl.CallChan <- callB

	// Send callA (should match the waiter)
	ctrl.CallChan <- callA

	// Wait for the waiter to receive callA
	time.Sleep(100 * time.Millisecond)

	if !receivedA.Load() {
		t.Error("Eventually waiter did not receive callA")
	}

	// Verify callB is still in the queue
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	result := ctrl.GetCall(1*time.Second, func(call *testCall) error {
		if call.name != "callB" {
			return fmt.Errorf("expected callB, got %q", call.name)
		}

		return nil
	})

	if result != callB {
		t.Errorf("Expected callB to be queued, got %v", result)
	}
}

// TestGetCallEventually_WaitsForMatch verifies that GetCallEventually waits
// for a matching call, allowing non-matching calls to be queued.
//
// This test verifies that:
// 1. GetCallEventually waiter is registered for "callA"
// 2. Non-matching calls (callB, callC) arrive and are queued
// 3. When the matching call (callA) arrives, the waiter receives it
// 4. Previously queued calls remain available for later retrieval
// 5. Helper() is called but Fatalf() is NOT called (no timeout)
//
//nolint:varnamelen,funlen // Standard Go test parameter name; test requires setup
func TestGetCallEventually_WaitsForMatch(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callA := &testCall{name: "callA"}
	callB := &testCall{name: "callB"}
	callC := &testCall{name: "callC"}

	var receivedA atomic.Bool

	// Handle the Helper() calls (one for GetCallEventually, two for later GetCall checks)
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Create an eventually waiter for "callA"
	go func() {
		result := ctrl.GetCallEventually(func(call *testCall) error {
			if call.name != "callA" {
				return fmt.Errorf("expected callA, got %q", call.name)
			}

			return nil
		})
		if result == callA {
			receivedA.Store(true)
		}
	}()

	// Give the waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send callB first (should be queued)
	ctrl.CallChan <- callB

	// Send callC second (should be queued)
	ctrl.CallChan <- callC

	// Send callA third (should match the waiter)
	ctrl.CallChan <- callA

	// Wait for the waiter to receive callA
	time.Sleep(100 * time.Millisecond)

	if !receivedA.Load() {
		t.Error("GetCallEventually did not receive callA")
	}

	// Verify callB and callC are still in the queue (in order)
	resultB := ctrl.GetCall(1*time.Second, func(call *testCall) error {
		if call.name != "callB" {
			return fmt.Errorf("expected callB, got %q", call.name)
		}

		return nil
	})

	if resultB != callB {
		t.Errorf("Expected callB to be queued, got %v", resultB)
	}

	resultC := ctrl.GetCall(1*time.Second, func(call *testCall) error {
		if call.name != "callC" {
			return fmt.Errorf("expected callC, got %q", call.name)
		}

		return nil
	})

	if resultC != callC {
		t.Errorf("Expected callC to be queued, got %v", resultC)
	}
}

// TestGetCallOrdered_FailsOnMismatch verifies that GetCallOrdered fails fast
// when a non-matching call is at the front of the queue.
//
// This test verifies that:
// 1. When a call is queued before an ordered waiter arrives
// 2. And the queued call doesn't match the ordered waiter's criteria
// 3. GetCallOrdered fails immediately with an informative error message
// 4. The error message includes both expected and actual call names
//
//nolint:varnamelen // Standard Go test parameter name
func TestGetCallOrdered_FailsOnMismatch(t *testing.T) {
	t.Parallel()

	testerMock := NewTesterImp(t)

	// Handle Helper() call
	go func() {
		testerMock.ExpectCallIs.Helper().Resolve()
	}()

	ctrl := imptest.NewController[*testCall](testerMock.Mock)

	callB := &testCall{name: "callB"}

	// Queue callB BEFORE creating the ordered waiter
	ctrl.CallChan <- callB

	// Give dispatcher time to queue it
	time.Sleep(50 * time.Millisecond)

	// Handle expected Fatalf call
	go func() {
		fatalfCall := testerMock.ExpectCallIs.Fatalf().ExpectArgsShould(imptest.Any(), imptest.Any())

		// Verify error message mentions both expected and actual
		fullMsg := fmt.Sprintf(fatalfCall.format, fatalfCall.args...)
		if !contains(fullMsg, "callA") || !contains(fullMsg, "callB") {
			t.Errorf("expected error message to mention both 'callA' and 'callB', got: %s", fullMsg)
		}

		fatalfCall.Resolve()
	}()

	// Now call GetCallOrdered for callA - should fail-fast on queued mismatch
	ctrl.GetCallOrdered(1*time.Second, func(call *testCall) error {
		if call.name != "callA" {
			return fmt.Errorf("expected callA, got %q", call.name)
		}

		return nil
	})
}

// TestGetCall_ConcurrentWaiters verifies that the waiter registration pattern
// correctly handles concurrent goroutines waiting for different calls.
//
// This test verifies that:
// 1. Multiple goroutines can wait concurrently
// 2. Each goroutine receives the correct call matching its validator
// 3. No calls are lost or delivered to the wrong waiter
// 4. Helper() is called but Fatalf() is NOT called (no timeout).
//
//nolint:varnamelen,funlen // Standard Go test parameter name; concurrent test requires setup
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

		result := ctrl.GetCall(1*time.Second, func(call *testCall) error {
			if call.name != "callA" {
				return fmt.Errorf("expected callA, got %q", call.name)
			}

			return nil
		})

		if result == callA {
			receivedA.Store(true)
		}
	}()

	// Goroutine waiting for callB
	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		result := ctrl.GetCall(1*time.Second, func(call *testCall) error {
			if call.name != "callB" {
				return fmt.Errorf("expected callB, got %q", call.name)
			}

			return nil
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
	result := ctrl.GetCall(1*time.Second, func(call *testCall) error {
		if call.name != "call2" {
			return fmt.Errorf("expected call2, got %q", call.name)
		}

		return nil
	})

	if result != call2 {
		t.Errorf("Expected to receive call2, got %v", result)
	}

	// call1 should still be in the queue
	result = ctrl.GetCall(1*time.Second, func(call *testCall) error {
		if call.name != "call1" {
			return fmt.Errorf("expected call1, got %q", call.name)
		}

		return nil
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

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
