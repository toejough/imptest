// Package storage provides storage interfaces for dot-import testing.
package storage

type Repository interface {
	// Save persists data with a key
	Save(key string, data []byte) error

	// Load retrieves data by key
	Load(key string) ([]byte, error)

	// Delete removes data by key
	Delete(key string) error
}
