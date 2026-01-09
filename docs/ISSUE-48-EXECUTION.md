# Issue #48 Execution Plan

**Goal**: Make Eventually() async with test handle pattern (`h.Mock`, `h.Method`, `h.Controller`)

**Reference**: See `docs/ISSUE-48-PLAN.md` for full design details.

---

## Commit Format

Use conventional commits with the `AI-Used` trailer:

```
<type>(scope): description

Optional body explaining why, not what.

AI-Used: [claude]
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

---

## Progress Log

| Phase | Status | Commit |
|-------|--------|--------|
| 1 | complete | dad78de |
| 2 | complete | 8252e8d |
| 3 | complete | 4cb58c6 |

---

## Phase 1: Test Handle Pattern (All-at-Once Breaking Change)

Codegen change affects ALL tests at once. Must update everything together.

### 1.1 RED: Update ALL test files to new pattern
- [x] Update all `*_test.go` files:
  - `mock := MockFoo(t)` → `h := MockFoo(t)`
  - `mock.MethodName.Expect...` → `h.Method.MethodName.Expect...`
  - `mock.Interface()` → `h.Mock`
  - `mock.Func()` → `h.Mock` (for function mocks)
- [x] Tests will fail (generated code still old pattern)
- [x] Commit: `test(uat): update all tests to test handle pattern (RED)`

### 1.2 GREEN: Update ALL codegen templates
- [x] Update `depConstructorTmpl` - interface/struct mocks
- [x] Update `funcDepConstructorTmpl` - function/functype mocks
- [x] Update `targetConstructorTmpl` - function wrappers
- [x] Update `interfaceTargetConstructorTmpl` - interface/struct wrappers
- [x] Add Handle struct templates
- [x] Add Methods struct templates
- [x] Regenerate all: `go generate ./...`
- [x] All tests pass
- [x] Commit: `refactor(api): transform mock API to test handle pattern` (dad78de)

### 1.3 REFACTOR: mage check
- [x] Run `mage check`
- [x] Fix any linter issues
- [x] (Combined with 1.2 commit)

**CHECKPOINT**: Complete

---

## Phase 2: Async Eventually() - Core

### 2.1 RED: Write failing test for async Eventually
- [x] In `UAT/variations/concurrency/eventually/`
- [x] Test expects Eventually() to be non-blocking
- [x] Test expects `h.Controller.Wait()` to block
- [x] (Combined with 2.2)

### 2.2 GREEN: Implement PendingExpectation + Controller.Wait()
- [x] Add `PendingExpectation` struct to `imptest/controller.go`
- [x] Add `Controller.Wait()` and `Controller.SetTimeout()`
- [x] Modify dispatch loop to check pending expectations
- [x] Modify `DependencyMethod` for async registration
- [x] Implement callback pattern on `InjectReturnValues`/`InjectPanicValue`
- [x] Tests pass
- [x] Commit: `feat(eventually): implement async Eventually with Wait()` (8252e8d)

### 2.3 REFACTOR: mage check
- [x] Fix any linter issues
- [x] (Combined with 2.2 commit)

**CHECKPOINT**: Complete

---

## Phase 3: Target Wrapper Eventually

### 3.1 RED: Test async Eventually on target wrappers
- [x] Test `handle.Eventually().ExpectReturnsEqual()`
- [x] Test `handle.Eventually().ExpectPanicEquals()`
- [x] (Combined with 3.2)

### 3.2 GREEN: Implement
- [x] Add `TargetController` and `PendingCompletion` structs
- [x] Add `Eventually()` to generated CallHandle types
- [x] Add `Controller` field to wrapper Handle
- [x] Update templates for target wrappers
- [x] Tests pass
- [x] Commit: `feat(target-wrapper): implement async Eventually() with Controller.Wait()` (4cb58c6)

### 3.3 REFACTOR: mage check
- [x] Fixed linter issues (elseif, varnamelen)
- [x] (Combined with 3.2 commit)

**CHECKPOINT**: Complete

---

## Phase 4: Documentation

### 4.1: Update docs
- [ ] README.md examples
- [ ] TAXONOMY.md concurrency section
- [ ] Commit: `docs: update for async Eventually and test handle pattern`

**DONE**

---

## Current Phase: 4 (Documentation)

**Next Action**: Update documentation for the new test handle pattern and async Eventually() API.

Phases 1-3 complete. Both dependency mocks and target wrappers support async Eventually() with `h.Controller.Wait()`.
