package imptest

import (
	"sync"

	"github.com/toejough/imptest/internal/core"
)

// registry stores Imp instances keyed by TestReporter.
// This enables multiple mocks/wrappers in the same test to share coordination.
var (
	registryMu sync.Mutex
	registry   = make(map[TestReporter]*Imp)
)

// cleanupRegistrar is the interface needed for registering cleanup functions.
// This is satisfied by *testing.T and *testing.B.
type cleanupRegistrar interface {
	Cleanup(func())
}

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

	imp := core.NewImp(t)
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
