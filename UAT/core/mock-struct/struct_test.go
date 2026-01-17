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
	// Returns (mock, imp) where mock implements the interface and imp sets expectations.
	mock, imp := MockCalculator(t)

	// Run the code under test in a goroutine.
	go mockstruct.UseCalculator(mock)

	// Interactive Control Pattern - identical to interface mocking.

	// 1. Intercept 'Add' and provide a return value.
	imp.Add.Expect(1, 2).Return(3)

	// 2. Intercept 'Store' and provide a return value.
	imp.Store.Expect(42).Return(0)

	// 3. Intercept 'Get' and provide multiple return values.
	imp.Get.Expect().Return(42, nil)

	// 4. Intercept 'Reset' (void method) and signal completion.
	imp.Reset.Expect().Return()
}

// TestStructMockingWithError demonstrates returning errors from mocked struct methods.
func TestStructMockingWithError(t *testing.T) {
	t.Parallel()

	mock, imp := MockCalculator(t)

	// Run a simple operation that calls Get.
	go func() {
		_, _ = mock.Get()
	}()

	// Return an error from Get.
	imp.Get.Expect().Return(0, mockstruct.ErrNotFound)
}
