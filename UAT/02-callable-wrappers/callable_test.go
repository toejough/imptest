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

	// Initialize the mock dependency and the callable wrapper.
	mockSvc := NewExternalServiceImp(t)
	logic := NewBusinessLogicImp(t, callable.BusinessLogic)

	// Start the business logic in a goroutine.
	// We pass the mock and the input arguments.
	logic.Start(mockSvc.Mock, 42)

	// 1. Expect call to FetchData and provide response.
	mockSvc.ExpectCallIs.FetchData().ExpectArgsAre(42).InjectResults("raw data", nil)

	// 2. Expect call to Process and provide response.
	mockSvc.ExpectCallIs.Process().ExpectArgsAre("raw data").InjectResult("processed data")

	// 3. Verify the final output of the business logic.
	logic.ExpectReturnedValuesAre("Result: processed data", nil)
}

var errNotFound = errors.New("not found")

// TODO: add comments explaining what all the generated code is doing, too.

func TestBusinessLogicError(t *testing.T) {
	t.Parallel()

	mockSvc := NewExternalServiceImp(t)
	logic := NewBusinessLogicImp(t, callable.BusinessLogic)

	logic.Start(mockSvc.Mock, 99)

	// Simulate an error from the service.
	mockSvc.ExpectCallIs.FetchData().ExpectArgsAre(99).InjectResults("", errNotFound)

	// TODO: remove this test? I'm not sure what else it's demonstrating, and it uses advanced matching too early.
	// Verify that the business logic returns the error.
	logic.ExpectReturnedValuesShould("", imptest.Satisfies(func(err error) bool {
		return errors.Is(err, errNotFound)
	}))
}
