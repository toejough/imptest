// Package zero_params demonstrates mock generation for interfaces with zero-parameter methods.
// This tests edge cases in parameter handling and field counting logic.
//
//nolint:revive // Package name intentionally uses underscore for clarity
package zero_params

//go:generate impgen NoParams

// NoParams is an interface with methods that have no parameters.
// This tests the edge case where parameter lists are empty or nil.
type NoParams interface {
	// Get has no parameters and returns a value
	Get() int
	// Execute has no parameters and no meaningful return
	Execute() error
}
