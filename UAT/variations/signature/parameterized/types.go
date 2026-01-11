// Package parameterized demonstrates mocking interfaces that use instantiated
// generic types like Container[string] in method signatures.
package parameterized

// Container is a generic container type.
type Container[T any] struct {
	Value T
}

// DataProcessor demonstrates using instantiated generic types in interface methods.
// This tests that the generator correctly handles types like Container[string] and Pair[int, bool].
type DataProcessor interface {
	// ProcessContainer takes a Container instantiated with string
	ProcessContainer(data Container[string]) error

	// ProcessPair takes a Pair instantiated with int and bool
	ProcessPair(pair Pair[int, bool]) string

	// ReturnContainer returns a Container instantiated with int
	ReturnContainer() Container[int]
}

// Pair is a generic pair type with two type parameters.
type Pair[K, V any] struct {
	Key   K
	Value V
}
