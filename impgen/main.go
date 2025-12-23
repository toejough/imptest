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
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"github.com/toejough/imptest/impgen/run"
	"golang.org/x/tools/go/packages"
)

var (
	errNoPackagesFound = errors.New("no packages found")
	errPackageErrors   = errors.New("package errors")
)

// main is the entry point of the impgen tool.
func main() {
	if os.Args == nil {
		return
	}

	err := run.WithCache(os.Args, os.Getenv, &realFileSystem{}, &realPackageLoader{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// realFileSystem implements FileSystem using os package.
type realFileSystem struct{}

// realPackageLoader implements PackageLoader using golang.org/x/tools/go/packages.
type realPackageLoader struct{}

// Load loads a package by import path and returns its AST files, FileSet, and type information.
func (pl *realPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax,
		Tests: true,
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	}

	// Collect all AST files from all packages (including test packages)
	var (
		allFiles  []*ast.File
		fset      *token.FileSet
		typesInfo *types.Info
	)

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		if fset == nil {
			fset = pkg.Fset
		}

		// Use type info from the first valid package
		if typesInfo == nil && pkg.TypesInfo != nil {
			typesInfo = pkg.TypesInfo
		}

		allFiles = append(allFiles, pkg.Syntax...)
	}

	if len(allFiles) == 0 {
		if len(pkgs[0].Errors) > 0 {
			return nil, nil, nil, fmt.Errorf("%w: %v", errPackageErrors, pkgs[0].Errors)
		}

		return nil, nil, nil, fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	}

	return allFiles, fset, typesInfo, nil
}

// WriteFile writes data to the file named by name.
func (fs *realFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", name, err)
	}

	return nil
}

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
