package concurrency_test

import (
	"testing"
	"time"

	concurrency "github.com/toejough/imptest/UAT/variations/concurrency/eventually"
)

// TestAsyncEventuallyInjectPanicValue tests InjectPanicValue in async mode.
func TestAsyncEventuallyInjectPanicValue(t *testing.T) {
	t.Parallel()

	mock := MockSlowService(t)

	// Register expectation with panic FIRST (async pattern)
	mock.Method.DoA.Eventually().ExpectCalledWithExactly(999).InjectPanicValue("test panic")

	// Capture panic from goroutine
	panicChan := make(chan any, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicChan <- r
			}
		}()

		_ = mock.Mock.DoA(999)
	}()

	// Wait for all expectations to be satisfied
	mock.Controller.Wait()

	// Wait for panic to be received
	p := <-panicChan
	if p != "test panic" {
		t.Errorf("expected panic 'test panic', got %v", p)
	}
}

// TestAsyncEventuallyWithControllerWait demonstrates the new async Eventually() API.
//
// This test REQUIRES non-blocking Eventually() to work:
// - Expectations are registered BEFORE starting the code under test
// - With blocking Eventually(), this would deadlock (nothing to receive the call)
// - With async Eventually(), expectations register immediately and Wait() blocks
//
// Key Requirements Met:
//  1. Non-blocking Eventually(): Expectations register immediately without blocking
//  2. Controller.Wait(): Single call blocks until all expectations are satisfied
//  3. Callback pattern: InjectReturnValues can be called before call arrives
func TestAsyncEventuallyWithControllerWait(t *testing.T) {
	t.Parallel()

	mock := MockSlowService(t)

	// Register expectations FIRST - these must NOT block (new async behavior)
	// With blocking Eventually(), this would deadlock because no goroutine is making calls yet
	mock.Method.DoA.Eventually().ExpectCalledWithExactly(789).InjectReturnValues("Async A")
	mock.Method.DoB.Eventually().ExpectCalledWithExactly(789).InjectReturnValues("Async B")

	// NOW start code under test - the expectations are already registered
	go func() {
		_ = concurrency.RunConcurrent(mock.Mock, 789)
	}()

	// Controller.Wait() blocks until all expectations are satisfied
	mock.Controller.Wait()
}

//go:generate impgen concurrency.SlowService --dependency

// TestConcurrentOutOfOrder demonstrates how imptest handles code that executes
// dependencies in parallel or in non-deterministic order.
//
// Key Requirements Met:
//  1. Thread-Safe Expectations: The internal call queue allows expectations to be
//     defined in one order while the code under test calls them in another.
//  2. Timing Control: Use .Eventually(duration) to tell imptest to wait for a call,
//     preventing flaky tests in concurrent environments.
func TestConcurrentOutOfOrder(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock implementation.
	mock := MockSlowService(t)

	// resultChan will collect the results from the concurrent execution.
	resultChan := make(chan []string, 1)

	// Run the code under test. It will call DoA and DoB concurrently.
	go func() {
		resultChan <- concurrency.RunConcurrent(mock.Mock, 123)
	}()

	// Requirement: We can expect DoA then DoB, even if the code calls them in reverse order.
	// The .Eventually() modifier tells imptest to wait indefinitely for the call.

	// 1. Expect DoA(123) to be called.
	mock.Method.DoA.Eventually().ExpectCalledWithExactly(123).InjectReturnValues("Result A")

	// 2. Expect DoB(123) to be called.
	mock.Method.DoB.Eventually().ExpectCalledWithExactly(123).InjectReturnValues("Result B")

	// Wait for the code under test to finish and verify results.
	results := <-resultChan
	if results[0] != "Result A" || results[1] != "Result B" {
		t.Errorf("unexpected results: %v", results)
	}
}

// TestExplicitReversedExpectation intentionally defines expectations in the
// opposite order of their execution to prove the power of the queueing mechanism.
//
// Key Requirements Met:
//  1. Order Independence: Tests remain robust even when dependency call order
//     is not guaranteed or changes due to implementation details.
func TestExplicitReversedExpectation(t *testing.T) {
	t.Parallel()

	mock := MockSlowService(t)
	resultChan := make(chan []string, 1)

	go func() {
		resultChan <- concurrency.RunConcurrent(mock.Mock, 456)
	}()

	// Requirement: Demonstrate that we can wait for DoB first, then DoA,
	// regardless of which one the system-under-test triggers first.
	mock.Method.DoB.Eventually().ExpectCalledWithExactly(456).InjectReturnValues("Result B")
	mock.Method.DoA.Eventually().ExpectCalledWithExactly(456).InjectReturnValues("Result A")

	results := <-resultChan
	if results[0] != "Result A" || results[1] != "Result B" {
		t.Errorf("unexpected results: %v", results)
	}
}

// TestSetTimeoutAPI verifies SetTimeout can be called on the controller.
func TestSetTimeoutAPI(t *testing.T) {
	t.Parallel()

	mock := MockSlowService(t)

	// SetTimeout should be callable (currently a stub)
	mock.Controller.SetTimeout(5 * time.Second)

	go func() {
		_ = mock.Mock.DoA(1)
	}()

	mock.Method.DoA.Eventually().ExpectCalledWithExactly(1).InjectReturnValues("ok")
	mock.Controller.Wait()
}
