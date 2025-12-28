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
- Functions can be extrapolated from: named types, function definitions, function literals
- Interfaces can be extrapolated from: named types, struct definitions with methods, struct literals with methods
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

### Generator Command

```go
//go:generate impgen --[target|dependency] (package-alias.)[interface|struct|function-type|function]
```

### Setup

```go
// Central coordinator
imp := NewImp(t)

// Targets (things we're testing)
runTarget := NewRunTargetFunction(imp, callable)            // function
runInterfaceTarget := NewRunTargetInterface(imp, instance)  // interface

// Dependencies (things we're mocking)
serviceDep := NewServiceDependencyFunction(imp)             // function
serviceInterfaceDep := NewServiceDependencyInterface(imp)   // interface
```

### Target API (Verifying Behavior)

```go
// Call the target function
instance := runTarget.CallWith(arg1, arg2)  // compile-time typesafe

// For interface targets
instance := runInterfaceTarget.Method1.CallWith(args)

// Ordering
instance = instance.Eventually()  // switch to unordered mode

// Verify returns
instance.ExpectReturnsEqual(val1, val2)     // exact match, compile-time typesafe
instance.ExpectReturnsMatch(matcher1, matcher2) // matchers, runtime checked

// Verify panics
instance.ExpectPanicEquals(value)   // exact match
instance.ExpectPanicMatches(matcher) // matcher

// Get actual values (typesafe)
instance.GetReturns().R1  // fails test if no return
instance.GetPanic()       // fails test if no panic
```

### Dependency API (Mocking Behavior)

```go
// Expect a call to dependency function
instance := serviceDep.ExpectCalledWithExactly(arg1, arg2)  // compile-time typesafe
instance := serviceDep.ExpectCalledWithMatches(matcher1, matcher2) // matchers

// For interface dependencies
instance := serviceInterfaceDep.Method1.ExpectCalledWithExactly(args)
instance := serviceInterfaceDep.Method1.ExpectCalledWithMatches(matchers)

// Ordering
instance = instance.Eventually()  // switch to unordered mode

// Get actual args (typesafe)
instance.GetArgs().A1

// Inject response
instance.InjectReturnValues(val1, val2)  // compile-time typesafe
instance.InjectPanicValue(value)
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
- **Source**: Named types, definitions, literals
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
| Source | named type, definition, literal |
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

2. **Naming**: `TargetFunction`/`DependencyFunction` and `TargetInterface`/`DependencyInterface` naming is clear and approved.

3. **GetReturns()/GetArgs()/GetPanic() semantics**: Follow the same ordered/unordered semantics as Expect methods.
   - **Ordered**: Expect the very next interaction to match (call, panic, or return for their instance); fail test if no match
   - **Unordered**: Wait for matching interaction, queueing any interactions that come in before it
   - **Both cases**: Wait as long as necessary (no timeout)

4. **GetReturns()/GetArgs() field naming**: Use indexed names since Go doesn't allow both slice indexing and field access.
   ```go
   // If source has: func Foo(name string, count int) (result string, err error)
   instance.GetArgs().A1    // first arg (name)
   instance.GetArgs().A2    // second arg (count)
   instance.GetReturns().R1 // first return (result)
   instance.GetReturns().R2 // second return (err)
   ```

5. **Func types in interfaces**: NO auto-generation. When an interface method has func params/returns, users generate wrappers separately and wire them manually. This keeps the implementation simpler and gives users explicit control.

---

## Summary

This redesign provides:
- **Clearer mental model**: Target vs Dependency, Ordered vs Unordered
- **Symmetric API**: Same patterns for verification and mocking
- **Unified matchers**: Same matcher API for args, returns, and panics
- **Explicit structure**: Func vs Interface is surfaced in the type names
- **Focused scope**: Channels deferred, shared state out of scope
