package noncomparable_test

import (
	"testing"

	noncomparable "github.com/toejough/imptest/UAT/variations/signature/non-comparable"
)

//go:generate impgen noncomparable.DataProcessor --dependency

// TestNonComparableArguments demonstrates how imptest handles Go types that
// cannot be compared with the == operator (like slices and maps).
//
// Key Requirements Met:
//  1. Automatic Deep Equality: The generator detects non-comparable types
//     and automatically uses reflect.DeepEqual for validation.
//  2. Developer Ease: Expect works seamlessly regardless of the
//     argument types involved.
func TestNonComparableArguments(t *testing.T) {
	t.Parallel()

	mock, imp := MockDataProcessor(t)

	go noncomparable.RunProcessor(mock)

	// Intercept ProcessSlice with a slice argument.
	// Requirement: reflect.DeepEqual is used automatically for slices.
	imp.ProcessSlice.ArgsEqual([]string{"a", "b", "c"}).Return(3)

	// Intercept ProcessMap with a map argument.
	// Requirement: reflect.DeepEqual is used automatically for maps.
	imp.ProcessMap.ArgsEqual(map[string]int{"threshold": 10}).
		Return(true)
}
