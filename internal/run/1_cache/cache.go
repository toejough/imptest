// Package cache provides caching for generated code signatures to support
// incremental regeneration in the impgen tool.
package cache

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Exported constants.
const (
	// DirName is the name of the local cache directory.
	DirName = ".impgen"
	// DirPerm is the default directory permission.
	DirPerm = 0o755
	// FilePerm is the default file permission.
	FilePerm = 0o600
)

type Data struct {
	Entries map[string]Entry `json:"entries"`
}

type Entry struct {
	Signature string `json:"signature"`
	Content   string `json:"content"`
	Filename  string `json:"filename"`
}

type FileSystem interface {
	Open(path string) (io.ReadCloser, error)
	Create(path string) (io.WriteCloser, error)
	MkdirAll(path string, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	Getwd() (string, error)
}

// CalculatePackageSignature generates a unique hash based on CLI arguments
// and the Go source files in the current directory.

// 1. Hash the arguments (skip program name)

// 2. Hash all .go files in the current directory (where go:generate runs)

// Skip generated files to avoid circular dependency in signature

// FindProjectRoot locates the nearest directory containing a go.mod file.
func FindProjectRoot(cfs FileSystem) (string, error) {
	curr, err := cfs.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		_, err = cfs.Stat(filepath.Join(curr, "go.mod"))
		if err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", errProjectRootNotFound
		}

		curr = parent
	}
}

// unexported variables.
var (
	errProjectRootNotFound = errors.New("could not find project root (go.mod)")
)
