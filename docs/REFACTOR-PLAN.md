# API Refactor Plan: Controller/Wait Simplification

## Goal

Simplify imptest's API by:

1. Eliminating explicit `NewImp()` calls - auto-share coordination via `*testing.T`
2. Two-return style for mocks - `mock, imp := MockX(t)` instead of handle struct
3. Package-level wait - `imptest.Wait(t)` instead of `h.Controller.Wait()`

## Before/After Comparison

```go
// BEFORE
func Test_PrintSum(t *testing.T) {
    t.Parallel()
    imp := imptest.NewImp(t)
    h := MockIntOps(imp)
    wrapper := WrapPrintSum(imp, run.PrintSum).Start(10, 32, h.Mock)

    h.Method.Add.ExpectCalledWithExactly(10, 32).InjectReturnValues(42)
    h.Method.Format.ExpectCalledWithExactly(42).InjectReturnValues("42")

    wrapper.ExpectReturnsEqual(10, 32, "42")
}

// AFTER
func Test_PrintSum(t *testing.T) {
    t.Parallel()
    mock, imp := MockIntOps(t)
    wrapper := WrapPrintSum(t, run.PrintSum).Start(10, 32, mock)

    imp.Add.ExpectCalledWithExactly(10, 32).InjectReturnValues(42)
    imp.Format.ExpectCalledWithExactly(42).InjectReturnValues("42")

    wrapper.ExpectReturnsEqual(10, 32, "42")
}
```

## Changes Summary

| Aspect          | Current                                       | Proposed                                 |
| --------------- | --------------------------------------------- | ---------------------------------------- |
| Coordination    | `imp := imptest.NewImp(t)` then pass `imp`    | Auto-share via `TestReporter` registry   |
| Mock creation   | `h := MockIntOps(imp)` returns handle         | `mock, imp := MockIntOps(t)` two returns |
| Interface usage | `h.Mock`                                      | `mock` (first return)                    |
| Expectations    | `h.Method.Add.ExpectCalledWith...`            | `imp.Add.ExpectCalledWith...`            |
| Async           | `h.Method.Add.Eventually.ExpectCalledWith...` | `imp.Add.Eventually.ExpectCalledWith...` |
| Wait            | `h.Controller.Wait()`                         | `imptest.Wait(t)`                        |

## Decisions

- **Export `GetOrCreateImp`?** - Yes (changed from original "no"). Generated code needs to call it, and keeping it internal would require awkward workarounds.
- **Backward compatibility?** - No. Remove `NewImp()` outright. Document the new way.
- **Generated type names?** - `*IntOpsImp` - fits the style/character of the repo.
- **Function mocks?** - Yes, `mockFn, imp := MockMyFunc(t)` with `imp.ExpectCalledWith...` directly.
- **Use `TestReporter` vs `*testing.T`?** - Use `TestReporter` interface for flexibility (supports `*testing.T`, `*testing.B`, and test doubles).

---

## TDD Workflow

Each phase follows this cycle:

1. **RED**: Write/update tests expressing intent of the change. Commit: `test(scope): ...`
2. **GREEN**: Implement to make tests pass. Commit: `feat(scope): ...` or `refactor(scope): ...`
3. **REFACTOR**: Fix lints, run `targ check`, clean up. Commit: `refactor(scope): ...`
4. **REVIEW**: Stop for user review before proceeding to next phase.

**Testing tools**: Use imptest itself, gomega matchers, and rapid for property-based tests. Extract dependencies as interfaces where needed to enable testing.

---

## Implementation Phases

### Phase 1: Core Library - Registry Infrastructure ✅ COMPLETE

**Goal**: Add internal registry keyed by `*testing.T` and public `Wait(t)` function.

**Status**: Complete (commits `ef6cd67`, `32bd5ac`, `317463f`)

**Files to create/modify:**

- [x] New file: `registry.go` — Created
- [x] Modify: `imptest.go` (remove `NewImp` export) — Done

**Changes:**

- [x] Add internal registry: `map[TestReporter]*Imp` with mutex protection
  - *Deviation*: Used `TestReporter` interface instead of `*testing.T` for flexibility
- [x] Add `GetOrCreateImp(t TestReporter) *Imp` function
  - *Deviation*: Exported (not internal) because generated code needs to call it
- [x] Add `imptest.Wait(t TestReporter)` public function
- [x] Add cleanup hook via `t.Cleanup()` to remove entry when test completes
- [x] Remove `NewImp()` from public API

**Tests (using imptest, gomega, rapid):**

- [x] Registry returns same Imp for same `t` (`TestGetOrCreateImp_SameT_ReturnsSameImp`)
- [x] Registry returns different Imp for different `t` (`TestGetOrCreateImp_DifferentT_ReturnsDifferentImp`)
- [x] `Wait(t)` blocks until async expectations satisfied (`TestWait_BlocksUntilAsyncExpectationsSatisfied`)
- [x] Cleanup removes entry after test completes (`TestCleanup_RemovesEntryAfterTestCompletes`)
- [x] Property: concurrent access to registry is safe (`TestGetOrCreateImp_ConcurrentAccess`, `TestGetOrCreateImp_ConcurrentAccess_Rapid`)
- [x] Additional: `Wait(t)` returns immediately with no expectations (`TestWait_NoExpectations_ReturnsImmediately`)

