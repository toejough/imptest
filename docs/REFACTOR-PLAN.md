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
| Coordination    | `imp := imptest.NewImp(t)` then pass `imp`    | Auto-share via `*testing.T` registry     |
| Mock creation   | `h := MockIntOps(imp)` returns handle         | `mock, imp := MockIntOps(t)` two returns |
| Interface usage | `h.Mock`                                      | `mock` (first return)                    |
| Expectations    | `h.Method.Add.ExpectCalledWith...`            | `imp.Add.ExpectCalledWith...`            |
| Async           | `h.Method.Add.Eventually.ExpectCalledWith...` | `imp.Add.Eventually.ExpectCalledWith...` |
| Wait            | `h.Controller.Wait()`                         | `imptest.Wait(t)`                        |

## Decisions

- **Export `getOrCreateImp`?** - No, keep it internal.
- **Backward compatibility?** - No. Remove `NewImp()` outright. Document the new way.
- **Generated type names?** - `*IntOpsImp` - fits the style/character of the repo.
- **Function mocks?** - Yes, `mockFn, imp := MockMyFunc(t)` with `imp.ExpectCalledWith...` directly.

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

### Phase 1: Core Library - Registry Infrastructure

**Goal**: Add internal registry keyed by `*testing.T` and public `Wait(t)` function.

**Files to create/modify:**

- New file: `registry.go`
- Modify: `imptest.go` (remove `NewImp` export)

**Changes:**

- [ ] Add internal registry: `map[*testing.T]*Imp` with mutex protection
- [ ] Add `getOrCreateImp(t *testing.T) *Imp` internal function
- [ ] Add `imptest.Wait(t *testing.T)` public function
- [ ] Add cleanup hook via `t.Cleanup()` to remove entry when test completes
- [ ] Remove `NewImp()` from public API

**Tests (using imptest, gomega, rapid):**

- [ ] Registry returns same Imp for same `t`
- [ ] Registry returns different Imp for different `t`
- [ ] `Wait(t)` blocks until async expectations satisfied
- [ ] Cleanup removes entry after test completes
- [ ] Property: concurrent access to registry is safe

**Docs**: Update README examples that mention `NewImp()`.

---

### Phase 2: Code Generation - Mock Templates

**Goal**: Change generated mocks to two-return style accepting `*testing.T`.

**Files to modify:**

- `impgen/run/5_generate/dependency.go` (template file)

**Changes:**

- [ ] Change generated function signature:
  ```go
  // FROM:
  func MockIntOps(imp *imptest.Imp) *IntOpsHandle
  // TO:
  func MockIntOps(t *testing.T) (IntOps, *IntOpsImp)
  ```
- [ ] Generated function calls internal `getOrCreateImp(t)` (export minimal helper if needed)
- [ ] First return: mock implementation
- [ ] Second return: expectation handle (rename from `*IntOpsMethod` to `*IntOpsImp`)
- [ ] Remove `Handle` struct generation
- [ ] Remove `Controller` field

**Tests:**

- [ ] Generated code compiles with new signature
- [ ] Generated mock returns correct types
- [ ] Multiple mocks with same `t` share coordination

**Docs**: Update README mock creation examples.

---

### Phase 3: Code Generation - Wrapper Templates

**Goal**: Change generated wrappers to accept `*testing.T` directly.

**Files to modify:**

- `impgen/run/5_generate/target.go` (template file)

**Changes:**

- [ ] Change generated function signature:
  ```go
  // FROM:
  func WrapPrintSum(imp *imptest.Imp, fn func(...) ...) *PrintSumWrapper
  // TO:
  func WrapPrintSum(t *testing.T, fn func(...) ...) *PrintSumWrapper
  ```
- [ ] Generated function calls internal `getOrCreateImp(t)`

**Tests:**

- [ ] Generated code compiles with new signature
- [ ] Wrapper coordinates with mocks via shared registry

**Docs**: Update README wrapper examples.

---

### Phase 4: Update UAT Tests & Final Cleanup

**Goal**: Migrate all existing tests to new API, verify everything works.

**Files to modify:**

- All files in `UAT/core/`
- All files in `UAT/variations/`
- All generated mock/wrapper files

**Changes:**

- [ ] Regenerate all mocks and wrappers: `go generate ./...`
- [ ] Update all test files:
  - Remove `imp := imptest.NewImp(t)` lines
  - Change `h := MockX(imp)` to `mock, imp := MockX(t)`
  - Change `h.Mock` to `mock`
  - Change `h.Method.X` to `imp.X`
  - Change `h.Controller.Wait()` to `imptest.Wait(t)`
  - Change `WrapX(imp, ...)` to `WrapX(t, ...)`
- [ ] Remove any remaining `Handle` types or `Controller` references

**Docs**: Final pass on README and TAXONOMY.md to ensure all examples use new API.

**Verification:**

- [ ] All UAT tests pass
- [ ] `targ check` passes
- [ ] Generated code compiles
- [ ] README examples are accurate

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
