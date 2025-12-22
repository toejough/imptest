//nolint:testpackage // Need same package to test unexported isComparableExpr
package run

import (
	"testing"
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

	for _, tt := range tests {
		got := extractPackageName(tt.input)
		if got != tt.want {
			t.Errorf("extractPackageName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
