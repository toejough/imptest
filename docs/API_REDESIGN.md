# imptest API Redesign Plan

## Overview

Redesign the imptest API around a cleaner conceptual model before implementing additional features. This provides a unified, symmetric API that scales to all interaction patterns.

---

## Core Concepts

### Function Interactions

| Interaction Type | As Target (Wrap) | As Dependency (Mock) |
|------------------|------------------|----------------------|
| **Function** | TargetFunction | DependencyFunction |
| **Interface** | TargetInterface | DependencyInterface |
| **Channel** | TargetChannel | DependencyChannel |
| **Shared State** | *(not interceptable)* | *(not interceptable)* |

**Type Sources:**

The generator can create wrappers/mocks from two categories of sources:

**Type Sources** - Generate from named type declarations:
- Functions: `type Fetcher func(int) (string, error)`
- Interfaces: `type DataStore interface { Get(...) Save(...) }`

**Definition Sources** - Generate from concrete implementations:
- Functions: `func Add(a, b int) int { return a + b }`
- Interfaces: `type BasicCalculator struct{}` with methods implementing an interface

**Source Usage by Role:**
- **Targets**: Can use both Type and Definition sources (wrapping implementations to verify behavior)
- **Dependencies**: Use Type sources only (mocking type contracts, not implementations)

**Excluded:**
- Literal types (anonymous functions, inline struct literals) are not supported
- Users must name all things they want to wrap/mock
- Channels: deferred until after major refactor

### Direction Determines Role

| Role | What We Do | Arg Direction | Result Direction |
|------|------------|---------------|------------------|
| **Target** | We wrap it | We set | We get |
| **Dependency** | We mock it | We get | We set |

### Ordering Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| **Ordered** (default) | Expect next interaction immediately | Sequential code |
| **Unordered** (Eventually) | Wait for matching interaction, queueing misses | Concurrent/async code |

### Matching Modes

| Mode | Args | Returns | Panics |
|------|------|---------|--------|
| **Exact** | `ExpectCalledWithExactly()` | `ExpectReturnsEqual()` | `ExpectPanicEquals()` |
| **Matcher** | `ExpectCalledWithMatches()` | `ExpectReturnsMatch()` | `ExpectPanicMatches()` |

---

## Proposed API

### Core Interface

```go
// TestReporter is the minimal interface imptest needs from test frameworks.
// testing.T, testing.B, and *Imp all implement this interface.
type TestReporter interface {
    Helper()
    Fatalf(format string, args ...any)
}
```

### Generator Command

```go
//go:generate impgen --target Add           // Generates WrapAdd(t TestReporter, fn ...) *AddTarget
//go:generate impgen --dependency Fetcher   // Generates MockFetcher(t TestReporter) *FetcherDependency
```

### Simple API (Pure Functions, Single Interactions)

```go
// Concise syntax - generated wrapper accepts TestReporter
// If given testing.T, creates internal Imp coordinator
WrapAdd(t, Add).Start(2, 3).ExpectReturnsEqual(5)

// With matchers
WrapDivide(t, Divide).Start(10, 2).ExpectReturnsMatch(
    imptest.Satisfies(func(v any) bool {
        result, ok := v.(int)
        return ok && result > 0
    }),
)

// Verify panics
WrapDivide(t, Divide).Start(10, 0).ExpectPanicEquals("division by zero")
```

### Complex API (Multiple Interactions, Shared Coordinator)

```go
// Create shared coordinator for orchestrating multiple interactions
imp := imptest.NewImp(t)

// Wrap target functions and interfaces (pass imp which implements TestReporter)
target := WrapProcessData(imp, ProcessData)
fetcher := MockFetcher(imp)

// Execute target - it will call the mock (conversational flow)
result := target.Start(42, fetcher.Func())

// Interactively verify dependency call and inject response
call := fetcher.ExpectCalledWithExactly(42)
call.InjectReturnValues("test data", nil)

// Verify target result
result.ExpectReturnsEqual("processed: test data", nil)
```

### Target API (Wrapping Code Under Test)

```go
// Functions
WrapAdd(t, Add).Start(2, 3).ExpectReturnsEqual(5)

// Interfaces
calc := &BasicCalculator{}
WrapCalculator(t, calc).Add.Start(2, 3).ExpectReturnsEqual(5)

// Unordered mode (for async/concurrent code)
call := WrapAsyncOp(t, AsyncOp).Start(args)
call.Eventually().ExpectReturnsEqual(result)

// Get actual values
returns := WrapDivide(t, Divide).Start(10, 2).GetReturns()
if returns.R1 != 5 {
    t.Errorf("expected 5, got %d", returns.R1)
}

// Verify panics
WrapDivide(t, Divide).Start(10, 0).ExpectPanicMatches(imptest.Any())
```

### Dependency API (Mocking Dependencies)

