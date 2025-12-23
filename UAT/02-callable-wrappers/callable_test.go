package callable_test

import (
	"errors"
	"fmt"
	"testing"

	callable "github.com/toejough/imptest/UAT/02-callable-wrappers"
	"github.com/toejough/imptest/imptest"
)

// Generate a mock for the dependency.
//go:generate go run ../../impgen/main.go callable.ExternalService --name ExternalServiceImp

// Generate a wrapper for the function under test.
//go:generate go run ../../impgen/main.go callable.BusinessLogic --name BusinessLogicImp

// TestBusinessLogic demonstrates how to use type-safe wrappers for functions.
//
// Key Requirements Met:
//  1. Function Verification: Verify return values from standalone functions or
//     methods that are not part of an interface.
//  2. Coordinated Control: Synchronously control dependencies while verifying
//     the behavior of the "impure" logic that coordinates them.
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

// TestBusinessLogicError demonstrates error path validation using matchers.
//
// Key Requirements Met:
//  1. Error Chain Validation: Use Satisfies with errors.Is to verify that the
//     correct error is propagated, without requiring strict pointer equality.
func TestBusinessLogicError(t *testing.T) {
	t.Parallel()

	impSvc := NewExternalServiceImp(t)
	logicImp := NewBusinessLogicImp(t, callable.BusinessLogic)

	logicImp.Start(impSvc.Mock, 99)

	// Simulate an error from the service.
	impSvc.ExpectCallIs.FetchData().ExpectArgsAre(99).InjectResults("", errNotFound)

	// Requirement: Verify that the business logic returns the error.
	// We use Satisfies to check if the error is (or wraps) errNotFound.
	logicImp.ExpectReturnedValuesShould("", imptest.Satisfies(func(err error) error {
		if !errors.Is(err, errNotFound) {
			return fmt.Errorf("expected error %w, got %w", errNotFound, err)
		}

		return nil
	}))
}
