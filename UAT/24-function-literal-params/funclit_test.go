// Package funclit_test demonstrates that impgen correctly handles function literal
// parameters, which are an extremely common Go pattern for callbacks and transformations.
package funclit_test

import (
	"errors"
	"testing"

	funclit "github.com/toejough/imptest/UAT/24-function-literal-params"
	"github.com/toejough/imptest/imptest"
)

// Generate dependency mock for interface with function literal params
//go:generate impgen funclit.DataProcessor --dependency

// Generate target wrappers for functions with function literal params
//go:generate impgen funclit.Executor.Run --target
//go:generate impgen funclit.Map --target
//go:generate impgen funclit.Filter --target

// TestDependencyWithFunctionLiterals demonstrates that dependency mocks work correctly
// with interfaces that have function literal parameters.
func TestDependencyWithFunctionLiterals(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)
	items := []int{1, 2, 3}
	transformFn := func(x int) (int, error) { return x * 2, nil }

	// Run code under test
	go func() {
		result, err := mock.Interface().Transform(items, transformFn)
		_ = result
		_ = err
	}()

	// Verify mock handles function literal parameter correctly
	// Note: Function literals can't be compared with ==, so use matcher
	mock.Transform.ExpectCalledWithMatches(items, imptest.Any()).
		InjectReturnValues([]int{2, 4, 6}, nil)
}

// TestDependencyWithPredicate demonstrates mocking with predicate function literals.
func TestDependencyWithPredicate(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)
	items := []int{1, 2, 3, 4, 5}
	isEven := func(x int) bool { return x%2 == 0 }

	go func() {
		result := mock.Interface().Filter(items, isEven)
		_ = result
	}()

	mock.Filter.ExpectCalledWithMatches(items, imptest.Any()).
		InjectReturnValues([]int{2, 4})
}

// TestDependencyWithReducer demonstrates mocking with reducer function literals.
// Note: Using single-param function due to impgen limitation with multi-param function literals
func TestDependencyWithReducer(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)
	items := []int{1, 2, 3, 4}
	double := func(x int) int { return x * 2 }

	go func() {
		result := mock.Interface().Reduce(items, double)
		_ = result
	}()

	mock.Reduce.ExpectCalledWithMatches(items, imptest.Any()).
		InjectReturnValues([]int{2, 4, 6, 8})
}

// TestFunctionWithPredicate demonstrates wrapping functions with predicate literals.
func TestFunctionWithPredicate(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3, 4, 5, 6}
	isOdd := func(x int) bool { return x%2 == 1 }

	wrapper := WrapFilter(t, funclit.Filter)

	wrapper.Start(items, isOdd).ExpectReturnsEqual([]int{1, 3, 5})
}

// TestFunctionWithTransform demonstrates wrapping standalone functions
// that accept function literal parameters.
func TestFunctionWithTransform(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3}
	double := func(x int) int { return x * 2 }

	wrapper := WrapMap(t, funclit.Map)

	wrapper.Start(items, double).ExpectReturnsEqual([]int{2, 4, 6})
}

// TestMultipleFunctionLiterals demonstrates handling multiple function literal
// parameters in a single method call.
func TestMultipleFunctionLiterals(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)
	items := []int{1, 2, 3, 4, 5}

	// First call: Transform
	transformFn := func(x int) (int, error) { return x * 3, nil }

	go func() {
		result, err := mock.Interface().Transform(items, transformFn)
		_ = result
		_ = err
	}()

	mock.Transform.ExpectCalledWithMatches(items, imptest.Any()).
		InjectReturnValues([]int{3, 6, 9, 12, 15}, nil)

	// Second call: Filter on same mock
	predicateFn := func(x int) bool { return x > 10 }

	go func() {
		result := mock.Interface().Filter(items, predicateFn)
		_ = result
	}()

	mock.Filter.ExpectCalledWithMatches(items, imptest.Any()).
		InjectReturnValues([]int{12, 15})
}

// TestTargetWithCallback demonstrates that target wrappers work correctly
// with methods that have function literal parameters.
func TestTargetWithCallback(t *testing.T) {
	t.Parallel()

	executor := funclit.Executor{}
	callbackCalled := false
	callback := func() error {
		callbackCalled = true
		return nil
	}

	// Wrap the Run method for testing
	wrapper := WrapExecutorRun(t, executor.Run)

	// Execute and verify
	wrapper.Start(callback).ExpectReturnsEqual(nil)

	if !callbackCalled {
		t.Error("expected callback to be called")
	}
}

// TestTargetWithCallbackError demonstrates error handling with function literal params.
func TestTargetWithCallbackError(t *testing.T) {
	t.Parallel()

	executor := funclit.Executor{}
	expectedErr := errors.New("callback failed")
	callback := func() error {
		return expectedErr
	}

	wrapper := WrapExecutorRun(t, executor.Run)

	wrapper.Start(callback).ExpectReturnsEqual(expectedErr)
}
