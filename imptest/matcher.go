package imptest

import (
	"fmt"
	"reflect"
)

// Matcher defines the interface for flexible value matching.
// Compatible with gomega.GomegaMatcher via duck typing - any type
// implementing Match and FailureMessage will work.
type Matcher interface {
	Match(actual any) (success bool, err error)
	FailureMessage(actual any) string
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

// Any returns a matcher that matches any value.
// Useful when you don't care about a particular argument or return value.
func Any() Matcher {
	return anyMatcher{}
}

// anyMatcher is the implementation of the Any() matcher.
type anyMatcher struct{}

// Match always returns true - matches any value.
func (anyMatcher) Match(any) (bool, error) {
	return true, nil
}

// FailureMessage returns an empty string since Any() always matches.
func (anyMatcher) FailureMessage(any) string {
	return ""
}
