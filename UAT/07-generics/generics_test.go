package generics_test

import (
	"errors"
	"fmt"
	"testing"

	generics "github.com/toejough/imptest/UAT/07-generics"
	"github.com/toejough/imptest/imptest"
)

// Generate a mock for the generic interface.
// Note: We use the base interface name; the generator handles the type parameters.
//go:generate go run ../../impgen/main.go generics.Repository --name RepositoryImp

// Generate a wrapper for the generic function.
//go:generate go run ../../impgen/main.go generics.ProcessItem --name ProcessItemImp --call

// TODO: automatically identify whether the generate arg is an interface or a function, and skip --call for interfaces?

var errTest = errors.New("test error")

func TestGenericMocking(t *testing.T) {
	t.Parallel()

	// Initialize the mock implementation for a specific type (string).
	// The generated constructor is generic.
	repoImp := NewRepositoryImp[string](t)

	// Run the code under test in a goroutine.
	go func() {
		transformer := func(s string) string { return s + "!" }
		_ = generics.ProcessItem[string](repoImp.Mock, "123", transformer)
	}()

	// Expectations are type-safe based on the generic parameter.
	repoImp.ExpectCallIs.Get().ExpectArgsAre("123").InjectResults("hello", nil)
	repoImp.ExpectCallIs.Save().ExpectArgsAre("hello!").InjectResult(nil)
}

func TestGenericCallable(t *testing.T) {
	t.Parallel()

	repoImp := NewRepositoryImp[int](t)

	// Initialize the callable wrapper implementation for a specific instantiation of the generic function.
	// NewProcessItemImp is generic.
	logicImp := NewProcessItemImp[int](t, generics.ProcessItem[int])

	// Start the function.
	transformer := func(i int) int { return i * 2 }
	logicImp.Start(repoImp.Mock, "456", transformer)

	repoImp.ExpectCallIs.Get().ExpectArgsAre("456").InjectResults(21, nil)
	repoImp.ExpectCallIs.Save().ExpectArgsAre(42).InjectResult(nil)

	// Verify it returned successfully (nil error).
	logicImp.ExpectReturnedValuesAre(nil)
}

// TODO: I'm not sure what these are demonstrating that is unique.
func TestProcessItem_Error(t *testing.T) {
	t.Parallel()

	t.Run("Get error", func(t *testing.T) {
		t.Parallel()
		repoImp := NewRepositoryImp[string](t)
		logicImp := NewProcessItemImp[string](t, generics.ProcessItem[string])

		logicImp.Start(repoImp.Mock, "123", func(s string) string { return s })

		repoImp.ExpectCallIs.Get().ExpectArgsAre("123").InjectResults("", errTest)

		logicImp.ExpectReturnedValuesShould(imptest.Satisfies(func(err error) error {
			if !errors.Is(err, errTest) {
				return fmt.Errorf("expected error %w, got %w", errTest, err)
			}

			return nil
		}))
	})

	t.Run("Save error", func(t *testing.T) {
		t.Parallel()
		repoImp := NewRepositoryImp[string](t)
		logicImp := NewProcessItemImp[string](t, generics.ProcessItem[string])

		logicImp.Start(repoImp.Mock, "123", func(s string) string { return s })

		repoImp.ExpectCallIs.Get().ExpectArgsAre("123").InjectResults("data", nil)
		repoImp.ExpectCallIs.Save().ExpectArgsAre("data").InjectResult(errTest)

		logicImp.ExpectReturnedValuesShould(imptest.Satisfies(func(err error) error {
			if !errors.Is(err, errTest) {
				return fmt.Errorf("expected error %w, got %w", errTest, err)
			}

			return nil
		}))
	})
}
