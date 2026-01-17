package callable_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/toejough/imptest"
	callable "github.com/toejough/imptest/UAT/core/wrapper-function"
)

// Generate a mock for the dependency using v2 API.
//go:generate impgen callable.ExternalService --dependency

// Generate a wrapper for the function under test using v2 API.
//go:generate impgen callable.BusinessLogic --target

// Generate wrappers for Calculator methods to demonstrate method wrapping using v2 API.
//go:generate impgen callable.Calculator.Add --target
//go:generate impgen callable.Calculator.Multiply --target
//go:generate impgen callable.Calculator.Divide --target
//go:generate impgen callable.Calculator.ProcessValue --target

// TestBusinessLogic demonstrates how to use type-safe wrappers for functions.
//
// Key Requirements Met:
//  1. Function Verification: Verify return values from standalone functions or
//     methods that are not part of an interface.
//  2. Coordinated Control: Synchronously control dependencies while verifying
//     the behavior of the "impure" logic that coordinates them.
func TestBusinessLogic(t *testing.T) {
	t.Parallel()

	// Initialize the mock implementation dependency and the callable wrapper using v2 API.
	// Returns (mock, imp) where mock implements the interface and imp sets expectations.
	mockSvc, svcImp := MockExternalService(t)

	// Start the business logic in a goroutine.
	// We pass the mock implementation and the input arguments.
	call := StartBusinessLogic(t, callable.BusinessLogic, mockSvc, 42)

	// 1. Expect call to FetchData and provide response.
	svcImp.FetchData.Expect(42).Return("raw data", nil)

	// 2. Expect call to Process and provide response.
	svcImp.Process.Expect("raw data").Return("processed data")

	// 3. Verify the final output of the business logic.
	call.ExpectReturn("Result: processed data", nil)
}

// TestBusinessLogicError demonstrates error path validation using matchers.
//
// Key Requirements Met:
//  1. Error Chain Validation: Use Satisfies with errors.Is to verify that the
//     correct error is propagated, without requiring strict pointer equality.
func TestBusinessLogicError(t *testing.T) {
	t.Parallel()

	mockSvc, svcImp := MockExternalService(t)

	call := StartBusinessLogic(t, callable.BusinessLogic, mockSvc, 99)

	// Simulate an error from the service.
	svcImp.FetchData.Expect(99).Return("", errNotFound)

	// Requirement: Verify that the business logic returns the error.
	// We use Satisfies to check if the error is (or wraps) errNotFound.
	call.ExpectReturnMatch("", imptest.Satisfies(func(err error) error {
		if !errors.Is(err, errNotFound) {
			return fmt.Errorf("expected error %w, got %w", errNotFound, err)
		}

		return nil
	}))
}

// TestCalculatorAdd demonstrates wrapping a simple method with multiple parameters.
func TestCalculatorAdd(t *testing.T) {
	t.Parallel()

	calc := callable.NewCalculator(2)

	// Start the method with test arguments using v2 API
	StartCalculatorAdd(t, calc.Add, 5, 3).ExpectReturn(8)
}

// TestCalculatorDivide demonstrates wrapping a method with multiple return values.
func TestCalculatorDivide(t *testing.T) {
	t.Parallel()

	calc := callable.NewCalculator(1)

	// Test successful division using v2 API
	StartCalculatorDivide(t, calc.Divide, 10, 2).ExpectReturn(5, true)
}

// TestCalculatorDivideByZero demonstrates testing error conditions.
func TestCalculatorDivideByZero(t *testing.T) {
	t.Parallel()

	calc := callable.NewCalculator(1)

	// Test division by zero returns false using v2 API
	StartCalculatorDivide(t, calc.Divide, 10, 0).ExpectReturn(0, false)
}

// TestCalculatorMultiply demonstrates wrapping a method that uses receiver state.
func TestCalculatorMultiply(t *testing.T) {
	t.Parallel()

	// Create calculator with multiplier=3
	calc := callable.NewCalculator(3)

	// Test that it correctly applies the multiplier using v2 API
	StartCalculatorMultiply(t, calc.Multiply, 7).ExpectReturn(21)
}

// TestCalculatorProcessValuePanic demonstrates testing panic behavior.
func TestCalculatorProcessValuePanic(
	t *testing.T,
) {
	t.Parallel()

	calc := callable.NewCalculator(5)

	// Test that negative values cause a panic using v2 API
	StartCalculatorProcessValue(t, calc.ProcessValue, -1).ExpectPanic("negative values not supported")
}

// TestCalculatorProcessValueSuccess demonstrates normal execution path.
func TestCalculatorProcessValueSuccess(
	t *testing.T,
) {
	t.Parallel()

	calc := callable.NewCalculator(5)

	// Test normal case: (3 * 5) + 10 = 25 using v2 API
	StartCalculatorProcessValue(t, calc.ProcessValue, 3).ExpectReturn(25)
}

// unexported variables.
var (
	errNotFound = errors.New("not found")
)
