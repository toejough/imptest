# imptest

**Zero manual mocking. Full control.**

## What is imptest?

**Test impure functions without writing mock implementations.**

imptest generates type-safe mocks from your interfaces. Each test interactively controls the mock—expect calls, inject responses, and validate behavior—all with compile-time safety or flexible matchers. No manual mock code. No complex setup. Just point at an interface and test.

## Why though?

Sometimes you want to test those pesky impure functions that call out to other services, databases, or systems.
Traditional mocking libraries often require you to write mock implementations by hand or configure complex expectations
upfront. imptest changes the game by generating mocks automatically and letting you control them interactively during
tests.

## Quick Start

```go
package mypackage_test

import (
    "testing"
    "github.com/toejough/imptest/UAT/run"
)

//go:generate go run github.com/toejough/imptest/impgen run.IntOps --name IntOpsImp
//go:generate go run github.com/toejough/imptest/impgen run.PrintSum --name PrintSumImp --call

func Test_PrintSum(t *testing.T) {
    t.Parallel()

    // Create the generated mock
    imp := NewIntOpsImp(t)

    // Start the function under test
    printSumImp := NewPrintSumImp(t, run.PrintSum).Start(10, 32, imp.Mock)

    // Expect calls in order, inject responses
    imp.ExpectCallIs.Add().ExpectArgsAre(10, 32).InjectResult(42)
    imp.ExpectCallIs.Format().ExpectArgsAre(42).InjectResult("42")
    imp.ExpectCallIs.Print().ExpectArgsAre("42").Resolve()

    // Validate return values
    printSumImp.ExpectReturnedValuesAre(10, 32, "42")
}
```

**What just happened?**
1. `//go:generate` directives created type-safe mocks from interfaces
2. The test controls the mock interactively—each `Expect*` call waits for the actual call
3. Results are injected on-demand, simulating any behavior you want
4. Return values and panics are validated synchronously

## Flexible Matching with Gomega

Use [gomega](https://github.com/onsi/gomega) matchers for flexible assertions:

```go
import . "github.com/onsi/gomega"
import "github.com/toejough/imptest/imptest"

func Test_PrintSum_Flexible(t *testing.T) {
    t.Parallel()

    imp := NewIntOpsImp(t)
    printSumImp := NewPrintSumImp(t, run.PrintSum).Start(10, 32, imp.Mock)

    // Flexible matching with gomega
    imp.ExpectCallIs.Add().ExpectArgsShould(
        BeNumerically(">", 0),
        BeNumerically(">", 0),
    ).InjectResult(42)

    imp.ExpectCallIs.Format().ExpectArgsShould(imptest.Any()).InjectResult("42")
    imp.ExpectCallIs.Print().InjectResult() // Don't care about args

    printSumImp.ExpectReturnedValuesShould(
        Equal(10),
        Equal(32),
        ContainSubstring("4"),
    )
}
```

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Interface Mocks** | Generate type-safe mocks from any interface with `//go:generate go run .../impgen <package.Interface> --name <Name>Imp` |
| **Callable Wrappers** | Wrap functions to validate returns/panics with the `--call` flag: `//go:generate go run .../impgen <package.Function> --name <Name>Imp --call` |
| **Two-Step Matching** | Match methods first (`ExpectCallIs.Method()`), then arguments (`ExpectArgsAre()` for exact, `ExpectArgsShould()` for matchers) |
| **Type Safety** | `ExpectArgsAre(int, int)` is compile-time checked; `ExpectArgsShould(any, any)` accepts matchers |
| **Concurrent Support** | Use `Within(timeout)` to handle out-of-order calls: `imp.Within(time.Second).ExpectCallIs.Add().ExpectArgsAre(1, 2)` |
| **Matcher Compatibility** | Works with any gomega matcher via duck typing—implement `Match(any) (bool, error)` and `FailureMessage(any) string` |

## Examples

### Handling Concurrent Calls

```go
func Test_Concurrent(t *testing.T) {
    imp := NewCalculatorImp(t)

    go func() { imp.Mock.Add(1, 2) }()
    go func() { imp.Mock.Add(5, 6) }()

    // Match specific calls out-of-order within timeout
    imp.Within(time.Second).ExpectCallIs.Add().ExpectArgsAre(5, 6).InjectResult(11)
    imp.Within(time.Second).ExpectCallIs.Add().ExpectArgsAre(1, 2).InjectResult(3)
}
```

### Expecting Panics

```go
func Test_PrintSum_Panic(t *testing.T) {
    imp := NewIntOpsImp(t)
    printSumImp := NewPrintSumImp(t, run.PrintSum).Start(10, 32, imp.Mock)

    // Inject a panic
    imp.ExpectCallIs.Add().ExpectArgsAre(10, 32).InjectPanic("math overflow")

    // Expect the function to panic with matching value
    printSumImp.ExpectPanicWith(ContainSubstring("overflow"))
}
```

### Manual Control

For maximum control, use `GetCurrentCall()` to manually inspect and resolve calls:

```go
func Test_Manual(t *testing.T) {
    imp := NewIntOpsImp(t)

    go func() { imp.Mock.Add(1, 2) }()

    call := imp.GetCurrentCall()
    if call.Name() != "Add" {
        t.Fatalf("expected Add, got %s", call.Name())
    }

    addCall := call.AsAdd()
    addCall.InjectResult(addCall.a + addCall.b)
}
```

## Installation

```bash
go get github.com/toejough/imptest
```

Then add `//go:generate` directives to your test files and run `go generate`:

```bash
go generate ./...
```

## Learn More

- **API Reference**: [pkg.go.dev/github.com/toejough/imptest](https://pkg.go.dev/github.com/toejough/imptest)
- **More Examples**: See the UAT directory for comprehensive examples
- **How It Works**: imptest generates mocks that communicate via channels, enabling synchronous test control of asynchronous function behavior

## Why imptest?

**Traditional mocking libraries** require you to:
- Write mock implementations by hand, or
- Configure complex expectations upfront, then run the code

**imptest** lets you:
- Generate mocks automatically from interfaces
- Control mocks interactively—inject responses as calls happen
- Choose type-safe exact matching OR flexible gomega-style matchers
- Test concurrent behavior with timeout-based call matching

**Zero manual mocking. Full control.**
