// Package funclit demonstrates mocking interfaces with function literal parameters.
package funclit

import "errors"

// Exported variables.
var (
	ErrTransformFailed = errors.New("transform failed")
)

// DataProcessor demonstrates an interface with function literal parameters.
// This is a common pattern for data transformation and processing pipelines.
type DataProcessor interface {
	// Transform applies a transformation function to each item.
	// The function literal parameter allows flexible transformations at runtime.
	Transform(items []int, fn func(int) (int, error)) ([]int, error)

	// Filter applies a predicate function to select items.
	Filter(items []int, predicate func(int) bool) []int

	// Reduce aggregates items using a reducer function with accumulator.
	// Multi-parameter function literals like func(acc, item int) are common in reduce operations.
	Reduce(items []int, initial int, reducer func(acc, item int) int) int
}

// Executor demonstrates a struct with methods accepting function literals.
type Executor struct{}

// Run executes a callback function with error handling.
// Function literal parameters are commonly used for callbacks and hooks.
func (e Executor) Run(callback func() error) error {
	err := callback()
	if err != nil {
		return err
	}

	return nil
}

// Filter demonstrates a standalone function with predicate function literal.
func Filter(items []int, predicate func(int) bool) []int {
	result := []int{}

	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}

	return result
}

// Map demonstrates a standalone function with function literal parameter.
// This is the classic functional programming map operation.
func Map(items []int, transform func(int) int) []int {
	result := make([]int, len(items))
	for i, item := range items {
		result[i] = transform(item)
	}

	return result
}
