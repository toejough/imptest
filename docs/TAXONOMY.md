# What imptest Can (and Cannot) Do

<!-- Last reviewed: 2025-12-30 | Review triggers: new UAT added, major feature change -->

## Table of Contents

- [Introduction](#introduction)
- [Capability Matrix (Types)](#capability-matrix-types)
- [Package Variations Matrix](#package-variations-matrix)
- [Signature Variations Matrix](#signature-variations-matrix)
- [Concurrency Support](#concurrency-support)
- [Target Examples](#target-examples)
- [Dependency Examples](#dependency-examples)
- [Cannot Do (With Workarounds)](#cannot-do-with-workarounds)
- [UAT Directory Index](#uat-directory-index)

---

## Introduction

This document is the comprehensive reference for imptest capabilities. For getting started guides and tutorials, see the [README](https://github.com/toejough/imptest).

imptest enables testing of **impure functions**—functions that interact with external dependencies. It provides two mechanisms:

- **Targets**: Wrap the function under test to verify its behavior by observing its interactions not only with the caller, but also with dependencies it uses internally (`--target`)
- **Dependencies**: Mock the dependencies of the function under test to validate when & how they are called, and to control what they return to the function under test (`--dependency`)

The matrices below document what imptest can wrap, mock, and handle. Each cell links to examples and UAT coverage.

---

## Capability Matrix (Types)

| What | As Target | As Dependency | Notes |
|------|-----------|---------------|-------|
| Function | [Yes](../UAT/02-callable-wrappers/) | ? | `impgen pkg.MyFunc --target` |
| Function type | [Yes](../UAT/16-function-type-wrapping/) | ? | `impgen pkg.HandlerFunc --target` |
| Anonymous function | [No](#1-anonymous-functions-function-literals) | [No](#1-anonymous-functions-function-literals) | Create named function type first |
| Struct type | [Yes](../UAT/02-callable-wrappers/) | ? | `impgen pkg.Calculator.Add --target` |
| Interface type | ? | [Yes](../UAT/01-basic-interface-mocking/) | `impgen pkg.MyInterface --dependency` |
| Anonymous type | ? | ? | Struct literal (untested) |

**Legend**:
- **Yes** = Supported with UAT coverage
- **?** = Untested or limited coverage
- **No** = Not supported (see workaround)

---

## Package Variations Matrix

This matrix documents where the types to be wrapped/mocked can be defined. Package handling has historically required careful implementation.

| Package Location | Targets | Dependencies | UAT Coverage | Notes |
|------------------|---------|--------------|--------------|-------|
| Same package | Yes | Yes | [14](../UAT/14-same-package-interfaces/), [12](../UAT/12-whitebox-testing/) | Type in same package as test |
| Different package | Yes | Yes | [01](../UAT/01-basic-interface-mocking/), [02](../UAT/02-callable-wrappers/), [22](../UAT/22-test-package-import/) | Local or external module |
| Standard library | Yes | Yes | [18](../UAT/18-external-function-types/), [08](../UAT/08-embedded-interfaces/) | `http.HandlerFunc`, `io.Reader` |
| Standard library shadowing | Yes | Yes | [11](../UAT/11-package-name-conflicts/) | 4-tier resolution (see below) |
| Aliased import | Yes | Yes | — | `import alias "github.com/foo/bar"` |
| Dot import | Yes | Yes | [26](../UAT/26-dot-imports/) | `import . "pkg"` - symbols available without qualification |

**Legend**: Yes = supported, ? = untested

### Standard Library Shadowing Resolution

**Standard library shadowing** ([UAT-11](../UAT/11-package-name-conflicts/)) is now **fully supported** via a 4-tier resolution strategy (implemented in commit cae3385):

**The Problem**: When a local package shadows a stdlib package name (e.g., local `time` package shadowing stdlib `time`), and the test doesn't import either package, impgen previously couldn't determine which package was intended:

```go
// Local "time" package exists
// Test doesn't import any "time" package
//go:generate impgen time.Timer --dependency  // Which "time"?
```

**The Solution**: 4-tier resolution strategy:
1. **Explicit `--import-path` flag** (highest priority): `impgen --import-path=time time.Timer` for stdlib or `impgen --import-path=github.com/user/project/time time.Timer` for local
2. **Infer from test file imports**: If test imports the package (aliased or not), use that import path automatically
3. **Detect ambiguity**: If both stdlib and local package exist and neither is imported, error with helpful message suggesting `--import-path` or adding an import
4. **Fallback**: Existing resolution logic for non-ambiguous cases

---

## Signature Variations Matrix

This matrix documents what parameter and return types are supported in wrapped/mocked signatures.

### Parameter/Return Count and Naming

| Feature | Parameters | Returns | UAT Coverage |
|---------|------------|---------|--------------|
| Zero (0) | Yes | Yes | [01](../UAT/01-basic-interface-mocking/), [09](../UAT/09-edge-zero-returns/) |
| One (1) | Yes | Yes | [02](../UAT/02-callable-wrappers/) |
| Multiple (n) | Yes | Yes | [10](../UAT/10-edge-many-params/) |
| Variadic | Yes | — | [01](../UAT/01-basic-interface-mocking/) |
| Named | Yes | Yes | [23](../UAT/23-named-params-returns/) |
| Anonymous | Yes | Yes | Most UATs |

### Type Complexity

| Feature | In Args | In Returns | UAT Coverage |
|---------|---------|------------|--------------|
| Concrete types | Yes | Yes | Most UATs |
| Generic types | Yes | Yes | [07](../UAT/07-generics/), [21](../UAT/21-parameterized-types/) |
| Comparable types | Yes | Yes | [02](../UAT/02-callable-wrappers/) |
| Non-comparable types | Yes | Yes | [03](../UAT/03-non-comparable-arguments/) |
| Simple types (int, string, etc.) | Yes | Yes | Most UATs |
| Struct literal | ? | ? | — |
| Function literal | Yes | Yes | [24](../UAT/24-function-literal-params/) | Multi-param function literals now supported |
| Interface literal | Yes | Yes | [25](../UAT/25-interface-literal-params/) |
| Channel | Yes | Yes | [20](../UAT/20-channel-types/) |
| Directional channel | Yes | Yes | [20](../UAT/20-channel-types/) |
| Slice | Yes | Yes | [03](../UAT/03-non-comparable-arguments/) |
| Map | Yes | Yes | [03](../UAT/03-non-comparable-arguments/) |
| Pointer | Yes | Yes | [13](../UAT/13-external-type-imports/) |

**Legend**: Yes = supported, ? = untested, — = not applicable

---

## Concurrency Support

imptest provides mechanisms to handle concurrent and potentially unordered calls:

| Feature | Support | UAT Coverage | Notes |
|---------|---------|--------------|-------|
| `Eventually()` | Yes | [06](../UAT/06-concurrency/) | Matches calls that may arrive out of order due to concurrency |
| Ordered expectations | Yes | All UATs | Default behavior - expects calls in order |
| Race-free mocking | Yes | [06](../UAT/06-concurrency/) | Thread-safe call recording and matching |

### Eventually()

Use `Eventually()` when testing concurrent code where call order is not deterministic:

```go
mock.Method.Eventually().ExpectCalledWithExactly(arg1, arg2)
```

This allows the test to match calls that arrive in any order, rather than requiring strict sequential ordering.

---

## Target Examples

### Example 1: Basic Function Wrapping

Wrap a package-level function to verify its behavior and interactions with dependencies.

**Directive**:
```go
//go:generate impgen callable.BusinessLogic --target
```

**Usage**:
```go
// Create wrapper and configure expectations
wrapper := callable.WrapBusinessLogic()
wrapper.ExpectCalledWithExactly(input)
wrapper.ExpectReturnsEqual(expectedOutput)

// Execute function under test
result := wrapper.Call(input)

// Verify all expectations met
wrapper.Verify(t)
```

**Coverage**: [UAT/02-callable-wrappers](../UAT/02-callable-wrappers/)

**Notes**:
- Wrapping functions allows testing of business logic that calls external dependencies
- Use `Call()` to invoke the wrapped function with controlled inputs
- Combine with dependency mocks to isolate unit under test

---

### Example 2: Struct Method Wrapping

Wrap methods on a struct type to verify method behavior.

**Directive**:
```go
//go:generate impgen callable.Calculator.Add --target
//go:generate impgen callable.Calculator.Multiply --target
//go:generate impgen callable.Calculator.Divide --target
```

**Usage**:
```go
// Create wrapper for specific method
addWrapper := callable.WrapCalculatorAdd()
addWrapper.ExpectCalledWithExactly(5, 3)
addWrapper.ExpectReturnsEqual(8)

// Create Calculator instance and invoke
calc := Calculator{}
result := addWrapper.Call(calc, 5, 3)

// Verify expectations
addWrapper.Verify(t)
```

**Coverage**: [UAT/02-callable-wrappers](../UAT/02-callable-wrappers/)

**Notes**:
- Generate separate wrappers for each method you want to test
- Pass the struct instance as first argument to `Call()`
- Useful for testing stateful objects with complex method interactions

---

### Example 3: Local Function Type Wrapping

Wrap a locally-defined function type to control callback behavior.

**Directive**:
```go
//go:generate impgen visitor.WalkFunc --target
```

**Usage**:
```go
// Create wrapper for function type
walkWrapper := visitor.WrapWalkFunc()
walkWrapper.ExpectCalledWithExactly("/path/to/file", fileInfo)
walkWrapper.ExpectReturnsEqual(nil)

// Pass wrapped function as callback
walker := TreeWalker{walkFn: walkWrapper.Callable()}
err := walker.Walk("/path")

// Verify callback was invoked correctly
walkWrapper.Verify(t)
```

**Coverage**: [UAT/15-callback-visitor](../UAT/15-callback-visitor/), [UAT/16-function-type-wrapping](../UAT/16-function-type-wrapping/)

**Notes**:
- Function type wrappers expose `Callable()` method to get the wrapped function
- Ideal for testing code that accepts callbacks or handlers
- Can verify callback parameters using `GetArgs()` on dependency mocks

---

### Example 4: External Function Type Wrapping (stdlib)

Wrap function types from standard library or external packages.

**Directive**:
```go
//go:generate impgen http.HandlerFunc --target
```

**Usage**:
```go
// Create wrapper for http.HandlerFunc
handlerWrapper := http.WrapHandlerFunc()
handlerWrapper.ExpectCalled()

// Use wrapped handler in HTTP server
mux := http.NewServeMux()
mux.HandleFunc("/api", handlerWrapper.Callable())

// Simulate request
req := httptest.NewRequest("GET", "/api", nil)
recorder := httptest.NewRecorder()
mux.ServeHTTP(recorder, req)

// Verify handler was invoked
handlerWrapper.Verify(t)
```

**Coverage**: [UAT/18-external-function-types](../UAT/18-external-function-types/)

**Notes**:
- Works with any named function type from external packages
- impgen automatically imports the external package
- Common for testing HTTP handlers, middleware, and callbacks

---

### Example 5: Function with Panic Verification

Wrap functions to verify panic behavior during testing.

**Directive**:
```go
//go:generate impgen safety.UnsafeRunner --target
```

**Usage**:
```go
// Create wrapper expecting panic
wrapper := safety.WrapUnsafeRunner()
wrapper.ExpectPanic("division by zero")

// Execute function that should panic
func() {
    defer func() { recover() }()
    wrapper.Call(10, 0)
}()

// Verify panic occurred with expected message
wrapper.Verify(t)
```

**Coverage**: [UAT/04-error-and-panic-handling](../UAT/04-error-and-panic-handling/)

**Notes**:
- Use `ExpectPanic(message)` to verify panic occurs with specific message
- Combine with dependency mocks that use `InjectPanic()` to test error paths
- Critical for testing error handling and recovery mechanisms

---

### Example 6: Function with Callback Parameter

Wrap functions that accept callbacks to verify both function and callback behavior.

**Directive**:
```go
//go:generate impgen visitor.CountFiles --target
```

**Usage**:
```go
// Create wrapper with callback expectation
wrapper := visitor.WrapCountFiles()
wrapper.ExpectCalledWith(gomega.Any())

// Create mock for callback dependency
mockWalker := visitor.MockTreeWalker()
mockWalker.Walk.ExpectCalled()

// Execute function under test
count := wrapper.Call(mockWalker.Interface())

// Extract and verify callback from mock
args := mockWalker.Walk.GetArgs(0)
callback := args[0].(visitor.WalkFunc)
// ... test callback behavior

wrapper.Verify(t)
mockWalker.Verify(t)
```

**Coverage**: [UAT/15-callback-visitor](../UAT/15-callback-visitor/)

**Notes**:
- Use `GetArgs()` on dependency mocks to extract callback parameters
- Callbacks can be tested independently after extraction
- Powerful pattern for testing visitor, observer, and strategy patterns

---

## Dependency Examples

### Example 1: Basic Interface Mocking

Mock a simple interface to control dependency behavior in tests.

**Directive**:
```go
//go:generate impgen basic.Ops --dependency
```

**Usage**:
```go
// Create mock and configure expectations
mock := basic.MockOps()
mock.Notify.ExpectCalledWithExactly("Task complete", 1, 2, 3)
mock.Finish.ExpectCalled()

// Pass mock to code under test
service := NewService(mock.Interface())
service.ProcessTask()

// Verify all expected calls occurred
mock.Verify(t)
```

**Coverage**: [UAT/01-basic-interface-mocking](../UAT/01-basic-interface-mocking/)

**Notes**:
- Each interface method gets its own expectation API (e.g., `mock.Notify`, `mock.Finish`)
- Use `ExpectCalledWithExactly()` for variadic parameters
- Call `mock.Interface()` to get the interface value to pass to code under test

---

### Example 2: Generic Interface Mocking

Mock interfaces with type parameters to test generic code.

**Directive**:
```go
//go:generate impgen generics.Repository[T] --dependency
```

**Usage**:
```go
// Create mock for specific type parameter
mock := generics.MockRepository[User]()
mock.Get.ExpectCalledWithExactly(123)
mock.Get.ExpectReturnsEqual(User{ID: 123, Name: "Alice"}, nil)

// Use mock with generic function
repo := mock.Interface()
user, err := ProcessUser(repo, 123)

// Verify behavior
assert.NoError(t, err)
assert.Equal(t, "Alice", user.Name)
mock.Verify(t)
```

**Coverage**: [UAT/07-generics](../UAT/07-generics/), [UAT/21-parameterized-types](../UAT/21-parameterized-types/)

**Notes**:
- Specify concrete type parameter when creating mock (e.g., `MockRepository[User]()`)
- Works with both simple type parameters (`[T any]`) and constrained types (`[T Numeric]`)
- Type safety maintained throughout test

---

### Example 3: Embedded Interface Mocking

Mock interfaces that embed other interfaces, automatically expanding all methods.

**Directive**:
```go
//go:generate impgen embedded.ReadCloser --dependency
```

**Usage**:
```go
// Create mock - embeds both io.Reader and io.Closer
mock := embedded.MockReadCloser()

// Configure expectations for all embedded methods
mock.Read.ExpectCalledWith(gomega.Any())
mock.Read.ExpectReturnsEqual(10, nil)
mock.Close.ExpectCalled()
mock.Close.ExpectReturnsEqual(nil)

// Use mock
rc := mock.Interface()
buf := make([]byte, 100)
n, err := rc.Read(buf)
rc.Close()

// Verify
assert.NoError(t, err)
assert.Equal(t, 10, n)
mock.Verify(t)
```

**Coverage**: [UAT/08-embedded-interfaces](../UAT/08-embedded-interfaces/)

**Notes**:
- impgen automatically expands embedded interfaces into individual method expectations
- Works with standard library interfaces (e.g., `io.Reader`, `io.Closer`)
- No need to manually implement each embedded method

---

### Example 4: Interface with External Type Parameters

Mock interfaces that use external types (e.g., stdlib types) in method signatures.

**Directive**:
```go
//go:generate impgen middleware.HTTPMiddleware --dependency
```

**Usage**:
```go
// Create mock with methods using http.HandlerFunc
mock := middleware.MockHTTPMiddleware()

// Configure expectations for method with external type
var capturedHandler http.HandlerFunc
mock.Wrap.ExpectCalledWith(gomega.Any())
mock.Wrap.ExpectReturns(func(h http.HandlerFunc) http.HandlerFunc {
    capturedHandler = h
    return func(w http.ResponseWriter, r *http.Request) {
        // Middleware logic
        h(w, r)
    }
})

// Use mock
mw := mock.Interface()
wrapped := mw.Wrap(originalHandler)

// Verify and test captured handler
mock.Verify(t)
assert.NotNil(t, capturedHandler)
```

**Coverage**: [UAT/19-interface-external-func-type](../UAT/19-interface-external-func-type/)

**Notes**:
- impgen automatically imports external packages (e.g., `net/http`)
- External types are fully qualified in generated code
- Useful for mocking middleware, adapters, and wrappers

---

### Example 5: Same-Package Interface Mocking (Whitebox Testing)

Mock interfaces defined in the same package as the test for whitebox testing.

**Directive**:
```go
//go:generate impgen Ops --dependency
```

**Usage**:
```go
// In same package (no _test suffix), mock local interface
mock := MockOps()
mock.Process.ExpectCalledWithExactly(42)
mock.Process.ExpectReturnsEqual(84)

// Test internal implementation details
result := internalFunction(mock.Interface(), 42)

// Verify
assert.Equal(t, 84, result)
mock.Verify(t)
```

**Coverage**: [UAT/12-whitebox-testing](../UAT/12-whitebox-testing/), [UAT/14-same-package-interfaces](../UAT/14-same-package-interfaces/)

**Notes**:
- No package prefix in directive when interface is in same package
- Test file uses same package name (no `_test` suffix)
- Enables testing of internal/unexported implementations
- Multiple interfaces in same package can each have separate mocks

---

## Cannot Do (With Workarounds)

### 1. Anonymous Functions (Function Literals)

**Problem**: Cannot directly wrap or mock inline function literals.

```go
// This won't work - no named type to reference
//go:generate impgen ??? --target
result := func(x int) int { return x * 2 }(5)
```

**Workaround**: Create a named function type, then wrap that type.

```go
// Define named function type
type Transformer func(int) int

//go:generate impgen mypackage.Transformer --target

// Use in code
var transform Transformer = func(x int) int { return x * 2 }
wrapper := mypackage.WrapTransformer()
result := wrapper.Call(5)
```

**Example UATs**: [UAT/15-callback-visitor](../UAT/15-callback-visitor/), [UAT/16-function-type-wrapping](../UAT/16-function-type-wrapping/)

---

### 2. Anonymous Types (Struct Literals)

**Problem**: Cannot wrap methods on struct literals or unnamed types.

```go
// This won't work - struct has no name
var calculator = struct {
    Add func(int, int) int
}{
    Add: func(a, b int) int { return a + b },
}
```

**Workaround**: Define a named struct type, then wrap its methods.

```go
// Define named struct type
type Calculator struct{}

func (c Calculator) Add(a, b int) int { return a + b }

//go:generate impgen mypackage.Calculator.Add --target

// Now you can wrap the method
wrapper := mypackage.WrapCalculatorAdd()
```

**Example UATs**: [UAT/02-callable-wrappers](../UAT/02-callable-wrappers/)

---

### 3. Stdlib Package Shadowing (Now Supported ✓)

**Problem**: When a local package shadows a stdlib package name and the test doesn't import the shadowed package, impgen needs to determine which package is intended.

```go
// Local package named "time" exists
// Test doesn't import either "time" package

//go:generate impgen time.Timer --dependency  // Which "time"?
```

**Status**: **Fully supported** as of commit cae3385 via 4-tier resolution strategy.

**Solutions** (in priority order):
1. **Use `--import-path` flag**: `impgen --import-path=time time.Timer --dependency` (stdlib) or `impgen --import-path=github.com/user/project/time time.Timer --dependency` (local)
2. **Import the package in your test**: impgen will automatically infer from imports
3. **Let impgen detect ambiguity**: If both exist and neither imported, impgen errors with helpful suggestions
4. **Fallback resolution**: Works automatically for non-ambiguous cases

See [Package Variations Matrix](#package-variations-matrix) for details on the 4-tier resolution strategy.

**Example UAT**: [UAT/11-package-name-conflicts](../UAT/11-package-name-conflicts/)

---

### 4. Dot Imports (Now Supported ✓)

**Previous Problem**: Dot imports (`import . "pkg"`) made package boundaries unclear during symbol resolution.

**Example that now works**:

```go
import . "github.com/example/helpers"

// impgen can now find Storage from the dot-imported package
//go:generate impgen Storage --dependency

func TestWithDotImport(t *testing.T) {
	mock := MockStorage(t)
	// Storage is available without package qualification
	var _ Storage = mock.Interface()
}
```

**Implementation**: When a symbol is not found in the current package, impgen now checks all dot-imported packages, loads each one, and searches for the symbol. The generated mock correctly imports the source package where the symbol was found.

**Supported**: Yes (as of [UAT-26](../UAT/26-dot-imports/))

**Note**: While dot imports are now supported for mocking, they are generally discouraged in Go code for clarity. Use explicit imports when practical.

---

### 5. Multi-Parameter Function Literals (Now Fixed ✓)

**Previous Problem**: Function literal parameters with multiple arguments had their parameters incorrectly parsed by impgen code generation, dropping all but the first parameter.

**Example that now works**:
```go
type DataProcessor interface {
    // Multi-parameter function literals now work correctly
    Reduce(items []int, initial int, reducer func(acc, item int) int) int
}

//go:generate impgen DataProcessor --dependency
```

**Fix Applied**: Created `expandFieldListTypes` helper function that properly expands fields with multiple names. In Go AST, `func(a, b int)` is represented as ONE field with `Names=[a,b]` and `Type=int`. The fix ensures the type string is repeated for each name, correctly rendering `func(int, int)` instead of `func(int)`.

**Important Note**: When testing with function literal parameters, you **must** use matchers instead of exact equality, because Go functions cannot be compared with `==`:

```go
// In tests - ALWAYS use matchers for function literal params
reducer := func(acc, item int) int { return acc + item }

// Don't use ExpectCalledWithExactly - will hang!
// mock.Reduce.ExpectCalledWithExactly(items, 0, reducer)

// DO use ExpectCalledWithMatches with imptest.Any()
mock.Reduce.ExpectCalledWithMatches(items, imptest.Any(), imptest.Any()).
    InjectReturnValues(10)
```

**Example UAT**: [UAT/24-function-literal-params](../UAT/24-function-literal-params/)

**Status**: ✅ Fixed in issue #10 (2025-12-31). Multi-parameter function literals now work correctly. Use matchers for function parameters in tests to avoid comparison issues.

---

## UAT Directory Index

Quick reference for locating User Acceptance Tests by feature coverage.

| UAT | Name | Primary Feature | Type Coverage | Package Coverage | Signature Features |
|-----|------|-----------------|---------------|------------------|-------------------|
| [01](../UAT/01-basic-interface-mocking/) | basic-interface-mocking | Core interface mocking | Interface (dep) | Local | Variadic params, zero params |
| [02](../UAT/02-callable-wrappers/) | callable-wrappers | Function & method wrapping | Function, Methods (target)<br>Interface (dep) | Local | Multiple params/returns |
| [03](../UAT/03-non-comparable-arguments/) | non-comparable-arguments | Non-comparable types | Interface (dep) | Local | Slice args, map args |
| [04](../UAT/04-error-and-panic-handling/) | error-and-panic-handling | Panic injection | Interface (dep)<br>Functions (target) | Local | Error returns, panics |
| [05](../UAT/05-advanced-matching/) | advanced-matching | Gomega matchers | Interface (dep) | Local | Various return types |
| [06](../UAT/06-concurrency/) | concurrency | Eventually() async | Interface (dep) | Local | Concurrent calls |
| [07](../UAT/07-generics/) | generics | Generic types | Generic interface (dep)<br>Generic function (target) | Local | Type parameters `[T any]` |
| [08](../UAT/08-embedded-interfaces/) | embedded-interfaces | Embedded interfaces | Embedded interface (dep) | Local + Stdlib (`io`) | Standard methods |
| [09](../UAT/09-edge-zero-returns/) | edge-zero-returns | Zero return values | Function (target) | Local | Multiple params, no returns |
| [10](../UAT/10-edge-many-params/) | edge-many-params | Many parameters | Interface (dep) | Local | 10 parameters |
| [11](../UAT/11-package-name-conflicts/) | package-name-conflicts | Import aliasing | Interfaces (dep) | Local shadowing Stdlib | **Known hole** |
| [12](../UAT/12-whitebox-testing/) | whitebox-testing | Same-package testing | Interface (dep) | Same package | Standard methods |
| [13](../UAT/13-external-type-imports/) | external-type-imports | External type params | Interface (dep) | Local + External types | External types (`fs.DirEntry`), pointers |
| [14](../UAT/14-same-package-interfaces/) | same-package-interfaces | Multiple same-pkg mocks | Interfaces (dep) | Same package | Multiple interfaces |
| [15](../UAT/15-callback-visitor/) | callback-visitor | Callback extraction | Interface (dep)<br>Function + Func type (target) | Local | Function params, callbacks |
| [16](../UAT/16-function-type-wrapping/) | function-type-wrapping | Local func type wrapping | Named func type (target) | Local | Function type signature |
| [17](../UAT/17-typesafe-getargs/) | typesafe-getargs | Type-safe GetArgs() | Interface (manual) | Local | Mixed param types |
| [18](../UAT/18-external-function-types/) | external-function-types | External func type wrapping | Named func type (target) | Stdlib (`net/http`) | Zero returns |
| [19](../UAT/19-interface-external-func-type/) | interface-external-func-type | Interface with external types | Interface (dep) | Local + Stdlib refs | External func type params |
| [20](../UAT/20-channel-types/) | channel-types | Channel types | Interface (dep) | Local | Channels (all directions) |
| [21](../UAT/21-parameterized-types/) | parameterized-types | Constrained generics | Generic interface (dep) | Local | Type constraints |
| [22](../UAT/22-test-package-import/) | test-package-import | External module imports | Interface (dep) | External module | Standard methods |
| [23](../UAT/23-named-params-returns/) | named-params-returns | Named parameters/returns | Interface (dep)<br>Method + Function (target) | Local | Named params/returns |
| [24](../UAT/24-function-literal-params/) | function-literal-params | Function literal parameters | Interface (dep)<br>Method + Function (target) | Local | Function literal params; matcher usage |
| [25](../UAT/25-interface-literal-params/) | interface-literal-params | Interface literal parameters | Interface (dep) | Local | Interface literal params |
| [26](../UAT/26-dot-imports/) | dot-imports | Dot imports in test code | Interface (dep) | Dot-imported | Storage, Processor interfaces |
| [27](../UAT/27-business-logic-dot-imports/) | business-logic-dot-imports | Dot imports in production code | Interface (dep) | Dot-imported (production) | Repository interface |
| [28](../UAT/28-ordered-eventually-modes/) | ordered-eventually-modes | Ordered vs Eventually modes | Interface (dep) | Local | Call matching modes |
| [29](../UAT/29-cross-file-external-imports/) | cross-file-external-imports | Cross-file import resolution | Interface (dep) | Local + External types (cross-file) | External types from interface file |

### Legend

- **(dep)** = Generated with `--dependency` flag (mock)
- **(target)** = Generated with `--target` flag (wrapper)
- **Local** = Same module, different package
- **Same package** = Test and code in same package (no `_test` suffix)
- **Stdlib** = Standard library package
- **External** = Third-party module
