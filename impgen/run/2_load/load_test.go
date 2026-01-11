//nolint:paralleltest // Tests use t.Chdir which is incompatible with t.Parallel
package load_test

import (
	"os"
	"path/filepath"
	"testing"

	load "github.com/toejough/imptest/impgen/run/2_load"
)

// Create a temp directory with a Go file

// Change to temp directory

// Test loading current directory

// Create a temp directory with no Go files

// Create a txt file (not .go)

// Change to temp directory

// Test loading current directory with no .go files

func TestPackageDST_ExcludesTestFilesForExternalPackages(t *testing.T) {
	// Create a temp directory with both regular and test Go files
	tmpDir := t.TempDir()
	subPkgDir := filepath.Join(tmpDir, "extpkg")

	err := os.Mkdir(subPkgDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create extpkg dir: %v", err)
	}

	// Create a regular Go file
	regularFile := filepath.Join(subPkgDir, "ext.go")

	err = os.WriteFile(regularFile, []byte("package extpkg\n\nfunc ExtFunc() {}\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write regular go file: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(subPkgDir, "ext_test.go")
	testContent := "package extpkg\n\nimport \"testing\"\n\nfunc TestExtFunc(t *testing.T) {}\n"

	err = os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("failed to write test go file: %v", err)
	}

	// Change to temp directory
	t.Chdir(tmpDir)

	// Load as a subdirectory package (not ".") - should exclude test files
	files, _, err := load.PackageDST("extpkg")
	if err != nil {
		t.Errorf("PackageDST(\"extpkg\") unexpected error: %v", err)
	}

	// Should only have 1 file (the regular one, not the test)
	if len(files) != 1 {
		t.Errorf("expected 1 file (excluding test), got %d", len(files))
	}
}

// Create a temp directory with a subdirectory containing Go files

// Change to temp directory

// Test loading local subdirectory package (shadows any stdlib package with same name)

// Create a temp directory with both parseable and unparseable Go files

// Create a valid Go file

// Create an invalid Go file

// Change to temp directory

// Test loading directory with mixed files - should succeed with parseable files

func TestPackageDST_NonexistentPackage(t *testing.T) {
	t.Parallel()

	// Test loading a non-existent package
	_, _, err := load.PackageDST("nonexistent/package/xyz123")
	if err == nil {
		t.Error("PackageDST for non-existent package should return error")
	}
}

// Test loading a standard library package

// Create a temp directory with an unparseable Go file

// Create an invalid Go file

// Change to temp directory

// Test loading directory with only unparseable files should error

// returns temp dir if needed
// expect result == importPath

// Create a temp directory with a subdirectory containing NO .go files

// Create a non-.go file in the directory

// Change to temp directory

// Should return original since no .go files

// Create a temp directory with a subdirectory containing a .go file

// Create a .go file in the package

// Change to temp directory

// Resolve symlinks for comparison (e.g., /var -> /private/var on macOS)
