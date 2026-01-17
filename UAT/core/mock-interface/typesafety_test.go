package basic_test

import (
	"errors"
	"testing"
)

// TestTypedInjectReturnValues demonstrates compile-time type safety for return value injection.
// This test verifies that:
// 1. Typed InjectReturnValues methods exist on call wrappers
// 2. Correct types can be injected
// 3. Wrong types would cause compile errors (see commented examples below)
func TestTypedInjectReturnValues(t *testing.T) {
	t.Parallel()

	// Test Store method (returns int, error)
	t.Run("Store_CorrectTypes", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockOps(t)

		// Launch goroutine that will call Store
		go func() {
			_, _ = mock.Store("key", "value")
		}()

		// Use typed InjectReturnValues - should compile with correct types
		call := imp.Store.ExpectCalledWithExactly("key", "value")
		call.InjectReturnValues(42, nil) // Correct types: int, error

		// Verify the mock received the call
		args := call.GetArgs()
		if args.Key != "key" {
			t.Fatalf("expected key 'key', got %q", args.Key)
		}
	})

	t.Run("Store_WithError", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockOps(t)

		// Launch goroutine that will call Store
		go func() {
			_, _ = mock.Store("foo", "bar")
		}()

		// Use typed InjectReturnValues with error
		testErr := errors.New("storage failure")
		call := imp.Store.ExpectCalledWithExactly("foo", "bar")
		call.InjectReturnValues(0, testErr) // Correct types: int, error
	})

	// Test Add method (returns int)
	t.Run("Add_SingleReturn", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockOps(t)

		// Launch goroutine that will call Add
		go func() {
			_ = mock.Add(5, 3)
		}()

		// Use typed InjectReturnValues for single return value
		call := imp.Add.ExpectCalledWithExactly(5, 3)
		call.InjectReturnValues(8) // Correct type: int
	})

	// Test Notify method (returns bool)
	t.Run("Notify_VariadicWithReturn", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockOps(t)

		// Launch goroutine that will call Notify
		go func() {
			_ = mock.Notify("alert", 1, 2, 3)
		}()

		// Use typed InjectReturnValues for bool return
		call := imp.Notify.ExpectCalledWithExactly("alert", 1, 2, 3)
		call.InjectReturnValues(true) // Correct type: bool
	})
}

// The following would cause COMPILE ERRORS with typed InjectReturnValues:
//
// func TestWrongTypes(t *testing.T) {
//     mock, imp := MockOps(t)
//     go func() { _, _ = mock.Store("key", "value") }()
//     call := imp.Store.ExpectCalledWithExactly("key", "value")
//
//     // COMPILE ERROR: cannot use "wrong" (type string) as type int
//     call.InjectReturnValues("wrong", nil)
//
//     // COMPILE ERROR: cannot use 123 (type int) as type error
//     call.InjectReturnValues(42, 123)
//
//     // COMPILE ERROR: not enough arguments (expected 2, got 1)
//     call.InjectReturnValues(42)
//
//     // COMPILE ERROR: too many arguments (expected 2, got 3)
//     call.InjectReturnValues(42, nil, "extra")
// }
