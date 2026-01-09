package core_test

// This file contains regression tests for data races in the mockTester pattern.

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toejough/imptest"
)

// TestProperSynchronization_AtomicBased demonstrates the CORRECT pattern
// using atomic operations and sync.WaitGroup.
//
// SOLUTION: Use atomic operations for the boolean flag and a sync.WaitGroup
// or mutex for the string message.
func TestProperSynchronization_AtomicBased(t *testing.T) {
	t.Parallel()

	// CORRECT: Use atomic for boolean, mutex for string
	var fatalfCalled atomic.Bool

	var msgMu sync.Mutex

	var fatalfMsg string

	// Use the simple mockTester that captures format+args properly
	mockT := &mockTester{
		helper: func() {},
		fatalf: func(format string, args ...any) {
			fatalfCalled.Store(true) // Atomic write

			msgMu.Lock()

			fatalfMsg = fmt.Sprintf(format, args...)

			msgMu.Unlock()
		},
	}

	ctrl := imptest.NewController[*testCall](mockT)
	callB := &testCall{name: "CallB"}

	// Explicit method calls to prevent deadcode removal (required for interface impl)
	_ = callB.Name()
	_ = callB.Done()

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

	if !strings.Contains(msg, "CallA") || !strings.Contains(msg, "CallB") {
		t.Errorf("expected message to mention both calls, got: %s", msg)
	}
}

// unexported variables.
var (
	_ imptest.Call = (*testCall)(nil)
)

// mockTester is a simple mock for demonstrating race conditions in these regression tests.
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

// testCall is a test type for Controller tests.
// Implements imptest.Call interface.
type testCall struct {
	name string
	done atomic.Bool
}

func (c *testCall) Done() bool {
	return c.done.Load()
}

func (c *testCall) Name() string {
	return c.name
}
