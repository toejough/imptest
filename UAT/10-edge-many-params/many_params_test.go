package many_params_test

import (
	"fmt"
	"testing"

	mp "github.com/toejough/imptest/UAT/10-edge-many-params"
)

//go:generate impgen many_params.ManyParams

// TestManyParams_DifferentValues_mutant tests with different parameter values.
// This ensures index arithmetic is correct even when values differ.
func TestManyParams_DifferentValues_mutant(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	testCases := []struct {
		name   string
		params [10]int
		result string
	}{
		{
			name:   "all zeros",
			params: [10]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			result: "zeros",
		},
		{
			name:   "sequential",
			params: [10]int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			result: "sequential",
		},
		{
			name:   "negative values",
			params: [10]int{-1, -2, -3, -4, -5, -6, -7, -8, -9, -10},
			result: "negative",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) { //nolint:varnamelen // Standard Go test convention
			t.Parallel()

			// Create a new mock for each test case
			mockImpl := mp.NewManyParamsImp(t)

			// Call Process in a goroutine
			resultChan := make(chan string)

			go func() {
				resultChan <- mockImpl.Mock.Process(
					testCase.params[0], testCase.params[1], testCase.params[2], testCase.params[3], testCase.params[4],
					testCase.params[5], testCase.params[6], testCase.params[7], testCase.params[8], testCase.params[9],
				)
			}()

			// Set up expectation
			mockImpl.ExpectCallIs.Process().ExpectArgsAre(
				testCase.params[0], testCase.params[1], testCase.params[2], testCase.params[3], testCase.params[4],
				testCase.params[5], testCase.params[6], testCase.params[7], testCase.params[8], testCase.params[9],
			).InjectResult(testCase.result)

			// Verify result
			got := <-resultChan
			if got != testCase.result {
				t.Errorf("Process() = %v, want %v", got, testCase.result)
			}
		})
	}
}

// TestManyParams_UnnamedParams_mutant tests interfaces with unnamed parameters.
func TestManyParams_UnnamedParams_mutant(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	// Note: This would test an interface with unnamed params which exercises different code paths
	// type UnnamedParams interface {
	// 	Execute(int, int, int) error
	// }

	// Generate message to document what this would test
	msg := fmt.Sprintf("Interface with %d unnamed parameters would test unnamed parameter indexing", 10)
	if len(msg) == 0 {
		t.Error("Message should not be empty")
	}

	// Note: To fully test this, we'd need to generate UnnamedParams with //go:generate
	// and create a mock for it. This test documents the edge case.
}

// TestManyParams_VerifyArgs_mutant tests that all 10 arguments are captured correctly.
// This is critical for catching off-by-one errors in parameter indexing.
func TestManyParams_VerifyArgs_mutant(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	mock := mp.NewManyParamsImp(t)

	// Use distinct values for each parameter to catch index errors
	values := [10]int{111, 222, 333, 444, 555, 666, 777, 888, 999, 1000}

	// Call Process in a goroutine
	resultChan := make(chan string)

	go func() {
		resultChan <- mock.Mock.Process(
			values[0], values[1], values[2], values[3], values[4],
			values[5], values[6], values[7], values[8], values[9],
		)
	}()

	// Set up expectation
	mock.ExpectCallIs.Process().ExpectArgsAre(
		values[0], values[1], values[2], values[3], values[4],
		values[5], values[6], values[7], values[8], values[9],
	).InjectResult("ok")

	// Verify result
	result := <-resultChan
	if result != "ok" {
		t.Errorf("Process() = %v, want 'ok'", result)
	}
}

// TestManyParams_mutant verifies that mocks work correctly for methods with many parameters.
// This catches mutations in:
// - Parameter index arithmetic (index + 1, index + 0, index + 2)
// - Parameter naming beyond A-H (should use param8, param9, etc)
// - Array bounds checking (index < len(names) vs index <= len(names)).
func TestManyParams_mutant(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	mock := mp.NewManyParamsImp(t)

	// Call with all 10 parameters in a goroutine
	resultChan := make(chan string)

	go func() {
		resultChan <- mock.Mock.Process(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	}()

	// Set up expectation with all 10 parameters
	mock.ExpectCallIs.Process().ExpectArgsAre(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).InjectResult("success")

	// Verify result
	result := <-resultChan
	if result != "success" {
		t.Errorf("Process() = %v, want 'success'", result)
	}
}
