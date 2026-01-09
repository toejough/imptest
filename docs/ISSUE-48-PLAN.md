# Issue #48: Make Eventually() Truly Async with h.Controller.Wait()

## Summary

Transform Eventually() from blocking to non-blocking. Expectations register immediately with pre-configured return values, then `h.Controller.Wait()` blocks until all are satisfied.

**Breaking Changes**:
1. Mock constructor now returns a test handle struct with `h.Mock`, `h.Method`, `h.Controller`
2. Existing tests using Eventually() must add `h.Controller.Wait()` after expectations
3. Eliminates ALL naming conflicts by separating mock, methods, and controller

---

## Current vs New Behavior

### Current (Blocking)
```go
mock := MockFoo(t)
go func() { code_under_test(mock.Interface()) }()
mock.MethodA.Eventually().ExpectCalledWithExactly(args).InjectReturnValues(vals) // BLOCKS
```

### New (Async with Test Handle)
```go
h := MockFoo(t)
h.Controller.SetTimeout(5 * time.Second) // Optional - applies to ALL blocking ops

// Start code under test FIRST (imptest's core value - no upfront setup required)
go func() { code_under_test(h.Mock) }()  // h.Mock IS the interface, no .Interface() needed

// Set expectations (non-blocking now)
h.Method.MethodA.Eventually().ExpectCalledWithExactly(args).InjectReturnValues(vals) // NON-BLOCKING
h.Method.MethodB.Eventually().ExpectCalledWithMatches(matcher).InjectPanicValue(err) // NON-BLOCKING

h.Controller.Wait() // BLOCKS until all expectations matched (respects timeout)
```

**Key Points**:
- **Zero naming conflicts**: `h.Mock` (interface), `h.Method` (expectations), `h.Controller` (wait/timeout)
- Methods are on `h.Method.MethodName`, not directly on the mock struct
- InjectReturnValues()/InjectPanicValue() can be called before OR after Wait() - uses callback pattern

---

## Design Decisions (User Confirmed)

1. **API Change**: Replace current Eventually() behavior (breaking change)
2. **Unmatched calls**: Queue for later matching (same as current)
3. **Timeout**: Configured on Controller via `h.Controller.SetTimeout(d)`, applies to ALL blocking ops
4. **InjectReturnValues/InjectPanicValue timing**: Can be called before OR after Wait() (callback pattern)
5. **Migration**: Big bang - existing tests must update to test handle pattern
6. **No naming conflicts**: Test handle struct separates Mock/Method/Controller completely

---

## Implementation Plan

### Phase 1: Core Infrastructure (`imptest/`)

#### Step 1.1: Add PendingExpectation struct
**File:** `imptest/imp.go`

```go
type PendingExpectation struct {
    MethodName     string
    Validator      func([]any) error
    ReturnValues   []any           // nil until InjectReturnValues called
    PanicValue     any
    IsPanic        bool
    ValidatorMatch bool            // true when a call matched the validator
    Injected       bool            // true when Inject* was called
    responseChan   chan<- GenericResponse // set when validator matches, used for lazy injection
    done           chan struct{}   // signals when BOTH matched AND injected
}
```

**Callback pattern for InjectReturnValues AND InjectPanicValue:**
- When validator matches: set `ValidatorMatch=true`, store `responseChan`
- If `Injected` already true: send response immediately
- When InjectReturnValues/InjectPanicValue called: set `Injected=true`, store values
- If `ValidatorMatch` already true: send response immediately

**Applies to all mock AND wrapper types:**
- Interface mocks (--dependency on interface)
- Struct mocks (--dependency on struct)
- Function mocks (--dependency on function)
- FuncType mocks (--dependency on function type)
- Function wrappers (--target on function)
- Interface wrappers (--target on interface)
- Struct wrappers (--target on struct)

#### Step 1.2: Extend Controller struct
**File:** `imptest/controller.go`

Add fields:
- `timeout time.Duration` // 0 = no timeout (block forever)
- `pendingMu sync.Mutex`
- `pendingExpectations []*PendingExpectation`

Add methods:
- `SetTimeout(d time.Duration)` - configure timeout for all blocking ops
- `Wait()` - blocks until all pending expectations satisfied
- `RegisterPendingExpectation(methodName string, validator func([]any) error) *PendingExpectation`

#### Step 1.3: Update Imp struct
**File:** `imptest/imp.go`

- Imp already has `Controller` field - no changes needed for pending expectations
- Pending state lives on Controller, not Imp

