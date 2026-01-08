package mockmethod_test

import (
	"testing"

	mockmethod "github.com/toejough/imptest/UAT/core/mock-method"
)

// TestRealCounter exercises the actual Counter implementation.
// This test ensures the struct methods are not marked as dead code.
func TestRealCounter(t *testing.T) {
	t.Parallel()

	counter := &mockmethod.Counter{}

	// Use all the methods to prevent deadcode tool from removing them
	_ = counter.Add(5)
	_ = counter.Inc()
	_ = counter.Dec()
	_ = counter.Value()
}
