// Package helpers provides helper interfaces for dot-import testing.
package helpers

type Processor interface {
	// Process transforms input data
	Process(input string) string
}

type Storage interface {
	// Save stores a value with a key
	Save(key, value string) error

	// Load retrieves a value by key
	Load(key string) (string, error)
}
