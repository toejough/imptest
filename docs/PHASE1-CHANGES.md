# Phase 1 Changes: Registry Infrastructure

This document details all changes made during Phase 1 of the API refactor, following TDD (RED → GREEN → REFACTOR).

## Summary

Added a package-level registry that maps `*testing.T` to `*Imp`, enabling:
- Multiple mocks/wrappers in the same test to automatically share coordination
- A single `imptest.Wait(t)` call to wait for all async expectations

## Commits

| Commit | Type | Description |
|--------|------|-------------|
| `ef6cd67` | RED | Tests for registry infrastructure (fail: not implemented) |
| `32bd5ac` | GREEN | Implement registry and Wait(t) |
| `317463f` | REFACTOR | Fix lint issues and test parallelism |

---

## Commit 1: RED Phase

**Commit:** `ef6cd67` - `test(registry): add tests for *testing.T registry infrastructure`

### New File: `/registry.go`

```go
package imptest

// GetOrCreateImp returns the Imp for the given test, creating one if needed.
// Multiple calls with the same TestReporter return the same Imp instance.
// This enables coordination between mocks and wrappers in the same test.
func GetOrCreateImp(t TestReporter) *Imp {
	// TODO: implement registry lookup
	panic("not implemented")
}

// Wait blocks until all async expectations registered under t are satisfied.
// This is the package-level wait that coordinates across all mocks/wrappers
// sharing the same TestReporter.
func Wait(t TestReporter) {
	// TODO: implement
	panic("not implemented")
}
```

**Why:** Stub implementations that define the public API. Tests can compile against these signatures but will fail at runtime with "not implemented" - proper RED state.

### New File: `/registry_test.go`

Tests written to express the desired behavior:

1. **`TestGetOrCreateImp_SameT_ReturnsSameImp`** - Same `*testing.T` should return the same `*Imp` (identity check)

2. **`TestGetOrCreateImp_DifferentT_ReturnsDifferentImp`** - Different tests get different Imps (isolation)

3. **`TestGetOrCreateImp_ConcurrentAccess`** - 100 goroutines calling with same `t` all get identical Imp (thread safety)

4. **`TestGetOrCreateImp_ConcurrentAccess_Rapid`** - Property-based test with randomized goroutine counts (robustness)

5. **`TestWait_BlocksUntilAsyncExpectationsSatisfied`** - Wait(t) blocks until expectations complete

6. **`TestWait_NoExpectations_ReturnsImmediately`** - Wait(t) returns immediately when no pending expectations

7. **`TestCleanup_RemovesEntryAfterTestCompletes`** - Registry cleanup via `t.Cleanup()`

**Why:** Tests express intent before implementation. Using gomega for assertions and rapid for property-based testing.

---

## Commit 2: GREEN Phase

**Commit:** `32bd5ac` - `feat(registry): implement *testing.T registry for Imp coordination`

### Changes to `/registry.go`

```diff
 package imptest

+import (
+	"sync"
+
+	"github.com/toejough/imptest/internal/core"
+)
+
+// registry stores Imp instances keyed by TestReporter.
+// This enables multiple mocks/wrappers in the same test to share coordination.
+var (
+	registryMu sync.Mutex
+	registry   = make(map[TestReporter]*Imp)
+)
+
+// cleanupRegistrar is the interface needed for registering cleanup functions.
+// This is satisfied by *testing.T and *testing.B.
+type cleanupRegistrar interface {
+	Cleanup(func())
+}
```

**Why:**
- `registry` map stores Imp by TestReporter (usually `*testing.T`)
- `registryMu` mutex protects concurrent access
- `cleanupRegistrar` interface allows checking if `t` supports `Cleanup()` without importing `testing` package

```diff
 func GetOrCreateImp(t TestReporter) *Imp {
-	// TODO: implement registry lookup
-	panic("not implemented")
+	registryMu.Lock()
+	defer registryMu.Unlock()
+
+	if imp, ok := registry[t]; ok {
+		return imp
+	}
+
+	imp := core.NewImp(t)
+	registry[t] = imp
+
+	// Register cleanup if the TestReporter supports it
+	if cr, ok := t.(cleanupRegistrar); ok {
+		cr.Cleanup(func() {
+			registryMu.Lock()
+			delete(registry, t)
+			registryMu.Unlock()
+		})
+	}
+
+	return imp
 }
```

**Why:**
- Lock protects concurrent access
- Check if Imp already exists (return existing)
- Create new Imp via `core.NewImp(t)`
- Register cleanup to remove entry when test completes (prevents memory leaks across test runs)

