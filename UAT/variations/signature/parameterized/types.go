// Package parameterized demonstrates mocking interfaces that use instantiated
// generic types like Container[string] in method signatures.
package parameterized

type Container[T any] struct {
	Value T
}

type DataProcessor interface {
	// ProcessContainer takes a Container instantiated with string
	ProcessContainer(data Container[string]) error

	// ProcessPair takes a Pair instantiated with int and bool
	ProcessPair(pair Pair[int, bool]) string

	// ReturnContainer returns a Container instantiated with int
	ReturnContainer() Container[int]
}

type Pair[K, V any] struct {
	Key   K
	Value V
}
