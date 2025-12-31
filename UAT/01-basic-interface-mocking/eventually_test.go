package basic_test

import (
	"testing"
)

// TestEventuallyTypePreservation verifies that Eventually() returns the typed wrapper.
// This is a compile-time type safety check that ensures Eventually() preserves
// the typed wrapper and allows chaining with type-safe methods.
func TestEventuallyTypePreservation(t *testing.T) {
	t.Parallel()

	mock := MockOps(t)

	// These are compile-time checks that verify the types are correct.
	// If Eventually() didn't return the right type, these would fail to compile.

	// Verify Eventually() returns *OpsMockAddMethod
	var addMethod *OpsMockAddMethod = mock.Add.Eventually()
	if addMethod == nil {
		t.Fatal("Eventually() should return non-nil typed wrapper")
	}

	// Verify Eventually() returns *OpsMockStoreMethod
	var storeMethod *OpsMockStoreMethod = mock.Store.Eventually()
	if storeMethod == nil {
		t.Fatal("Eventually() should return non-nil typed wrapper")
	}

	// Verify Eventually() returns *OpsMockLogMethod
	var logMethod *OpsMockLogMethod = mock.Log.Eventually()
	if logMethod == nil {
		t.Fatal("Eventually() should return non-nil typed wrapper")
	}

	// Verify Eventually() returns *OpsMockNotifyMethod
	var notifyMethod *OpsMockNotifyMethod = mock.Notify.Eventually()
	if notifyMethod == nil {
		t.Fatal("Eventually() should return non-nil typed wrapper")
	}
}
