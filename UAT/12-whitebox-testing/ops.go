// Package whitebox demonstrates whitebox testing where tests are in the same package.
package whitebox

// Ops is an interface to demonstrate whitebox testing.
// It includes both exported and unexported methods to show that whitebox tests
// can access unexported members of the package.
type Ops interface {
	internalMethod(x int) int // unexported - only accessible in whitebox tests
	PublicMethod(x int) int
}
