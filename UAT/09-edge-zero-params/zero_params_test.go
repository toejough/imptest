package zero_params_test

import (
	"errors"
	"testing"

	zp "github.com/toejough/imptest/UAT/09-edge-zero-params"
)

// TestNoParams_mutant verifies that mocks work correctly for interfaces with zero-parameter methods.
// This catches mutations in parameter counting and nil parameter list handling.
func TestNoParams_mutant(t *testing.T) {
	t.Parallel()

	mock := zp.NewNoParamsImp(t)

	// Test Get() with no parameters
	resultChan := make(chan int)

	go func() {
		resultChan <- mock.Mock.Get()
	}()

	mock.ExpectCallIs.Get().InjectResult(42)

	result := <-resultChan
	if result != 42 {
		t.Errorf("Get() = %v, want 42", result)
	}

	// Test Execute() with no parameters
	expectedErr := errors.New("test error")
	errChan := make(chan error)

	go func() {
		errChan <- mock.Mock.Execute()
	}()

	mock.ExpectCallIs.Execute().InjectResult(expectedErr)

	err := <-errChan
	if !errors.Is(err, expectedErr) {
		t.Errorf("Execute() = %v, want %v", err, expectedErr)
	}
}

// TestNoParams_MultipleCalls_mutant verifies multiple calls work correctly.
func TestNoParams_MultipleCalls_mutant(t *testing.T) {
	t.Parallel()

	mock := zp.NewNoParamsImp(t)

	// Call Get() three times in a goroutine
	results := make(chan int, 3)

	go func() {
		results <- mock.Mock.Get()

		results <- mock.Mock.Get()

		results <- mock.Mock.Get()
	}()

	// Provide responses for each call
	mock.ExpectCallIs.Get().InjectResult(1)
	mock.ExpectCallIs.Get().InjectResult(2)
	mock.ExpectCallIs.Get().InjectResult(3)

	// Verify results
	if got := <-results; got != 1 {
		t.Errorf("First call: got %v, want 1", got)
	}

	if got := <-results; got != 2 {
		t.Errorf("Second call: got %v, want 2", got)
	}

	if got := <-results; got != 3 {
		t.Errorf("Third call: got %v, want 3", got)
	}
}