#### Step 1.4: Modify dispatch loop
**File:** `imptest/controller.go` or `imptest/imp.go`

When call arrives:
1. First check pending expectations for match (validator passes)
2. If match found AND already injected: send response, mark done
3. If match found but NOT injected: store responseChan, wait for injection
4. If no match: fall through to existing queue/waiter logic (for Ordered mode)

#### Step 1.5: Modify DependencyMethod
**File:** `imptest/dependency.go`

Change `ExpectCalledWithExactly()` AND `ExpectCalledWithMatches()`:
- When `eventually=true`: register pending expectation, return immediately (non-blocking)
- Return `*PendingExpectation` (or wrapper) with `InjectReturnValues()` AND `InjectPanicValue()`
- Both Inject methods use callback pattern: if validator already matched, send response; otherwise store for later

This affects all mock types since they all use DependencyMethod:
- Interface mocks: each method has a DependencyMethod
- Struct mocks: each method has a DependencyMethod
- Function mocks: the function itself has a DependencyMethod
- FuncType mocks: the function type has a DependencyMethod

#### Step 1.6: Add Eventually/Wait to CallableController (Target Wrappers)
**File:** `imptest/callable_controller.go` or new file

Target wrappers (--target) use CallableController for function execution. Add Eventually support:

```go
// Usage pattern:
h := WrapFoo(t, realFoo)

callHandle := h.Method.Start(args)
callHandle.Eventually().ExpectReturnsEqual(vals) // NON-BLOCKING - registers expectation

h.Controller.Wait() // BLOCKS until all expectations satisfied
```

**New PendingCompletion struct** (similar to PendingExpectation):
```go
type PendingCompletion struct {
    ExpectedReturns []any           // for ExpectReturnsEqual
    ExpectedPanic   any             // for ExpectPanicEquals
    Matcher         func(any) error // for Match variants
    Completed       bool            // true when result received
    AssertionDone   bool            // true when assertion called
    done            chan struct{}   // signals completion
}
```

**CallableController changes:**
- Add `Eventually()` method that returns an async handle
- Async handle has `ExpectReturnsEqual()`, `ExpectPanicEquals()`, etc. that register pending completions
- These return immediately (non-blocking)
- `controller.Wait()` blocks until all pending completions are satisfied

**Callback pattern for ExpectReturns/ExpectPanic:**
- When result arrives (return or panic): set `Completed=true`, store result
- If `AssertionDone` already true: run assertion
- When ExpectReturns*/ExpectPanic* called: set `AssertionDone=true`, store expected
- If `Completed` already true: run assertion immediately

### Phase 2: Code Generation (`impgen/`)

#### Step 2.1: Change ALL constructors to return test handle struct
**File:** `impgen/run/text_templates.go`

**New generated types per mock:**
```go
// Test handle struct (returned by constructor)
type MockFooHandle struct {
    Mock       FooInterface              // The interface implementation
    Method     *MockFooMethods           // Method wrappers for expectations
    Controller *_imptest.Controller[...] // Wait(), SetTimeout()
}

// Methods struct (holds all method wrappers)
type MockFooMethods struct {
    MethodA *MethodAWrapper
    MethodB *MethodBWrapper
    // ...
}
```

**Templates to update:**
- `depConstructorTmpl` - interface/struct dependency mocks
- `funcDepConstructorTmpl` - function/functype dependency mocks
- `targetConstructorTmpl` - function wrappers
- `interfaceTargetConstructorTmpl` - interface/struct wrappers

Change constructor signatures from:
```go
func MockFoo(t *testing.T) *mockFoo { ... }
func WrapFoo(t *testing.T, fn FuncType) *wrapperFoo { ... }
```

To:
```go
func MockFoo(t *testing.T) *MockFooHandle { ... }
func WrapFoo(t *testing.T, fn FuncType) *WrapFooHandle { ... }
```

Usage (same pattern for all mock/wrapper types):
```go
// Dependency mocks
h := MockFoo(t)
h.Controller.SetTimeout(5 * time.Second)
h.Method.MethodA.Eventually().ExpectCalledWithExactly(args).InjectReturnValues(vals)
h.Controller.Wait()

// Target wrappers
h := WrapFoo(t, realFoo)
h.Controller.SetTimeout(5 * time.Second)
h.Method.Start(args).Eventually().ExpectReturnsEqual(vals)  // for functions
h.Method.MethodA.Start(args).Eventually().ExpectReturnsEqual(vals)  // for interface methods
h.Controller.Wait()
```

