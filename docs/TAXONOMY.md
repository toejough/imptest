# imptest Documentation

<!-- Last reviewed: 2026-01-05 | Review triggers: new UAT added, major feature change -->

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
    mockPayment := MockPaymentService(t)
    wrapper := WrapProcessOrder(t, orders.ProcessOrder)

    call := wrapper.Start(mockPayment.Interface(), order)

    mockPayment.Charge.ExpectCalledWithExactly(order.Amount).
        InjectReturnValues(receipt, nil)

    call.ExpectReturnsEqual(expectedResult, nil)
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
    mockRepo := MockRepository(t)
    service := NewUserService(mockRepo.Interface())

    go service.GetUser(123)

    mockRepo.FindByID.ExpectCalledWithExactly(123).
        InjectReturnValues(User{ID: 123, Name: "Alice"}, nil)
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
| Function | `impgen pkg.MyFunc --target` | `WrapMyFunc` | [02](../UAT/02-callable-wrappers/) |
| Struct method | `impgen pkg.Calculator.Add --target` | `WrapCalculatorAdd` | [02](../UAT/02-callable-wrappers/) |
| Struct (all methods) | `impgen pkg.Calculator --target` | `WrapCalculator` | [33](../UAT/33-struct-as-target/) |
| Interface | `impgen pkg.Logger --target` | `WrapLogger` | [32](../UAT/32-interface-as-target/) |
| Function type | `impgen pkg.HandlerFunc --target` | `WrapHandlerFunc` | [16](../UAT/16-function-type-wrapping/) |

#### API Pattern

All wrappers follow the same pattern:

```go
// 1. Create wrapper with testing.T and the callable
wrapper := WrapMyFunc(t, pkg.MyFunc)

// 2. Start execution (returns a call handle)
call := wrapper.Start(arg1, arg2, ...)

// 3. Handle any dependency interactions (see Mock Pattern)
mockDep.Method.ExpectCalledWithExactly(...).InjectReturnValues(...)

// 4. Verify the result
call.ExpectReturnsEqual(expectedReturn1, expectedReturn2)
// or
call.ExpectPanicEquals("expected panic message")
```

#### Wrapper Examples

##### Function Wrapper

```go
//go:generate impgen orders.ProcessOrder --target

func TestProcessOrder(t *testing.T) {
    mockService := MockExternalService(t)
    wrapper := WrapProcessOrder(t, orders.ProcessOrder)

    call := wrapper.Start(mockService.Interface(), 42)

    mockService.FetchData.ExpectCalledWithExactly(42).
        InjectReturnValues("data", nil)

    call.ExpectReturnsEqual("Result: data", nil)
}
```

##### Method Wrapper

```go
//go:generate impgen calculator.Calculator.Add --target

func TestCalculatorAdd(t *testing.T) {
    calc := calculator.NewCalculator()
    wrapper := WrapCalculatorAdd(t, calc.Add)

    call := wrapper.Start(10, 20)
    call.ExpectReturnsEqual(30)
}
```

##### Struct Wrapper (All Methods)

```go
//go:generate impgen calculator.Calculator --target

func TestCalculator(t *testing.T) {
    calc := calculator.NewCalculator()
    wrapper := WrapCalculator(t, calc)

    addCall := wrapper.Add.Start(10, 20)
    addCall.ExpectReturnsEqual(30)

    mulCall := wrapper.Multiply.Start(5, 6)
    mulCall.ExpectReturnsEqual(30)
}
```

##### Interface Wrapper

```go
//go:generate impgen handlers.Logger --target

func TestLogger(t *testing.T) {
    impl := handlers.NewConsoleLogger()
    wrapper := WrapLogger(t, impl)

    infoCall := wrapper.Info.Start("test message")
    infoCall.ExpectReturnsEqual()

    warnCall := wrapper.Warn.Start("warning!")
    warnCall.ExpectReturnsEqual()
}
```

##### Function Type Wrapper

```go
//go:generate impgen visitor.WalkFunc --target

func TestWalkFunc(t *testing.T) {
    myWalker := func(path string, info os.FileInfo) error {
        return nil
    }
    wrapper := WrapWalkFunc(t, myWalker)

    call := wrapper.Start("/path", mockFileInfo)
    call.ExpectReturnsEqual(nil)
}
```

##### Panic Verification

