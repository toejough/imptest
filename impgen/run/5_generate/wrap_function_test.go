package generate

import (
	"testing"

	"github.com/dave/dst"
)

func TestTargetGenerator_hasMultipleResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		results  *dst.FieldList
		expected bool
	}{
		{
			name:     "nil results",
			results:  nil,
			expected: false,
		},
		{
			name:     "empty results",
			results:  &dst.FieldList{List: []*dst.Field{}},
			expected: false,
		},
		{
			name: "single unnamed result",
			results: &dst.FieldList{
				List: []*dst.Field{
					{Type: &dst.Ident{Name: "int"}},
				},
			},
			expected: false,
		},
		{
			name: "single named result",
			results: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{{Name: "x"}},
						Type:  &dst.Ident{Name: "int"},
					},
				},
			},
			expected: false,
		},
		{
			name: "multiple result fields",
			results: &dst.FieldList{
				List: []*dst.Field{
					{Type: &dst.Ident{Name: "int"}},
					{Type: &dst.Ident{Name: "error"}},
				},
			},
			expected: true,
		},
		{
			name: "single field with multiple names",
			results: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{{Name: "a"}, {Name: "b"}},
						Type:  &dst.Ident{Name: "int"},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := &targetGenerator{
				funcDecl: &dst.FuncDecl{
					Type: &dst.FuncType{
						Results: tt.results,
					},
				},
			}

			if got := gen.hasMultipleResults(); got != tt.expected {
				t.Errorf("hasMultipleResults() = %v, want %v", got, tt.expected)
			}
		})
	}
}
