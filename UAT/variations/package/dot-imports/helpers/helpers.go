// Package helpers provides helper interfaces for dot-import testing.
package helpers

// Processor demonstrates a second interface for comprehensive testing.
type Processor interface {
	// Process transforms input data
	Process(input string) string
}

// Storage demonstrates an interface that will be dot-imported and mocked.
// Dot imports allow using exported symbols without package qualification.
type Storage interface {
	// Save stores a value with a key
	Save(key, value string) error

	// Load retrieves a value by key
	Load(key string) (string, error)
}
