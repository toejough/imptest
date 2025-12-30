package functype_test

//go:generate impgen --target functype.WalkFunc

import (
	"errors"
	"testing"

	_ "github.com/toejough/imptest/UAT/16-function-type-wrapping"
)

// TestWrapFunctionType verifies that we can wrap a named function type.
// According to the design, wrapping a function type should:
// 1. Create a WrapWalkFunc(t, fn) constructor
// 2. Allow calling Start(args...) to invoke the function
// 3. Allow expecting return values with ExpectReturnsEqual/ExpectReturnsMatch
func TestWrapFunctionType(t *testing.T) {
	t.Parallel()

	testFn := func(path string, info string) error {
		if path == "/error" {
			return errors.New("test error")
		}
		return nil
	}

	// Wrap the function type instance
	wrapped := WrapWalkFunc(t, testFn)

	// Start the function with test arguments
	returns := wrapped.Start("/test", "info")

	// Expect the function to return nil (no error)
	returns.ExpectReturnsEqual(nil)
}

// TestWrapFunctionTypeWithError verifies error handling.
func TestWrapFunctionTypeWithError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("test error")
	testFn := func(path string, info string) error {
		return expectedErr
	}

	wrapped := WrapWalkFunc(t, testFn)
	returns := wrapped.Start("/error", "info")

	// Should match the expected error
	returns.ExpectReturnsEqual(expectedErr)
}
