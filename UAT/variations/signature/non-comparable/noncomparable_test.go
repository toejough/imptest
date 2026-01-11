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
//  2. Developer Ease: ExpectCalledWithExactly works seamlessly regardless of the
//     argument types involved.
func TestNonComparableArguments(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)

	go noncomparable.RunProcessor(mock.Mock)

	// Intercept ProcessSlice with a slice argument.
	// Requirement: reflect.DeepEqual is used automatically for slices.
	mock.Method.ProcessSlice.ExpectCalledWithExactly([]string{"a", "b", "c"}).InjectReturnValues(3)

	// Intercept ProcessMap with a map argument.
	// Requirement: reflect.DeepEqual is used automatically for maps.
	mock.Method.ProcessMap.ExpectCalledWithExactly(map[string]int{"threshold": 10}).
		InjectReturnValues(true)
}
