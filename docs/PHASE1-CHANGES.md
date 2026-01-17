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
| `ea2d848` | DOCS | Update plan with Phase 1 status and deviations |
| `6e9fc5e` | REFACTOR | Remove NewImp export, update templates and docs |
| `cb6dacb` | REFACTOR | Use dot imports and short var names in tests |

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
+	. "github.com/onsi/gomega" //nolint:revive // dot import preferred for test readability
```

**Why:** The linter discourages dot imports, but they improve test readability with gomega. Added nolint directive to allow this.

**Issue 4: `varnamelen` errors**

Updated `dev/golangci-lint.toml` to allow conventional short names in tests:

```diff
-ignore-names = ['t', 'b', 'rt', 'a']
+ignore-names = ['t', 'b', 'rt', 'a', 'g', 'wg', 'mu']
```

**Why:** Short names like `g` (gomega), `wg` (WaitGroup), and `mu` (mutex) are idiomatic in Go tests. Updated linter config rather than lengthening names.

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
    mu.Lock()
    results["sub1"] = imptest.GetOrCreateImp(t)
    mu.Unlock()
})
t.Run("subtest2", func(t *testing.T) {
    t.Parallel()
    mu.Lock()
    results["sub2"] = imptest.GetOrCreateImp(t)
    mu.Unlock()
})
t.Cleanup(func() {
    // This runs after all subtests complete
    g.Expect(results["sub1"]).NotTo(BeIdenticalTo(results["sub2"]))
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

---

## Commit 4: Documentation Update

**Commit:** `ea2d848` - `docs(plan): update Phase 1 status and record deviations`

Updated `REFACTOR-PLAN.md` to record:
- Phase 1 completion status with commit references
- Deviations from original plan (exported `GetOrCreateImp`, used `TestReporter` interface)
- Updated Decisions section with rationale

---

## Commit 5: Remove NewImp Export

**Commit:** `6e9fc5e` - `refactor(api): remove NewImp, use GetOrCreateImp everywhere`

### Changes to `/imptest.go`

**Remove NewImp export:**

```diff
-// NewImp creates a new Imp coordinator.
-func NewImp(t TestReporter) *Imp {
-	return core.NewImp(t)
-}
```

**Update doc comment:**

```diff
 // # User API
 //
 // These are meant to be used directly in test code:
 //
 //   - [Any] - matcher that accepts any value
 //   - [Satisfies] - matcher using a custom predicate function
 //   - [TestReporter] - interface for test frameworks (usually *testing.T)
-//   - [NewImp] - create shared coordinator for centralized control of multiple mocks/wrappers
+//   - [GetOrCreateImp] - get/create shared coordinator for a test (used by generated code)
+//   - [Wait] - block until all async expectations for a test are satisfied
```

**Why:** `NewImp` is replaced by `GetOrCreateImp` which uses the registry. The internal `core.NewImp` is still used by `GetOrCreateImp`, but is no longer exported.

### Changes to `/impgen/run/5_generate/template_content.go`

**Update mock constructor template:**

```diff
-	ctrl := {{.PkgImptest}}.NewImp(t)
+	ctrl := {{.PkgImptest}}.GetOrCreateImp(t)
```

**Why:** Generated mocks now use the registry-based `GetOrCreateImp` instead of creating standalone `Imp` instances. This enables automatic coordination between multiple mocks in the same test.

### Changes to `/README.md`

All examples updated from:
```go
imp := imptest.NewImp(t)
```

To:
```go
imp := imptest.GetOrCreateImp(t)
```

**Files updated:** 5 occurrences across Quick Start, Flexible Matching, Expecting Panics, Channel Patterns, and Comparison sections.

### Changes to Generated Files

All 45+ generated mock files regenerated with new template:

```diff
 func MockIntOps(t _imptest.TestReporter) *IntOpsMockHandle {
-	ctrl := _imptest.NewImp(t)
+	ctrl := _imptest.GetOrCreateImp(t)
```

### Changes to Manual Test File

`/UAT/variations/behavior/typesafe-getargs/manual_typed_wrappers_test.go`:

```diff
 func NewTypesafeCalculatorMock(t imptest.TestReporter) *TypesafeCalculatorMockHandle {
-	ctrl := imptest.NewImp(t)
+	ctrl := imptest.GetOrCreateImp(t)
```

---

## Commit 6: Test Style Preferences

**Commit:** `cb6dacb` - `refactor(test): use dot imports and short var names`

### Changes to `/dev/golangci-lint.toml`

```diff
-ignore-names = ['t', 'b', 'rt', 'a']
+ignore-names = ['t', 'b', 'rt', 'a', 'g', 'wg', 'mu']
```

**Why:** Allow idiomatic short names in tests rather than forcing verbose alternatives.

### Changes to `/registry_test.go`

```diff
-	"github.com/onsi/gomega"
+	. "github.com/onsi/gomega" //nolint:revive // dot import preferred for test readability
```

```diff
-	expect := gomega.NewWithT(t)
+	g := NewWithT(t)

-	var waitGroup sync.WaitGroup
+	var wg sync.WaitGroup

-	var orderMutex sync.Mutex
+	var mu sync.Mutex
```

**Why:** Shorter, idiomatic names are preferred in tests. Dot import for gomega improves readability.

---

## Phase 1 Complete

**New Public API:**

```go
// Get or create shared Imp for a test (used by generated mocks/wrappers)
imp := imptest.GetOrCreateImp(t)

// Wait for all async expectations under t
imptest.Wait(t)
```

**Removed from Public API:**

```go
// No longer exported - use GetOrCreateImp instead
imptest.NewImp(t)  // REMOVED
```

**Key Design Decisions:**

1. **Exported `GetOrCreateImp`** (changed from original plan) - Generated code needs to call it
2. **Used `TestReporter` interface** instead of `*testing.T` - More flexible, supports test doubles
3. **Registry uses mutex** - Thread-safe for parallel tests
4. **Automatic cleanup** via `t.Cleanup()` - Prevents memory leaks
