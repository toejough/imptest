package mockstruct_test

import (
	"testing"

	mockstruct "github.com/toejough/imptest/UAT/core/mock-struct"
)

// TestRealCalculator exercises the actual Calculator implementation.
// This test ensures the struct methods are not marked as dead code.
func TestRealCalculator(t *testing.T) {
	t.Parallel()

	calc := &mockstruct.Calculator{}

	// Use the interface helper with the real Calculator
	mockstruct.UseCalculator(calc)
}
