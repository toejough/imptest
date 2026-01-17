// Package whitebox demonstrates whitebox testing where tests are in the same package.
package whitebox

type Ops interface {
	internalMethod(x int) int // unexported - only accessible in whitebox tests
	PublicMethod(x int) int
}
