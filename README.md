# imptest

<p align="center">
  <img src="docs/imptest-imp.png" alt="imptest logo - a purple Go gopher imp" width="200">
</p>

**Zero manual mocking. Full control.**

## What is imptest?

**Test impure functions without writing mock implementations.**

imptest generates type-safe mocks from your interfaces. Each test interactively controls the mock—expect calls, inject
responses, and validate behavior—all with compile-time safety or flexible matchers. No manual mock code. No complex
setup. Just point at your functions and dependencies and test.

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

    "github.com/toejough/imptest"
    "github.com/toejough/imptest/UAT/run"
)

// create mock for your dependency interface
//go:generate impgen run.IntOps --dependency

// create wrapper for your function under test
//go:generate impgen run.PrintSum --target

func Test_PrintSum(t *testing.T) {
    t.Parallel()

    // Create the dependency mock (returns mock and expectation handle)
    mock, mockImp := MockIntOps(t)

    // Start the function under test
    wrapper := WrapPrintSum(t, run.PrintSum).Start(10, 32, mock)

    // Expect calls in order, inject responses
    mockImp.Add.Expect(10, 32).Return(42)
    mockImp.Format.Expect(42).Return("42")
    mockImp.Print.Expect("42").Return()

    // Validate return values
    wrapper.ExpectReturn(10, 32, "42")
}
```

**What just happened?**

1. A `//go:generate` directive created a type-safe mock (`MockIntOps`) from the interface, providing interactive control
   over dependency behavior
1. A `//go:generate` directive created a type-safe wrapper (`WrapPrintSum`) for the function under test, enabling return
   value and panic validation
1. The test controls the dependency interactively—each `Expect` call waits for the actual call
1. Results are injected on-demand with `Return`, simulating any behavior you want
1. Return values are validated with `ExpectReturn`

## Flexible Matching with Gomega

