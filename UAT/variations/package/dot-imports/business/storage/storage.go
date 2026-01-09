package storage

// Repository demonstrates an interface that will be dot-imported by business logic.
// This simulates a common pattern where storage abstractions are imported
// without qualification for cleaner code.
type Repository interface {
	// Save persists data with a key
	Save(key string, data []byte) error

	// Load retrieves data by key
	Load(key string) ([]byte, error)

	// Delete removes data by key
	Delete(key string) error
}
