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
	mockSvc := MockExternalService(t)
	wrapper := WrapBusinessLogic(t, callable.BusinessLogic)

	// Start the business logic in a goroutine.
	// We pass the mock implementation and the input arguments.
	call := wrapper.Method.Start(mockSvc.Mock, 42)

	// 1. Expect call to FetchData and provide response.
	mockSvc.Method.FetchData.ExpectCalledWithExactly(42).InjectReturnValues("raw data", nil)

	// 2. Expect call to Process and provide response.
	mockSvc.Method.Process.ExpectCalledWithExactly("raw data").InjectReturnValues("processed data")

	// 3. Verify the final output of the business logic.
	call.ExpectReturnsEqual("Result: processed data", nil)
}

// TestBusinessLogicError demonstrates error path validation using matchers.
//
// Key Requirements Met:
//  1. Error Chain Validation: Use Satisfies with errors.Is to verify that the
//     correct error is propagated, without requiring strict pointer equality.
func TestBusinessLogicError(t *testing.T) {
	t.Parallel()

	mockSvc := MockExternalService(t)
	wrapper := WrapBusinessLogic(t, callable.BusinessLogic)

	call := wrapper.Method.Start(mockSvc.Mock, 99)

	// Simulate an error from the service.
	mockSvc.Method.FetchData.ExpectCalledWithExactly(99).InjectReturnValues("", errNotFound)

	// Requirement: Verify that the business logic returns the error.
	// We use Satisfies to check if the error is (or wraps) errNotFound.
	call.ExpectReturnsMatch("", imptest.Satisfies(func(err error) error {
		if !errors.Is(err, errNotFound) {
			return fmt.Errorf("expected error %w, got %w", errNotFound, err)
		}

		return nil
	}))
}

// TestCalculatorAdd demonstrates wrapping a simple method with multiple parameters.
func TestCalculatorAdd(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	calc := callable.NewCalculator(2)

	// Wrap the Add method for testing using v2 API
	wrapper := WrapCalculatorAdd(t, calc.Add)

	// Start the method with test arguments
	wrapper.Method.Start(5, 3).ExpectReturnsEqual(8)
}

// TestCalculatorDivide demonstrates wrapping a method with multiple return values.
func TestCalculatorDivide(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	calc := callable.NewCalculator(1)

	// Wrap the Divide method using v2 API
	wrapper := WrapCalculatorDivide(t, calc.Divide)

	// Test successful division
	wrapper.Method.Start(10, 2).ExpectReturnsEqual(5, true)
}

// TestCalculatorDivideByZero demonstrates testing error conditions.
func TestCalculatorDivideByZero(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	calc := callable.NewCalculator(1)

	// Wrap the Divide method using v2 API
	wrapper := WrapCalculatorDivide(t, calc.Divide)

	// Test division by zero returns false
	wrapper.Method.Start(10, 0).ExpectReturnsEqual(0, false)
}

// TestCalculatorMultiply demonstrates wrapping a method that uses receiver state.
func TestCalculatorMultiply(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	// Create calculator with multiplier=3
	calc := callable.NewCalculator(3)

	// Wrap the Multiply method using v2 API
	wrapper := WrapCalculatorMultiply(t, calc.Multiply)

	// Test that it correctly applies the multiplier
	wrapper.Method.Start(7).ExpectReturnsEqual(21)
}

// TestCalculatorProcessValuePanic demonstrates testing panic behavior.
func TestCalculatorProcessValuePanic(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	calc := callable.NewCalculator(5)

	// Wrap the ProcessValue method using v2 API
	wrapper := WrapCalculatorProcessValue(t, calc.ProcessValue)

	// Test that negative values cause a panic
	wrapper.Method.Start(-1).ExpectPanicEquals("negative values not supported")
}

// TestCalculatorProcessValueSuccess demonstrates normal execution path.
func TestCalculatorProcessValueSuccess(t *testing.T) { //nolint:varnamelen // Standard Go test convention
	t.Parallel()

	calc := callable.NewCalculator(5)

	// Wrap the ProcessValue method using v2 API
	wrapper := WrapCalculatorProcessValue(t, calc.ProcessValue)

	// Test normal case: (3 * 5) + 10 = 25
	wrapper.Method.Start(3).ExpectReturnsEqual(25)
}

// unexported variables.
var (
	errNotFound = errors.New("not found")
)
