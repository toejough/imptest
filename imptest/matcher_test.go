package imptest_test

import (
	"errors"
	"testing"

	"github.com/toejough/imptest/imptest"
)

var errMatcher = errors.New("matcher error")

// Test MatchValue with direct values using reflect.DeepEqual.
func TestMatchValue_DirectValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		actual   any
		expected any
		wantOK   bool
	}{
		{"equal ints", 42, 42, true},
		{"unequal ints", 42, 43, false},
		{"equal strings", "hello", "hello", true},
		{"unequal strings", "hello", "world", false},
		{"equal slices", []int{1, 2, 3}, []int{1, 2, 3}, true},
		{"unequal slices", []int{1, 2, 3}, []int{1, 2, 4}, false},
		{"nil vs nil", nil, nil, true},
		{"nil vs value", nil, 42, false},
	}

	for _, tt := range tests { //nolint:varnamelen // tt is idiomatic for table tests
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ok, msg := imptest.MatchValue(tt.actual, tt.expected) //nolint:varnamelen // ok is idiomatic
			if ok != tt.wantOK {
				t.Errorf("MatchValue() ok = %v, want %v; msg = %q", ok, tt.wantOK, msg)
			}

			if ok && msg != "" {
				t.Errorf("MatchValue() returned success but non-empty msg: %q", msg)
			}

			if !ok && msg == "" {
				t.Errorf("MatchValue() returned failure but empty msg")
			}
		})
	}
}

// Test MatchValue with the Any() matcher.
func TestMatchValue_AnyMatcher(t *testing.T) {
	t.Parallel()

	tests := []any{
		42,
		"hello",
		[]int{1, 2, 3},
		nil,
		struct{ x int }{x: 42},
	}

	for _, val := range tests {
		ok, msg := imptest.MatchValue(val, imptest.Any())
		if !ok {
			t.Errorf("Any() should match %v, but got ok=%v, msg=%q", val, ok, msg)
		}

		if msg != "" {
			t.Errorf("Any() should have empty message, got %q", msg)
		}
	}
}

// mockMatcher is a test double for testing MatchValue with custom matchers.
type mockMatcher struct {
	matchResult bool
	matchErr    error
	failMsg     string
}

func (m mockMatcher) Match(any) (bool, error) {
	return m.matchResult, m.matchErr
}

func (m mockMatcher) FailureMessage(any) string {
	return m.failMsg
}

// Test MatchValue with a matcher that succeeds.
func TestMatchValue_MatcherSuccess(t *testing.T) {
	t.Parallel()

	m := mockMatcher{matchResult: true}
	ok, msg := imptest.MatchValue(42, m)

	if !ok {
		t.Errorf("MatchValue() with successful matcher: ok = %v, want true", ok)
	}

	if msg != "" {
		t.Errorf("MatchValue() with successful matcher should have empty msg, got %q", msg)
	}
}

// Test MatchValue with a matcher that fails.
func TestMatchValue_MatcherFailure(t *testing.T) {
	t.Parallel()

	m := mockMatcher{matchResult: false, failMsg: "value too small"}
	ok, msg := imptest.MatchValue(42, m)

	if ok {
		t.Errorf("MatchValue() with failing matcher: ok = %v, want false", ok)
	}

	if msg != "value too small" {
		t.Errorf("MatchValue() msg = %q, want %q", msg, "value too small")
	}
}

// Test MatchValue with a matcher that returns an error.
func TestMatchValue_MatcherError(t *testing.T) {
	t.Parallel()

	m := mockMatcher{matchResult: false, matchErr: errMatcher}
	ok, msg := imptest.MatchValue(42, m)

	if ok {
		t.Errorf("MatchValue() with matcher error: ok = %v, want false", ok)
	}

	if msg != "matcher error" {
		t.Errorf("MatchValue() msg = %q, want %q", msg, "matcher error")
	}
}

// Test the Any() matcher directly.
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
