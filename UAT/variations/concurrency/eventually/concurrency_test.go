package concurrency_test

import (
	"testing"

	concurrency "github.com/toejough/imptest/UAT/variations/concurrency/eventually"
)

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
		resultChan <- concurrency.RunConcurrent(mock.Interface(), 123)
	}()

	// Requirement: We can expect DoA then DoB, even if the code calls them in reverse order.
	// The .Eventually() modifier tells imptest to wait indefinitely for the call.

	// 1. Expect DoA(123) to be called.
	mock.DoA.Eventually().ExpectCalledWithExactly(123).InjectReturnValues("Result A")

	// 2. Expect DoB(123) to be called.
	mock.DoB.Eventually().ExpectCalledWithExactly(123).InjectReturnValues("Result B")

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
		resultChan <- concurrency.RunConcurrent(mock.Interface(), 456)
	}()

	// Requirement: Demonstrate that we can wait for DoB first, then DoA,
	// regardless of which one the system-under-test triggers first.
	mock.DoB.Eventually().ExpectCalledWithExactly(456).InjectReturnValues("Result B")
	mock.DoA.Eventually().ExpectCalledWithExactly(456).InjectReturnValues("Result A")

	results := <-resultChan
	if results[0] != "Result A" || results[1] != "Result B" {
		t.Errorf("unexpected results: %v", results)
	}
}
