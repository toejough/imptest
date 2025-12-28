// Package v2 provides the redesigned imptest API with clearer Target/Dependency semantics.
package v2

import (
	"testing"
)

// Imp is the central coordinator for all test interactions.
// It manages the flow of calls, returns, and panics between targets and dependencies.
type Imp struct {
	t testing.TB
	// TODO: Add interaction channel and coordination logic
}

// NewImp creates a new Imp coordinator for a test.
func NewImp(t testing.TB) *Imp {
	return &Imp{
		t: t,
	}
}
