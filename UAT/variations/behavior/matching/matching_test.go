package matching_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega" //nolint:revive // Dot import intentional for Gomega matcher DSL

	"github.com/toejough/imptest"
	matching "github.com/toejough/imptest/UAT/variations/behavior/matching"
)

//go:generate impgen matching.ComplexService --dependency

// TestAdvancedMatching demonstrates how to verify complex structures where only a subset
// of fields matter.
//
// Key Requirements Met:
//  1. Partial Matching: Decouple tests from irrelevant or transient data (like timestamps)
//     using custom predicates or the Any() matcher.
//  2. Expressive Failures: Provide descriptive failure messages through error-returning
//     Satisfies predicates, making it easy to identify exactly why a match failed.
func TestAdvancedMatching(t *testing.T) {
	t.Parallel()

	mock, imp := MockComplexService(t)

	go matching.UseService(mock, "hello world")

	// Use Match with matchers to validate only the parts of the input we care about.
	// Requirement: We want to match the Payload exactly, ensure the ID is valid (positive),
	// but ignore the Timestamp because it is non-deterministic.
	imp.Process.Match(imptest.Satisfies(func(data matching.Data) error {
		if data.Payload != "hello world" {
			return fmt.Errorf("expected payload 'hello world', got %q", data.Payload)
		}

		if data.ID <= 0 {
			return fmt.Errorf("expected ID > 0, got %d", data.ID)
		}

		// We could use imptest.Any if this was a separate argument, but since it's
		// a field in a struct, we simply don't validate it in this predicate.
		return nil
	})).Return(true)
}

// TestGomegaIntegration demonstrates that imptest is compatible with third-party matchers.
//
// Key Requirements Met:
//  1. Extensibility: Use familiar libraries like Gomega without imptest having a hard
//     dependency on them.
//  2. Duck Typing: Any object implementing Match(any) (bool, error) and FailureMessage(any) string
//     can be used directly in Match or ExpectReturnsMatch.
func TestGomegaIntegration(t *testing.T) {
	t.Parallel()

	mock, imp := MockComplexService(t)

	go matching.UseService(mock, "gomega rules")

	// We use Gomega's And and HaveField matchers to verify the struct state.
	// This demonstrates how imptest's flexible matcher interface allows for
	// highly readable and powerful expectations.
	imp.Process.Match(
		And(
			HaveField("Payload", Equal("gomega rules")),
			HaveField("ID", BeNumerically(">", 100)),
			HaveField("Timestamp", BeNumerically(">", 0)),
		),
	).Return(true)
}

// TestMatchAny demonstrates the use of the Any() matcher for arguments we don't care about.
//
// Key Requirements Met:
// 1. Clarity: Explicitly signal that a particular value is not relevant to the test's intent.
func TestMatchAny(t *testing.T) {
	t.Parallel()

	mock, imp := MockComplexService(t)

	// In this scenario, we don't care about the input data at all, only that the call happened.
	go mock.Process(matching.Data{ID: 999})

	imp.Process.Match(imptest.Any).Return(true)
}
