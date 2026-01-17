// Package callable demonstrates wrapping functions and struct methods for testing.
package callable

import (
	"fmt"
	"time"
)

// Calculator provides arithmetic operations.
// In real applications, methods might have side effects (logging, metrics, database calls)
// that make them difficult to test directly.
type Calculator struct {
	multiplier int
}

// NewCalculator creates a calculator with a multiplier factor.
func NewCalculator(multiplier int) *Calculator {
	return &Calculator{multiplier: multiplier}
}

// Add returns the sum of two integers.
func (c *Calculator) Add(a, b int) int {
	return a + b
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

// ProcessValue demonstrates a method that might panic in production.
// This method applies a series of operations that could fail.
func (c *Calculator) ProcessValue(value int) int {
	if value < 0 {
		panic("negative values not supported")
	}

	result := c.Multiply(value)

	return result + processingOffset
}

// ExternalService is a dependency for our business logic.
type ExternalService interface {
	FetchData(id int) (string, error)
	Process(data string) string
}

// BusinessLogic is the function we want to test.
// It orchestrates calls to an ExternalService.
func BusinessLogic(svc ExternalService, id int) (string, error) {
	data, err := svc.FetchData(id)
	if err != nil {
		return "", fmt.Errorf("failed to fetch data: %w", err)
	}

	result := svc.Process(data)

	return "Result: " + result, nil
}

// PanicWithMessage panics with the given message if non-empty.
// Used for testing panic handling in wrapped functions.
func PanicWithMessage(msg string) {
	if msg != "" {
		panic(msg)
	}
}

// Function type definitions for testing CallHandle behavior.
// These define signatures used in callhandle_test.go.
type (
	// SlowAddFunc adds two integers with a delay for concurrent testing.
	SlowAddFunc func(a, b int, delay time.Duration) int

	// SlowFuncFunc is a no-arg function that returns int with potential delay.
	SlowFuncFunc func() int

	// ProcessFunc processes an int and returns string+error.
	ProcessFunc func(x int) (string, error)

	// MultiplyFunc multiplies two integers.
	MultiplyFunc func(a, b int) int

	// SlowMultiplyFunc multiplies with delay for concurrent testing.
	SlowMultiplyFunc func(a int, delay time.Duration) int

	// DivideFunc divides and returns result with success flag.
	DivideFunc func(a, b int) (int, bool)

	// PanicIntFunc is a func that returns int and may panic.
	PanicIntFunc func() int

	// ComputeFunc computes and returns multiple values.
	ComputeFunc func(x int) (int, string, bool)

	// SideEffectFunc performs side effects with no return.
	SideEffectFunc func(x int)

	// ConditionalFunc conditionally panics or returns.
	ConditionalFunc func(x int) int

	// PanicFunc is a no-arg function that may panic.
	PanicFunc func()

	// AddFunc adds two integers.
	AddFunc func(a, b int) int
)

// unexported constants.
const (
	processingOffset = 10
)
