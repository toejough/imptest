// Package typesafeargs demonstrates type-safe GetArgs() for dependency mocks.
package typesafeargs

// Calculator demonstrates type-safe GetArgs() for dependency mocks.
type Calculator interface {
	// Add has two int parameters
	Add(a, b int) int

	// Multiply has two named parameters of the same type
	Multiply(x, y int) int

	// Store has mixed parameter types
	Store(key string, value any) error
}
