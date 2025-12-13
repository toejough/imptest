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
	"os"

	"github.com/toejough/imptest/impgen/run"
	"golang.org/x/tools/go/packages"
)

var (
	errNoPackagesFound = errors.New("no packages found")
	errPackageErrors   = errors.New("package errors")
)

// main is the entry point of the impgen tool.
func main() {
	err := run.Run(os.Args, os.Getenv, &RealFileSystem{}, &RealPackageLoader{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// RealFileSystem implements FileSystem using os package.
type RealFileSystem struct{}

// WriteFile writes data to the file named by name.
func (fs *RealFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", name, err)
	}

	return nil
}

// RealPackageLoader implements PackageLoader using golang.org/x/tools/go/packages.
type RealPackageLoader struct{}

// Load loads a package by import path and returns its AST files and FileSet.
func (pl *RealPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, error) {
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	}

	// Collect all AST files from all packages (including test packages)
	var (
		allFiles []*ast.File
		fset     *token.FileSet
	)

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		if fset == nil {
			fset = pkg.Fset
		}

		allFiles = append(allFiles, pkg.Syntax...)
	}

	if len(allFiles) == 0 {
		if len(pkgs[0].Errors) > 0 {
			return nil, nil, fmt.Errorf("%w: %v", errPackageErrors, pkgs[0].Errors)
		}

		return nil, nil, fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	}

	return allFiles, fset, nil
}
