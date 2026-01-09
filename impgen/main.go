// imptest/impgen is a tool to generate test mocks for Go interfaces.
// To use it, install it with `go install github.com/toejough/imptest/impgen@latest`
// and in your test files, add a `//go:generate impgen <interface>` comment to generate a mock for the specified
// interface. By default, the mocked struct will be named <interface>Imp. Add a `--name <mockname>` flag to specify a
// custom name for the generated mock struct. The generated mock will be placed in a file named <mockname>_test.go,
// in the same package as the test file containing the `//go:generate` comment.
package main

import (
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"github.com/dave/dst"
	"github.com/toejough/imptest/impgen/run"
	load "github.com/toejough/imptest/impgen/run/2_load"
)

// main is the entry point of the impgen tool.
func main() {
	if os.Args == nil {
		return
	}

	// Caching disabled per user request - do not re-enable without explicit approval
	err := run.Run(os.Args, os.Getenv, &realFileSystem{}, &realPackageLoader{}, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// realFileSystem implements FileSystem using os package.
type realFileSystem struct{}

// Glob returns the names of all files matching pattern.
func (fs *realFileSystem) Glob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob failed for pattern %s: %w", pattern, err)
	}

	return matches, nil
}

// ReadFile reads the file named by name and returns the contents.
func (fs *realFileSystem) ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", name, err)
	}

	return data, nil
}

// WriteFile writes data to the file named by name.
func (fs *realFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", name, err)
	}

	return nil
}

// realPackageLoader implements PackageLoader using direct DST parsing.
type realPackageLoader struct{}

// Load loads a package by import path and returns its DST files and FileSet.
// Uses the shared load.PackageDST function for direct DST parsing with no type checking.
func (pl *realPackageLoader) Load(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error) {
	files, fset, err := load.PackageDST(importPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load package %q: %w", importPath, err)
	}

	// Return nil for typesInfo - we use syntax-based type detection instead
	return files, fset, nil, nil
}
