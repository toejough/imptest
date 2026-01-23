//nolint:testpackage // Fuzz tests for internal functions
package generate

import (
	"testing"

	"pgregory.net/rapid"
)

// FuzzIsBuiltinType tests isBuiltinType with coverage-guided fuzzing.
// Uses rapid.MakeFuzz to combine property-based generation with Go's fuzzer.
func FuzzIsBuiltinType(f *testing.F) {
	f.Fuzz(rapid.MakeFuzz(func(t *rapid.T) {
		// Decide whether to test a known builtin or random input
		testBuiltin := rapid.Bool().Draw(t, "testBuiltin")

		if testBuiltin {
			// Property: all known builtins should return true
			input := rapid.SampledFrom(allBuiltinTypes).Draw(t, "builtin")
			if !isBuiltinType(input) {
				t.Fatalf("isBuiltinType(%q) should be true", input)
			}
		} else {
			// Property: should never panic on any input
			input := rapid.OneOf(
				rapid.StringMatching(`[A-Za-z_][A-Za-z0-9_]{0,20}`),
				rapid.Just(""),
				rapid.String(),
			).Draw(t, "input")
			_ = isBuiltinType(input)
		}
	}))
}

// FuzzIsExportedIdent tests isExportedIdent with coverage-guided fuzzing.
func FuzzIsExportedIdent(f *testing.F) {
	// Seed corpus
	for _, s := range []string{
		"", "a", "A", "foo", "Foo", "_foo", "_Foo", "HTTPHandler",
		"myFunc", "MyFunc", "123", "a1", "A1",
	} {
		f.Add([]byte(s))
	}

	f.Fuzz(rapid.MakeFuzz(func(t *rapid.T) {
		input := rapid.OneOf(
			rapid.StringMatching(`[A-Z][A-Za-z0-9_]{0,20}`), // Exported
			rapid.StringMatching(`[a-z][A-Za-z0-9_]{0,20}`), // Unexported
			rapid.StringMatching(`_[A-Za-z0-9_]{0,20}`),     // Underscore prefix
			rapid.Just(""),
			rapid.String(),
		).Draw(t, "input")

		// Should never panic
		result := isExportedIdent(input)

		// Property: empty string is not exported
		if input == "" && result {
			t.Fatal("empty string should not be exported")
		}

		// Property: strings starting with uppercase ASCII are exported
		if len(input) > 0 && input[0] >= 'A' && input[0] <= 'Z' && !result {
			t.Fatalf("isExportedIdent(%q) should be true for uppercase start", input)
		}
	}))
}

// FuzzResultDataBuilder tests ResultDataBuilder.Build with random inputs.
func FuzzResultDataBuilder(f *testing.F) {
	f.Fuzz(rapid.MakeFuzz(func(t *rapid.T) {
		// Generate 0-5 result types
		resultTypes := rapid.SliceOfN(
			rapid.StringMatching(`[a-z*\[\]]{0,20}`),
			0, 5,
		).Draw(t, "resultTypes")

		// Generate a valid prefix
		prefix := rapid.StringMatching(`[a-z]{1,5}`).Draw(t, "prefix")
		if prefix == "" {
			prefix = "r"
		}

		// Should never panic
		builder := &ResultDataBuilder{
			ResultTypes: resultTypes,
			VarPrefix:   prefix,
		}

		result := builder.Build()

		// Property: HasResults matches whether we have types
		expectedHasResults := len(resultTypes) > 0
		if result.HasResults != expectedHasResults {
			t.Fatalf("HasResults = %v, expected %v for types %v",
				result.HasResults, expectedHasResults, resultTypes)
		}

		// Property: Field count matches result type count
		if len(result.Fields) != len(resultTypes) {
			t.Fatalf("len(Fields) = %d, expected %d", len(result.Fields), len(resultTypes))
		}
	}))
}
