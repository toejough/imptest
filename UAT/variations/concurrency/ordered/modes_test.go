package orderedvsmode_test

import (
	"errors"
	"testing"
)

// NOTE: Testing ordered mode failure (fail-fast) requires MockTester setup.
// This is demonstrated in imptest/controller_test.go - see TestGetCallOrdered_FailsOnMismatch.

// TestEventually_CallsOutOfOrder verifies eventually mode succeeds with out-of-order calls.
// Operations are launched concurrently so they can arrive in any order.
func TestEventually_CallsOutOfOrder(t *testing.T) {
	t.Parallel()

	mock := MockService(t)

	// Launch operations in separate goroutines so they don't block each other
	// They'll arrive out of order: B, then A, then C
	done := make(chan bool, 3)

	go func() {
		_ = mock.Mock.OperationB(2)

		done <- true
	}()

	go func() {
		_ = mock.Mock.OperationA(1)

		done <- true
	}()

	go func() {
		_ = mock.Mock.OperationC(3)

		done <- true
	}()

	// Use Eventually() to handle out-of-order calls
	// Eventually mode queues mismatches and waits for matches
	mock.Method.OperationA.Eventually().ExpectCalledWithExactly(1).InjectReturnValues(nil)
	mock.Method.OperationB.Eventually().ExpectCalledWithExactly(2).InjectReturnValues(nil)
	mock.Method.OperationC.Eventually().ExpectCalledWithExactly(3).InjectReturnValues(nil)

	// Wait for all goroutines
	<-done
	<-done
	<-done
}

// TestEventually_ConcurrentCalls verifies eventually mode handles truly concurrent calls.
func TestEventually_ConcurrentCalls(t *testing.T) {
	t.Parallel()

	mock := MockService(t)

	// Launch three concurrent goroutines calling operations simultaneously
	done := make(chan bool, 3)

	go func() {
		_ = mock.Mock.OperationC(3)

		done <- true
	}()

	go func() {
		_ = mock.Mock.OperationA(1)

		done <- true
	}()

	go func() {
		_ = mock.Mock.OperationB(2)

		done <- true
	}()

	// Eventually mode handles any arrival order
	mock.Method.OperationA.Eventually().ExpectCalledWithExactly(1).InjectReturnValues(nil)
	mock.Method.OperationB.Eventually().ExpectCalledWithExactly(2).InjectReturnValues(nil)
	mock.Method.OperationC.Eventually().ExpectCalledWithExactly(3).InjectReturnValues(nil)

	// Wait for all goroutines
	<-done
	<-done
	<-done
}

// TestEventually_PreservesTypeSafety verifies Eventually() returns typed wrapper.
func TestEventually_PreservesTypeSafety(t *testing.T) {
	t.Parallel()

	mock := MockService(t)

	go func() {
		_ = mock.Mock.OperationA(42)
	}()

	// Eventually() returns *ServiceMockOperationAMethod, not *DependencyMethod
	call := mock.Method.OperationA.Eventually().ExpectCalledWithExactly(42)

	// Type-safe GetArgs() access
	args := call.GetArgs()

	if args.Id != 42 {
		t.Errorf("expected Id=42, got %v", args.Id)
	}

	call.InjectReturnValues(errors.New("test error"))
}

// TestMixed_OrderedAndEventually verifies mixing ordered and eventually expectations.
func TestMixed_OrderedAndEventually(t *testing.T) {
	t.Parallel()

	mock := MockService(t)

	// Launch concurrent operations
	done := make(chan bool, 3)

	go func() {
		_ = mock.Mock.OperationB(2)

		done <- true
	}()

	go func() {
		_ = mock.Mock.OperationC(3)

		done <- true
	}()

	go func() {
		_ = mock.Mock.OperationA(1)

		done <- true
	}()

	// Mix ordered and eventually:
	// - A with eventually (can arrive in any order)
	// - B with eventually (can arrive in any order)
	// - C ordered (but since we already got all calls, it will work)
	mock.Method.OperationA.Eventually().ExpectCalledWithExactly(1).InjectReturnValues(nil)
	mock.Method.OperationB.Eventually().ExpectCalledWithExactly(2).InjectReturnValues(nil)

	// By the time we get here, C has already arrived and been queued
	// Ordered mode will find it in the queue
	mock.Method.OperationC.ExpectCalledWithExactly(3).InjectReturnValues(nil)

	<-done
	<-done
	<-done
}

//go:generate impgen orderedvsmode.Service --dependency

// TestOrdered_CallsInOrder verifies ordered mode succeeds when calls arrive sequentially.
func TestOrdered_CallsInOrder(t *testing.T) {
	t.Parallel()

	mock := MockService(t)

	// Start goroutine that calls operations in order: A, B, C
	done := make(chan bool)

	go func() {
		svc := mock.Mock
		_ = svc.OperationA(1)
		_ = svc.OperationB(2)
		_ = svc.OperationC(3)

		done <- true
	}()

	// Expect calls in order (ordered mode = default)
	mock.Method.OperationA.ExpectCalledWithExactly(1).InjectReturnValues(nil)
	mock.Method.OperationB.ExpectCalledWithExactly(2).InjectReturnValues(nil)
	mock.Method.OperationC.ExpectCalledWithExactly(3).InjectReturnValues(nil)

	<-done
}
