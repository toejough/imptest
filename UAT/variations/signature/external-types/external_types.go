// Package externalimports demonstrates mocking interfaces with external types.
package externalimports

import (
	"io"
	"os"
)

// FileHandler demonstrates using external types in method signatures.
type FileHandler interface {
	// ReadAll uses io.Reader in parameter
	ReadAll(r io.Reader) ([]byte, error)
	// OpenFile uses os.FileMode in parameter and *os.File in return
	OpenFile(path string, mode os.FileMode) (*os.File, error)
	// Stats uses os.FileInfo in return
	Stats(path string) (os.FileInfo, error)
}
