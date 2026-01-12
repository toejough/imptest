//nolint:funlen // Test file
package detect_test

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

func TestCollectStructMethods(t *testing.T) {
	t.Parallel()

	// Create temp file with a struct and methods
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "mystruct.go")

	fileContent := `package testpkg

type MyStruct struct {
	value int
}

func (m *MyStruct) GetValue() int {
	return m.value
}

func (m *MyStruct) SetValue(v int) {
	m.value = v
}

func (m MyStruct) String() string {
	return "mystruct"
}

// Not a method of MyStruct
func HelperFunc() {}
`

	err := os.WriteFile(tmpFile, []byte(fileContent), 0o600)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Parse the file using DST
	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	dstFile, err := dec.ParseFile(tmpFile, nil, 0)
	if err != nil {
		t.Fatalf("failed to parse temp file: %v", err)
	}

	// Test CollectStructMethods
	methods := detect.CollectStructMethods([]*dst.File{dstFile}, fset, "MyStruct")

	// Should have 3 methods
	if len(methods) != 3 {
		t.Errorf("CollectStructMethods() returned %d methods, want 3", len(methods))
	}

	// Check specific methods exist
	expectedMethods := []string{"GetValue", "SetValue", "String"}
	for _, name := range expectedMethods {
		if _, ok := methods[name]; !ok {
			t.Errorf("CollectStructMethods() missing method %q", name)
		}
	}
}

func TestExtractPackageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		qualifiedName string
		want          string
	}{
		{
			name:          "qualified name returns package",
			qualifiedName: "http.Request",
			want:          "http",
		},
		{
			name:          "multiple dots returns first part",
			qualifiedName: "pkg.Type.Method",
			want:          "pkg",
		},
		{
			name:          "no dot returns empty string",
			qualifiedName: "SomeType",
			want:          "",
		},
		{
			name:          "empty string returns empty",
			qualifiedName: "",
			want:          "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := detect.ExtractPackageName(testCase.qualifiedName)
			if got != testCase.want {
				t.Errorf(
					"ExtractPackageName(%q) = %q, want %q",
					testCase.qualifiedName,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestInferImportPathFromTestFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fileContent string
		pkgName     string
		want        string
		wantErr     bool
	}{
		{
			name: "named import match",
			fileContent: `package foo_test

import (
	renamed "github.com/example/original"
)
`,
			pkgName: "renamed",
			want:    "github.com/example/original",
			wantErr: false,
		},
		{
			name: "suffix match",
			fileContent: `package foo_test

import (
	"github.com/example/mypkg"
)
`,
			pkgName: "mypkg",
			want:    "github.com/example/mypkg",
			wantErr: false,
		},
		{
			name: "exact match",
			fileContent: `package foo_test

import (
	"fmt"
)
`,
			pkgName: "fmt",
			want:    "fmt",
			wantErr: false,
		},
		{
			name: "package not found",
			fileContent: `package foo_test

import (
	"fmt"
)
`,
			pkgName: "nothere",
			want:    "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create temp file with test content
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test_file.go")

			err := os.WriteFile(tmpFile, []byte(testCase.fileContent), 0o600)
			if err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			got, err := detect.InferImportPathFromTestFile(tmpFile, testCase.pkgName)
			if (err != nil) != testCase.wantErr {
				t.Errorf(
					"InferImportPathFromTestFile() error = %v, wantErr %v",
					err,
					testCase.wantErr,
				)

				return
			}

			if got != testCase.want {
				t.Errorf("InferImportPathFromTestFile() = %v, want %v", got, testCase.want)
			}
		})
	}
}
