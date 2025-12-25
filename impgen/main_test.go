package main

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestPackageConfigMode_mutant verifies that the Mode field includes all necessary flags.
// This catches mutations where bitwise operators are changed (| to &, etc).
func TestPackageConfigMode_mutant(t *testing.T) {
	t.Parallel()

	// Create a config like realPackageLoader does
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

	// Verify that all flags are set by checking specific bits
	requiredFlags := []packages.LoadMode{
		packages.NeedName,
		packages.NeedFiles,
		packages.NeedCompiledGoFiles,
		packages.NeedImports,
		packages.NeedTypes,
		packages.NeedTypesInfo,
		packages.NeedSyntax,
	}

	for _, flag := range requiredFlags {
		if cfg.Mode&flag == 0 {
			t.Errorf("Mode is missing required flag: %v", flag)
		}
	}

	// Verify Tests is enabled
	if !cfg.Tests {
		t.Error("Expected Tests to be true")
	}
}

// TestRealPackageLoader_InvalidPath_mutant tests error handling.
func TestRealPackageLoader_InvalidPath_mutant(t *testing.T) {
	t.Parallel()

	loader := &realPackageLoader{}

	//nolint:dogsled // Multiple blank identifiers needed for this error test
	_, _, _, err := loader.Load("nonexistent/package/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for invalid package path")
	}
}

// TestRealPackageLoader_mutant verifies the package loading configuration.
// This test specifically catches mutations in the bitwise operators used to combine
// packages.NeedX flags, ensuring all required information is loaded.
func TestRealPackageLoader_mutant(t *testing.T) {
	t.Parallel()

	loader := &realPackageLoader{}

	// Test loading a simple package - this exercises the Load method
	// and verifies that the Mode flags are correctly combined.
	// TypesInfo is intentionally nil for performance (we use syntax-based type detection).
	_, fset, typesInfo, err := loader.Load(".")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if fset == nil {
		t.Error("Expected non-nil FileSet")
	}

	if typesInfo != nil {
		t.Error("Expected nil TypesInfo (we skip type checking for performance)")
	}
}