**Docs**: Updated README.md examples to use `GetOrCreateImp(t)` instead of `NewImp(t)`.

---

### Phase 2: Code Generation - Mock Templates ✅ COMPLETE

**Goal**: Change generated mocks to two-return style accepting `*testing.T`.

**Status**: Complete (commit `e3bec4c`)

**Files modified:**

- [x] `impgen/run/5_generate/template_content.go` — Template strings for mock generation
- [x] `impgen/run/5_generate/templates.go` — Template data structures
- [x] `impgen/run/5_generate/text_templates.go` — Template write functions
- [x] `impgen/run/5_generate/mock_interface.go` — Interface mock generation logic
- [x] `internal/core/controller.go` — Removed `Controller` struct
- *Deviation*: Plan listed `dependency.go` but actual templates were in different files

**Changes:**

- [x] Change generated function signature:
  ```go
  // FROM:
  func MockIntOps(imp *imptest.Imp) *IntOpsHandle
  // TO:
  func MockIntOps(t imptest.TestReporter) (IntOps, *IntOpsImp)
  ```
- [x] Generated function calls `imptest.GetOrCreateImp(t)` (already exported in Phase 1)
- [x] First return: mock implementation
- [x] Second return: expectation handle (renamed to `*IntOpsImp`)
- [x] Remove `Handle` struct generation
- [x] Remove `Controller` field
- [x] Function mocks also updated: `mockFn, imp := MockMyFunc(t)` with `imp.ExpectCalledWith...`
  - *Note*: `WriteFuncDepMockStruct` template emptied (Handle no longer needed)

**Tests:**

- [x] Generated code compiles with new signature
- [x] Generated mock returns correct types
- [x] Multiple mocks with same `t` share coordination
- [x] All 84 affected files regenerated and tests updated

**Additional fixes during implementation:**

- Fixed nilaway false positives for generic mocks (added `//nolint:nilaway` directives)
- Fixed coverage issue for `WriteFuncDepMockStruct` (empty template, removed unreachable panic)
- Fixed race condition in `interfaces_test.go` (incorrect expectations for mock interfaces)

**Docs**: Updated README with new two-return style examples throughout.

---

### Phase 3: Code Generation - Wrapper Templates ✅ COMPLETE (in Phase 1)

**Goal**: Change generated wrappers to accept `*testing.T` directly.

**Status**: Complete — Wrappers were updated as part of Phase 1 when `GetOrCreateImp(t)` was introduced.

**Changes:**

- [x] Wrapper constructors accept `TestReporter` (e.g., `WrapPrintSum(t, fn)`)
- [x] Generated function calls `imptest.GetOrCreateImp(t)`
- [x] Wrapper coordinates with mocks via shared registry

*Note*: This was effectively combined with Phase 1 since wrappers needed the same registry infrastructure.

---

### Phase 4: Update UAT Tests & Final Cleanup ✅ COMPLETE (combined with Phase 2)

**Goal**: Migrate all existing tests to new API, verify everything works.

**Status**: Complete — Combined with Phase 2 (commit `e3bec4c`)

**Files modified:**

- [x] All files in `UAT/core/` (18 files)
- [x] All files in `UAT/variations/` (48 files)
- [x] All generated mock/wrapper files (66 files)
- Total: 84 files changed, +975 insertions, -1438 deletions

**Changes:**

- [x] Regenerate all mocks and wrappers: `go generate ./...`
- [x] Update all test files:
  - Remove `imp := imptest.NewImp(t)` lines (already done in Phase 1)
  - Change `h := MockX(imp)` to `mock, imp := MockX(t)`
  - Change `h.Mock` to `mock`
  - Change `h.Method.X` to `imp.X`
  - Change `h.Controller.Wait()` to `imptest.Wait(t)`
  - Change `WrapX(imp, ...)` to `WrapX(t, ...)` (already done in Phase 1)
- [x] Remove `Handle` types from generated code
- [x] Remove `Controller` struct from `internal/core/controller.go`

**Docs**: README.md updated with all new API examples.

**Verification:**

- [x] All UAT tests pass
- [x] `targ check` passes
- [x] Generated code compiles
- [x] README examples are accurate

**Deviations from plan:**

- Phases 2-4 were combined into a single implementation effort
- Added blank imports to some test files to help impgen resolve external packages
- Fixed pre-existing race condition in `interfaces_test.go`

---

## Migration Guide (for users)

```diff
func Test_Example(t *testing.T) {
    t.Parallel()
-   imp := imptest.NewImp(t)
-   h := MockDependency(imp)
-   wrapper := WrapFunction(imp, Function).Start(h.Mock, arg1)
+   mock, imp := MockDependency(t)
+   wrapper := WrapFunction(t, Function).Start(mock, arg1)

-   h.Method.DoThing.ExpectCalledWithExactly(arg1).InjectReturnValues(result)
+   imp.DoThing.ExpectCalledWithExactly(arg1).InjectReturnValues(result)

-   h.Controller.Wait()
+   imptest.Wait(t)

    wrapper.ExpectReturnsEqual(expected)
}
```