```diff
 func Wait(t TestReporter) {
-	// TODO: implement
-	panic("not implemented")
+	registryMu.Lock()
+	imp, ok := registry[t]
+	registryMu.Unlock()
+
+	if !ok {
+		return
+	}
+
+	imp.Wait()
 }
```

**Why:**
- Look up Imp for this test
- If no Imp exists, nothing to wait for - return immediately
- Otherwise delegate to `imp.Wait()` which blocks until all async expectations are satisfied

---

## Commit 3: REFACTOR Phase

**Commit:** `317463f` - `refactor(registry): fix lint issues and test parallelism`

### Changes to `/registry.go`

**Issue 1: `gochecknoglobals` lint error**

```diff
+// unexported variables.
 var (
+	//nolint:gochecknoglobals // Package-level registry is intentional for test coordination
 	registry   = make(map[TestReporter]*Imp)
+	//nolint:gochecknoglobals // Mutex for registry
 	registryMu sync.Mutex
 )
```

**Why:** The linter flags global variables, but a package-level registry is intentional here - it's the mechanism for coordinating across mock/wrapper instantiations within a single test. The `nolint` directive explains this is deliberate.

**Issue 2: `inamedparam` lint error**

```diff
 type cleanupRegistrar interface {
-	Cleanup(func())
+	Cleanup(cleanupFunc func())
 }
```

**Why:** Linter requires interface method parameters to be named. Changed from anonymous `func()` to `cleanupFunc func()`.

### Changes to `/registry_test.go`

**Issue 3: `revive` dot-imports error**

```diff
-	. "github.com/onsi/gomega"
+	"github.com/onsi/gomega"
```

**Why:** Dot imports are discouraged - they pollute the namespace and make it unclear where symbols come from.

**Issue 4: `varnamelen` errors**

```diff
-	g := NewWithT(t)
+	expect := gomega.NewWithT(t)

-	var wg sync.WaitGroup
+	var waitGroup sync.WaitGroup

-	var mu sync.Mutex
+	var orderMutex sync.Mutex
```

**Why:** Single-letter or very short variable names are flagged when the variable has a larger scope. More descriptive names improve readability.

**Issue 5: `tparallel` / `paralleltest` errors - Subtests need `t.Parallel()`**

```diff
 	t.Run("subtest1", func(t *testing.T) {
+		t.Parallel()
 		imp1 = imptest.GetOrCreateImp(t)
 	})
```

**Why:** When the parent test calls `t.Parallel()`, subtests should too for consistency.

**Issue 6: Test timing/synchronization bug**

The original test had a bug - parallel subtests don't block the parent, so assertions ran before subtests completed:

```go
// BROKEN: imp1 and imp2 are still nil when this runs
t.Run("subtest1", func(t *testing.T) {
    t.Parallel()
    imp1 = imptest.GetOrCreateImp(t)
})
t.Run("subtest2", func(t *testing.T) {
    t.Parallel()
    imp2 = imptest.GetOrCreateImp(t)
})
g.Expect(imp1).NotTo(BeIdenticalTo(imp2))  // FAILS: both nil
```

**Fix:** Use `t.Cleanup()` to defer assertion until subtests complete:

```go
t.Run("subtest1", func(t *testing.T) {
    t.Parallel()
    resultsMutex.Lock()
    results["sub1"] = imptest.GetOrCreateImp(t)
    resultsMutex.Unlock()
})
t.Run("subtest2", func(t *testing.T) {
    t.Parallel()
    resultsMutex.Lock()
    results["sub2"] = imptest.GetOrCreateImp(t)
    resultsMutex.Unlock()
})
t.Cleanup(func() {
    // This runs after all subtests complete
    expect.Expect(results["sub1"]).NotTo(BeIdenticalTo(results["sub2"]))
})
```

**Why:** Go's test framework waits for subtests to complete before running parent cleanup. This ensures the assertion runs after both subtests have stored their results.

---

## Final State

### `/registry.go` (69 lines)

```go
package imptest

import (
	"sync"

	"github.com/toejough/imptest/internal/core"
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
```

---

## New Public API

```go
// Get or create shared Imp for a test (used by generated mocks/wrappers)
imp := imptest.GetOrCreateImp(t)

// Wait for all async expectations under t
imptest.Wait(t)
```

These will be used by the generated mock/wrapper code in Phase 2 and Phase 3 to automatically coordinate via the test's `*testing.T`.
