package load

import (
	"errors"
	"fmt"
	"go/build"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// PackageDST loads a package by import path and returns its DST files and FileSet.
// This is the shared implementation used by all PackageLoader implementations.
// Uses fast DST parsing with no type checking for better performance.
//
//nolint:cyclop,funlen // Package loading and file parsing require multiple steps
func PackageDST(importPath string) ([]*dst.File, *token.FileSet, error) {
	// Resolve import path to directory
	var dir string

	//nolint:nestif // Path resolution requires conditional logic
	if importPath == "." {
		// Current directory
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	} else {
		// Check if it's a local subdirectory package (e.g., "./time" shadowing stdlib "time")
		resolvedPath := ResolveLocalPackagePath(importPath)

		if resolvedPath != importPath {
			// It's a local package - use the resolved absolute path
			dir = resolvedPath
		} else {
			// Use go/build to resolve the import path
			srcDir, _ := os.Getwd()

			pkg, err := build.Import(importPath, srcDir, build.FindOnly)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to find package %q: %w", importPath, err)
			}

			dir = pkg.Dir
		}
	}

	// Find all .go files
	// For local packages (importPath == "."), include test files
	// For external/stdlib packages, exclude test files to avoid parse errors
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
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
		return nil, nil, fmt.Errorf("%w: no .go files in %s", errNoPackagesFound, dir)
	}

	// Parse all files using DST (no conversion needed)
	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	allFiles := make([]*dst.File, 0, len(goFiles))

	for _, goFile := range goFiles {
		// Parse using DST with filename for proper FileSet registration
		dstFile, err := dec.ParseFile(goFile, nil, 0)
		if err != nil {
			// Skip files with parse errors
			continue
		}

		allFiles = append(allFiles, dstFile)
	}

	if len(allFiles) == 0 {
		return nil, nil, fmt.Errorf(
			"%w: failed to parse any .go files in %s",
			errNoPackagesFound,
			dir,
		)
	}

	return allFiles, fset, nil
}

// ResolveLocalPackagePath checks if importPath refers to a local subdirectory package.
// For simple package names (no slashes), it checks if there's a local subdirectory
// with that name containing .go files. This handles cases where local packages
// shadow stdlib packages (e.g., a local "time" package shadowing stdlib "time").
//
// Returns the absolute path to the local package directory if found, or the
// original importPath if it should be resolved normally.
//
//nolint:cyclop // Early returns for different resolution paths
func ResolveLocalPackagePath(importPath string) string {
	// Only check for simple package names (no slashes, not ".", not absolute paths)
	if importPath == "." || strings.HasPrefix(importPath, "/") ||
		strings.Contains(importPath, "/") {
		return importPath
	}

	srcDir, err := os.Getwd()
	if err != nil {
		return importPath
	}

	localDir := filepath.Join(srcDir, importPath)

	info, err := os.Stat(localDir)
	if err != nil || !info.IsDir() {
		return importPath
	}

	// Check if it contains .go files
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return importPath
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") && !e.IsDir() {
			// Found a local package - return the absolute path
			return localDir
		}
	}

	return importPath
}

// unexported variables.
var (
	errNoPackagesFound = errors.New("no packages found")
)
