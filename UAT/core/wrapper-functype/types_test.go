package functype_test

//go:generate impgen --target functype.WalkFunc

import (
	"errors"
	"testing"

	_ "github.com/toejough/imptest/UAT/core/wrapper-functype"
)

// TestStartFunctionType verifies that we can wrap a named function type.
// According to the design, wrapping a function type should:
// 1. Create a StartWalkFunc(t, fn, args...) function
// 2. Allow expecting return values with ReturnsEqual/ReturnsShould
func TestStartFunctionType(t *testing.T) {
	t.Parallel()

	testFn := func(path, _ string) error {
		if path == "/error" {
			return errors.New("test error")
		}

		return nil
	}

	// Start the function with test arguments
	returns := StartWalkFunc(t, testFn, "/test", "info")

	// Expect the function to return nil (no error)
	returns.ReturnsEqual(nil)
}

// TestStartFunctionTypeWithError verifies error handling.
func TestStartFunctionTypeWithError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("test error")
	testFn := func(_, _ string) error {
		return expectedErr
	}

	returns := StartWalkFunc(t, testFn, "/error", "info")

	// Should match the expected error
	returns.ReturnsEqual(expectedErr)
}
