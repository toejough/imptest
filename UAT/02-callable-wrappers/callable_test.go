package callable_test

import (
	"errors"
	"testing"

	callable "github.com/toejough/imptest/UAT/02-callable-wrappers"
	"github.com/toejough/imptest/imptest"
)

// Generate a mock for the dependency.
//go:generate go run ../../impgen/main.go callable.ExternalService --name ExternalServiceImp

// Generate a wrapper for the function under test.
//go:generate go run ../../impgen/main.go callable.BusinessLogic --name BusinessLogicImp --call

func TestBusinessLogic(t *testing.T) {
	t.Parallel()

	// Initialize the mock implementation dependency and the callable wrapper.
	impSvc := NewExternalServiceImp(t)
	logicImp := NewBusinessLogicImp(t, callable.BusinessLogic)

	// Start the business logic in a goroutine.
	// We pass the mock implementation and the input arguments.
	logicImp.Start(impSvc.Mock, 42)

	// 1. Expect call to FetchData and provide response.
	impSvc.ExpectCallIs.FetchData().ExpectArgsAre(42).InjectResults("raw data", nil)

	// 2. Expect call to Process and provide response.
	impSvc.ExpectCallIs.Process().ExpectArgsAre("raw data").InjectResult("processed data")

	// 3. Verify the final output of the business logic.
	logicImp.ExpectReturnedValuesAre("Result: processed data", nil)
}

var errNotFound = errors.New("not found")

// TODO: add comments explaining what all the generated code is doing, too.
// TODO: remove the existing unit tests besides UAT, and see what coverage remains. Only add back tests that provided
// necessary coverage that the UAT tests did not.

func TestBusinessLogicError(t *testing.T) {
	t.Parallel()

	impSvc := NewExternalServiceImp(t)
	logicImp := NewBusinessLogicImp(t, callable.BusinessLogic)

	logicImp.Start(impSvc.Mock, 99)

	// Simulate an error from the service.
	impSvc.ExpectCallIs.FetchData().ExpectArgsAre(99).InjectResults("", errNotFound)

	// TODO: remove this test? I'm not sure what else it's demonstrating, and it uses advanced matching too early.
	// Verify that the business logic returns the error.
	logicImp.ExpectReturnedValuesShould("", imptest.Satisfies(func(err error) bool {
		return errors.Is(err, errNotFound)
	}))
}
