package run //nolint:testpackage // Testing unexported function

import (
	"testing"

	"github.com/dave/dst"
)

//nolint:funlen // Table-driven test with multiple cases
func TestGetDotImportPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		files    []*dst.File
		expected []string
	}{
		{
			name:     "no imports",
			files:    []*dst.File{{Imports: nil}},
			expected: nil,
		},
		{
			name: "no dot imports",
			files: []*dst.File{{
				Imports: []*dst.ImportSpec{
					{Path: &dst.BasicLit{Value: `"fmt"`}},
					{Path: &dst.BasicLit{Value: `"testing"`}},
				},
			}},
			expected: nil,
		},
		{
			name: "single dot import",
			files: []*dst.File{{
				Imports: []*dst.ImportSpec{
					{Name: &dst.Ident{Name: "."}, Path: &dst.BasicLit{Value: `"github.com/example/helpers"`}},
				},
			}},
			expected: []string{"github.com/example/helpers"},
		},
		{
			name: "multiple dot imports",
			files: []*dst.File{{
				Imports: []*dst.ImportSpec{
					{Name: &dst.Ident{Name: "."}, Path: &dst.BasicLit{Value: `"github.com/example/helpers"`}},
					{Path: &dst.BasicLit{Value: `"fmt"`}},
					{Name: &dst.Ident{Name: "."}, Path: &dst.BasicLit{Value: `"github.com/example/utils"`}},
				},
			}},
			expected: []string{"github.com/example/helpers", "github.com/example/utils"},
		},
		{
			name: "aliased import (not dot)",
			files: []*dst.File{{
				Imports: []*dst.ImportSpec{
					{Name: &dst.Ident{Name: "helpers"}, Path: &dst.BasicLit{Value: `"github.com/example/helpers"`}},
				},
			}},
			expected: nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := getDotImportPaths(testCase.files)
			if len(result) != len(testCase.expected) {
				t.Fatalf("expected %d imports, got %d", len(testCase.expected), len(result))
			}

			for i, expected := range testCase.expected {
				if result[i] != expected {
					t.Errorf("import %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}
