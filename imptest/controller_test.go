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

// raceCoordinator ensures B only accepts calls A has rejected (except the first).
type raceCoordinator struct {
	mu             sync.Mutex
	aRejectedCalls map[*testCall]bool
	firstARejected *testCall
}

func (rc *raceCoordinator) ValidatorA(call *testCall) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.aRejectedCalls == nil {
		rc.aRejectedCalls = make(map[*testCall]bool)
	}

	if rc.firstARejected == nil {
		rc.firstARejected = call // Track first rejection
	}

	rc.aRejectedCalls[call] = true

	return false // Always reject → call goes to queue
}

func (rc *raceCoordinator) ValidatorB(call *testCall) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// B accepts calls A rejected, EXCEPT the first one
	// This ensures B enters select before accepting anything
	if rc.aRejectedCalls[call] && call != rc.firstARejected {
		return true // Accept - but this call is in queue!
	}

	return false // Reject - either A hasn't rejected, or it's the first
}

// TestGetCall_RaceCondition demonstrates the race condition in GetCall where
// a goroutine can be stuck waiting on CallChan while its matching call sits
// in the queue.
//
// The bug: After unlocking queueLock (controller.go:55) and before entering
// select (controller.go:65), another goroutine can receive from CallChan,
// reject the call, and add it to the queue. The first goroutine never
// rechecks the queue.
func TestGetCall_RaceCondition(t *testing.T) {
	t.Parallel()

	tester := &mockTester{t: t}
	ctrl := imptest.NewController[*testCall](tester)
	coordinator := &raceCoordinator{}

	var wg sync.WaitGroup

	// Goroutine A: always rejects, tracking all rejections
	wg.Add(1)

	go func() {
		defer wg.Done()

		// A will timeout because it always rejects
		_ = ctrl.GetCall(100*time.Millisecond, coordinator.ValidatorA)
	}()

	// Goroutine B: accepts calls A rejected (except the first)
	wg.Add(1)

	go func() {
		defer wg.Done()

		// B will timeout waiting for A's second reject (which is in queue)
		_ = ctrl.GetCall(100*time.Millisecond, coordinator.ValidatorB)
	}()

	// Send calls to trigger the race condition
	// We need at least 2 calls so A has multiple rejects
	go func() {
		// Small delay to let goroutines enter GetCall
		time.Sleep(10 * time.Millisecond)

		// Send first call - A will reject this (becomes firstARejected)
		ctrl.CallChan <- &testCall{name: "call1"}

		// Small delay
		time.Sleep(10 * time.Millisecond)

		// Send second call - A will reject this too
		// B would accept this, but it will be in the queue
		ctrl.CallChan <- &testCall{name: "call2"}

		// Send third call for good measure
		time.Sleep(10 * time.Millisecond)
		ctrl.CallChan <- &testCall{name: "call3"}
	}()

	wg.Wait()

	// Verify that the race condition was triggered (goroutines timed out)
	if !tester.DidFatal() {
		t.Error("Expected timeout to occur, demonstrating the race condition, but test passed. " +
			"This might indicate the race condition was not triggered, or the bug has been fixed.")
	} else {
		t.Log("SUCCESS: Race condition demonstrated - goroutines timed out while matching calls sat in queue")
	}
}
