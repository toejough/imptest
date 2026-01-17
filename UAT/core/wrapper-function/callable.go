// Package callable demonstrates wrapping functions and struct methods for testing.
package callable

import (
	"fmt"
	"time"
)

type AddFunc func(a, b int) int

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

type ComputeFunc func(x int) (int, string, bool)

type ConditionalFunc func(x int) int

type DivideFunc func(a, b int) (int, bool)

type ExternalService interface {
	FetchData(id int) (string, error)
	Process(data string) string
}

type MultiplyFunc func(a, b int) int

type PanicFunc func()

type PanicIntFunc func() int

type ProcessFunc func(x int) (string, error)

type SideEffectFunc func(x int)

type SlowAddFunc func(a, b int, delay time.Duration) int

type SlowFuncFunc func() int

type SlowMultiplyFunc func(a int, delay time.Duration) int

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

// unexported constants.
const (
	processingOffset = 10
)