#### Step 2.2: Update dependency method wrappers for async
**File:** `impgen/run/text_templates.go`

Generated `ExpectCalledWithExactly` must handle async mode - returns type that allows `InjectReturnValues()` to complete registration.

#### Step 2.3: Add Eventually() to target wrapper CallHandle
**File:** `impgen/run/text_templates.go`

Templates to update:
- `targetCallHandleStructTmpl` - function wrapper call handles
- `interfaceTargetMethodCallHandleStructTmpl` - interface/struct method call handles

Add `Eventually()` method to CallHandle that returns an async handle with non-blocking `ExpectReturnsEqual()`, `ExpectPanicEquals()`, etc.

### Phase 3: Update All UATs

**All tests** need updating for new test handle pattern:
- Change `mock := MockFoo(t)` to `h := MockFoo(t)`
- Change `mock.MethodA.ExpectCalledWith...` to `h.Method.MethodA.ExpectCalledWith...`
- Change `mock.Interface()` to `h.Mock`

Tests using `Eventually()` also need `h.Controller.Wait()`:
- `UAT/variations/concurrency/eventually/concurrency_test.go`
- Any other tests using Eventually()

### Phase 4: Documentation

- Update README.md Eventually() examples
- Update TAXONOMY.md concurrency section
- Update any doc files referencing Eventually()

---

## Key Files to Modify

| File | Changes |
|------|---------|
| `imptest/controller.go` | PendingExpectation, PendingCompletion, SetTimeout(), Wait(), dispatch loop changes |
| `imptest/dependency.go` | Modify ExpectCalledWith* for async, callback pattern on InjectReturnValues AND InjectPanicValue |
| `imptest/callable_controller.go` | Add Eventually() for target wrappers, async ExpectReturns*/ExpectPanic* |
| `impgen/run/text_templates.go` | New Handle/Methods structs; constructors return handle; add Eventually() to CallHandle |
| `impgen/run/codegen_dependency.go` | Template data for Handle/Methods struct generation |
| `impgen/run/codegen_target.go` | Template data for target wrapper Handle struct generation |
| `UAT/**/*_test.go` | Update all tests: `h := MockFoo(t)`, `h.Method.X...`, `h.Mock` |
| `UAT/variations/concurrency/eventually/*` | Add `h.Controller.Wait()` |
| `UAT/core/wrapper-function/*` | Update for new handle pattern |
| `UAT/core/mock-struct/*` | Update for new handle pattern |

---

## Implementation Order

1. **RED**: Write failing test for async Eventually() + h.Controller.Wait() with test handle pattern
2. **GREEN**: Add PendingExpectation + Controller.Wait() + SetTimeout()
3. **GREEN**: Modify dispatch loop to check pending expectations first
4. **GREEN**: Modify DependencyMethod for async registration with callback pattern
5. **GREEN**: Update code generation to return test handle struct (h.Mock, h.Method, h.Controller)
6. **REFACTOR**: Update all UATs to test handle pattern (`h := MockFoo(t)`, `h.Method.X...`, `h.Mock`)
7. **REFACTOR**: Add h.Controller.Wait() to Eventually tests
8. **REFACTOR**: Update documentation
9. **VERIFY**: mage check passes

---

## Success Criteria

**Test Handle Pattern:**
- [ ] Constructors return test handle struct with `h.Mock`, `h.Method`, `h.Controller`
- [ ] `h.Mock` is the interface/function implementation (no extra methods)
- [ ] `h.Method` contains method wrappers for setting expectations
- [ ] `h.Controller` provides Wait(), SetTimeout()
- [ ] **Zero naming conflicts** - all imptest internals on separate structs

**Dependency Mocks:**
- [ ] Eventually() is non-blocking (returns immediately after registering)
- [ ] InjectReturnValues() AND InjectPanicValue() use callback pattern (before OR after Wait())
- [ ] Unmatched calls queue for later matching

**Target Wrappers:**
- [ ] `handle.Eventually().ExpectReturnsEqual(vals)` is non-blocking
- [ ] `handle.Eventually().ExpectPanicEquals(val)` is non-blocking
- [ ] ExpectReturns*/ExpectPanic* use callback pattern (before OR after Wait())

**Shared:**
- [ ] `h.Controller.SetTimeout(d)` configures timeout for all blocking operations
- [ ] `h.Controller.Wait()` blocks until all Eventually() expectations satisfied
- [ ] Timeout affects all blocking ops (Wait, Ordered mode expectations)
- [ ] All UATs updated and passing
- [ ] mage check clean
