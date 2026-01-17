package basic_test

import (
	"errors"
	"testing"
)

// TestTypedReturn demonstrates compile-time type safety for return value injection.
// This test verifies that:
// 1. Typed Return methods exist on call wrappers
// 2. Correct types can be injected
// 3. Wrong types would cause compile errors (see commented examples below)
func TestTypedReturn(t *testing.T) {
	t.Parallel()

	// Test Store method (returns int, error)
	t.Run("Store_CorrectTypes", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockOps(t)

		// Launch goroutine that will call Store
		go func() {
			_, _ = mock.Store("key", "value")
		}()

		// Use typed Return - should compile with correct types
		call := imp.Store.ArgsEqual("key", "value")
		call.Return(42, nil) // Correct types: int, error

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

		// Use typed Return with error
		testErr := errors.New("storage failure")
		call := imp.Store.ArgsEqual("foo", "bar")
		call.Return(0, testErr) // Correct types: int, error
	})

	// Test Add method (returns int)
	t.Run("Add_SingleReturn", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockOps(t)

		// Launch goroutine that will call Add
		go func() {
			_ = mock.Add(5, 3)
		}()

		// Use typed Return for single return value
		call := imp.Add.ArgsEqual(5, 3)
		call.Return(8) // Correct type: int
	})

	// Test Notify method (returns bool)
	t.Run("Notify_VariadicWithReturn", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockOps(t)

		// Launch goroutine that will call Notify
		go func() {
			_ = mock.Notify("alert", 1, 2, 3)
		}()

		// Use typed Return for bool return
		call := imp.Notify.ArgsEqual("alert", 1, 2, 3)
		call.Return(true) // Correct type: bool
	})
}

// The following would cause COMPILE ERRORS with typed Return:
//
// func TestWrongTypes(t *testing.T) {
//     mock, imp := MockOps(t)
//     go func() { _, _ = mock.Store("key", "value") }()
//     call := imp.Store.Expect("key", "value")
//
//     // COMPILE ERROR: cannot use "wrong" (type string) as type int
//     call.Return("wrong", nil)
//
//     // COMPILE ERROR: cannot use 123 (type int) as type error
//     call.Return(42, 123)
//
//     // COMPILE ERROR: not enough arguments (expected 2, got 1)
//     call.Return(42)
//
//     // COMPILE ERROR: too many arguments (expected 2, got 3)
//     call.Return(42, nil, "extra")
// }
