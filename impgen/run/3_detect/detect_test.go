package detect_test

import (
	"os"
	"path/filepath"
	"testing"

	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := detect.ExtractPackageName(tt.qualifiedName)
			if got != tt.want {
				t.Errorf("ExtractPackageName(%q) = %q, want %q", tt.qualifiedName, got, tt.want)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp file with test content
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test_file.go")

			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0o600)
			if err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			got, err := detect.InferImportPathFromTestFile(tmpFile, tt.pkgName)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferImportPathFromTestFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("InferImportPathFromTestFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInferImportPathFromTestFile_EmptyPath(t *testing.T) {
	t.Parallel()

	_, err := detect.InferImportPathFromTestFile("", "pkg")
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestInferImportPathFromTestFile_InvalidFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.go")

	err := os.WriteFile(tmpFile, []byte("this is not valid go code {{{"), 0o600)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err = detect.InferImportPathFromTestFile(tmpFile, "pkg")
	if err == nil {
		t.Error("expected error for invalid file, got nil")
	}
}

func TestInferImportPathFromTestFile_NonexistentFile(t *testing.T) {
	t.Parallel()

	_, err := detect.InferImportPathFromTestFile("/nonexistent/path/file.go", "pkg")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
