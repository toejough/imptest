// imptest/impgen is a tool to generate test mocks for Go interfaces.
// To use it, install it with `go install github.com/toejough/imptest/impgen@latest`
// and in your test files, add a `//go:generate impgen <interface>` comment to generate a mock for the specified
// interface. By default, the mocked struct will be named <interface>Imp. Add a `--name <mockname>` flag to specify a
// custom name for the generated mock struct. The generated mock will be placed in a file named <mockname>_test.go,
// in the same package as the test file containing the `//go:generate` comment.
package main

import (
	"errors"
	"fmt"
	"go/build"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/toejough/imptest/impgen/run"
)

// main is the entry point of the impgen tool.
func main() {
	if os.Args == nil {
		return
	}

	err := run.WithCache(os.Args, os.Getenv, &realFileSystem{}, &realPackageLoader{}, &realCacheFileSystem{}, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// unexported variables.
var (
	errNoPackagesFound = errors.New("no packages found")
)

// realCacheFileSystem implements CacheFileSystem using os package.
type realCacheFileSystem struct{}

// Create creates the named file for writing.
func (cfs *realCacheFileSystem) Create(path string) (io.WriteCloser, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", path, err)
	}

	return file, nil
}

// Getwd returns the current working directory.
func (cfs *realCacheFileSystem) Getwd() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return dir, nil
}

// MkdirAll creates a directory path and all parents.
func (cfs *realCacheFileSystem) MkdirAll(path string, perm os.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

// Open opens the named file for reading.
func (cfs *realCacheFileSystem) Open(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}

	return file, nil
}

// Stat returns file info for the named file.
func (cfs *realCacheFileSystem) Stat(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	return info, nil
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

// realPackageLoader implements PackageLoader using golang.org/x/tools/go/packages.
type realPackageLoader struct{}

// Load loads a package by import path and returns its DST files and FileSet.
// Uses fast DST parsing with no type checking for better performance.
//
//nolint:cyclop,funlen,gocognit,nestif // Package resolution requires checking multiple paths and conditions
func (pl *realPackageLoader) Load(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error) {
	// Resolve import path to directory
	var dir string

	if importPath == "." {
		// Current directory
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	} else {
		// First check if it's a local subdirectory (e.g., "./time" when import path is "time")
		// This is needed to handle cases where a local package shadows a stdlib package
		srcDir, _ := os.Getwd()

		localDir := filepath.Join(srcDir, importPath)

		info, err := os.Stat(localDir)
		if err == nil && info.IsDir() {
			// It's a local subdirectory - check if it contains .go files to confirm it's a package
			entries, _ := os.ReadDir(localDir)
			hasGoFiles := false

			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".go") && !e.IsDir() {
					hasGoFiles = true
					break
				}
			}

			if hasGoFiles {
				dir = localDir
			}
		}

		// If not found as local directory, use go/build to resolve
		if dir == "" {
			pkg, err := build.Import(importPath, srcDir, build.FindOnly)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to find package %q: %w", importPath, err)
			}

			dir = pkg.Dir
		}
	}

	// Find all .go files
	// For local packages (importPath == "."), include test files
	// For external/stdlib packages, exclude test files to avoid parse errors
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	includeTests := (importPath == ".")

	goFiles := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		// Skip test files for non-local packages
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			continue
		}

		goFiles = append(goFiles, filepath.Join(dir, name))
	}

	if len(goFiles) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: no .go files in %s", errNoPackagesFound, dir)
	}

	// Parse all files using DST (no conversion needed)
	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	allFiles := make([]*dst.File, 0, len(goFiles))

	for _, goFile := range goFiles {
		// Read file content
		content, err := os.ReadFile(goFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read %s: %w", goFile, err)
		}

		// Parse using DST (fast, no type checking)
		dstFile, err := dec.Parse(string(content))
		if err != nil {
			// Skip files with parse errors
			continue
		}

		allFiles = append(allFiles, dstFile)
	}

	if len(allFiles) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: failed to parse any .go files in %s", errNoPackagesFound, dir)
	}

	// Return nil for typesInfo - we use syntax-based type detection instead
	return allFiles, fset, nil, nil
}
