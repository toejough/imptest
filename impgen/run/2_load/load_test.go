package load_test

import (
	"os"
	"path/filepath"
	"testing"

	load "github.com/toejough/imptest/impgen/run/2_load"
)

func TestPackageDST_CurrentDirectory(t *testing.T) {
	// Not parallel - uses working directory

	// Create a temp directory with a Go file
	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "pkg.go")
	err := os.WriteFile(goFile, []byte("package mypkg\n\nfunc Hello() {}\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Test loading current directory
	files, fset, err := load.PackageDST(".")
	if err != nil {
		t.Errorf("PackageDST(\".\") unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Error("PackageDST(\".\") returned no files")
	}

	if fset == nil {
		t.Error("PackageDST(\".\") returned nil FileSet")
	}
}

func TestPackageDST_EmptyDirectory(t *testing.T) {
	// Not parallel - changes working directory

	// Create a temp directory with no Go files
	tmpDir := t.TempDir()

	// Create a txt file (not .go)
	txtFile := filepath.Join(tmpDir, "readme.txt")
	err := os.WriteFile(txtFile, []byte("not a go file\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Test loading current directory with no .go files
	_, _, err = load.PackageDST(".")
	if err == nil {
		t.Error("PackageDST for directory with no .go files should return error")
	}
}

func TestPackageDST_ExcludesTestFilesForExternalPackages(t *testing.T) {
	// Not parallel - changes working directory

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
	err = os.WriteFile(testFile, []byte("package extpkg\n\nimport \"testing\"\n\nfunc TestExtFunc(t *testing.T) {}\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write test go file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

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

func TestPackageDST_LocalSubdirPackage(t *testing.T) {
	// Not parallel - changes working directory

	// Create a temp directory with a subdirectory containing Go files
	tmpDir := t.TempDir()
	subPkgDir := filepath.Join(tmpDir, "subpkg")

	err := os.Mkdir(subPkgDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create subpkg dir: %v", err)
	}

	goFile := filepath.Join(subPkgDir, "sub.go")
	err = os.WriteFile(goFile, []byte("package subpkg\n\nfunc SubFunc() {}\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Test loading local subdirectory package (shadows any stdlib package with same name)
	files, fset, err := load.PackageDST("subpkg")
	if err != nil {
		t.Errorf("PackageDST(\"subpkg\") unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Error("PackageDST(\"subpkg\") returned no files")
	}

	if fset == nil {
		t.Error("PackageDST(\"subpkg\") returned nil FileSet")
	}
}

func TestPackageDST_MixedParseableAndUnparseable(t *testing.T) {
	// Not parallel - changes working directory

	// Create a temp directory with both parseable and unparseable Go files
	tmpDir := t.TempDir()

	// Create a valid Go file
	goodGoFile := filepath.Join(tmpDir, "good.go")
	err := os.WriteFile(goodGoFile, []byte("package mixed\n\nfunc GoodFunc() {}\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write good go file: %v", err)
	}

	// Create an invalid Go file
	badGoFile := filepath.Join(tmpDir, "bad.go")
	err = os.WriteFile(badGoFile, []byte("package mixed\n\nfunc incomplete(\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write bad go file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Test loading directory with mixed files - should succeed with parseable files
	files, fset, err := load.PackageDST(".")
	if err != nil {
		t.Errorf("PackageDST with mixed files unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	if fset == nil {
		t.Error("PackageDST returned nil FileSet")
	}
}

func TestPackageDST_NonexistentPackage(t *testing.T) {
	t.Parallel()

	// Test loading a non-existent package
	_, _, err := load.PackageDST("nonexistent/package/xyz123")
	if err == nil {
		t.Error("PackageDST for non-existent package should return error")
	}
}

func TestPackageDST_StdlibPackage(t *testing.T) {
	t.Parallel()

	// Test loading a standard library package
	files, fset, err := load.PackageDST("fmt")
	if err != nil {
		t.Errorf("PackageDST(\"fmt\") unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Error("PackageDST(\"fmt\") returned no files")
	}

	if fset == nil {
		t.Error("PackageDST(\"fmt\") returned nil FileSet")
	}
}

func TestPackageDST_UnparseableFile(t *testing.T) {
	// Not parallel - changes working directory

	// Create a temp directory with an unparseable Go file
	tmpDir := t.TempDir()

	// Create an invalid Go file
	badGoFile := filepath.Join(tmpDir, "bad.go")
	err := os.WriteFile(badGoFile, []byte("package broken\n\nfunc incomplete(\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write bad go file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Test loading directory with only unparseable files should error
	_, _, err = load.PackageDST(".")
	if err == nil {
		t.Error("PackageDST for directory with only unparseable .go files should return error")
	}
}

func TestResolveLocalPackagePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		importPath string
		setup      func(t *testing.T) string // returns temp dir if needed
		wantSame   bool                      // expect result == importPath
	}{
		{
			name:       "dot path unchanged",
			importPath: ".",
			wantSame:   true,
		},
		{
			name:       "absolute path unchanged",
			importPath: "/some/absolute/path",
			wantSame:   true,
		},
		{
			name:       "path with slash unchanged",
			importPath: "github.com/example/pkg",
			wantSame:   true,
		},
		{
			name:       "nonexistent local dir returns original",
			importPath: "nonexistent_pkg_xyz123",
			wantSame:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.setup != nil {
				tt.setup(t)
			}

			result := load.ResolveLocalPackagePath(tt.importPath)

			if tt.wantSame && result != tt.importPath {
				t.Errorf("expected %q to remain unchanged, got %q", tt.importPath, result)
			}
		})
	}
}

func TestResolveLocalPackagePath_DirWithoutGoFiles(t *testing.T) {
	// Not parallel - changes working directory

	// Create a temp directory with a subdirectory containing NO .go files
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "emptydir")

	err := os.Mkdir(pkgDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	// Create a non-.go file in the directory
	txtFile := filepath.Join(pkgDir, "readme.txt")

	err = os.WriteFile(txtFile, []byte("not a go file\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Should return original since no .go files
	result := load.ResolveLocalPackagePath("emptydir")
	if result != "emptydir" {
		t.Errorf("expected 'emptydir', got %q", result)
	}
}

func TestResolveLocalPackagePath_LocalDir(t *testing.T) {
	// Not parallel - changes working directory

	// Create a temp directory with a subdirectory containing a .go file
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "mypkg")

	err := os.Mkdir(pkgDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	// Create a .go file in the package
	goFile := filepath.Join(pkgDir, "pkg.go")

	err = os.WriteFile(goFile, []byte("package mypkg\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	result := load.ResolveLocalPackagePath("mypkg")

	// Resolve symlinks for comparison (e.g., /var -> /private/var on macOS)
	expectedResolved, _ := filepath.EvalSymlinks(pkgDir)
	resultResolved, _ := filepath.EvalSymlinks(result)

	if resultResolved != expectedResolved {
		t.Errorf("expected %q, got %q", expectedResolved, resultResolved)
	}
}
