// Package match provides matchers for use with imptest's ArgsShould, ReturnsShould, and PanicShould.
// This package is designed to be dot-imported alongside gomega matchers:
//
//	import (
//	    . "github.com/onsi/gomega"
//	    . "github.com/toejough/imptest/match"
//	)
//
//	expect.Add.ArgsShould(BeNumerically(">", 0), BeAny).Return(42)
package match

import (
	"errors"
	"fmt"
)

// errTypeMismatch is a sentinel error for type assertion failures.
var errTypeMismatch = errors.New("type mismatch")

// Matcher defines the interface for flexible value matching.
// Compatible with gomega.GomegaMatcher via duck typing - any type
// implementing Match and FailureMessage will work.
type Matcher interface {
	Match(actual any) (success bool, err error)
	FailureMessage(actual any) string
}

// BeAny is a matcher that matches any value.
// Useful when you don't care about a particular argument or return value.
//
//nolint:gochecknoglobals // Intentional exported constant-like value
var BeAny Matcher = anyMatcher{}

// Satisfy returns a matcher that uses a predicate function to check for a match.
// The predicate should return nil if the value matches, or an error describing
// the mismatch if it does not.
//
// Example:
//
//	expect.Add.ArgsShould(Satisfy(func(x int) error {
//	    if x < 0 { return fmt.Errorf("expected positive, got %d", x) }
//	    return nil
//	}))
func Satisfy[T any](predicate func(T) error) Matcher {
	return &satisfyMatcher[T]{predicate: predicate}
}

// anyMatcher is the implementation of the BeAny matcher.
type anyMatcher struct{}

// FailureMessage returns an empty string since BeAny always matches.
func (anyMatcher) FailureMessage(any) string {
	return ""
}

// Match always returns true - matches any value.
func (anyMatcher) Match(any) (bool, error) {
	return true, nil
}

type satisfyMatcher[T any] struct {
	predicate func(T) error
	lastErr   error
}

func (m *satisfyMatcher[T]) FailureMessage(actual any) string {
	if m.lastErr != nil {
		return fmt.Sprintf("value %v does not satisfy predicate: %v", actual, m.lastErr)
	}

	return fmt.Sprintf("value %v does not satisfy predicate", actual)
}

func (m *satisfyMatcher[T]) Match(actual any) (bool, error) {
	val, ok := actual.(T)

	if !ok {
		return false, fmt.Errorf("%w: expected %T, got %T", errTypeMismatch, *new(T), actual)
	}

	m.lastErr = m.predicate(val)

	return m.lastErr == nil, nil
}
