package methods_test

import (
	"testing"

	methods "github.com/toejough/imptest/UAT/13-method-wrapping"
)

//go:generate impgen methods.Calculator.Add --name CalculatorAddImp
//go:generate impgen methods.Calculator.Multiply --name CalculatorMultiplyImp
//go:generate impgen methods.Calculator.Divide --name CalculatorDivideImp
//go:generate impgen methods.Calculator.ProcessValue --name CalculatorProcessValueImp

// TestCalculatorAdd demonstrates wrapping a simple method with multiple parameters.
func TestCalculatorAdd(t *testing.T) {
	t.Parallel()

	calc := methods.NewCalculator(2)

	// Wrap the Add method for testing
	addImp := NewCalculatorAddImp(t, calc.Add)

	// Start the method with test arguments
	addImp.Start(5, 3).ExpectReturnedValuesAre(8)
}

// TestCalculatorDivide demonstrates wrapping a method with multiple return values.
func TestCalculatorDivide(t *testing.T) {
	t.Parallel()

	calc := methods.NewCalculator(1)

	// Wrap the Divide method
	divideImp := NewCalculatorDivideImp(t, calc.Divide)

	// Test successful division
	divideImp.Start(10, 2).ExpectReturnedValuesAre(5, true)
}

// TestCalculatorDivideByZero demonstrates testing error conditions.
func TestCalculatorDivideByZero(t *testing.T) {
	t.Parallel()

	calc := methods.NewCalculator(1)

	// Wrap the Divide method
	divideImp := NewCalculatorDivideImp(t, calc.Divide)

	// Test division by zero returns false
	divideImp.Start(10, 0).ExpectReturnedValuesAre(0, false)
}

// TestCalculatorMultiply demonstrates wrapping a method that uses receiver state.
func TestCalculatorMultiply(t *testing.T) {
	t.Parallel()

	// Create calculator with multiplier=3
	calc := methods.NewCalculator(3)

	// Wrap the Multiply method
	multiplyImp := NewCalculatorMultiplyImp(t, calc.Multiply)

	// Test that it correctly applies the multiplier
	multiplyImp.Start(7).ExpectReturnedValuesAre(21)
}

// TestCalculatorProcessValuePanic demonstrates testing panic behavior.
func TestCalculatorProcessValuePanic(t *testing.T) {
	t.Parallel()

	calc := methods.NewCalculator(5)

	// Wrap the ProcessValue method
	processImp := NewCalculatorProcessValueImp(t, calc.ProcessValue)

	// Test that negative values cause a panic
	processImp.Start(-1).ExpectPanicWith("negative values not supported")
}

// TestCalculatorProcessValueSuccess demonstrates normal execution path.
func TestCalculatorProcessValueSuccess(t *testing.T) {
	t.Parallel()

	calc := methods.NewCalculator(5)

	// Wrap the ProcessValue method
	processImp := NewCalculatorProcessValueImp(t, calc.ProcessValue)

	// Test normal case: (3 * 5) + 10 = 25
	processImp.Start(3).ExpectReturnedValuesAre(25)
}
