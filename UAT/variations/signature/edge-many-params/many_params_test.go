package many_params_test

//go:generate impgen many_params.ManyParams --dependency

import (
	"testing"

	_ "github.com/toejough/imptest/UAT/variations/signature/edge-many-params"
)

// TestV2_ManyParams demonstrates v2 API for interfaces with many parameters.
func TestV2_ManyParams(t *testing.T) {
	t.Parallel()

	mock := MockManyParams(t)

	// Call with all 10 parameters in a goroutine
	resultChan := make(chan string)

	go func() {
		resultChan <- mock.Mock.Process(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	}()

	// Set up expectation with all 10 parameters
	mock.Method.Process.ExpectCalledWithExactly(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).
		InjectReturnValues("success")

	// Verify result
	result := <-resultChan
	if result != "success" {
		t.Errorf("Process() = %v, want 'success'", result)
	}
}

// TestV2_ManyParams_DifferentValues tests with different parameter values.
func TestV2_ManyParams_DifferentValues(t *testing.T) {
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
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mock := MockManyParams(t)

			// Call Process in a goroutine
			resultChan := make(chan string)

			go func() {
				resultChan <- mock.Mock.Process(
					testCase.params[0], testCase.params[1], testCase.params[2], testCase.params[3], testCase.params[4],
					testCase.params[5], testCase.params[6], testCase.params[7], testCase.params[8], testCase.params[9],
				)
			}()

			// Set up expectation
			mock.Method.Process.ExpectCalledWithExactly(
				testCase.params[0],
				testCase.params[1],
				testCase.params[2],
				testCase.params[3],
				testCase.params[4],
				testCase.params[5],
				testCase.params[6],
				testCase.params[7],
				testCase.params[8],
				testCase.params[9],
			).InjectReturnValues(testCase.result)

			// Verify result
			got := <-resultChan
			if got != testCase.result {
				t.Errorf("Process() = %v, want %v", got, testCase.result)
			}
		})
	}
}

// TestV2_ManyParams_VerifyArgs tests that all 10 arguments are captured correctly.
func TestV2_ManyParams_VerifyArgs(t *testing.T) {
	t.Parallel()

	mock := MockManyParams(t)

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
	mock.Method.Process.ExpectCalledWithExactly(
		values[0], values[1], values[2], values[3], values[4],
		values[5], values[6], values[7], values[8], values[9],
	).InjectReturnValues("ok")

	// Verify result
	result := <-resultChan
	if result != "ok" {
		t.Errorf("Process() = %v, want 'ok'", result)
	}
}
