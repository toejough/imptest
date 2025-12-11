// imptest/impgen is a tool to generate test mocks for Go interfaces.
// To use it, install it with `go install github.com/toejough/imptest/impgen@latest`
// and in your test files, add a `//go:generate impgen <interface>` comment to generate a mock for the specified
// interface. By default, the mocked struct will be named <interface>Imp. Add a `--name <mockname>` flag to specify a
// custom name for the generated mock struct. The generated mock will be placed in a file named <mockname>_test.go,
// in the same package as the test file containing the `//go:generate` comment.
package main

import (
	"fmt"
	"os"

	"github.com/toejough/imptest/impgen/run"
)

// main is the entry point of the impgen tool.
func main() {
	err := run.Run(os.Args, os.Getenv, &RealFileSystem{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// RealFileSystem implements FileSystem using os package.
type RealFileSystem struct{}

// Getwd returns the current working directory.
func (fs *RealFileSystem) Getwd() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return wd, nil
}

// ReadDir reads the contents of the directory named by name.
func (fs *RealFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", name, err)
	}

	return entries, nil
}

// ReadFile reads the file named by name and returns the contents.
func (fs *RealFileSystem) ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", name, err)
	}

	return data, nil
}

// WriteFile writes data to the file named by name.
func (fs *RealFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", name, err)
	}

	return nil
}