Use [gomega](https://github.com/onsi/gomega)-style matchers for flexible assertions:

```go
import . "github.com/onsi/gomega"
import "github.com/toejough/imptest"

func Test_PrintSum_Flexible(t *testing.T) {
    t.Parallel()

    mock, mockImp := MockIntOps(t)
    wrapper := WrapPrintSum(t, run.PrintSum).Start(10, 32, mock)

    // Flexible matching with gomega-style matchers
    mockImp.Add.Match(
        BeNumerically(">", 0),
        BeNumerically(">", 0),
    ).Return(42)

    mockImp.Format.Match(imptest.Any).Return("42")
    mockImp.Print.Match(imptest.Any).Return()

    wrapper.ExpectReturnMatch(
        Equal(10),
        Equal(32),
        ContainSubstring("4"),
    )
}
```

## Key Concepts

| Concept                   | Description                                                                                                               |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Interface Mocks**       | Generate type-safe mocks from any interface with `//go:generate impgen <package.Interface> --dependency`                  |
| **Callable Wrappers**     | Wrap functions to validate returns/panics with `//go:generate impgen <package.Function> --target`                         |
| **Two-Return Pattern**    | Mocks return `(mock, imp)`: `mock` is the interface, `imp` holds method expectations                                      |
| **Two-Step Matching**     | Access methods via `imp.X`, then specify matching mode (`Expect()` or `Match()`)       |
| **Type Safety**           | `Expect(int, int)` is compile-time checked; `Match(matcher, matcher)` accepts matchers |
| **Concurrent Support**    | Use `.Eventually` for async expectations, then `imptest.Wait(t)` to block until satisfied                                 |
| **Matcher Compatibility** | Works with any gomega-style matcher via duck typing—implement `Match(any) (bool, error)` and `FailureMessage(any) string` |

## Examples

### Handling Concurrent Calls

```go
func Test_Concurrent(t *testing.T) {
    mock, imp := MockCalculator(t)

    go func() { mock.Add(1, 2) }()
    go func() { mock.Add(5, 6) }()

    // Register async expectations (non-blocking)
    imp.Add.Eventually.Expect(5, 6).Return(11)
    imp.Add.Eventually.Expect(1, 2).Return(3)

    // Wait for all expectations to be satisfied
    imptest.Wait(t)
}
```

### Expecting Panics

```go
func Test_PrintSum_Panic(t *testing.T) {
    mock, mockImp := MockIntOps(t)
    wrapper := WrapPrintSum(t, run.PrintSum).Start(10, 32, mock)

    // Inject a panic
    mockImp.Add.Expect(10, 32).Panic("math overflow")

    // Expect the function to panic with matching value
    wrapper.ExpectPanicMatch(ContainSubstring("overflow"))
}
```

### Manual Control

For maximum control, use type-safe `GetArgs()` or raw `RawArgs()` to manually inspect arguments:

```go
func Test_Manual(t *testing.T) {
    mock, imp := MockCalculator(t)

    go func() { mock.Add(1, 2) }()

    call := imp.Add.Expect(1, 2)

    // Access typed arguments
    args := call.GetArgs()
    result := args.A + args.B

    call.Return(result)
}
```

## Testing Callbacks

When your code passes callback functions to mocked dependencies, imptest makes it easy to extract and test those callbacks:

```go
// Create mock for dependency that receives callbacks
mock, imp := MockTreeWalker(t)

// Wait for the call with a callback parameter (use imptest.Any to match any function)
call := imp.Walk.Eventually.Match("/test", imptest.Any)

// Extract the callback from the arguments (blocks until call arrives and matches)
rawArgs := call.GetMatchedArgs()
callback := rawArgs[1].(func(string, fs.DirEntry, error) error)

// Invoke the callback with test data
err := callback("/test/file.txt", mockDirEntry{name: "file.txt"}, nil)

// Verify callback behavior and complete the mock call
call.Return(nil)
```

## Channel Patterns

When your code communicates via channels, imptest gives you full control. The key insight: channels are just values—you can inject them as return values or access them from arguments.

### Returning a Test-Controlled Channel

When a dependency returns a channel, inject one you control:

```go
// Interface: type EventSource interface { Events() <-chan Event }

func Test_ChannelReturn(t *testing.T) {
    mock, mockImp := MockEventSource(t)
    wrapper := WrapProcessEvents(t, ProcessEvents).Start(mock)

    // Create a channel the test controls
    eventChan := make(chan Event)

    // Inject it as the return value
    mockImp.Events.ExpectCalled().Return(eventChan)

    // Send events when you want
    eventChan <- Event{Type: "start"}
    eventChan <- Event{Type: "data", Payload: "hello"}
    close(eventChan) // Signal completion

    wrapper.ExpectReturn(2, nil) // Processed 2 events
}
```

### Accessing Channel Arguments

When the function under test passes a channel to a dependency, access it via `GetArgs()`:

```go
// Interface: type Worker interface { StartJob(id int, results chan<- Result) error }

func Test_ChannelArg(t *testing.T) {
    mock, imp := MockWorker(t)

    go func() {
        results := make(chan Result, 1)
        mock.StartJob(42, results)
        // Function blocks waiting for result
        r := <-results
        fmt.Println(r.Status)
    }()

    // Capture the call and access the channel argument
    call := imp.StartJob.Match(Equal(42), imptest.Any)
    resultsChan := call.GetArgs().Results

    // Send a result on the captured channel
    resultsChan <- Result{Status: "done"}

    call.Return(nil)
}
```

### Bidirectional Channel Communication

For request/response patterns over channels:

```go
// Interface: type RPC interface { Call(req <-chan Request, resp chan<- Response) }

func Test_Bidirectional(t *testing.T) {
    mock, imp := MockRPC(t)

    // Channels the function under test will create
    go func() {
        reqChan := make(chan Request)
        respChan := make(chan Response)
        go mock.Call(reqChan, respChan)
        reqChan <- Request{ID: 1, Data: "ping"}
        resp := <-respChan
        // ... use resp
    }()

    call := imp.Call.Eventually.ExpectCalled()

    // Access both channels from args
    args := call.GetArgs()

    // Read from request channel, write to response channel
    req := <-args.Req
    args.Resp <- Response{ID: req.ID, Data: "pong"}

    call.Return()
    imptest.Wait(t)
}
```

The pattern is consistent: channels are values. Inject them as returns, access them from args, then send/receive as your test requires.

## Installation

Install the library with:

```bash
go get github.com/toejough/imptest
```

Install the code generator tool:

```bash
go install github.com/toejough/imptest/impgen@latest
```

Then add `//go:generate impgen <interface|callable> --dependency` (for interfaces) or `//go:generate impgen <callable> --target` (for functions) directives to your test files and run `go generate`:

```bash
go generate ./...
```

## Learn More

- **Capability Reference**: [TAXONOMY.md](./docs/TAXONOMY.md) - comprehensive matrix of what imptest can and cannot do, with examples and workarounds
- **API Reference**: [pkg.go.dev/github.com/toejough/imptest](https://pkg.go.dev/github.com/toejough/imptest)
- **More Examples**: See the [UAT/core](https://github.com/toejough/imptest/tree/main/UAT/core) for basic patterns and [UAT/variations](https://github.com/toejough/imptest/tree/main/UAT/variations) for edge cases
- **How It Works**: imptest generates mocks that communicate via channels, enabling interactive test control of even asynchronous function behavior

## Why imptest?

**Traditional mocking libraries** require you to:

- Write mock implementations by hand, or
- Configure complex expectations upfront, then run the code

**imptest** lets you:

- Generate mocks automatically from interfaces
- Control mocks interactively—inject responses as calls happen
- Choose type-safe exact matching OR flexible gomega-style matchers
- Test concurrent behavior with timeout-based call matching

### Comparison Example

Let's test a function that processes user data by calling an external service. Here's how different testing approaches compare:

**The Function Under Test:**

```go
func ProcessUser(svc ExternalService, userID int) (string, error) {
    data, err := svc.FetchData(userID)
    if err != nil {
        return "", err
    }
    return svc.Process(data), nil
}
```

### Approach 1: Basic Go Testing

```go
func TestProcessUser_Basic(t *testing.T) {
    // ❌ Problem: Must write a complete mock implementation by hand
    mock := &MockService{
        fetchResult: "test data",
        processResult: "processed",
    }

    result, err := ProcessUser(mock, 42)

    // ❌ Problem: Manual assertions, verbose error messages
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if result != "processed" {
        t.Fatalf("expected 'processed', got '%s'", result)
    }
    // ❌ Problem: Can't verify FetchData was called with correct args
}
```

### Approach 2: Using others

```go

func TestProcessUser_Other(t *testing.T) {
    // ❌ Still need to write mock implementation
    mock := &MockService{
        fetchResult: "test data",
        processResult: "processed",
    }

    result, err := ProcessUser(mock, 42)

    // ✅ Better: Cleaner assertions
    assert.NoError(t, err)
    assert.Equal(t, "processed", result)

    // ❌ Problem: can't control behavior per call interactively
}
```

### Approach 3: Using imptest

**For testing with dependencies:**

```go
//go:generate impgen ExternalService --dependency
//go:generate impgen ProcessUser --target

func TestProcessUser_Imptest(t *testing.T) {
    t.Parallel()

    // ✅ Generated mock, no manual implementation
    mock, mockImp := MockExternalService(t)

    // ✅ Wrap function for return value validation
    wrapper := WrapProcessUser(t, ProcessUser).Start(mock, 42)

    // ✅ Interactive control: expect calls and inject responses
    mockImp.FetchData.Expect(42).Return("test data", nil)
    mockImp.Process.Expect("test data").Return("processed")

    // ✅ Validate return values (can use gomega matchers too!)
    wrapper.ExpectReturn("processed", nil)
}
```

**For simple return value assertions (without dependencies):**

```go
// generate the wrapper for the Add function
//go:generate impgen Add --target

func Add(a, b int) int {
    return a + b
}

func TestAdd_Simple(t *testing.T) {
    t.Parallel()

    // ✅ Wrap function and validate returns in one line
    // ✅ Args are type-safe and checked at compile time - your IDE can autocomplete them or inform you of mismatches!
    // ✅ Panics are caught cleanly and reported in failure messages
    WrapAdd(t, Add).Start(2, 3).ExpectReturn(5)
}
```

**Key Differences:**

| Feature                  | Basic Go     | others          | imptest          |
| ------------------------ | ------------ | --------------- | ---------------- |
| **Clean Assertions**     | ❌ Verbose   | ✅ Yes          | ✅ Yes           |
| **Auto-Generated Mocks** | ❌ No        | ✅ Yes          | ✅ Yes           |
| **Verify Call Order**    | ❌ Manual    | ❌ Complex      | ✅ Easy          |
| **Verify Call Args**     | ❌ Manual    | ⚠️ Per function | ✅ Per call      |
| **Interactive Control**  | ❌ Difficult | ❌ Difficult    | ✅ Easy          |
| **Concurrent Testing**   | ❌ Difficult | ⚠️ Possible     | ✅ Easy          |
| **Return Validation**    | ❌ Manual    | ✅ Yes          | ✅ Yes           |
| **Panic Validation**     | ❌ Manual    | ❌ Manual       | ✅ Yes/Automatic |

**Zero manual mocking. Full control.**
