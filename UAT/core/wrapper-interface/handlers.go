// Package handlers demonstrates wrapping interface implementations.
package handlers

import (
	"context"
)

type Calculator interface {
	Add(a, b int) int
	Divide(numerator, denominator int) (int, bool)
	Multiply(value int) int
	ProcessValue(value int) int
}

type CalculatorImpl struct {
	multiplier int
}

// NewCalculatorImpl creates a new CalculatorImpl.
func NewCalculatorImpl(multiplier int) *CalculatorImpl {
	return &CalculatorImpl{multiplier: multiplier}
}

// Add returns multiplier + a + b.
func (c *CalculatorImpl) Add(a, b int) int {
	return c.multiplier + a + b
}

// Divide returns numerator / denominator and success flag.
func (c *CalculatorImpl) Divide(numerator, denominator int) (int, bool) {
	if denominator == 0 {
		return 0, false
	}

	return numerator / denominator, true
}

// Multiply returns value * multiplier.
func (c *CalculatorImpl) Multiply(value int) int {
	return value * c.multiplier
}

// ProcessValue returns Multiply(value) + 10, panics if value < 0.
func (c *CalculatorImpl) ProcessValue(value int) int {
	const offset = 10

	if value < 0 {
		panic("negative values not supported")
	}

	return c.Multiply(value) + offset
}

type Logger interface {
	// Log writes a log message and returns any error encountered.
	Log(msg string) error

	// LogWithContext writes a log message with context and returns any error.
	LogWithContext(ctx context.Context, msg string) error
}

type Service struct {
	// Service fields would go here in a real implementation
}
