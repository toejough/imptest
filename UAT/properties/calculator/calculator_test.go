// Package calculator_test demonstrates property-based testing with imptest and rapid.
//
// This replaces hand-written example-based tests with property-based tests that
// prove correctness across many randomly generated inputs.
package calculator_test

import (
	"testing"

	"pgregory.net/rapid"

	callable "github.com/toejough/imptest/UAT/core/wrapper-function"
)

// Generate wrappers for Calculator methods.
//go:generate impgen callable.Calculator.Add --target
//go:generate impgen callable.Calculator.Multiply --target
//go:generate impgen callable.Calculator.Divide --target
//go:generate impgen callable.Calculator.ProcessValue --target

// TestAdd_Property proves: Add(a, b) == a + b for all integers.
func TestAdd_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate random inputs
		a := rapid.Int().Draw(rt, "a")
		b := rapid.Int().Draw(rt, "b")

		// Calculator multiplier doesn't affect Add
		calc := callable.NewCalculator(1)
		wrapper := WrapCalculatorAdd(t, calc.Add)

		// Property: Add returns the sum
		wrapper.Method.Start(a, b).ExpectReturnsEqual(a + b)
	})
}

// TestDivide_NonZero_Property proves: Divide(a, b) where b != 0 returns (a/b, true).
func TestDivide_NonZero_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.Int().Draw(rt, "a")
		// Generate non-zero denominator
		b := rapid.Int().Draw(rt, "b")
		if b == 0 {
			b = 1 // Ensure non-zero
		}

		calc := callable.NewCalculator(1)
		wrapper := WrapCalculatorDivide(t, calc.Divide)

		// Property: Non-zero division succeeds with correct quotient
		wrapper.Method.Start(a, b).ExpectReturnsEqual(a/b, true)
	})
}

// TestDivide_Zero_Property proves: Divide(a, 0) returns (0, false) for all a.
func TestDivide_Zero_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.Int().Draw(rt, "a")

		calc := callable.NewCalculator(1)
		wrapper := WrapCalculatorDivide(t, calc.Divide)

		// Property: Division by zero returns (0, false)
		wrapper.Method.Start(a, 0).ExpectReturnsEqual(0, false)
	})
}

// TestMultiply_Property proves: Multiply(v) == v * multiplier for all integers.
func TestMultiply_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate random multiplier and value
		multiplier := rapid.IntRange(1, 1000).Draw(rt, "multiplier")
		value := rapid.IntRange(-1000, 1000).Draw(rt, "value")

		calc := callable.NewCalculator(multiplier)
		wrapper := WrapCalculatorMultiply(t, calc.Multiply)

		// Property: Multiply applies the multiplier
		wrapper.Method.Start(value).ExpectReturnsEqual(value * multiplier)
	})
}

// TestProcessValue_Negative_Property proves: ProcessValue(v < 0) panics.
func TestProcessValue_Negative_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		value := rapid.IntRange(-1000, -1).Draw(rt, "negative_value")

		calc := callable.NewCalculator(1)
		wrapper := WrapCalculatorProcessValue(t, calc.ProcessValue)

		// Property: Negative values cause panic
		wrapper.Method.Start(value).ExpectPanicEquals("negative values not supported")
	})
}

// TestProcessValue_NonNegative_Property proves: ProcessValue(v >= 0) == (v * multiplier) + 10.
func TestProcessValue_NonNegative_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		multiplier := rapid.IntRange(1, 100).Draw(rt, "multiplier")
		value := rapid.IntRange(0, 1000).Draw(rt, "value") // Non-negative

		calc := callable.NewCalculator(multiplier)
		wrapper := WrapCalculatorProcessValue(t, calc.ProcessValue)

		// Property: ProcessValue applies multiplier and adds offset (10)
		expected := (value * multiplier) + 10
		wrapper.Method.Start(value).ExpectReturnsEqual(expected)
	})
}
