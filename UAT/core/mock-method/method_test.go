package mockmethod_test

import (
	"testing"

	mockmethod "github.com/toejough/imptest/UAT/core/mock-method"
)

//go:generate impgen Counter.Add --dependency
//go:generate impgen Counter.Inc --dependency

// TestMethodMock demonstrates mocking a single struct method.
// The mock extracts the method's signature and creates a mock function.
func TestMethodMock(t *testing.T) {
	t.Parallel()

	const (
		inputValue  = 5
		returnValue = 10
	)

	// Create mock for Counter.Add method signature
	mock := MockCounterAdd(t)

	// Start goroutine that uses the mock function
	go func() {
		result := mockmethod.UseCounterAdd(mock.Mock, inputValue)
		if result != returnValue {
			t.Errorf("expected %d, got %d", returnValue, result)
		}
	}()

	// Verify the mock was called with expected argument
	call := mock.Method.ExpectCalledWithExactly(inputValue)
	args := call.GetArgs()

	if args.N != inputValue {
		t.Errorf("expected N=%d, got N=%d", inputValue, args.N)
	}

	// Inject return value
	call.InjectReturnValues(returnValue)
}

// TestMethodMockMatchers demonstrates using matchers with method mocks.
func TestMethodMockMatchers(t *testing.T) {
	t.Parallel()

	mock := MockCounterAdd(t)

	const expectedReturn = 42

	go func() {
		_ = mockmethod.UseCounterAdd(mock.Mock, 999)
	}()

	// Use matchers for flexible argument matching
	call := mock.Method.ExpectCalledWithMatches(999)
	call.InjectReturnValues(expectedReturn)
}

// TestMultipleMethodMocks demonstrates mocking multiple methods from same struct.
func TestMultipleMethodMocks(t *testing.T) {
	t.Parallel()

	const (
		addInput  = 7
		addReturn = 14
		incReturn = 1
	)

	// Create mocks for different methods
	addMock := MockCounterAdd(t)
	incMock := MockCounterInc(t)

	// Use both mocks
	go func() {
		_ = mockmethod.UseCounterAdd(addMock.Mock, addInput)
	}()

	go func() {
		_ = mockmethod.UseCounterInc(incMock.Mock)
	}()

	// Verify both mocks independently
	addMock.Method.Eventually.ExpectCalledWithExactly(addInput).InjectReturnValues(addReturn)
	incMock.Method.Eventually.ExpectCalledWithMatches().InjectReturnValues(incReturn)
}
