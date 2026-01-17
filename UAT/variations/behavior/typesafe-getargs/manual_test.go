package typesafeargs_test

import (
	"testing"
)

// TestManualTypeSafeGetArgs tests the manually created typed wrappers.
func TestManualTypeSafeGetArgs(t *testing.T) {
	t.Parallel()

	mock, imp := NewTypesafeCalculatorMock(t)

	go func() {
		_ = mock.Add(10, 20)
	}()

	// Use the typed wrapper
	call := imp.Add.Eventually.ArgsEqual(10, 20)

	// GetArgs should return typed struct - no casting!
	args := call.GetArgs()
	if args.A != 10 {
		t.Fatalf("expected A=10, got %d", args.A)
	}

	if args.B != 20 {
		t.Fatalf("expected B=20, got %d", args.B)
	}

	call.Return(30)
}
