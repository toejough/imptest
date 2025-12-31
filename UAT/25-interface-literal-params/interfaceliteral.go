package interfaceliteral

// DataProcessor demonstrates interface literals in method signatures.
// Interface literals are anonymous interfaces defined inline, commonly used for
// ad-hoc contracts without needing to define a named interface type.
type DataProcessor interface {
	// Process accepts an object with a single getter method.
	// This is a common pattern for simple read-only access to data.
	Process(obj interface{ Get() string }) string

	// Transform accepts an object with multiple methods.
	// Tests that interface literals with multiple methods are handled correctly.
	Transform(obj interface {
		GetValue() int
		SetValue(value int)
	}) int

	// Validate accepts an object with method returning error.
	// Common pattern for validation interfaces.
	Validate(validator interface{ Check(input string) error }) error

	// ProcessWithReturn demonstrates interface literal as return type.
	// This tests both parameter and return type handling.
	ProcessWithReturn(input string) interface{ Result() string }
}
