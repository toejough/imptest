package matching_test

import (
	"testing"

	matching "github.com/toejough/imptest/UAT/05-advanced-matching"
	"github.com/toejough/imptest/imptest"
)

//go:generate go run ../../impgen/main.go matching.ComplexService --name ComplexServiceImp

func TestAdvancedMatching(t *testing.T) {
	t.Parallel()

	mock := NewComplexServiceImp(t)

	go matching.UseService(mock.Mock, "important message")

	// Use ExpectArgsShould with matchers.
	// We want to match the Payload exactly, but use a predicate for ID
	// and Any() for the Timestamp because we don't care about the exact value.
	mock.ExpectCallIs.Process().ExpectArgsShould(imptest.Satisfies(func(d matching.Data) bool {
		// Custom matching logic:
		if d.ID != 123 {
			return false
		}

		if d.Payload != "important message" {
			return false
		}
		// Timestamp must be positive
		return d.Timestamp > 0
	})).InjectResult(true)
}

func TestSimplifiedMatching(t *testing.T) {
	t.Parallel()

	mock := NewComplexServiceImp(t)

	go mock.Mock.Process(matching.Data{ID: 1, Payload: "a", Timestamp: 2})

	// Match only part of the struct using Satisfies
	mock.ExpectCallIs.Process().ExpectArgsShould(imptest.Satisfies(func(d matching.Data) bool {
		return d.Payload == "a"
	})).InjectResult(true)
}
