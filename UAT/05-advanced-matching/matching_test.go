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

	go matching.UseService(mock.Mock, "hello world")

	// Use ExpectArgsShould with matchers.
	// We want to match the Payload exactly, but use a predicate for ID
	// and Any() for the Timestamp because we don't care about the exact value.
	// Use a predicate for more complex logic.
	mock.ExpectCallIs.Process().ExpectArgsShould(imptest.Satisfies(func(data matching.Data) bool {
		return data.Payload == "hello world" && data.ID > 0
	})).InjectResult(true)
}

// TODO: not sure this actually adds anything beyond the above test? One of these could use Any() to show that.
// TODO: add an actual test that uses gomega matchers?
// TODO: make satisfies take a predicate that returns error, so we can give better failure messages?
func TestSimplifiedMatching(t *testing.T) {
	t.Parallel()

	mock := NewComplexServiceImp(t)

	go mock.Mock.Process(matching.Data{ID: 1, Payload: "a", Timestamp: 2})

	// Match only part of the struct using Satisfies
	mock.ExpectCallIs.Process().ExpectArgsShould(imptest.Satisfies(func(d matching.Data) bool {
		return d.Payload == "a"
	})).InjectResult(true)
}
