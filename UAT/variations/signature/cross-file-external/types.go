package crossfile

import (
	"os"
	"time"
)

// FileSystem demonstrates an interface using external types in a separate file from tests.
// The go:generate directive will be in a different file.
type FileSystem interface {
	// Stat returns file info including os.FileMode and time.Time
	Stat(path string) (os.FileMode, time.Time, error)
	// Create creates a file with the given permissions
	Create(path string, mode os.FileMode) error
}
