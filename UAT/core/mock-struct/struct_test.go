// Package mockstruct_test demonstrates mocking struct types with imptest.
// This tests Issue #44: Struct as dependency (mock a struct's methods).
package mockstruct_test

import (
	"testing"

	mockstruct "github.com/toejough/imptest/UAT/core/mock-struct"
)

// impgen detects Calculator as a struct type and generates a mock for all its methods.
// The generated mock implements the same methods as the struct, allowing interception.
//go:generate impgen mockstruct.Calculator --dependency

// TestStructMocking demonstrates mocking a struct type's methods.
// The mock provides the same API as interface mocks.
func TestStructMocking(t *testing.T) {
	t.Parallel()

	// Initialize the generated mock - works just like interface mocking.
	mock := MockCalculator(t)

	// Run the code under test in a goroutine.
	go mockstruct.UseCalculator(mock.Mock)

	// Interactive Control Pattern - identical to interface mocking.

	// 1. Intercept 'Add' and provide a return value.
	mock.Method.Add.ExpectCalledWithExactly(1, 2).InjectReturnValues(3)

	// 2. Intercept 'Store' and provide a return value.
	mock.Method.Store.ExpectCalledWithExactly(42).InjectReturnValues(0)

	// 3. Intercept 'Get' and provide multiple return values.
	mock.Method.Get.ExpectCalledWithExactly().InjectReturnValues(42, nil)

	// 4. Intercept 'Reset' (void method) and signal completion.
	mock.Method.Reset.ExpectCalledWithExactly().InjectReturnValues()
}

// TestStructMockingWithError demonstrates returning errors from mocked struct methods.
func TestStructMockingWithError(t *testing.T) {
	t.Parallel()

	mock := MockCalculator(t)

	// Run a simple operation that calls Get.
	go func() {
		_, _ = mock.Mock.Get()
	}()

	// Return an error from Get.
	mock.Method.Get.ExpectCalledWithExactly().InjectReturnValues(0, mockstruct.ErrNotFound)
}
