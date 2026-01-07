package zero_returns_test

import (
	"testing"

	zr "github.com/toejough/imptest/UAT/variations/signature/edge-zero-returns"
)

// TestV2_ProcessData demonstrates v2 API for zero-return functions.
func TestV2_ProcessData(t *testing.T) {
	t.Parallel()

	// Wrap and start the zero-return function
	WrapProcessData(t, zr.ProcessData).Start("test data", 42).ExpectCompletes()
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

			WrapProcessData(t, zr.ProcessData).Start(tc.data, tc.count).ExpectCompletes()
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

	WrapProcessData(t, panicFunc).Start("test", 42).ExpectPanicEquals("test panic")
}
