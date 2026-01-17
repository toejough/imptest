// Package funclit_test demonstrates that impgen correctly handles function literal
// parameters, which are an extremely common Go pattern for callbacks and transformations.
package funclit_test

import (
	"errors"
	"testing"

	funclit "github.com/toejough/imptest/UAT/variations/signature/function-literal"
	"github.com/toejough/imptest/match"
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

	mock, imp := MockDataProcessor(t)
	items := []int{1, 2, 3}
	transformFn := func(x int) (int, error) { return x * 2, nil }

	// Run code under test
	go func() {
		result, err := mock.Transform(items, transformFn)
		_ = result
		_ = err
	}()

	// Verify mock handles function literal parameter correctly
	// Note: Function literals can't be compared with ==, so use matcher
	imp.Transform.ArgsShould(items, match.BeAny).
		Return([]int{2, 4, 6}, nil)
}

// TestDependencyWithPredicate demonstrates mocking with predicate function literals.
func TestDependencyWithPredicate(t *testing.T) {
	t.Parallel()

	mock, imp := MockDataProcessor(t)
	items := []int{1, 2, 3, 4, 5}
	isEven := func(x int) bool { return x%2 == 0 }

	go func() {
		result := mock.Filter(items, isEven)
		_ = result
	}()

	imp.Filter.ArgsShould(items, match.BeAny).
		Return([]int{2, 4})
}

// TestDependencyWithReducer demonstrates mocking with multi-parameter function literals.
// The reducer function takes an accumulator and item, a common pattern in functional programming.
func TestDependencyWithReducer(t *testing.T) {
	t.Parallel()

	mock, imp := MockDataProcessor(t)
	items := []int{1, 2, 3, 4}
	sum := func(acc, item int) int { return acc + item }

	go func() {
		result := mock.Reduce(items, 0, sum)
		_ = result
	}()

	imp.Reduce.ArgsShould(items, match.BeAny, match.BeAny).
		Return(10)
}

// TestFunctionWithPredicate demonstrates wrapping functions with predicate literals.
func TestFunctionWithPredicate(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3, 4, 5, 6}
	isOdd := func(x int) bool { return x%2 == 1 }

	StartFilter(t, funclit.Filter, items, isOdd).ReturnsEqual([]int{1, 3, 5})
}

// TestFunctionWithTransform demonstrates wrapping standalone functions
// that accept function literal parameters.
func TestFunctionWithTransform(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3}
	double := func(x int) int { return x * 2 }

	StartMap(t, funclit.Map, items, double).ReturnsEqual([]int{2, 4, 6})
}

// TestMultipleFunctionLiterals demonstrates handling multiple function literal
// parameters in a single method call.
func TestMultipleFunctionLiterals(t *testing.T) {
	t.Parallel()

	mock, imp := MockDataProcessor(t)
	items := []int{1, 2, 3, 4, 5}

	// First call: Transform
	transformFn := func(x int) (int, error) { return x * 3, nil }

	go func() {
		result, err := mock.Transform(items, transformFn)
		_ = result
		_ = err
	}()

	imp.Transform.ArgsShould(items, match.BeAny).
		Return([]int{3, 6, 9, 12, 15}, nil)

	// Second call: Filter on same mock
	predicateFn := func(x int) bool { return x > 10 }

	go func() {
		result := mock.Filter(items, predicateFn)
		_ = result
	}()

	imp.Filter.ArgsShould(items, match.BeAny).
		Return([]int{12, 15})
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

	// Execute and verify
	StartExecutorRun(t, executor.Run, callback).ReturnsEqual(nil)

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

	StartExecutorRun(t, executor.Run, callback).ReturnsEqual(expectedErr)
}
