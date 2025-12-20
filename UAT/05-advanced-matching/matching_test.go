package matching_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega" //nolint:revive
	matching "github.com/toejough/imptest/UAT/05-advanced-matching"
	"github.com/toejough/imptest/imptest"
)

//go:generate go run ../../impgen/main.go matching.ComplexService --name ComplexServiceImp

func TestAdvancedMatching(t *testing.T) {
	t.Parallel()

	imp := NewComplexServiceImp(t)

	go matching.UseService(imp.Mock, "hello world")

	// Use ExpectArgsShould with matchers.
	// We want to match the Payload exactly, but use a predicate for ID
	// and Any() for the Timestamp because we don't care about the exact value.
	// Use a predicate for more complex logic.
	imp.ExpectCallIs.Process().ExpectArgsShould(imptest.Satisfies(func(data matching.Data) error {
		if data.Payload != "hello world" {
			return fmt.Errorf("expected payload 'hello world', got %q", data.Payload)
		}

		if data.ID <= 0 {
			return fmt.Errorf("expected ID > 0, got %d", data.ID)
		}

		return nil
	})).InjectResult(true)
}

// TODO: not sure this actually adds anything beyond the above test?

func TestGomegaIntegration(t *testing.T) {
	t.Parallel()

	imp := NewComplexServiceImp(t)

	go matching.UseService(imp.Mock, "gomega rules")

	// imptest is compatible with third-party matchers like Gomega via duck typing.
	// Any object implementing Match(any) (bool, error) and FailureMessage(any) string works.
	imp.ExpectCallIs.Process().ExpectArgsShould(
		And(
			HaveField("Payload", Equal("gomega rules")),
			HaveField("ID", BeNumerically(">", 100)),
		),
	).InjectResult(true)
}

func TestSimplifiedMatching(t *testing.T) {
	t.Parallel()

	imp := NewComplexServiceImp(t)

	go imp.Mock.Process(matching.Data{ID: 1, Payload: "a", Timestamp: 2})

	// Match only part of the struct using Satisfies
	imp.ExpectCallIs.Process().ExpectArgsShould(imptest.Satisfies(func(d matching.Data) error {
		if d.Payload != "a" {
			return fmt.Errorf("expected payload 'a', got %q", d.Payload)
		}

		return nil
	})).InjectResult(true)
}
