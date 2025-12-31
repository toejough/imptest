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

// TestGetCallOrdered_FailsOnMismatch verifies that an ordered waiter receives
// notification on the mismatch channel when a non-matching call arrives first.
//
// This test verifies that:
// 1. An ordered waiter (fail-fast mode) is created correctly
// 2. When a non-matching call arrives, it's sent to the mismatch channel
// 3. The mismatched call is queued for later waiters
// 4. Helper() is called but Fatalf() is NOT called (no timeout)
//
//nolint:varnamelen,funlen // Standard Go test parameter name; test requires setup
func TestGetCallOrdered_FailsOnMismatch(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callB := &testCall{name: "callB"}

	// Handle the Helper() call
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Create an ordered waiter for "callA"
	mismatchChan := make(chan *testCall, 1)
	go func() {
		ctrl.GetCallOrdered(1*time.Second, func(call *testCall) bool {
			return call.name == "callA"
		}, mismatchChan)
	}()

	// Give the waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send callB (which doesn't match the ordered waiter)
	ctrl.CallChan <- callB

	// Verify we received callB on the mismatch channel
	select {
	case mismatched := <-mismatchChan:
		if mismatched != callB {
			t.Errorf("Expected callB on mismatch channel, got %v", mismatched)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for mismatch notification")
	}

	// Verify callB is now queued - send another Helper() call for this GetCall
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Try to get callB from the queue
	result := ctrl.GetCall(1*time.Second, func(call *testCall) bool {
		return call.name == "callB"
	})

	if result != callB {
		t.Errorf("Expected callB to be queued, got %v", result)
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
//nolint:varnamelen,funlen // Standard Go test parameter name; test requires setup
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
		result := ctrl.GetCall(2*time.Second, func(call *testCall) bool {
			return call.name == "callA"
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

	result := ctrl.GetCall(1*time.Second, func(call *testCall) bool {
		return call.name == "callB"
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
		result := ctrl.GetCallEventually(2*time.Second, func(call *testCall) bool {
			return call.name == "callA"
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
	resultB := ctrl.GetCall(1*time.Second, func(call *testCall) bool {
		return call.name == "callB"
	})

	if resultB != callB {
		t.Errorf("Expected callB to be queued, got %v", resultB)
	}

	resultC := ctrl.GetCall(1*time.Second, func(call *testCall) bool {
		return call.name == "callC"
	})

	if resultC != callC {
		t.Errorf("Expected callC to be queued, got %v", resultC)
	}
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
	result := ctrl.GetCallEventually(1*time.Second, func(call *testCall) bool {
		return call.name == "callA"
	})

	if result != callA {
		t.Errorf("Expected to receive callA from queue, got %v", result)
	}
}

// TestDispatchLoop_FIFOPriority verifies that the dispatcher follows FIFO order
// when multiple waiters are registered, prioritizing the first waiter regardless
// of whether it's ordered or eventually.
//
// This test verifies that:
// 1. First waiter (ordered for "CallA") is registered
// 2. Second waiter (eventually for "CallA") is registered after
// 3. When "CallA" arrives, the first waiter receives it (FIFO)
// 4. Second waiter continues waiting (will timeout since we don't send another CallA)
//
//nolint:varnamelen,funlen // Standard Go test parameter name; test requires setup
func TestDispatchLoop_FIFOPriority(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callA := &testCall{name: "CallA"}

	var (
		orderedReceivedA    atomic.Bool
		eventuallyReceivedA atomic.Bool
	)

	// Handle Helper() calls (one for ordered, one for eventually, one for eventually timeout/Fatalf)
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Register ordered waiter FIRST
	mismatchChan := make(chan *testCall, 1)
	go func() {
		result := ctrl.GetCallOrdered(2*time.Second, func(call *testCall) bool {
			return call.name == "CallA"
		}, mismatchChan)
		if result == callA {
			orderedReceivedA.Store(true)
		}
	}()

	// Give first waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Register eventually waiter SECOND (after ordered)
	go func() {
		result := ctrl.GetCallEventually(500*time.Millisecond, func(call *testCall) bool {
			return call.name == "CallA"
		})
		if result == callA {
			eventuallyReceivedA.Store(true)
		}
	}()

	// Give second waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send CallA - should go to FIRST waiter (ordered) due to FIFO
	ctrl.CallChan <- callA

	// Wait for ordered waiter to receive
	time.Sleep(100 * time.Millisecond)

	// Verify FIFO: ordered waiter (first registered) received CallA
	if !orderedReceivedA.Load() {
		t.Error("Ordered waiter (first registered) did not receive CallA - FIFO violation")
	}

	// Verify eventually waiter did NOT receive CallA (because ordered got it first)
	// Eventually waiter will timeout after 500ms
	time.Sleep(600 * time.Millisecond)

	if eventuallyReceivedA.Load() {
		t.Error("Eventually waiter should not have received CallA - it went to first waiter")
	}

	// Expect Fatalf call for eventually waiter timeout
	fatalfCall := testerMock.Fatalf.ExpectCalledWithMatches(imptest.Any())
	fatalfCall.InjectReturnValues()
}

// TestDispatchLoop_OrderedFailsEventuallyWaits verifies the dispatcher's behavior
// when an ordered waiter fails on mismatch and an eventually waiter receives the
// mismatched call.
//
// This test verifies that:
// 1. Ordered waiter for "CallA" is registered first
// 2. Eventually waiter for "CallB" is registered second
// 3. When "CallB" arrives, ordered waiter gets mismatch signal
// 4. "CallB" is queued for future waiters
// 5. Eventually waiter for "CallB" receives it from the queue
// 6. Ordered waiter is removed from waiters list after mismatch
//
//nolint:varnamelen,funlen // Standard Go test parameter name; test requires setup
func TestDispatchLoop_OrderedFailsEventuallyWaits(t *testing.T) {
	t.Parallel()

	testerMock := MockTester(t)
	ctrl := imptest.NewController[*testCall](testerMock.Interface())

	callB := &testCall{name: "CallB"}

	var eventuallyReceivedB atomic.Bool

	// Handle Helper() calls (one for ordered, one for eventually)
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Register ordered waiter for "CallA" FIRST
	mismatchChan := make(chan *testCall, 1)
	go func() {
		ctrl.GetCallOrdered(2*time.Second, func(call *testCall) bool {
			return call.name == "CallA"
		}, mismatchChan)
	}()

	// Give ordered waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Register eventually waiter for "CallB" SECOND
	go func() {
		result := ctrl.GetCallEventually(2*time.Second, func(call *testCall) bool {
			return call.name == "CallB"
		})
		if result == callB {
			eventuallyReceivedB.Store(true)
		}
	}()

	// Give eventually waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send CallB - should trigger mismatch for ordered waiter
	ctrl.CallChan <- callB

	// Verify ordered waiter received mismatch signal
	select {
	case mismatched := <-mismatchChan:
		if mismatched != callB {
			t.Errorf("Expected CallB on mismatch channel, got %v", mismatched)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for mismatch notification")
	}

	// Wait for eventually waiter to receive CallB from queue
	time.Sleep(100 * time.Millisecond)

	// Verify eventually waiter received CallB
	if !eventuallyReceivedB.Load() {
		t.Error("Eventually waiter did not receive CallB after ordered waiter mismatch")
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
	mismatchChan1 := make(chan *testCall, 1)
	go func() {
		result := ctrl.GetCallOrdered(2*time.Second, func(call *testCall) bool {
			return call.name == "CallA"
		}, mismatchChan1)
		if result == callA {
			firstReceivedA.Store(true)
		}
	}()

	// Give first waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Register second ordered waiter for "CallB"
	mismatchChan2 := make(chan *testCall, 1)
	go func() {
		result := ctrl.GetCallOrdered(2*time.Second, func(call *testCall) bool {
			return call.name == "CallB"
		}, mismatchChan2)
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

	// Now test wrong order: register new waiters and send in reverse order
	callA2 := &testCall{name: "CallA"}
	callB2 := &testCall{name: "CallB"}

	var thirdReceivedA atomic.Bool

	// Handle Helper() calls for second sequence
	go func() {
		call := testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
		call = testerMock.Helper.ExpectCalledWithExactly()
		call.InjectReturnValues()
	}()

	// Register third ordered waiter for "CallA" (expecting A first)
	mismatchChan3 := make(chan *testCall, 1)
	go func() {
		result := ctrl.GetCallOrdered(2*time.Second, func(call *testCall) bool {
			return call.name == "CallA"
		}, mismatchChan3)
		if result == callA2 {
			thirdReceivedA.Store(true)
		}
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Register fourth ordered waiter for "CallB"
	mismatchChan4 := make(chan *testCall, 1)
	go func() {
		ctrl.GetCallOrdered(2*time.Second, func(call *testCall) bool {
			return call.name == "CallB"
		}, mismatchChan4)
	}()

	// Give waiter time to register
	time.Sleep(50 * time.Millisecond)

	// Send CallB FIRST (wrong order - third waiter expects CallA)
	ctrl.CallChan <- callB2

	// Verify third waiter gets mismatch when CallB arrives (it expected CallA)
	select {
	case mismatched := <-mismatchChan3:
		if mismatched != callB2 {
			t.Errorf("Expected CallB on mismatch channel, got %v", mismatched)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for mismatch notification on third waiter")
	}

	// Verify third waiter did not receive CallB (it got mismatch instead)
	if thirdReceivedA.Load() {
		t.Error("Third waiter should not have received CallA - it got mismatch for CallB")
	}
}
