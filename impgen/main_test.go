package main

import (
	"testing"
)

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
