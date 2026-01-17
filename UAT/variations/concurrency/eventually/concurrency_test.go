package concurrency_test

import (
	"testing"
	"time"

	"github.com/toejough/imptest"
	concurrency "github.com/toejough/imptest/UAT/variations/concurrency/eventually"
)

// TestAsyncEventuallyPanic tests Panic in async mode.
func TestAsyncEventuallyPanic(t *testing.T) {
	t.Parallel()

	mock, imp := MockSlowService(t)

	// Register expectation with panic FIRST (async pattern)
	imp.Eventually.DoA.ArgsEqual(999).Panic("test panic")

	// Capture panic from goroutine
	panicChan := make(chan any, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicChan <- r
			}
		}()

		_ = mock.DoA(999)
	}()

	// Wait for all expectations to be satisfied
	imptest.Wait(t)

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
//  3. Callback pattern: Return can be called before call arrives
func TestAsyncEventuallyWithControllerWait(t *testing.T) {
	t.Parallel()

	mock, imp := MockSlowService(t)

	// Register expectations FIRST - these must NOT block (new async behavior)
	// With blocking Eventually(), this would deadlock because no goroutine is making calls yet
	imp.Eventually.DoA.ArgsEqual(789).Return("Async A")
	imp.Eventually.DoB.ArgsEqual(789).Return("Async B")

	// NOW start code under test - the expectations are already registered
	go func() {
		_ = concurrency.RunConcurrent(mock, 789)
	}()

	// imptest.Wait() blocks until all expectations are satisfied
	imptest.Wait(t)
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
	mock, imp := MockSlowService(t)

	// resultChan will collect the results from the concurrent execution.
	resultChan := make(chan []string, 1)

	// Run the code under test. It will call DoA and DoB concurrently.
	go func() {
		resultChan <- concurrency.RunConcurrent(mock, 123)
	}()

	// Requirement: We can expect DoA then DoB, even if the code calls them in reverse order.
	// The .Eventually modifier tells imptest to wait indefinitely for the call.

	// 1. Expect DoA(123) to be called.
	imp.Eventually.DoA.ArgsEqual(123).Return("Result A")

	// 2. Expect DoB(123) to be called.
	imp.Eventually.DoB.ArgsEqual(123).Return("Result B")

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

	mock, imp := MockSlowService(t)
	resultChan := make(chan []string, 1)

	go func() {
		resultChan <- concurrency.RunConcurrent(mock, 456)
	}()

	// Requirement: Demonstrate that we can wait for DoB first, then DoA,
	// regardless of which one the system-under-test triggers first.
	imp.Eventually.DoB.ArgsEqual(456).Return("Result B")
	imp.Eventually.DoA.ArgsEqual(456).Return("Result A")

	results := <-resultChan
	if results[0] != "Result A" || results[1] != "Result B" {
		t.Errorf("unexpected results: %v", results)
	}
}

// TestSetTimeoutAPI verifies SetTimeout can be called via imptest.SetTimeout.
func TestSetTimeoutAPI(t *testing.T) {
	t.Parallel()

	mock, imp := MockSlowService(t)

	// SetTimeout is callable at package level
	imptest.SetTimeout(t, 5*time.Second)

	go func() {
		_ = mock.DoA(1)
	}()

	imp.Eventually.DoA.ArgsEqual(1).Return("ok")
	imptest.Wait(t)
}
