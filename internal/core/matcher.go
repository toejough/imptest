package core

import (
	"errors"
	"fmt"
	"reflect"
)

// Exported variables.
var (
	// BeAny matches any value - useful for "don't care" arguments.
	BeAny Matcher = anyMatcher{} //nolint:gochecknoglobals // BeAny is an exported sentinel value
)

type Matcher interface {
	Match(actual any) (success bool, err error)
	FailureMessage(actual any) string
}

// Satisfy returns a matcher that uses a predicate function to check for a match.
// The predicate should return nil if the value matches, or an error describing
// the mismatch if it does not.
func Satisfy[T any](predicate func(T) error) Matcher {
	return &satisfyMatcher[T]{predicate: predicate}
}

// unexported variables.
var (
	errTypeMismatch = errors.New("type mismatch")
)

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

// MatchValue checks if actual matches expected.
// If expected implements the Matcher interface, uses its Match method.
// Otherwise, uses reflect.DeepEqual for comparison.
// Returns (success, errorMessage). If success is true, errorMessage is empty.
func MatchValue(actual, expected any) (bool, string) {
	// Check if expected is a Matcher
	if matcher, ok := expected.(Matcher); ok {
		success, err := matcher.Match(actual)
		if err != nil {
			return false, err.Error()
		}

		if !success {
			return false, matcher.FailureMessage(actual)
		}

		return true, ""
	}

	// Fall back to reflect.DeepEqual for non-matchers
	if reflect.DeepEqual(actual, expected) {
		return true, ""
	}

	return false, fmt.Sprintf("expected %v, got %v", expected, actual)
}
