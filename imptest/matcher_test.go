package imptest_test

import (
	"errors"
	"testing"

	"github.com/toejough/imptest/imptest"
)

// Test the Any() matcher directly.
//
//nolint:varnamelen // Standard Go test parameter name
func TestAny(t *testing.T) {
	t.Parallel()

	matcher := imptest.Any()

	// Test Match always returns true
	ok, err := matcher.Match(42) //nolint:varnamelen // ok is idiomatic
	if !ok || err != nil {
		t.Errorf("Any().Match(42) = (%v, %v), want (true, nil)", ok, err)
	}

	ok, err = matcher.Match(nil)
	if !ok || err != nil {
		t.Errorf("Any().Match(nil) = (%v, %v), want (true, nil)", ok, err)
	}

	// Test FailureMessage is empty
	msg := matcher.FailureMessage(42)
	if msg != "" {
		t.Errorf("Any().FailureMessage(42) = %q, want empty string", msg)
	}
}

// Test the Satisfies() matcher.
//
//nolint:varnamelen // Standard Go test parameter name
func TestSatisfies_MatchFailure(t *testing.T) {
	t.Parallel()

	matcher := imptest.Satisfies(func(val int) error {
		if val <= 10 {
			return errors.New("must be greater than 10")
		}

		return nil
	})

	ok, err := matcher.Match(5)

	if ok || err != nil {
		t.Errorf("Satisfies().Match(5) = (%v, %v), want (false, nil)", ok, err)
	}

	msg := matcher.FailureMessage(5)

	expected := "value 5 does not satisfy predicate: must be greater than 10"

	if msg != expected {
		t.Errorf("Satisfies().FailureMessage(5) = %q, want %q", msg, expected)
	}
}

//nolint:varnamelen // Standard Go test parameter name
func TestSatisfies_MatchSuccess(t *testing.T) {
	t.Parallel()

	matcher := imptest.Satisfies(func(val int) error {
		if val <= 10 {
			return errors.New("must be greater than 10")
		}

		return nil
	})

	ok, err := matcher.Match(42)

	if !ok || err != nil {
		t.Errorf("Satisfies().Match(42) = (%v, %v), want (true, nil)", ok, err)
	}

	// Test FailureMessage even on success (for coverage)

	msg := matcher.FailureMessage(42)

	expected := "value 42 does not satisfy predicate"

	if msg != expected {
		t.Errorf("Satisfies().FailureMessage(42) = %q, want %q", msg, expected)
	}
}
