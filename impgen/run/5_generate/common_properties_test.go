//nolint:testpackage // Tests internal functions
package generate

import (
	"testing"
	"unicode"

	"pgregory.net/rapid"
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

// TestIsBuiltinType_NonBuiltins_Property proves random identifiers that aren't
// builtins return false.
func TestIsBuiltinType_NonBuiltins_Property(t *testing.T) {
	t.Parallel()

	// Build lookup set for efficient checking
	builtinSet := make(map[string]bool, len(allBuiltinTypes))
	for _, b := range allBuiltinTypes {
		builtinSet[b] = true
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Generate random Go identifier-like strings
		name := rapid.StringMatching(`[A-Za-z_][A-Za-z0-9_]{0,20}`).Draw(rt, "name")

		// Skip if it happens to be a builtin
		if builtinSet[name] {
			rt.Skip("generated a builtin")
		}

		if isBuiltinType(name) {
			rt.Fatalf("isBuiltinType(%q) should be false for non-builtin", name)
		}
	})
}

// TestIsExportedIdent_Empty returns false for empty string.
func TestIsExportedIdent_Empty(t *testing.T) {
	t.Parallel()

	if isExportedIdent("") {
		t.Error("isExportedIdent(\"\") should be false")
	}
}

// TestIsExportedIdent_Lowercase_Property proves identifiers starting with
// lowercase are unexported.
func TestIsExportedIdent_Lowercase_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate identifier starting with lowercase letter
		name := rapid.StringMatching(`[a-z][A-Za-z0-9_]{0,20}`).Draw(rt, "name")

		if isExportedIdent(name) {
			rt.Fatalf("isExportedIdent(%q) should be false for lowercase start", name)
		}
	})
}

// TestIsExportedIdent_Underscore_Property proves identifiers starting with
// underscore are unexported.
func TestIsExportedIdent_Underscore_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate identifier starting with underscore
		name := rapid.StringMatching(`_[A-Za-z0-9_]{0,20}`).Draw(rt, "name")

		if isExportedIdent(name) {
			rt.Fatalf("isExportedIdent(%q) should be false for underscore start", name)
		}
	})
}

// TestIsExportedIdent_Unicode_Property proves Unicode uppercase letters
// are correctly identified as exported.
func TestIsExportedIdent_Unicode_Property(t *testing.T) {
	t.Parallel()

	// Sample of Unicode uppercase letters that are valid Go identifier starts
	uppercaseRunes := []rune{'A', 'Z', 'Ä', 'Ö', 'Ü', 'É', 'Ñ'}

	for _, r := range uppercaseRunes {
		name := string(r) + "test"
		expected := unicode.IsUpper(r)

		if isExportedIdent(name) != expected {
			t.Errorf("isExportedIdent(%q) = %v, want %v", name, !expected, expected)
		}
	}
}

// TestIsExportedIdent_Uppercase_Property proves identifiers starting with
// uppercase are exported.
func TestIsExportedIdent_Uppercase_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate identifier starting with uppercase letter
		name := rapid.StringMatching(`[A-Z][A-Za-z0-9_]{0,20}`).Draw(rt, "name")

		if !isExportedIdent(name) {
			rt.Fatalf("isExportedIdent(%q) should be true for uppercase start", name)
		}
	})
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
