//nolint:testpackage // Fuzz tests for internal functions
package detect

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// FuzzExtractPackageName tests ExtractPackageName with coverage-guided fuzzing.
// Uses rapid.MakeFuzz for smart input generation.
func FuzzExtractPackageName(f *testing.F) {
	f.Fuzz(rapid.MakeFuzz(func(t *rapid.T) {
		expect := gomega.NewWithT(t)

		// Generate qualified names with dots
		input := rapid.OneOf(
			rapid.Just(""),
			rapid.StringMatching(`[a-z]{1,10}`),
			rapid.StringMatching(`[a-z]{1,10}\.[A-Z][a-zA-Z]{0,10}`),
			rapid.StringMatching(`[a-z]{1,10}\.[A-Z][a-zA-Z]{0,10}\.[A-Z][a-zA-Z]{0,10}`),
			rapid.String(),
		).Draw(t, "input")

		result := ExtractPackageName(input)

		// Property: if result is non-empty, input must start with "result."
		if result != "" {
			expect.Expect(input).To(gomega.HavePrefix(result + "."))
		}

		// Property: if input has a dot (not at start), result should be non-empty
		if strings.Contains(input, ".") && !strings.HasPrefix(input, ".") {
			expect.Expect(result).NotTo(gomega.BeEmpty())
		}
	}))
}

// FuzzIsStdlibPackage tests IsStdlibPackage with random inputs.
// Property: Function never panics on any string input.
func FuzzIsStdlibPackage(f *testing.F) {
	f.Fuzz(rapid.MakeFuzz(func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")

		// Should never panic
		_ = IsStdlibPackage(input)
	}))
}
