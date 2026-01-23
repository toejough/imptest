//nolint:testpackage // Tests internal functions
package generate

import (
	"testing"
)

// TestIsBuiltinType_AllBuiltins proves all known builtins return true.
func TestIsBuiltinType_AllBuiltins(t *testing.T) {
	t.Parallel()

	for _, builtin := range allBuiltinTypes {
		if !isBuiltinType(builtin) {
			t.Errorf("isBuiltinType(%q) should be true", builtin)
		}
	}
}

// TestIsExportedIdent_Empty returns false for empty string.
func TestIsExportedIdent_Empty(t *testing.T) {
	t.Parallel()

	if isExportedIdent("") {
		t.Error("isExportedIdent(\"\") should be false")
	}
}

// unexported variables.
var (
	//nolint:gochecknoglobals // Test data constant
	allBuiltinTypes = []string{
		"bool", "byte", "complex64", "complex128", "error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64", "rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"comparable", "any",
	}
)
