# imptest Documentation

<!-- Last reviewed: 2026-01-09 | Review triggers: new UAT added, major feature change -->

## Table of Contents

- [Quick Start](#quick-start)
  - [Mental Model](#mental-model)
  - [Decision Tree](#decision-tree-what-should-i-do)
- [Testing Patterns](#testing-patterns)
  - [Wrapper Pattern (--target)](#wrapper-pattern---target)
  - [Mock Pattern (--dependency)](#mock-pattern---dependency)
- [Variations Reference](#variations-reference)
  - [Package Handling](#package-handling)
  - [Signature Handling](#signature-handling)
  - [Concurrency](#concurrency)
- [Limitations](#limitations)
- [UAT Index](#uat-index)

---

## Quick Start

### Mental Model

imptest has two conceptual layers:

| Layer | Question | Options |
|-------|----------|---------|
| **Core** | What do you want to do? | Wrapper (test your code) or Mock (control dependencies) |
| **Variations** | Any special cases? | Package, signature, behavior, concurrency |

**Core = Pattern + Symbol**: You always pick a testing pattern AND a symbol type together. "I want to wrap this function" or "I want to mock this interface."

**Variations are additive**: Once you know your core case works, variations handle edge cases like generics, stdlib packages, or concurrent calls.

### Decision Tree: What Should I Do?

```
What are you testing?
│
├─► A function/method that CALLS dependencies
│   └─► Use WRAPPER pattern (--target)
│       "I want to verify how my code interacts with its dependencies"
│
└─► A dependency that your code USES
    └─► Use MOCK pattern (--dependency)
        "I want to control what the dependency returns to my code"
```

### When to Use --target (Wrapper Pattern)

Use `--target` when you want to **test a function or method** by:
- Running it in a controlled goroutine
- Verifying its return values
- Observing when it interacts with mocked dependencies
- Capturing panics it may throw

**Example scenario**: "I want to test my `ProcessOrder` function that calls a payment service"

```go
//go:generate impgen orders.ProcessOrder --target

func TestProcessOrder(t *testing.T) {
    mock, expect := MockPaymentService(t)

    call := StartProcessOrder(t, orders.ProcessOrder, mock, order)

    expect.Charge.ArgsEqual(order.Amount).
        Return(receipt, nil)

    call.ReturnsEqual(expectedResult, nil)
}
```

### When to Use --dependency (Mock Pattern)

Use `--dependency` when you want to **mock a dependency** by:
- Controlling what values it returns
- Verifying it was called with expected arguments
- Injecting errors or panics to test error paths

**Example scenario**: "I want to mock the database repository my service depends on"

```go
//go:generate impgen storage.Repository --dependency

func TestUserService(t *testing.T) {
    mock, expect := MockRepository(t)
    service := NewUserService(mock)

    go service.GetUser(123)

    expect.FindByID.ArgsEqual(123).
        Return(User{ID: 123, Name: "Alice"}, nil)
}
```

---

## Testing Patterns

### Wrapper Pattern (--target)

**What it does**: Wraps code under test to observe and control its execution.

**When to use it**: When the code you're testing is the "impure" function that coordinates multiple dependencies.

#### Supported Symbol Types

| Symbol Type | Directive Example | Generated | UAT |
|-------------|-------------------|-----------|-----|
| Function | `impgen pkg.MyFunc --target` | `StartMyFunc` | [wrapper-function](../UAT/core/wrapper-function/) |
| Struct method | `impgen pkg.Calculator.Add --target` | `StartCalculatorAdd` | [wrapper-function](../UAT/core/wrapper-function/) |
| Struct (all methods) | `impgen pkg.Calculator --target` | `StartCalculator` | [wrapper-struct](../UAT/core/wrapper-struct/) |
| Interface | `impgen pkg.Logger --target` | `StartLogger` | [wrapper-interface](../UAT/core/wrapper-interface/) |
| Function type | `impgen pkg.HandlerFunc --target` | `StartHandlerFunc` | [wrapper-functype](../UAT/core/wrapper-functype/) |

#### API Pattern

**Function/method/functype wrappers** (flattened API):
```go
// 1. Start execution directly with args (returns a call handle)
call := StartMyFunc(t, pkg.MyFunc, arg1, arg2, ...)

// 2. Handle any dependency interactions (see Mock Pattern)
expect.DepMethod.ArgsEqual(...).Return(...)

// 3. Verify the result
call.ReturnsEqual(expectedReturn1, expectedReturn2)
// or
call.PanicEquals("expected panic message")

// For async verification:
call.Eventually.ReturnsEqual(expected)
imptest.Wait(t)  // blocks until satisfied
```

**Struct/interface wrappers** (multi-method):
```go
// 1. Create wrapper with testing.T and the implementation
wrapper := StartCalculator(t, impl)

// 2. Start a method (returns a call handle)
call := wrapper.Add.Start(10, 20)

// 3. Verify the result
call.ReturnsEqual(30)
```

#### Wrapper Examples

##### Function Wrapper

```go
//go:generate impgen orders.ProcessOrder --target

func TestProcessOrder(t *testing.T) {
    mock, expect := MockExternalService(t)

    call := StartProcessOrder(t, orders.ProcessOrder, mock, 42)

    expect.FetchData.ArgsEqual(42).
        Return("data", nil)

    call.ReturnsEqual("Result: data", nil)
}
```

##### Method Wrapper

```go
//go:generate impgen calculator.Calculator.Add --target

func TestCalculatorAdd(t *testing.T) {
    calc := calculator.NewCalculator()

    call := StartCalculatorAdd(t, calc.Add, 10, 20)
    call.ReturnsEqual(30)
}
```

##### Struct Wrapper (All Methods)

```go
//go:generate impgen calculator.Calculator --target

func TestCalculator(t *testing.T) {
    calc := calculator.NewCalculator()
    wrapper := StartCalculator(t, calc)

    addCall := wrapper.Add.Start(10, 20)
    addCall.ReturnsEqual(30)

    mulCall := wrapper.Multiply.Start(5, 6)
    mulCall.ReturnsEqual(30)
}
```

##### Interface Wrapper

```go
//go:generate impgen handlers.Logger --target

func TestLogger(t *testing.T) {
    impl := handlers.NewConsoleLogger()
    wrapper := StartLogger(t, impl)

    infoCall := wrapper.Info.Start("test message")
    infoCall.ReturnsEqual()

    warnCall := wrapper.Warn.Start("warning!")
    warnCall.ReturnsEqual()
}
```

##### Function Type Wrapper

```go
//go:generate impgen visitor.WalkFunc --target

func TestWalkFunc(t *testing.T) {
    myWalker := func(path string, info os.FileInfo) error {
        return nil
    }

    call := StartWalkFunc(t, myWalker, "/path", mockFileInfo)
    call.ReturnsEqual(nil)
}
```

##### Panic Verification

```go
//go:generate impgen safety.UnsafeRunner --target

func TestUnsafeRunnerPanics(t *testing.T) {
    call := StartUnsafeRunner(t, safety.UnsafeRunner, 10, 0)  // Division by zero
    call.PanicEquals("division by zero")
}
```

---

### Mock Pattern (--dependency)

**What it does**: Creates a mock implementation of a dependency to control its behavior and verify interactions.

**When to use it**: When you need to isolate the code under test from its dependencies.

#### Supported Symbol Types

| Symbol Type | Directive Example | Generated | UAT |
|-------------|-------------------|-----------|-----|
| Interface | `impgen pkg.Service --dependency` | `MockService` | [mock-interface](../UAT/core/mock-interface/) |
| Function type | `impgen pkg.HandlerFunc --dependency` | `MockHandlerFunc` | [mock-functype](../UAT/core/mock-functype/) |

**Coming soon** (see [REORGANIZATION_PROPOSAL.md](./REORGANIZATION_PROPOSAL.md)):
- Function as dependency (#43)
- Struct as dependency (#44)
- Struct method as dependency (#45)

#### API Pattern

All mocks follow the same pattern:

```go
import "github.com/toejough/imptest/match"

// 1. Create mock with testing.T (returns mock and expectation handle)
mock, expect := MockService(t)

// 2. Pass mock (the interface implementation) to code under test
go myFunction(mock)

// 3. For each expected call (in Ordered mode):
expect.MethodName.ArgsEqual(arg1, arg2).
    Return(ret1, ret2)

// Or with matchers for flexible matching:
expect.MethodName.ArgsShould(match.BeAny, expectedArg2).
    Return(ret1, ret2)

// For concurrent code, use Eventually with imptest.Wait(t):
expect.Eventually.MethodName.ArgsEqual(arg1, arg2).
    Return(ret1, ret2)
imptest.Wait(t)  // blocks until all Eventually expectations satisfied
```

#### Mock Examples

##### Basic Interface Mock

```go
//go:generate impgen storage.Repository --dependency

func TestUserService(t *testing.T) {
    mock, expect := MockRepository(t)
    service := NewUserService(mock)

    go service.GetUser(123)

    expect.FindByID.ArgsEqual(123).
        Return(User{ID: 123}, nil)
}
```

##### Variadic Parameters

```go
//go:generate impgen logger.Logger --dependency

func TestWithVariadic(t *testing.T) {
    mock, expect := MockLogger(t)

    go doWork(mock)

    // Variadic args passed normally
    expect.Logf.ArgsEqual("format: %s %d", "hello", 42).
        Return()
}
```

##### Error Injection

```go
func TestErrorPath(t *testing.T) {
    mock, expect := MockRepository(t)
    service := NewUserService(mock)

    go service.GetUser(999)

    expect.FindByID.ArgsEqual(999).
        Return(User{}, ErrNotFound)
}
```

##### Panic Injection

```go
func TestPanicRecovery(t *testing.T) {
    mock, expect := MockRepository(t)
    service := NewUserService(mock)

    go service.GetUser(123)

    expect.FindByID.ArgsEqual(123).
        Panic("database connection lost")
}
```

##### Function Type Mock

```go
//go:generate impgen handlers.Router --dependency

func TestRouterDependency(t *testing.T) {
    mock, expect := MockRouter(t)

    go processRequests(mock)

    expect.ArgsEqual("/api/users").
        Return(handlerFunc)
}
```

##### Generic Interface Mock

```go
//go:generate impgen cache.Cache[T] --dependency

func TestGenericCache(t *testing.T) {
    mock, expect := MockCache[User](t)
    service := NewCachingService(mock)

    go service.GetCached("user:123")

    expect.Get.ArgsEqual("user:123").
        Return(User{ID: 123}, true)
}
```

##### Embedded Interface Mock

```go
//go:generate impgen io.ReadCloser --dependency

// Note: import "github.com/toejough/imptest/match" for matchers

func TestReadCloser(t *testing.T) {
    mock, expect := MockReadCloser(t)

    go processFile(mock)

    // Methods from embedded interfaces are accessible
    expect.Read.ArgsShould(match.BeAny).
        Return(10, nil)
    expect.Close.Called().
        Return(nil)
}
```

---

## Variations Reference

### Package Handling

imptest handles symbols from various package contexts:

| Package Location | Supported | UAT | Notes |
|------------------|-----------|-----|-------|
| Same package | Yes | [same-package](../UAT/variations/package/same-package/) | Whitebox testing, interface cross-refs |
| Different package | Yes | [mock-interface](../UAT/core/mock-interface/), [wrapper-function](../UAT/core/wrapper-function/) | Standard usage |
| Standard library | Yes | [external-functypes](../UAT/variations/behavior/external-functypes/), [embedded-interfaces](../UAT/variations/behavior/embedded-interfaces/) | `http.HandlerFunc`, `io.Reader` |
| Aliased import | Yes | — | `import alias "pkg"` |
| Dot import | Yes | [dot-imports](../UAT/variations/package/dot-imports/) | `import . "pkg"` |
| Stdlib shadowing | Yes | [shadowing](../UAT/variations/package/shadowing/) | 4-tier resolution |

#### Standard Library Shadowing Resolution

When a local package shadows a stdlib package name:

```go
// Local "time" package exists, stdlib "time" also exists
// Which one is intended?
//go:generate impgen time.Timer --dependency
```

**Resolution strategy** (in priority order):
1. **Explicit `--import-path` flag**: `impgen --import-path=time time.Timer` (stdlib)
2. **Infer from test file imports**: Uses the import path from your imports
3. **Detect ambiguity**: Errors with helpful suggestions if ambiguous
4. **Fallback**: Standard resolution for non-ambiguous cases

---

### Signature Handling

imptest handles various parameter and return type patterns:

#### Count and Naming

| Feature | Supported | UAT |
|---------|-----------|-----|
| Zero parameters | Yes | [mock-interface](../UAT/core/mock-interface/) |
| Multiple parameters | Yes | [edge-many-params](../UAT/variations/signature/edge-many-params/) |
| Zero returns | Yes | [edge-zero-returns](../UAT/variations/signature/edge-zero-returns/) |
| Multiple returns | Yes | [wrapper-function](../UAT/core/wrapper-function/) |
| Variadic parameters | Yes | [mock-interface](../UAT/core/mock-interface/) |
| Named parameters/returns | Yes | [named-params](../UAT/variations/signature/named-params/) |

#### Type Complexity

| Type | Supported | UAT | Notes |
|------|-----------|-----|-------|
| Concrete types | Yes | All | int, string, structs, etc. |
| Generic types | Yes | [generics](../UAT/variations/signature/generics/), [parameterized](../UAT/variations/signature/parameterized/) | `[T any]`, `[T Numeric]` |
| Non-comparable types | Yes | [non-comparable](../UAT/variations/signature/non-comparable/) | Slices, maps, functions |
| Struct literals | Yes | [struct-literal](../UAT/variations/signature/struct-literal/) | `struct{ Field int }` |
| Function literals | Yes | [function-literal](../UAT/variations/signature/function-literal/) | `func(int) int` |
| Interface literals | Yes | [interface-literal](../UAT/variations/signature/interface-literal/) | `interface{ Method() }` |
| Channels | Yes | [channels](../UAT/variations/signature/channels/) | All directions |
| External types | Yes | [external-types](../UAT/variations/signature/external-types/), [cross-file-external](../UAT/variations/signature/cross-file-external/) | `time.Time`, `os.FileInfo` |

#### Matching Function Parameters

When testing with function parameters, use matchers since Go functions cannot be compared:

```go
import "github.com/toejough/imptest/match"

// DON'T use ArgsEqual for functions
// expect.Map.ArgsEqual(items, mapFunc)  // Will hang!

// DO use matchers
expect.Map.ArgsShould(items, match.BeAny).
    Return(result)
```

---

### Concurrency

#### Ordered Mode (Default)

By default, imptest expects calls in the exact order you specify:

```go
expect.First.ArgsEqual(1).Return(1)
expect.Second.ArgsEqual(2).Return(2)
// Calls MUST arrive as: First, then Second
```

If calls arrive out of order, the test fails immediately.

#### Eventually Mode (Async)

For concurrent code where call order is non-deterministic, use `Eventually` with `imptest.Wait(t)`:

```go
mock, expect := MockService(t)

go func() { mock.TaskA("a") }()
go func() { mock.TaskB("b") }()
go func() { mock.TaskC("c") }()

// Register async expectations (non-blocking)
expect.Eventually.TaskA.ArgsEqual("a").Return()
expect.Eventually.TaskB.ArgsEqual("b").Return()
expect.Eventually.TaskC.ArgsEqual("c").Return()

// Block until all expectations are satisfied
imptest.Wait(t)
```

**Key points:**
- `Eventually` expectations are **non-blocking** - they register and return immediately
- `imptest.Wait(t)` blocks until **all** Eventually expectations are matched
- Calls that don't match any pending expectation are queued for later matching
- Use `imptest.SetTimeout(t, d)` to configure timeout for all blocking operations

**Target wrappers also support Eventually:**

```go
call1 := StartMyFunc(t, MyFunc, arg1)
call2 := StartMyFunc(t, MyFunc, arg2)

// Non-blocking expectations on wrapper calls
call1.Eventually.ReturnsEqual(expected1)
call2.Eventually.ReturnsEqual(expected2)

// Wait for all to complete
imptest.Wait(t)
```

**UAT**: [ordered](../UAT/variations/concurrency/ordered/), [eventually](../UAT/variations/concurrency/eventually/)

---

## Limitations

### Cannot Do: Anonymous Functions

Cannot wrap/mock inline function literals without a named type.

```go
// This won't work - no named type
result := func(x int) int { return x * 2 }(5)
```

**Workaround**: Create a named function type:

```go
type Transformer func(int) int

//go:generate impgen pkg.Transformer --target
```

### Cannot Do: Anonymous Structs

Cannot wrap methods on struct literals.

```go
// This won't work - struct has no name
var calc = struct{ Add func(int, int) int }{...}
```

**Workaround**: Define a named struct type:

```go
type Calculator struct{}
func (c Calculator) Add(a, b int) int { return a + b }

//go:generate impgen pkg.Calculator --target
```

---

## UAT Index

### By Testing Pattern

#### Wrapper Pattern (--target)

| UAT | Path | Symbol Type |
|-----|------|-------------|
| [wrapper-function](../UAT/core/wrapper-function/) | core/wrapper-function | Function, Method |
| [wrapper-functype](../UAT/core/wrapper-functype/) | core/wrapper-functype | Function type |
| [wrapper-interface](../UAT/core/wrapper-interface/) | core/wrapper-interface | Interface |
| [wrapper-struct](../UAT/core/wrapper-struct/) | core/wrapper-struct | Struct |

#### Mock Pattern (--dependency)

| UAT | Path | Symbol Type |
|-----|------|-------------|
| [mock-interface](../UAT/core/mock-interface/) | core/mock-interface | Interface |
| [mock-functype](../UAT/core/mock-functype/) | core/mock-functype | Function type |
| [mock-function](../UAT/core/mock-function/) | core/mock-function | Function |
| [mock-method](../UAT/core/mock-method/) | core/mock-method | Method |
| [mock-struct](../UAT/core/mock-struct/) | core/mock-struct | Struct |

### By Variation

#### Package Variations

| UAT | Path | Variation |
|-----|------|-----------|
| [shadowing](../UAT/variations/package/shadowing/) | variations/package/shadowing | Stdlib shadowing |
| [same-package](../UAT/variations/package/same-package/) | variations/package/same-package | Same package (whitebox + interface refs) |
| [test-package](../UAT/variations/package/test-package/) | variations/package/test-package | Test package import |
| [dot-imports](../UAT/variations/package/dot-imports/) | variations/package/dot-imports | Dot import (basic + business logic) |

#### Signature Variations

| UAT | Path | Variation |
|-----|------|-----------|
| [generics](../UAT/variations/signature/generics/) | variations/signature/generics | Generic types |
| [parameterized](../UAT/variations/signature/parameterized/) | variations/signature/parameterized | Constrained generics |
| [non-comparable](../UAT/variations/signature/non-comparable/) | variations/signature/non-comparable | Slices, maps |
| [named-params](../UAT/variations/signature/named-params/) | variations/signature/named-params | Named params/returns |
| [function-literal](../UAT/variations/signature/function-literal/) | variations/signature/function-literal | Function literal params |
| [interface-literal](../UAT/variations/signature/interface-literal/) | variations/signature/interface-literal | Interface literal params |
| [struct-literal](../UAT/variations/signature/struct-literal/) | variations/signature/struct-literal | Struct literal params |
| [channels](../UAT/variations/signature/channels/) | variations/signature/channels | Channel types |
| [external-types](../UAT/variations/signature/external-types/) | variations/signature/external-types | External types |
| [cross-file-external](../UAT/variations/signature/cross-file-external/) | variations/signature/cross-file-external | Cross-file imports |
| [external-functype](../UAT/variations/signature/external-functype/) | variations/signature/external-functype | External functype |
| [edge-many-params](../UAT/variations/signature/edge-many-params/) | variations/signature/edge-many-params | Many parameters |
| [edge-zero-returns](../UAT/variations/signature/edge-zero-returns/) | variations/signature/edge-zero-returns | Zero returns |

#### Behavior Variations

| UAT | Path | Variation |
|-----|------|-----------|
| [panic-handling](../UAT/variations/behavior/panic-handling/) | variations/behavior/panic-handling | Panic handling |
| [callbacks](../UAT/variations/behavior/callbacks/) | variations/behavior/callbacks | Callback patterns |
| [matching](../UAT/variations/behavior/matching/) | variations/behavior/matching | Argument matching |
| [embedded-interfaces](../UAT/variations/behavior/embedded-interfaces/) | variations/behavior/embedded-interfaces | Embedded interfaces |
| [embedded-structs](../UAT/variations/behavior/embedded-structs/) | variations/behavior/embedded-structs | Embedded structs |
| [external-functypes](../UAT/variations/behavior/external-functypes/) | variations/behavior/external-functypes | External function types |
| [typesafe-getargs](../UAT/variations/behavior/typesafe-getargs/) | variations/behavior/typesafe-getargs | Typesafe argument access |

#### Concurrency Variations

| UAT | Path | Variation |
|-----|------|-----------|
| [eventually](../UAT/variations/concurrency/eventually/) | variations/concurrency/eventually | Eventually mode |
| [ordered](../UAT/variations/concurrency/ordered/) | variations/concurrency/ordered | Ordered mode |