```go
// Mock functions
fetcher := MockFetcher(t)
call := fetcher.ExpectCalledWithExactly(42)
call.InjectReturnValues("data", nil)
ProcessData(42, fetcher.Func())

// Mock interfaces
store := MockDataStore(t)
call := store.Get.ExpectCalledWithExactly(42)
call.InjectReturnValues("data", nil)
service := &Service{store: store.Interface()}
service.LoadAndProcess(42)

// Matchers for args
call := fetcher.ExpectCalledWithMatches(imptest.Satisfies(func(v any) bool {
    id, ok := v.(int)
    return ok && id > 0
}))

// Inject errors or panics
call.InjectReturnValues("", errors.New("not found"))
call.InjectPanicValue("database error")

// Get actual args
args := call.GetArgs()
if args.A1 != 42 {
    t.Errorf("expected 42, got %d", args.A1)
}

// Unordered mode
call := fetcher.Eventually().ExpectCalledWithExactly(42)
```

### Low-Level API

```go
// For maximum flexibility
interaction := imp.NextInteraction()
```

---

## Key Differences from Current API

| Aspect | Current | Proposed |
|--------|---------|----------|
| Terminology | Callable, Imp, Mock | Target, Dependency, Imp |
| Panic matching | `ExpectPanicWith(value)` | `ExpectPanicEquals/Matches()` |
| Arg verification | `ExpectArgsAre/Should` | `ExpectCalledWithExactly/Matches` |
| Return verification | `ExpectReturnedValuesAre/Should` | `ExpectReturnsEqual/Match` |
| Ordering | Implicit + Within | Explicit ordered/unordered |
| Interface vs Func | Mixed | Explicit TargetInterface/DependencyInterface |

---

## Deferred Features

### Channels
TargetChannel and DependencyChannel are in scope but deferred until after the major API refactor. Implementation will require careful handling of edge cases around channel replacement.

---

## What's Out of Scope

### Shared State
No clear "call" to intercept. Actions on shared state are not interceptable. Recommend refactoring to message-passing patterns.

---

## Package Alias Handling

The generator must handle these scenarios:

| Scenario | Example | Resolution |
|----------|---------|------------|
| Single-word | `time` | `time` |
| Final-segment | `github.com/foo/bar` | `bar` |
| Obscured | Package at path has different name | Use actual package name |
| Aliased | `nick "github.com/foo/bar"` | Use alias `nick` |

**Rules:**
- Generator commands use package-alias as it appears in the source file
- Generated code goes into same package
- Generated code uses same type names and package aliases as source
- Framework imports use leading `_` to avoid conflicts

---

## Implementation Phases

### Phase 0: Record Plan
- Save this plan to `docs/API_REDESIGN.md` in the repository

### Phase 1: Write UAT Tests First

**Test Matrix Dimensions:**
- **Interaction Type**: Function, Interface (Channels deferred)
- **Role**: Target (wrap), Dependency (mock)
- **Source**: Type, Definition (Literals not supported)
- **Ordering**: Ordered (default), Unordered (Eventually)
- **Matching**: Exact, Matcher

**UAT Structure:**
- `UAT/target-function/` - wrapping functions
- `UAT/target-interface/` - wrapping interfaces
- `UAT/dependency-function/` - mocking functions
- `UAT/dependency-interface/` - mocking interfaces
- `UAT/package-alias/` - package alias scenarios

**Coverage per pattern:**
| Dimension | Options to Test |
|-----------|-----------------|
| Source | type, definition |
| Ordering | ordered, unordered (Eventually) |
| Matching | exact values, matchers |
| Outcome | returns, panics |

**Package Alias Coverage:**
| Scenario | Test Case |
|----------|-----------|
| Single-word | `time` → `time` |
| Final-segment | `github.com/foo/bar` → `bar` |
| Obscured | Package at path has different actual name |
| Aliased | `nick "github.com/foo/bar"` → `nick` |

**Package Alias Rules to Verify:**
- Generator commands use package-alias as it appears in source file
- Generated code goes into same package
- Generated code uses same type names and package aliases as source
- Framework imports use leading `_` to avoid conflicts

Write failing tests demonstrating the new API across the matrix.

### Phase 2: Core Restructure
- Implement `Imp` as central coordinator
- Implement Target/Dependency distinction
- Implement ordered/unordered modes (Eventually)
- Make UAT tests pass

### Phase 3: Codegen Updates
- Update `impgen` command syntax
- Generate new API structures
- Update golden tests

### Phase 4: Migration
- Migrate existing UAT tests to new API or archive them
- Ensure all tests pass with `mage check`

### Phase 5: Documentation
- Update README with new conceptual model
- Finalize docs/API_REDESIGN.md with implementation notes
- Add UAT READMEs explaining each pattern

---

## Design Decisions

1. **Backward compatibility**: Clean break (v2.0). No migration path - simpler implementation, cleaner codebase.

