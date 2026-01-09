// Package handlers_test demonstrates mocking function types directly
// with the --dependency flag (Issue #50).
//
// FUNCTION TYPE vs FUNCTION:
// - Function type: A named type for a function signature (type Validator func(string) error)
// - Package function: A declared function (func Validate(s string) error)
//
// This UAT tests function type mocking. For interface mocking (including interfaces
// that use function types), see mock-interface/.
package handlers_test

import (
	"errors"
	"testing"

	"github.com/toejough/imptest"
)

// Generate dependency mock for function type directly (Issue #50)
//go:generate impgen Validator --dependency

// TestFunctionTypeMock_BasicUsage demonstrates creating and using a mock
// for a function type directly (not wrapped in an interface).
func TestFunctionTypeMock_BasicUsage(t *testing.T) {
	t.Parallel()

	// MockValidator creates a mock for the Validator function type
	mock := MockValidator(t)

	// Get the function that can be passed where Validator is expected
	validatorFunc := mock.Mock

	// Use the mock function in a goroutine (simulating production code)
	errChan := make(chan error, 1)

	go func() {
		errChan <- validatorFunc("test@example.com")
	}()

	// Set up expectation and inject return value
	mock.Method.ExpectCalledWithExactly("test@example.com").
		InjectReturnValues(nil)

	// Verify result
	result := <-errChan
	if result != nil {
		t.Errorf("expected nil error, got %v", result)
	}
}

// TestFunctionTypeMock_GetArgs demonstrates type-safe argument access.
func TestFunctionTypeMock_GetArgs(t *testing.T) {
	t.Parallel()

	mock := MockValidator(t)
	validatorFunc := mock.Mock

	go func() {
		_ = validatorFunc("check-this-data")
	}()

	// Use ExpectCalledWithMatches to accept any call
	call := mock.Method.ExpectCalledWithMatches(imptest.Any())

	// Get typed args
	args := call.GetArgs()
	if args.Data != "check-this-data" {
		t.Errorf("expected Data to be %q, got %q", "check-this-data", args.Data)
	}

	call.InjectReturnValues(nil)
}

// TestFunctionTypeMock_ReturnError demonstrates injecting an error return.
func TestFunctionTypeMock_ReturnError(t *testing.T) {
	t.Parallel()

	mock := MockValidator(t)
	validatorFunc := mock.Mock

	expectedErr := errors.New("invalid email format")
	errChan := make(chan error, 1)

	go func() {
		errChan <- validatorFunc("invalid-email")
	}()

	// Inject error return
	mock.Method.ExpectCalledWithExactly("invalid-email").
		InjectReturnValues(expectedErr)

	// Verify error was returned
	result := <-errChan
	if result == nil {
		t.Error("expected error to be returned")
	}
}
