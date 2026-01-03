package calculator

import (
	"errors"
	"fmt"
)

// Exported variables.
var (
	ErrNegativeInput = errors.New("negative input not allowed")
)

// Calculator provides arithmetic operations with state.
// This demonstrates wrapping an entire struct type with --target flag to intercept all method calls.
// Unlike wrapping individual methods (UAT-02), struct wrapping creates wrappers for ALL methods at once.
type Calculator struct {
	multiplier int
	offset     int
}

// NewCalculator creates a calculator with a multiplier and offset.
func NewCalculator(multiplier, offset int) *Calculator {
	return &Calculator{
		multiplier: multiplier,
		offset:     offset,
	}
}

// Add returns the sum of two integers plus the calculator's offset.
func (c *Calculator) Add(a, b int) int {
	return a + b + c.offset
}

// Divide returns the quotient and whether the division was successful.
// Returns false if attempting to divide by zero.
func (c *Calculator) Divide(numerator, denominator int) (int, bool) {
	if denominator == 0 {
		return 0, false
	}

	return numerator / denominator, true
}

// Multiply applies the calculator's multiplier to the input.
func (c *Calculator) Multiply(value int) int {
	return value * c.multiplier
}

// Process demonstrates a method with complex logic that uses other methods.
// This shows that wrapping at the struct level intercepts all calls.
func (c *Calculator) Process(input int) (string, error) {
	if input < 0 {
		return "", fmt.Errorf("processing failed for input %d: %w", input, ErrNegativeInput)
	}

	// Use other methods
	multiplied := c.Multiply(input)
	sum := c.Add(multiplied, defaultOffset)

	return fmt.Sprintf("Result: %d", sum), nil
}

// unexported constants.
const (
	// defaultOffset is the offset value used in Process method calculations.
	defaultOffset = 5
)
