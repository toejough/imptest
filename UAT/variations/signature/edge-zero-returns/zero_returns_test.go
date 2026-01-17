//go:generate impgen zeroreturns.ProcessData --target

package zeroreturns_test

import (
	"testing"

	"github.com/toejough/imptest/UAT/variations/signature/edge-zero-returns"
)

// TestV2_ProcessData demonstrates v2 API for zero-return functions.
func TestV2_ProcessData(t *testing.T) {
	t.Parallel()

	// Start the zero-return function
	StartProcessData(t, zeroreturns.ProcessData, "test data", 42).ExpectCompletes()
}

// TestV2_ProcessData_MultipleArgs tests with various argument combinations.
func TestV2_ProcessData_MultipleArgs(t *testing.T) {
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

			StartProcessData(t, zeroreturns.ProcessData, tc.data, tc.count).ExpectCompletes()
		})
	}
}

// TestV2_ProcessData_Panic demonstrates panic verification for zero-return functions.
func TestV2_ProcessData_Panic(t *testing.T) {
	t.Parallel()

	// Define a function that panics
	panicFunc := func(_ string, _ int) {
		panic("test panic")
	}

	StartProcessData(t, panicFunc, "test", 42).ExpectPanic("test panic")
}
