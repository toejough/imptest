// Package manyparams demonstrates mock generation for interfaces with many parameters.
// This tests edge cases in parameter naming (beyond A-H) and index arithmetic.
package manyparams

type ManyParams interface {
	// Process has 10 parameters to test index arithmetic and naming logic
	Process(a, b, c, d, e, f, g, h, i, j int) string
}