2. **TestReporter interface**: Define minimal interface `TestReporter` with only `Helper()` and `Fatalf()`. Both `testing.T` and `*Imp` implement this interface. This:
   - Decouples from testing package implementation details
   - Makes clear what methods imptest actually uses
   - Simplifies testing the framework itself
   - Follows Interface Segregation Principle

3. **Concise syntax for simple cases**: Generated wrappers accept `TestReporter`, creating internal `Imp` when given plain `testing.T`:
   ```go
   // Simple: WrapAdd creates internal coordinator
   WrapAdd(t, Add).Start(2, 3).ExpectReturnsEqual(5)

   // Complex: pass shared Imp coordinator
   imp := imptest.NewImp(t)
   WrapAdd(imp, Add).Start(2, 3).ExpectReturnsEqual(5)
   MockFetcher(imp).ExpectCalledWithExactly(42).InjectReturnValues("data")
   ```

4. **Generated wrapper naming**:
   - Targets: `Wrap{Name}` (e.g., `WrapAdd`, `WrapCalculator`)
   - Dependencies: `Mock{Name}` (e.g., `MockFetcher`, `MockDataStore`)
   - Clear prefix indicates role; name matches the wrapped entity

5. **GetReturns()/GetArgs()/GetPanic() semantics**: Follow the same ordered/unordered semantics as Expect methods.
   - **Ordered**: Expect the very next interaction to match (call, panic, or return for their instance); fail test if no match
   - **Unordered**: Wait for matching interaction, queueing any interactions that come in before it
   - **Both cases**: Wait as long as necessary (no timeout)

6. **GetReturns()/GetArgs() field naming**: Use indexed names since Go doesn't allow both slice indexing and field access.
   ```go
   // If source has: func Foo(name string, count int) (result string, err error)
   instance.GetArgs().A1    // first arg (name)
   instance.GetArgs().A2    // second arg (count)
   instance.GetReturns().R1 // first return (result)
   instance.GetReturns().R2 // second return (err)
   ```

7. **Func types in interfaces**: NO auto-generation. When an interface method has func params/returns, users generate wrappers separately and wire them manually. This keeps the implementation simpler and gives users explicit control.

8. **Channel-based asynchronous execution**: Target wrappers use `.Start()` which runs the target function/method in a goroutine and returns immediately:
   - **Asynchronous execution**: `.Start()` launches function in goroutine, returns wrapper for method chaining
   - **Blocking verification**: `Expect*()` and `GetReturns()` methods call `WaitForResponse()` which blocks on channels
   - **State management**: Wrappers track `returned` and `panicked` state to avoid double-reading from channels
   - **Idempotent waiting**: `WaitForResponse()` checks state before blocking, safe to call multiple times
   - **Conversational flow**: This enables the pattern: "function calls mock → test verifies args → test injects response → function continues → test verifies result"
   - Implementation details:
     ```go
     // Each wrapper has channels and state
     type WrapAddWrapper struct {
         imp        *imptest.Imp
         fn         func(int, int) int
         returnChan chan WrapAddReturns
         panicChan  chan any
         returned   *WrapAddReturns
         panicked   any
     }

     // Start runs function in goroutine
     func (w *WrapAddWrapper) Start(a, b int) *WrapAddWrapper {
         w.returnChan = make(chan WrapAddReturns, 1)
         w.panicChan = make(chan any, 1)
         go func() {
             defer func() {
                 if r := recover(); r != nil {
                     w.panicChan <- r
                 }
             }()
             result := w.fn(a, b)
             w.returnChan <- WrapAddReturns{R1: result}
         }()
         return w
     }

     // WaitForResponse blocks until function completes
     func (w *WrapAddWrapper) WaitForResponse() {
         if w.returned != nil || w.panicked != nil {
             return
         }
         select {
         case ret := <-w.returnChan:
             w.returned = &ret
         case p := <-w.panicChan:
             w.panicked = p
         }
     }
     ```

---

## Summary

This redesign provides:
- **Clearer mental model**: Target vs Dependency, Ordered vs Unordered
- **Ergonomic for simple cases**: `WrapAdd(t, Add).Start(2, 3).ExpectReturnsEqual(5)`
- **Powerful for complex cases**: Shared `Imp` coordinator orchestrates multi-interaction scenarios
- **Minimal coupling**: Custom `TestReporter` interface (only `Helper()` and `Fatalf()`)
- **Symmetric API**: Same patterns for verification and mocking
- **Unified matchers**: Same matcher API for args, returns, and panics
- **Type-safe generated wrappers**: `Wrap{Name}` for targets, `Mock{Name}` for dependencies
- **Explicit structure**: Func vs Interface is surfaced in the type names
- **Channel-based flow control**: Asynchronous execution with blocking verification enables conversational testing
- **Focused scope**: Channels deferred, shared state out of scope
