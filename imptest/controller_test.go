package imptest_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toejough/imptest/imptest"
)

// testCall implements the Call interface for testing.
type testCall struct {
	name string
	done atomic.Bool
}

func (c *testCall) Name() string {
	return c.name
}

func (c *testCall) Done() bool {
	return c.done.Load()
}

func (c *testCall) MarkDone() {
	c.done.Store(true)
}

// mockTester implements the Tester interface for testing.
type mockTester struct {
	t       *testing.T
	fataled atomic.Bool
}

func (m *mockTester) Helper() {}

func (m *mockTester) Fatalf(format string, args ...any) {
	m.fataled.Store(true)
	m.t.Logf("Fatalf called: "+format, args...)
}

func (m *mockTester) DidFatal() bool {
	return m.fataled.Load()
}

// raceCoordinator ensures both goroutines enter select, then each wants what the other queued.
type raceCoordinator struct {
	mu        sync.Mutex
	alphaID   int  // 0=none, 1=A is Alpha, 2=B is Alpha
	betaReady chan struct{}
	callForA  *testCall
	callForB  *testCall
	aCalled   bool // Has A's validator been called once?
	bCalled   bool // Has B's validator been called once?
}

func (rc *raceCoordinator) ValidatorA(call *testCall) bool {
	rc.mu.Lock()

	// Determine role (happens once)
	if rc.alphaID == 0 {
		// A is Alpha (first validator called)
		rc.alphaID = 1
		rc.mu.Unlock()

		// Block until Beta is also in select
		<-rc.betaReady

		rc.mu.Lock()
	} else if rc.alphaID == 2 {
		// B is Alpha, A is Beta
		rc.mu.Unlock()

		// Signal Beta is ready
		select {
		case rc.betaReady <- struct{}{}:
		default:
		}

		rc.mu.Lock()
	}

	// First call: reject it (queue it for the other goroutine)
	if !rc.aCalled {
		rc.aCalled = true
		rc.mu.Unlock()

		return false // Reject to queue
	}

	// Subsequent calls (queue re-check): accept if it's what we want
	var wantCall *testCall

	if rc.alphaID == 1 {
		// A is Alpha: wants callB (what Beta will queue)
		wantCall = rc.callForB
	} else {
		// A is Beta: wants callA (what Alpha will queue)
		wantCall = rc.callForA
	}

	shouldAccept := call == wantCall
	rc.mu.Unlock()

	return shouldAccept
}

func (rc *raceCoordinator) ValidatorB(call *testCall) bool {
	rc.mu.Lock()

	// Determine role (happens once)
	if rc.alphaID == 0 {
		// B is Alpha (first validator called)
		rc.alphaID = 2
		rc.mu.Unlock()

		// Block until Beta is also in select
		<-rc.betaReady

		rc.mu.Lock()
	} else if rc.alphaID == 1 {
		// A is Alpha, B is Beta
		rc.mu.Unlock()

		// Signal Beta is ready
		select {
		case rc.betaReady <- struct{}{}:
		default:
		}

		rc.mu.Lock()
	}

	// First call: reject it (queue it for the other goroutine)
	if !rc.bCalled {
		rc.bCalled = true
		rc.mu.Unlock()

		return false // Reject to queue
	}

	// Subsequent calls (queue re-check): accept if it's what we want
	var wantCall *testCall

	if rc.alphaID == 2 {
		// B is Alpha: wants callA (what Beta will queue)
		wantCall = rc.callForA
	} else {
		// B is Beta: wants callB (what Alpha will queue)
		wantCall = rc.callForB
	}

	shouldAccept := call == wantCall
	rc.mu.Unlock()

	return shouldAccept
}

// TestGetCall_RaceCondition demonstrates the race condition in GetCall where
// a goroutine can be stuck waiting on CallChan while its matching call sits
// in the queue.
//
// The bug: After unlocking queueLock (controller.go:55) and before entering
// select (controller.go:65), another goroutine can receive from CallChan,
// reject the call, and add it to the queue. The first goroutine never
// rechecks the queue.
//
// This test FAILS when the bug exists (goroutines timeout) and PASSES when
// the bug is fixed (goroutines successfully find queued calls).
func TestGetCall_RaceCondition(t *testing.T) {
	t.Parallel()

	tester := &mockTester{t: t}
	ctrl := imptest.NewController[*testCall](tester)

	// Create the coordinator and the two calls
	callForA := &testCall{name: "callA"}
	callForB := &testCall{name: "callB"}

	coordinator := &raceCoordinator{
		betaReady: make(chan struct{}),
		callForA:  callForA,
		callForB:  callForB,
	}

	var wg sync.WaitGroup
	var gotA, gotB atomic.Bool

	// Goroutine A
	wg.Add(1)

	go func() {
		defer wg.Done()

		result := ctrl.GetCall(100*time.Millisecond, coordinator.ValidatorA)

		if result != nil {
			gotA.Store(true)
		}
	}()

	// Goroutine B
	wg.Add(1)

	go func() {
		defer wg.Done()

		result := ctrl.GetCall(100*time.Millisecond, coordinator.ValidatorB)

		if result != nil {
			gotB.Store(true)
		}
	}()

	// Send both calls after a small delay to ensure both goroutines are waiting
	go func() {
		time.Sleep(10 * time.Millisecond)

		// Send callA - Alpha will receive it (whichever goroutine goes first)
		ctrl.CallChan <- callForA

		// Send callB - Beta will receive it
		ctrl.CallChan <- callForB
	}()

	wg.Wait()

	// Test expectations:
	// - With bug: both goroutines timeout (didFatal=true, gotA=false, gotB=false)
	// - With fix: both goroutines succeed (didFatal=false, gotA=true, gotB=true)

	t.Logf("Test results: didFatal=%v, gotA=%v, gotB=%v", tester.DidFatal(), gotA.Load(), gotB.Load())
	t.Logf("Coordinator state: alphaID=%d, aCalled=%v, bCalled=%v",
		coordinator.alphaID, coordinator.aCalled, coordinator.bCalled)

	if tester.DidFatal() {
		// Goroutines timed out - bug exists
		t.Fatal("Race condition detected: goroutines timed out while their matching calls sat in queue. " +
			"This indicates GetCall does not re-check the queue after receiving a non-matching call.")
	}

	if !gotA.Load() || !gotB.Load() {
		t.Errorf("Expected both goroutines to succeed, but gotA=%v, gotB=%v", gotA.Load(), gotB.Load())
	}
}
