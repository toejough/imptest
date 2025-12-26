package run //nolint:testpackage

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func TestExtractPackageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"pkg.Name", "pkg"},
		{"Name", ""},
		{"a.b.c", "a"},
	}

	for _, testCase := range tests {
		got := extractPackageName(testCase.input)
		if got != testCase.want {
			t.Errorf("extractPackageName(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}

//nolint:paralleltest // Can't use t.Parallel with t.Chdir
func TestGetFullImportPath(t *testing.T) {
	// Note: Cannot use t.Parallel() because t.Chdir() changes process-wide state

	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	testPkg := filepath.Join(tmpDir, "testpkg")

	err := os.Mkdir(testPkg, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test package directory: %v", err)
	}

	// Create a simple go file in the test package
	goFile := filepath.Join(testPkg, "test.go")

	err = os.WriteFile(goFile, []byte("package testpkg\n"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test go file: %v", err)
	}

	// Create go.mod to make it a valid module
	goMod := filepath.Join(tmpDir, "go.mod")

	goModContent := []byte("module example.com/test\n\ngo 1.21\n")

	err = os.WriteFile(goMod, goModContent, 0o600)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Change to temp directory for test
	t.Chdir(tmpDir)

	tests := []struct {
		name        string
		pkgName     string
		wantContain string // Check if result contains this substring
		wantErr     bool
	}{
		{
			name:        "valid local package",
			pkgName:     "testpkg",
			wantContain: "example.com/test/testpkg",
			wantErr:     false,
		},
		{
			name:    "nonexistent package",
			pkgName: "nonexistent",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := getFullImportPath(testCase.pkgName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("getFullImportPath(%q) error = %v, wantErr %v", testCase.pkgName, err, testCase.wantErr)
				return
			}

			if !testCase.wantErr && testCase.wantContain != "" {
				if !strings.Contains(got, testCase.wantContain) {
					t.Errorf("getFullImportPath(%q) = %q, want to contain %q", testCase.pkgName, got, testCase.wantContain)
				}
			}
		})
	}
}

//nolint:paralleltest // Can't use t.Parallel with t.Chdir
func TestGetImportPathFromFiles(t *testing.T) {
	// Note: Cannot use t.Parallel() because t.Chdir() changes process-wide state
	_, fset, dstFile := setupTestPackage(t)

	tests := []struct {
		name        string
		wantContain string
		wantErr     bool
	}{
		{
			name:        "valid package with parsed files",
			wantContain: "example.com/test/testpkg",
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := getImportPathFromFiles([]*dst.File{dstFile}, fset, "testpkg")
			if (err != nil) != testCase.wantErr {
				t.Errorf("getImportPathFromFiles() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}

			if !testCase.wantErr && testCase.wantContain != "" {
				if !strings.Contains(got, testCase.wantContain) {
					t.Errorf("getImportPathFromFiles() = %q, want to contain %q", got, testCase.wantContain)
				}
			}
		})
	}
}

func setupTestPackage(t *testing.T) (string, *token.FileSet, *dst.File) {
	t.Helper()

	tmpDir := t.TempDir()
	testPkg := filepath.Join(tmpDir, "testpkg")

	err := os.Mkdir(testPkg, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test package directory: %v", err)
	}

	goFile := filepath.Join(testPkg, "test.go")
	goCode := []byte("package testpkg\n\nfunc Hello() string { return \"hello\" }\n")

	err = os.WriteFile(goFile, goCode, 0o600)
	if err != nil {
		t.Fatalf("Failed to create test go file: %v", err)
	}

	goMod := filepath.Join(tmpDir, "go.mod")
	goModContent := []byte("module example.com/test\n\ngo 1.21\n")

	err = os.WriteFile(goMod, goModContent, 0o600)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	dstFile, err := dec.ParseFile(goFile, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}

	t.Chdir(tmpDir)

	return tmpDir, fset, dstFile
}
