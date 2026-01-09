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
| 1 | pending | - |

---

## Phase 1: Test Handle Pattern (All-at-Once Breaking Change)

Codegen change affects ALL tests at once. Must update everything together.

### 1.1 RED: Update ALL test files to new pattern
- [ ] Update all `*_test.go` files:
  - `mock := MockFoo(t)` → `h := MockFoo(t)`
  - `mock.MethodName.Expect...` → `h.Method.MethodName.Expect...`
  - `mock.Interface()` → `h.Mock`
  - `mock.Func()` → `h.Mock` (for function mocks)
- [ ] Tests will fail (generated code still old pattern)
- [ ] Commit: `test(uat): update all tests to test handle pattern (RED)`

### 1.2 GREEN: Update ALL codegen templates
- [ ] Update `depConstructorTmpl` - interface/struct mocks
- [ ] Update `funcDepConstructorTmpl` - function/functype mocks
- [ ] Update `targetConstructorTmpl` - function wrappers
- [ ] Update `interfaceTargetConstructorTmpl` - interface/struct wrappers
- [ ] Add Handle struct templates
- [ ] Add Methods struct templates
- [ ] Regenerate all: `go generate ./...`
- [ ] All tests pass
- [ ] Commit: `feat(codegen): implement test handle pattern`

### 1.3 REFACTOR: mage check
- [ ] Run `mage check`
- [ ] Fix any linter issues
- [ ] Commit: `refactor: fix linter issues`

**CHECKPOINT**: Stop here, update log, check in with user.

---

## Phase 2: Async Eventually() - Core

### 2.1 RED: Write failing test for async Eventually
- [ ] In `UAT/variations/concurrency/eventually/`
- [ ] Test expects Eventually() to be non-blocking
- [ ] Test expects `h.Controller.Wait()` to block
- [ ] Commit: `test(eventually): expect async behavior (RED)`

### 2.2 GREEN: Implement PendingExpectation + Controller.Wait()
- [ ] Add `PendingExpectation` struct to `imptest/controller.go`
- [ ] Add `Controller.Wait()` and `Controller.SetTimeout()`
- [ ] Modify dispatch loop to check pending expectations
- [ ] Modify `DependencyMethod` for async registration
- [ ] Implement callback pattern on `InjectReturnValues`/`InjectPanicValue`
- [ ] Tests pass
- [ ] Commit: `feat(eventually): implement async Eventually with Wait()`

### 2.3 REFACTOR: mage check
- [ ] Fix any linter issues
- [ ] Commit: `refactor: fix linter issues`

**CHECKPOINT**

---

## Phase 3: Target Wrapper Eventually (if needed)

### 3.1 RED: Test async Eventually on target wrappers
- [ ] Test `handle.Eventually().ExpectReturnsEqual()`
- [ ] Commit RED

### 3.2 GREEN: Implement
- [ ] Add `PendingCompletion` struct
- [ ] Add `Eventually()` to `CallableController`
- [ ] Tests pass
- [ ] Commit GREEN

### 3.3 REFACTOR: mage check
- [ ] Commit if needed

**CHECKPOINT**

---

## Phase 4: Documentation

### 4.1: Update docs
- [ ] README.md examples
- [ ] TAXONOMY.md concurrency section
- [ ] Commit: `docs: update for async Eventually and test handle pattern`

**DONE**

---

## Current Phase: 1.1

**Next Action**: Update ALL `*_test.go` files to use new test handle pattern (`h := MockFoo(t)`, `h.Method.X`, `h.Mock`).
