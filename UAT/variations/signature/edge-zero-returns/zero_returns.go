// Package zeroreturns demonstrates mock generation for functions with zero return values.
// This tests edge cases in return value handling and result counting logic.
package zeroreturns

// ProcessData is a function that takes parameters but returns nothing.
// This tests the edge case where return value lists are empty or nil.
func ProcessData(data string, count int) {
	// Function body doesn't matter for mock generation
	_ = data
	_ = count
}
