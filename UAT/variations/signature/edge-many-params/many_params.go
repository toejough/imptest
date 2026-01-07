// Package many_params demonstrates mock generation for interfaces with many parameters.
// This tests edge cases in parameter naming (beyond A-H) and index arithmetic.
package many_params

// ManyParams is an interface with a method that has 10 parameters.
// This tests parameter naming beyond the first 8 (A-H), which should use param0, param1, etc.
type ManyParams interface {
	// Process has 10 parameters to test index arithmetic and naming logic
	Process(a, b, c, d, e, f, g, h, i, j int) string
}
