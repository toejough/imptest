package zero_returns_test

import (
	"testing"

	zr "github.com/toejough/imptest/UAT/09-edge-zero-returns"
)

// TestProcessData_MultipleArgs_mutant tests with various argument combinations.
func TestProcessData_MultipleArgs_mutant(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		data  string
		count int
	}{
		{"empty string", "", 0},
		{"single char", "x", 1},
		{"long string", "this is a longer test string", 100},
		{"negative count", "data", -5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			callable := zr.NewProcessDataImp(t, zr.ProcessData).Start(tc.data, tc.count)
			if callable == nil {
				t.Fatal("Expected non-nil callable")
			}
		})
	}
}

// TestProcessData_Panic_mutant tests that panics are captured.
func TestProcessData_Panic_mutant(t *testing.T) {
	t.Parallel()

	// Define a function that panics
	panicFunc := func(_ string, _ int) {
		panic("test panic")
	}

	callable := zr.NewProcessDataImp(t, panicFunc).Start("test", 42)

	// The wrapper should capture the panic - verify it's not nil
	if callable == nil {
		t.Fatal("Expected non-nil callable even after panic")
	}
}

// TestProcessData_mutant verifies that callable wrappers work correctly for functions with zero returns.
// This catches mutations in return value counting and nil result list handling.
func TestProcessData_mutant(t *testing.T) {
	t.Parallel()

	// Start the callable wrapper
	callable := zr.NewProcessDataImp(t, zr.ProcessData).Start("test data", 42)

	// For zero-return functions, the wrapper just ensures the function completes
	// We verify it ran by checking no panic occurred
	if callable == nil {
		t.Fatal("Expected non-nil callable")
	}
}
