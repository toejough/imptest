// Package testpkgimport demonstrates resolving unqualified interface names
// from test packages to their non-test package equivalents.
package testpkgimport

type Service interface {
	// Execute performs the service operation
	Execute(input string) (string, error)

	// Validate checks if the input is valid
	Validate(input string) bool
}
