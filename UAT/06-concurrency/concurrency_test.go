package concurrency_test

import (
	"testing"
	"time"

	concurrency "github.com/toejough/imptest/UAT/06-concurrency"
)

//go:generate go run ../../impgen/main.go concurrency.SlowService --name SlowServiceImp

func TestConcurrentOutOfOrder(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock.
	mock := NewSlowServiceImp(t)

	// resultChan will collect the results from the concurrent execution.
	resultChan := make(chan []string, 1)

	// Run the code under test. It will call DoA and DoB concurrently.
	go func() {
		resultChan <- concurrency.RunConcurrent(mock.Mock, 123)
	}()

	// imptest handles out-of-order calls via an internal queue.
	// We can expect DoA then DoB, even if the code calls them in reverse order.
	// The .Within() modifier tells imptest to wait up to the given duration for the call.

	// 1. Expect DoA(123) to be called within 1 second.
	mock.Within(time.Second).ExpectCallIs.DoA().ExpectArgsAre(123).InjectResult("Result A")

	// 2. Expect DoB(123) to be called within 1 second.
	mock.Within(time.Second).ExpectCallIs.DoB().ExpectArgsAre(123).InjectResult("Result B")

	// Wait for the code under test to finish and verify results.
	results := <-resultChan
	if results[0] != "Result A" || results[1] != "Result B" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestExplicitReversedExpectation(t *testing.T) {
	t.Parallel()

	mock := NewSlowServiceImp(t)
	resultChan := make(chan []string, 1)

	go func() {
		resultChan <- concurrency.RunConcurrent(mock.Mock, 456)
	}()

	// To demonstrate the power of the queueing mechanism, we can intentionally
	// expect the calls in the OPPOSITE order of how we think they might happen.
	// Here we wait for DoB first, then DoA.

	mock.Within(time.Second).ExpectCallIs.DoB().ExpectArgsAre(456).InjectResult("Result B")
	mock.Within(time.Second).ExpectCallIs.DoA().ExpectArgsAre(456).InjectResult("Result A")

	results := <-resultChan
	if results[0] != "Result A" || results[1] != "Result B" {
		t.Errorf("unexpected results: %v", results)
	}
}
