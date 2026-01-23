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

import "github.com/toejough/imptest/internal/core"

// BeAny matches any value - useful for "don't care" arguments.
var BeAny = core.BeAny //nolint:gochecknoglobals // BeAny is an exported sentinel value

type Matcher = core.Matcher

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
	return core.Satisfy(predicate)
}
