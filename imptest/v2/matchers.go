package v2

// Matcher represents a value matcher for validation.
type Matcher interface {
	Matches(value any) bool
	Description() string
}

// anyMatcher matches any value.
type anyMatcher struct{}

func (m *anyMatcher) Matches(value any) bool {
	return true
}

func (m *anyMatcher) Description() string {
	return "any value"
}

// Any returns a matcher that matches any value.
func Any() Matcher {
	return &anyMatcher{}
}

// satisfiesMatcher matches values that satisfy a predicate.
type satisfiesMatcher struct {
	predicate func(any) bool
	desc      string
}

func (m *satisfiesMatcher) Matches(value any) bool {
	return m.predicate(value)
}

func (m *satisfiesMatcher) Description() string {
	if m.desc != "" {
		return m.desc
	}
	return "satisfies predicate"
}

// Satisfies returns a matcher that checks if a value satisfies the given predicate.
func Satisfies(predicate func(any) bool) Matcher {
	return &satisfiesMatcher{predicate: predicate}
}
