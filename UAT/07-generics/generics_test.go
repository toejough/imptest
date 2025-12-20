package generics_test

import (
	"errors"
	"testing"

	generics "github.com/toejough/imptest/UAT/07-generics"
	"github.com/toejough/imptest/imptest"
)

// Generate a mock for the generic interface.
// Note: We use the base interface name; the generator handles the type parameters.
//go:generate go run ../../impgen/main.go generics.Repository --name RepositoryImp

// Generate a wrapper for the generic function.
//go:generate go run ../../impgen/main.go generics.ProcessItem --name ProcessItemImp --call

var errTest = errors.New("test error")

func TestGenericMocking(t *testing.T) {
	t.Parallel()

	// Initialize the mock for a specific type (string).
	// The generated constructor is generic.
	repo := NewRepositoryImp[string](t)

	// Run the code under test in a goroutine.
	go func() {
		transformer := func(s string) string { return s + "!" }
		_ = generics.ProcessItem[string](repo.Mock, "123", transformer)
	}()

	// Expectations are type-safe based on the generic parameter.
	repo.ExpectCallIs.Get().ExpectArgsAre("123").InjectResults("hello", nil)
	repo.ExpectCallIs.Save().ExpectArgsAre("hello!").InjectResult(nil)
}

func TestGenericCallable(t *testing.T) {
	t.Parallel()

	repo := NewRepositoryImp[int](t)

	// Initialize the callable wrapper for a specific instantiation of the generic function.
	// NewProcessItemImp is generic.
	logic := NewProcessItemImp[int](t, generics.ProcessItem[int])

	// Start the function.
	transformer := func(i int) int { return i * 2 }
	logic.Start(repo.Mock, "456", transformer)

	repo.ExpectCallIs.Get().ExpectArgsAre("456").InjectResults(21, nil)
	repo.ExpectCallIs.Save().ExpectArgsAre(42).InjectResult(nil)

	// Verify it returned successfully (nil error).
	logic.ExpectReturnedValuesAre(nil)
}

func TestProcessItem_Error(t *testing.T) {
	t.Parallel()

	t.Run("Get error", func(t *testing.T) {
		t.Parallel()
		repo := NewRepositoryImp[string](t)
		logic := NewProcessItemImp[string](t, generics.ProcessItem[string])

		logic.Start(repo.Mock, "123", func(s string) string { return s })

		repo.ExpectCallIs.Get().ExpectArgsAre("123").InjectResults("", errTest)

		logic.ExpectReturnedValuesShould(imptest.Satisfies(func(err error) bool {
			return errors.Is(err, errTest)
		}))
	})

	t.Run("Save error", func(t *testing.T) {
		t.Parallel()
		repo := NewRepositoryImp[string](t)
		logic := NewProcessItemImp[string](t, generics.ProcessItem[string])

		logic.Start(repo.Mock, "123", func(s string) string { return s })

		repo.ExpectCallIs.Get().ExpectArgsAre("123").InjectResults("data", nil)
		repo.ExpectCallIs.Save().ExpectArgsAre("data").InjectResult(errTest)

		logic.ExpectReturnedValuesShould(imptest.Satisfies(func(err error) bool {
			return errors.Is(err, errTest)
		}))
	})
}