```go
//go:generate impgen safety.UnsafeRunner --target

func TestUnsafeRunnerPanics(t *testing.T) {
    wrapper := WrapUnsafeRunner(t, safety.UnsafeRunner)

    call := wrapper.Start(10, 0)  // Division by zero
    call.ExpectPanicEquals("division by zero")
}
```

---

### Mock Pattern (--dependency)

**What it does**: Creates a mock implementation of a dependency to control its behavior and verify interactions.

**When to use it**: When you need to isolate the code under test from its dependencies.

#### Supported Symbol Types

| Symbol Type | Directive Example | Generated | UAT |
|-------------|-------------------|-----------|-----|
| Interface | `impgen pkg.Service --dependency` | `MockService` | [01](../UAT/01-basic-interface-mocking/) |
| Function type | `impgen pkg.HandlerFunc --dependency` | `MockHandlerFunc` | [31](../UAT/31-function-type-dependency/) |

**Coming soon** (see [REORGANIZATION_PROPOSAL.md](./REORGANIZATION_PROPOSAL.md)):
- Function as dependency (#43)
- Struct as dependency (#44)
- Struct method as dependency (#45)

#### API Pattern

All mocks follow the same pattern:

```go
// 1. Create mock with testing.T
mock := MockService(t)

// 2. Get the interface implementation to pass to code under test
svc := mock.Interface()

// 3. Run code under test (usually in a goroutine)
go myFunction(svc)

// 4. For each expected call:
mock.MethodName.ExpectCalledWithExactly(arg1, arg2).
    InjectReturnValues(ret1, ret2)

// Or with matchers for flexible matching:
mock.MethodName.ExpectCalledWithMatches(imptest.Any(), expectedArg2).
    InjectReturnValues(ret1, ret2)

// Or for eventual (out-of-order) calls in concurrent code:
mock.MethodName.Eventually().ExpectCalledWithExactly(arg1, arg2).
    InjectReturnValues(ret1, ret2)
```

#### Mock Examples

##### Basic Interface Mock

```go
//go:generate impgen storage.Repository --dependency

func TestUserService(t *testing.T) {
    mock := MockRepository(t)
    service := NewUserService(mock.Interface())

    go service.GetUser(123)

    mock.FindByID.ExpectCalledWithExactly(123).
        InjectReturnValues(User{ID: 123}, nil)
}
```

##### Variadic Parameters

```go
//go:generate impgen logger.Logger --dependency

func TestWithVariadic(t *testing.T) {
    mock := MockLogger(t)

    go doWork(mock.Interface())

    // Variadic args passed normally
    mock.Logf.ExpectCalledWithExactly("format: %s %d", "hello", 42).
        InjectReturnValues()
}
```

##### Error Injection

```go
func TestErrorPath(t *testing.T) {
    mock := MockRepository(t)
    service := NewUserService(mock.Interface())

    go service.GetUser(999)

    mock.FindByID.ExpectCalledWithExactly(999).
        InjectReturnValues(User{}, ErrNotFound)
}
```

##### Panic Injection

```go
func TestPanicRecovery(t *testing.T) {
    mock := MockRepository(t)
    service := NewUserService(mock.Interface())

    go service.GetUser(123)

    mock.FindByID.ExpectCalledWithExactly(123).
        InjectPanic("database connection lost")
}
```

##### Function Type Mock

```go
//go:generate impgen handlers.Router --dependency

func TestRouterDependency(t *testing.T) {
    mock := MockRouter(t)

    go processRequests(mock.Func())

    mock.ExpectCalledWithExactly("/api/users").
        InjectReturnValues(handlerFunc)
}
```

##### Generic Interface Mock

```go
//go:generate impgen cache.Cache[T] --dependency

func TestGenericCache(t *testing.T) {
    mock := MockCache[User](t)
    service := NewCachingService(mock.Interface())

    go service.GetCached("user:123")

    mock.Get.ExpectCalledWithExactly("user:123").
        InjectReturnValues(User{ID: 123}, true)
}
```

##### Embedded Interface Mock

```go
//go:generate impgen io.ReadCloser --dependency

func TestReadCloser(t *testing.T) {
    mock := MockReadCloser(t)

    go processFile(mock.Interface())

    // Methods from embedded interfaces are accessible
    mock.Read.ExpectCalledWithMatches(imptest.Any()).
        InjectReturnValues(10, nil)
    mock.Close.ExpectCalledWithExactly().
        InjectReturnValues(nil)
}
```

---

## Variations Reference

### Package Handling

imptest handles symbols from various package contexts:

| Package Location | Supported | UAT | Notes |
|------------------|-----------|-----|-------|
| Same package | Yes | [12](../UAT/12-whitebox-testing/), [14](../UAT/14-same-package-interfaces/) | Whitebox testing |
| Different package | Yes | [01](../UAT/01-basic-interface-mocking/), [02](../UAT/02-callable-wrappers/) | Standard usage |
| Standard library | Yes | [18](../UAT/18-external-function-types/), [08](../UAT/08-embedded-interfaces/) | `http.HandlerFunc`, `io.Reader` |
| Aliased import | Yes | — | `import alias "pkg"` |
| Dot import | Yes | [26](../UAT/26-dot-imports/), [27](../UAT/27-business-logic-dot-imports/) | `import . "pkg"` |
| Stdlib shadowing | Yes | [11](../UAT/11-package-name-conflicts/) | 4-tier resolution |

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
| Zero parameters | Yes | [01](../UAT/01-basic-interface-mocking/) |
| Multiple parameters | Yes | [10](../UAT/10-edge-many-params/) |
| Zero returns | Yes | [09](../UAT/09-edge-zero-returns/) |
| Multiple returns | Yes | [02](../UAT/02-callable-wrappers/) |
| Variadic parameters | Yes | [01](../UAT/01-basic-interface-mocking/) |
| Named parameters/returns | Yes | [23](../UAT/23-named-params-returns/) |

#### Type Complexity

| Type | Supported | UAT | Notes |
|------|-----------|-----|-------|
| Concrete types | Yes | All | int, string, structs, etc. |
| Generic types | Yes | [07](../UAT/07-generics/), [21](../UAT/21-parameterized-types/) | `[T any]`, `[T Numeric]` |
| Non-comparable types | Yes | [03](../UAT/03-non-comparable-arguments/) | Slices, maps, functions |
| Struct literals | Yes | [30](../UAT/30-struct-literal-params/) | `struct{ Field int }` |
| Function literals | Yes | [24](../UAT/24-function-literal-params/) | `func(int) int` |
| Interface literals | Yes | [25](../UAT/25-interface-literal-params/) | `interface{ Method() }` |
| Channels | Yes | [20](../UAT/20-channel-types/) | All directions |
| External types | Yes | [13](../UAT/13-external-type-imports/), [29](../UAT/29-cross-file-external-imports/) | `time.Time`, `os.FileInfo` |

#### Matching Function Parameters

When testing with function parameters, use matchers since Go functions cannot be compared:

```go
// DON'T use ExpectCalledWithExactly for functions
// mock.Map.ExpectCalledWithExactly(items, mapFunc)  // Will hang!

// DO use matchers
mock.Map.ExpectCalledWithMatches(items, imptest.Any()).
    InjectReturnValues(result)
```

---

### Concurrency

#### Ordered Mode (Default)

By default, imptest expects calls in the exact order you specify:

```go
mock.First.ExpectCalledWithExactly(1).InjectReturnValues(1)
mock.Second.ExpectCalledWithExactly(2).InjectReturnValues(2)
// Calls MUST arrive as: First, then Second
```

If calls arrive out of order, the test fails immediately.

#### Eventually Mode

For concurrent code where call order is non-deterministic, use `Eventually()`:

```go
// These can arrive in any order
mock.TaskA.Eventually().ExpectCalledWithExactly("a").InjectReturnValues()
mock.TaskB.Eventually().ExpectCalledWithExactly("b").InjectReturnValues()
mock.TaskC.Eventually().ExpectCalledWithExactly("c").InjectReturnValues()
```

**UAT**: [28](../UAT/28-ordered-eventually-modes/)

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

| UAT | Name | Symbol Type |
|-----|------|-------------|
| [02](../UAT/02-callable-wrappers/) | callable-wrappers | Function, Method |
| [04](../UAT/04-error-and-panic-handling/) | error-and-panic-handling | Function |
| [07](../UAT/07-generics/) | generics | Generic function |
| [09](../UAT/09-edge-zero-returns/) | edge-zero-returns | Function |
| [15](../UAT/15-callback-visitor/) | callback-visitor | Function, Function type |
| [16](../UAT/16-function-type-wrapping/) | function-type-wrapping | Function type |
| [18](../UAT/18-external-function-types/) | external-function-types | Function type (stdlib) |
| [23](../UAT/23-named-params-returns/) | named-params-returns | Function, Method |
| [24](../UAT/24-function-literal-params/) | function-literal-params | Function, Method |
| [30](../UAT/30-struct-literal-params/) | struct-literal-params | Function, Method |
| [32](../UAT/32-interface-as-target/) | interface-as-target | Interface |
| [33](../UAT/33-struct-as-target/) | struct-as-target | Struct |

#### Mock Pattern (--dependency)

| UAT | Name | Symbol Type |
|-----|------|-------------|
| [01](../UAT/01-basic-interface-mocking/) | basic-interface-mocking | Interface |
| [03](../UAT/03-non-comparable-arguments/) | non-comparable-arguments | Interface |
| [05](../UAT/05-advanced-matching/) | advanced-matching | Interface |
| [06](../UAT/06-concurrency/) | concurrency | Interface |
| [07](../UAT/07-generics/) | generics | Generic interface |
| [08](../UAT/08-embedded-interfaces/) | embedded-interfaces | Interface |
| [10](../UAT/10-edge-many-params/) | edge-many-params | Interface |
| [12](../UAT/12-whitebox-testing/) | whitebox-testing | Interface |
| [14](../UAT/14-same-package-interfaces/) | same-package-interfaces | Interface |
| [19](../UAT/19-interface-external-func-type/) | interface-external-func-type | Interface |
| [20](../UAT/20-channel-types/) | channel-types | Interface |
| [21](../UAT/21-parameterized-types/) | parameterized-types | Generic interface |
| [22](../UAT/22-test-package-import/) | test-package-import | Interface |
| [25](../UAT/25-interface-literal-params/) | interface-literal-params | Interface |
| [31](../UAT/31-function-type-dependency/) | function-type-dependency | Function type |

### By Variation

#### Package Variations

| UAT | Name | Variation |
|-----|------|-----------|
| [11](../UAT/11-package-name-conflicts/) | package-name-conflicts | Stdlib shadowing |
| [12](../UAT/12-whitebox-testing/) | whitebox-testing | Same package |
| [14](../UAT/14-same-package-interfaces/) | same-package-interfaces | Same package |
| [18](../UAT/18-external-function-types/) | external-function-types | Stdlib |
| [22](../UAT/22-test-package-import/) | test-package-import | External module |
| [26](../UAT/26-dot-imports/) | dot-imports | Dot import |
| [27](../UAT/27-business-logic-dot-imports/) | business-logic-dot-imports | Dot import |

#### Signature Variations

| UAT | Name | Variation |
|-----|------|-----------|
| [07](../UAT/07-generics/) | generics | Generic types |
| [21](../UAT/21-parameterized-types/) | parameterized-types | Constrained generics |
| [03](../UAT/03-non-comparable-arguments/) | non-comparable-arguments | Slices, maps |
| [23](../UAT/23-named-params-returns/) | named-params-returns | Named params/returns |
| [24](../UAT/24-function-literal-params/) | function-literal-params | Function literal params |
| [25](../UAT/25-interface-literal-params/) | interface-literal-params | Interface literal params |
| [30](../UAT/30-struct-literal-params/) | struct-literal-params | Struct literal params |
| [20](../UAT/20-channel-types/) | channel-types | Channel types |
| [13](../UAT/13-external-type-imports/) | external-type-imports | External types |
| [29](../UAT/29-cross-file-external-imports/) | cross-file-external-imports | Cross-file imports |
| [08](../UAT/08-embedded-interfaces/) | embedded-interfaces | Embedded interfaces |

#### Behavior Variations

| UAT | Name | Variation |
|-----|------|-----------|
| [04](../UAT/04-error-and-panic-handling/) | error-and-panic-handling | Panic handling |
| [15](../UAT/15-callback-visitor/) | callback-visitor | Callbacks |

#### Concurrency Variations

| UAT | Name | Variation |
|-----|------|-----------|
| [06](../UAT/06-concurrency/) | concurrency | Eventually() |
| [28](../UAT/28-ordered-eventually-modes/) | ordered-eventually-modes | Ordered vs Eventually |
