package concurrency_test

import (
	"testing"
	"time"

	concurrency "github.com/toejough/imptest/UAT/06-concurrency"
)

//go:generate go run ../../impgen/main.go concurrency.SlowService --name SlowServiceImp

// TestConcurrentOutOfOrder demonstrates how imptest handles code that executes
// dependencies in parallel or in non-deterministic order.
//
// Key Requirements Met:
//  1. Thread-Safe Expectations: The internal call queue allows expectations to be
//     defined in one order while the code under test calls them in another.
//  2. Timing Control: Use .Within(duration) to tell imptest to wait for a call,
//     preventing flaky tests in concurrent environments.
func TestConcurrentOutOfOrder(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock implementation.
	imp := NewSlowServiceImp(t)

	// resultChan will collect the results from the concurrent execution.
	resultChan := make(chan []string, 1)

	// Run the code under test. It will call DoA and DoB concurrently.
	go func() {
		resultChan <- concurrency.RunConcurrent(imp.Mock, 123)
	}()

	// Requirement: We can expect DoA then DoB, even if the code calls them in reverse order.
	// The .Within() modifier tells imptest to wait up to the given duration for the call.

	// 1. Expect DoA(123) to be called within 1 second.
	imp.Within(time.Second).ExpectCallIs.DoA().ExpectArgsAre(123).InjectResult("Result A")

	// 2. Expect DoB(123) to be called within 1 second.
	imp.Within(time.Second).ExpectCallIs.DoB().ExpectArgsAre(123).InjectResult("Result B")

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

	imp := NewSlowServiceImp(t)
	resultChan := make(chan []string, 1)

	go func() {
		resultChan <- concurrency.RunConcurrent(imp.Mock, 456)
	}()

	// Requirement: Demonstrate that we can wait for DoB first, then DoA,
	// regardless of which one the system-under-test triggers first.
	imp.Within(time.Second).ExpectCallIs.DoB().ExpectArgsAre(456).InjectResult("Result B")
	imp.Within(time.Second).ExpectCallIs.DoA().ExpectArgsAre(456).InjectResult("Result A")

	results := <-resultChan
	if results[0] != "Result A" || results[1] != "Result B" {
		t.Errorf("unexpected results: %v", results)
	}
}
