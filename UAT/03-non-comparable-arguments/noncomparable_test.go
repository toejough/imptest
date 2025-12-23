package noncomparable_test

import (
	"testing"

	noncomparable "github.com/toejough/imptest/UAT/03-non-comparable-arguments"
)

//go:generate impgen noncomparable.DataProcessor --name DataProcessorImp

// TestNonComparableArguments demonstrates how imptest handles Go types that
// cannot be compared with the == operator (like slices and maps).
//
// Key Requirements Met:
//  1. Automatic Deep Equality: The generator detects non-comparable types
//     and automatically uses reflect.DeepEqual for validation.
//  2. Developer Ease: ExpectArgsAre works seamlessly regardless of the
//     argument types involved.
func TestNonComparableArguments(t *testing.T) {
	t.Parallel()

	imp := NewDataProcessorImp(t)

	go noncomparable.RunProcessor(imp.Mock)

	// Intercept ProcessSlice with a slice argument.
	// Requirement: reflect.DeepEqual is used automatically for slices.
	imp.ExpectCallIs.ProcessSlice().ExpectArgsAre([]string{"a", "b", "c"}).InjectResult(3)

	// Intercept ProcessMap with a map argument.
	// Requirement: reflect.DeepEqual is used automatically for maps.
	imp.ExpectCallIs.ProcessMap().ExpectArgsAre(map[string]int{"threshold": 10}).InjectResult(true)
}
