package orderedvsmode_test

import (
	"errors"
	"testing"

	"github.com/toejough/imptest"
	// Import for impgen to resolve the package.
	_ "github.com/toejough/imptest/UAT/variations/concurrency/ordered"
)

// NOTE: Testing ordered mode failure (fail-fast) requires MockTester setup.
// This is demonstrated in imptest/controller_test.go - see TestGetCallOrdered_FailsOnMismatch.

// TestEventually_CallsOutOfOrder verifies eventually mode succeeds with out-of-order calls.
// Operations are launched concurrently so they can arrive in any order.
func TestEventually_CallsOutOfOrder(t *testing.T) {
	t.Parallel()

	mock, imp := MockService(t)

	// Launch operations in separate goroutines so they don't block each other
	// They'll arrive out of order: B, then A, then C
	done := make(chan bool, 3)

	go func() {
		_ = mock.OperationB(2)

		done <- true
	}()

	go func() {
		_ = mock.OperationA(1)

		done <- true
	}()

	go func() {
		_ = mock.OperationC(3)

		done <- true
	}()

	// Use Eventually() to handle out-of-order calls
	// Eventually mode queues mismatches and waits for matches
	imp.OperationA.Eventually.Expect(1).Return(nil)
	imp.OperationB.Eventually.Expect(2).Return(nil)
	imp.OperationC.Eventually.Expect(3).Return(nil)

	// Wait for all goroutines
	<-done
	<-done
	<-done
}

// TestEventually_ConcurrentCalls verifies eventually mode handles truly concurrent calls.
func TestEventually_ConcurrentCalls(t *testing.T) {
	t.Parallel()

	mock, imp := MockService(t)

	// Launch three concurrent goroutines calling operations simultaneously
	done := make(chan bool, 3)

	go func() {
		_ = mock.OperationC(3)

		done <- true
	}()

	go func() {
		_ = mock.OperationA(1)

		done <- true
	}()

	go func() {
		_ = mock.OperationB(2)

		done <- true
	}()

	// Eventually mode handles any arrival order
	imp.OperationA.Eventually.Expect(1).Return(nil)
	imp.OperationB.Eventually.Expect(2).Return(nil)
	imp.OperationC.Eventually.Expect(3).Return(nil)

	// Wait for all goroutines
	<-done
	<-done
	<-done
}

// TestEventually_PreservesTypeSafety verifies Eventually() returns typed wrapper.
func TestEventually_PreservesTypeSafety(t *testing.T) {
	t.Parallel()

	mock, imp := MockService(t)

	go func() {
		_ = mock.OperationA(42)
	}()

	// Eventually() returns *ServiceMockOperationAMethod, not *DependencyMethod
	call := imp.OperationA.Eventually.Expect(42)

	// Type-safe GetArgs() access
	args := call.GetArgs()

	if args.Id != 42 {
		t.Errorf("expected Id=42, got %v", args.Id)
	}

	call.Return(errors.New("test error"))
}

// TestMixed_OrderedAndEventually verifies mixing ordered and eventually expectations.
// This test uses a sequential call pattern where ordered calls happen first,
// then concurrent eventually calls can arrive in any order.
func TestMixed_OrderedAndEventually(t *testing.T) {
	t.Parallel()

	mock, imp := MockService(t)

	go func() {
		// First call is ordered - must match expectation order
		_ = mock.OperationA(1)
		// Remaining calls can arrive in any order via Eventually
		_ = mock.OperationC(3)
		_ = mock.OperationB(2)
	}()

	// First expectation is ordered (must be matched in order)
	imp.OperationA.Expect(1).Return(nil)

	// Remaining expectations use Eventually (can match in any order)
	imp.OperationB.Eventually.Expect(2).Return(nil)
	imp.OperationC.Eventually.Expect(3).Return(nil)

	// Wait for all Eventually expectations to be satisfied
	imptest.Wait(t)
}

//go:generate impgen orderedvsmode.Service --dependency

// TestOrdered_CallsInOrder verifies ordered mode succeeds when calls arrive sequentially.
func TestOrdered_CallsInOrder(t *testing.T) {
	t.Parallel()

	mock, imp := MockService(t)

	// Start goroutine that calls operations in order: A, B, C
	done := make(chan bool)

	go func() {
		_ = mock.OperationA(1)
		_ = mock.OperationB(2)
		_ = mock.OperationC(3)

		done <- true
	}()

	// Expect calls in order (ordered mode = default)
	imp.OperationA.Expect(1).Return(nil)
	imp.OperationB.Expect(2).Return(nil)
	imp.OperationC.Expect(3).Return(nil)

	<-done
}
