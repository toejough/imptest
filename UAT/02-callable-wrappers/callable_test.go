package callable_test

import (
	"errors"
	"fmt"
	"testing"

	callable "github.com/toejough/imptest/UAT/02-callable-wrappers"
	"github.com/toejough/imptest/imptest"
)

// Generate a mock for the dependency.
//go:generate impgen callable.ExternalService --name ExternalServiceImp

// Generate a wrapper for the function under test.
//go:generate impgen callable.BusinessLogic --name BusinessLogicImp

// Generate wrappers for Calculator methods to demonstrate method wrapping.
//go:generate impgen callable.Calculator.Add
//go:generate impgen callable.Calculator.Multiply
//go:generate impgen callable.Calculator.Divide
//go:generate impgen callable.Calculator.ProcessValue

// TestBusinessLogic demonstrates how to use type-safe wrappers for functions.
//
// Key Requirements Met:
//  1. Function Verification: Verify return values from standalone functions or
//     methods that are not part of an interface.
//  2. Coordinated Control: Synchronously control dependencies while verifying
//     the behavior of the "impure" logic that coordinates them.
func TestBusinessLogic(t *testing.T) {
	t.Parallel()

	// Initialize the mock implementation dependency and the callable wrapper.
	impSvc := NewExternalServiceImp(t)
	logicImp := NewBusinessLogicImp(t, callable.BusinessLogic)

	// Start the business logic in a goroutine.
	// We pass the mock implementation and the input arguments.
	logicImp.Start(impSvc.Mock, 42)

	// 1. Expect call to FetchData and provide response.
	impSvc.ExpectCallIs.FetchData().ExpectArgsAre(42).InjectResults("raw data", nil)

	// 2. Expect call to Process and provide response.
	impSvc.ExpectCallIs.Process().ExpectArgsAre("raw data").InjectResult("processed data")

	// 3. Verify the final output of the business logic.
	logicImp.ExpectReturnedValuesAre("Result: processed data", nil)
}

// TestBusinessLogicError demonstrates error path validation using matchers.
//
// Key Requirements Met:
//  1. Error Chain Validation: Use Satisfies with errors.Is to verify that the
//     correct error is propagated, without requiring strict pointer equality.
func TestBusinessLogicError(t *testing.T) {
	t.Parallel()

	impSvc := NewExternalServiceImp(t)
	logicImp := NewBusinessLogicImp(t, callable.BusinessLogic)

	logicImp.Start(impSvc.Mock, 99)

	// Simulate an error from the service.
	impSvc.ExpectCallIs.FetchData().ExpectArgsAre(99).InjectResults("", errNotFound)

	// Requirement: Verify that the business logic returns the error.
	// We use Satisfies to check if the error is (or wraps) errNotFound.
	logicImp.ExpectReturnedValuesShould("", imptest.Satisfies(func(err error) error {
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

	// Wrap the Add method for testing
	addImp := NewCalculatorAdd(t, calc.Add)

	// Start the method with test arguments
	addImp.Start(5, 3).ExpectReturnedValuesAre(8)
}

// TestCalculatorDivide demonstrates wrapping a method with multiple return values.
func TestCalculatorDivide(t *testing.T) {
	t.Parallel()

	calc := callable.NewCalculator(1)

	// Wrap the Divide method
	divideImp := NewCalculatorDivide(t, calc.Divide)

	// Test successful division
	divideImp.Start(10, 2).ExpectReturnedValuesAre(5, true)
}

// TestCalculatorDivideByZero demonstrates testing error conditions.
func TestCalculatorDivideByZero(t *testing.T) {
	t.Parallel()

	calc := callable.NewCalculator(1)

	// Wrap the Divide method
	divideImp := NewCalculatorDivide(t, calc.Divide)

	// Test division by zero returns false
	divideImp.Start(10, 0).ExpectReturnedValuesAre(0, false)
}

// TestCalculatorMultiply demonstrates wrapping a method that uses receiver state.
func TestCalculatorMultiply(t *testing.T) {
	t.Parallel()

	// Create calculator with multiplier=3
	calc := callable.NewCalculator(3)

	// Wrap the Multiply method
	multiplyImp := NewCalculatorMultiply(t, calc.Multiply)

	// Test that it correctly applies the multiplier
	multiplyImp.Start(7).ExpectReturnedValuesAre(21)
}

// TestCalculatorProcessValuePanic demonstrates testing panic behavior.
func TestCalculatorProcessValuePanic(t *testing.T) {
	t.Parallel()

	calc := callable.NewCalculator(5)

	// Wrap the ProcessValue method
	processImp := NewCalculatorProcessValue(t, calc.ProcessValue)

	// Test that negative values cause a panic
	processImp.Start(-1).ExpectPanicWith("negative values not supported")
}

// TestCalculatorProcessValueSuccess demonstrates normal execution path.
func TestCalculatorProcessValueSuccess(t *testing.T) {
	t.Parallel()

	calc := callable.NewCalculator(5)

	// Wrap the ProcessValue method
	processImp := NewCalculatorProcessValue(t, calc.ProcessValue)

	// Test normal case: (3 * 5) + 10 = 25
	processImp.Start(3).ExpectReturnedValuesAre(25)
}

// unexported variables.
var (
	errNotFound = errors.New("not found")
)
