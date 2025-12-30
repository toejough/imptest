package testpkgimport

// Service is an interface in the non-test package.
// This tests that the generator can resolve unqualified interface names
// from test packages to their non-test package equivalents.
type Service interface {
	// Execute performs the service operation
	Execute(input string) (string, error)

	// Validate checks if the input is valid
	Validate(input string) bool
}
