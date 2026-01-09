package generics_test

//go:generate impgen generics.Repository --dependency
//go:generate impgen generics.ProcessItem --target

import (
	"errors"
	"fmt"
	"testing"

	generics "github.com/toejough/imptest/UAT/variations/signature/generics"
	"github.com/toejough/imptest/imptest"
)

// TestGenericCallable demonstrates how imptest supports generic functions.
//
// Key Requirements Met:
//  1. Generic Function Support: Generate type-safe wrappers for generic
//     functions by specifying the type instantiation.
func TestGenericCallable(t *testing.T) {
	t.Parallel()

	repoImp := MockRepository[int](t)

	// Initialize the callable wrapper implementation for a specific instantiation of the generic function.
	// NewProcessItemImp is generic.
	logicImp := WrapProcessItem[int](t, generics.ProcessItem[int])

	// Start the function.
	transformer := func(i int) int { return i * 2 }
	call := logicImp.Method.Start(repoImp.Mock, "456", transformer)

	repoImp.Method.Get.ExpectCalledWithExactly("456").InjectReturnValues(21, nil)
	repoImp.Method.Save.ExpectCalledWithExactly(42).InjectReturnValues(nil)

	// Verify it returned successfully (nil error).
	call.ExpectReturnsEqual(nil)
}

// TestGenericMocking demonstrates how imptest supports generic interfaces.
//
// Key Requirements Met:
//  1. Generic Interface Support: Generate type-safe mocks for interfaces with
//     type parameters.
//  2. Type Safety: IDE and compiler assistance when defining expectations for
//     generic methods.
func TestGenericMocking(t *testing.T) {
	t.Parallel()

	// Initialize the mock implementation for a specific type (string).
	// The generated constructor is generic.
	repoImp := MockRepository[string](t)

	// Run the code under test in a goroutine.
	go func() {
		transformer := func(s string) string { return s + "!" }
		_ = generics.ProcessItem[string](repoImp.Mock, "123", transformer)
	}()

	// Expectations are type-safe based on the generic parameter.
	repoImp.Method.Get.ExpectCalledWithExactly("123").InjectReturnValues("hello", nil)
	repoImp.Method.Save.ExpectCalledWithExactly("hello!").InjectReturnValues(nil)
}

// TestProcessItem_Error demonstrates error handling in generic contexts.
func TestProcessItem_Error(t *testing.T) {
	t.Parallel()

	t.Run("Get error", func(t *testing.T) {
		t.Parallel()
		repoImp := MockRepository[string](t)
		logicImp := WrapProcessItem[string](t, generics.ProcessItem[string])

		call := logicImp.Method.Start(repoImp.Mock, "123", func(s string) string { return s })

		repoImp.Method.Get.ExpectCalledWithExactly("123").InjectReturnValues("", errTest)

		call.ExpectReturnsMatch(imptest.Satisfies(func(err error) error {
			if !errors.Is(err, errTest) {
				return fmt.Errorf("expected error %w, got %w", errTest, err)
			}

			return nil
		}))
	})

	t.Run("Save error", func(t *testing.T) {
		t.Parallel()
		repoImp := MockRepository[string](t)
		logicImp := WrapProcessItem[string](t, generics.ProcessItem[string])

		call := logicImp.Method.Start(repoImp.Mock, "123", func(s string) string { return s })

		repoImp.Method.Get.ExpectCalledWithExactly("123").InjectReturnValues("data", nil)
		repoImp.Method.Save.ExpectCalledWithExactly("data").InjectReturnValues(errTest)

		call.ExpectReturnsMatch(imptest.Satisfies(func(err error) error {
			if !errors.Is(err, errTest) {
				return fmt.Errorf("expected error %w, got %w", errTest, err)
			}

			return nil
		}))
	})
}

// unexported variables.
var (
	errTest = errors.New("test error")
)
