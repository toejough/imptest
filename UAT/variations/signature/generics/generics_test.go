package generics_test

//go:generate impgen generics.Repository --dependency
//go:generate impgen generics.ProcessItem --target

import (
	"errors"
	"fmt"
	"testing"

	"github.com/toejough/imptest/match"
	generics "github.com/toejough/imptest/UAT/variations/signature/generics"
)

// TestGenericCallable demonstrates how imptest supports generic functions.
//
// Key Requirements Met:
//  1. Generic Function Support: Generate type-safe wrappers for generic
//     functions by specifying the type instantiation.
func TestGenericCallable(t *testing.T) {
	t.Parallel()

	repoMock, repoImp := MockRepository[int](t)

	// Start the generic function with a specific type instantiation.
	transformer := func(i int) int { return i * 2 }
	call := StartProcessItem[int](t, generics.ProcessItem[int], repoMock, "456", transformer)

	//nolint:nilaway // false positive: repoImp assigned above
	repoImp.Get.ArgsEqual("456").Return(21, nil)
	//nolint:nilaway // false positive: repoImp assigned above
	repoImp.Save.ArgsEqual(42).Return(nil)

	// Verify it returned successfully (nil error).
	call.ReturnsEqual(nil)
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
	repoMock, repoImp := MockRepository[string](t)

	// Run the code under test in a goroutine.
	go func() {
		transformer := func(s string) string { return s + "!" }
		_ = generics.ProcessItem[string](repoMock, "123", transformer)
	}()

	// Expectations are type-safe based on the generic parameter.
	//nolint:nilaway // false positive: repoImp assigned above
	repoImp.Get.ArgsEqual("123").Return("hello", nil)
	//nolint:nilaway // false positive: repoImp assigned above
	repoImp.Save.ArgsEqual("hello!").Return(nil)
}

// TestProcessItem_Error demonstrates error handling in generic contexts.
func TestProcessItem_Error(t *testing.T) {
	t.Parallel()

	t.Run("Get error", func(t *testing.T) {
		t.Parallel()
		repoMock, repoImp := MockRepository[string](t)

		call := StartProcessItem[string](t, generics.ProcessItem[string], repoMock, "123", func(s string) string { return s })

		repoImp.Get.ArgsEqual("123").Return("", errTest)

		call.ReturnsShould(match.Satisfy(func(err error) error {
			if !errors.Is(err, errTest) {
				return fmt.Errorf("expected error %w, got %w", errTest, err)
			}

			return nil
		}))
	})

	t.Run("Save error", func(t *testing.T) {
		t.Parallel()
		repoMock, repoImp := MockRepository[string](t)

		call := StartProcessItem[string](t, generics.ProcessItem[string], repoMock, "123", func(s string) string { return s })

		repoImp.Get.ArgsEqual("123").Return("data", nil)
		repoImp.Save.ArgsEqual("data").Return(errTest)

		call.ReturnsShould(match.Satisfy(func(err error) error {
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
