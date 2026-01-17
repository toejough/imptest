package core

import (
	"sync"
	"time"
)

// GetOrCreateImp returns the Imp for the given test, creating one if needed.
// Multiple calls with the same TestReporter return the same Imp instance.
// This enables coordination between mocks and wrappers in the same test.
//
// If the TestReporter supports Cleanup (like *testing.T), the Imp is
// automatically removed from the registry when the test completes.
func GetOrCreateImp(t TestReporter) *Imp {
	registryMu.Lock()
	defer registryMu.Unlock()

	if imp, ok := registry[t]; ok {
		return imp
	}

	imp := NewImp(t)
	registry[t] = imp

	// Register cleanup if the TestReporter supports it
	if cr, ok := t.(cleanupRegistrar); ok {
		cr.Cleanup(func() {
			registryMu.Lock()
			delete(registry, t)
			registryMu.Unlock()
		})
	}

	return imp
}

// SetTimeout configures the timeout for all blocking operations in the test.
// A duration of 0 means no timeout (block forever).
//
// If no Imp has been created for t yet, one is created.
func SetTimeout(t TestReporter, d time.Duration) {
	GetOrCreateImp(t).SetTimeout(d)
}

// Wait blocks until all async expectations registered under t are satisfied.
// This is the package-level wait that coordinates across all mocks/wrappers
// sharing the same TestReporter.
//
// If no Imp has been created for t yet, Wait returns immediately.
func Wait(t TestReporter) {
	registryMu.Lock()

	imp, ok := registry[t]

	registryMu.Unlock()

	if !ok {
		return
	}

	imp.Wait()
}

// unexported variables.
var (
	//nolint:gochecknoglobals // Package-level registry is intentional for test coordination
	registry = make(map[TestReporter]*Imp)
	//nolint:gochecknoglobals // Mutex for registry
	registryMu sync.Mutex
)

// cleanupRegistrar is the interface needed for registering cleanup functions.
// This is satisfied by *testing.T and *testing.B.
type cleanupRegistrar interface {
	Cleanup(cleanupFunc func())
}
