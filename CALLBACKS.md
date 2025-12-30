# Testing Functions with Callback Parameters

This guide explains how to test code that passes callbacks to mocked dependencies using the imptest V2 API.

## Overview

When code under test passes a callback function to a mocked dependency, you can:
1. Extract the callback from the mock call arguments
2. Invoke the callback with test data
3. Verify the callback's behavior (return values, panics, side effects)

This pattern is demonstrated in UAT 15 (`UAT/15-callback-visitor`).

## Basic Pattern

### 1. Setup: Create Mock and Wrapper

```go
// Create mock for the dependency that receives callbacks
mock := MockTreeWalker(t)

// Wrap the function under test (optional, for return value verification)
wrapper := WrapCountFiles(t, visitor.CountFiles)

// Start the function under test
wrapper.Start(mock.Interface(), "/test")
```

### 2. Intercept: Wait for Call with Callback

```go
// Wait for the dependency call
// Use Eventually() for goroutine synchronization
// Use imptest.Any() to match any callback function
call := mock.Walk.Eventually(time.Second).ExpectCalledWithMatches("/test", imptest.Any())
```

### 3. Extract: Get the Callback from Arguments

When using `Eventually()`, you get a base `DependencyCall`, so use `RawArgs()`:

```go
rawArgs := call.RawArgs()
callback := rawArgs[1].(func(string, fs.DirEntry, error) error)
```

Without `Eventually()`, you can use type-safe `GetArgs()`:

```go
args := call.GetArgs()
callback := args.Fn  // Type-safe access
```

### 4. Invoke: Call the Callback with Test Data

```go
// Direct invocation
err := callback("/test/file.txt", mockDirEntry{name: "file.txt"}, nil)

// Verify return value
if err != nil {
    t.Errorf("Expected nil, got %v", err)
}
```

### 5. Complete: Let the Mock Return

```go
// Inject return value to complete the mock call
call.InjectReturnValues(nil)

// Optionally verify the function under test's return value
wrapper.ExpectReturnsEqual(1, nil)
```

## Common Patterns

### Testing Callback Invocation

**Example:** Verify a callback is invoked and returns expected values

```go
mock := MockTreeWalker(t)
wrapper := WrapCountFiles(t, visitor.CountFiles)
wrapper.Start(mock.Interface(), "/test")

call := mock.Walk.Eventually(time.Second).ExpectCalledWithMatches("/test", imptest.Any())

rawArgs := call.RawArgs()
callback := rawArgs[1].(func(string, fs.DirEntry, error) error)

// Invoke callback multiple times with different test data
if err := callback("/test/a.txt", mockDirEntry{name: "a.txt", isDir: false}, nil); err != nil {
    t.Errorf("Expected nil for a.txt, got %v", err)
}
if err := callback("/test/b.txt", mockDirEntry{name: "b.txt", isDir: false}, nil); err != nil {
    t.Errorf("Expected nil for b.txt, got %v", err)
}

call.InjectReturnValues(nil)
wrapper.ExpectReturnsEqual(2, nil)
```

### Testing Callback Panics

**Example:** Verify a callback panics with the expected value

```go
mock := MockTreeWalker(t)

go func() {
    _ = mock.Interface().Walk("/test", func(_ string, _ fs.DirEntry, _ error) error {
        panic("test panic")
    })
}()

call := mock.Walk.Eventually(time.Second).ExpectCalledWithMatches("/test", imptest.Any())

rawArgs := call.RawArgs()
callback := rawArgs[1].(func(string, fs.DirEntry, error) error)

// Invoke callback and catch panic
var panicValue any
func() {
    defer func() {
        panicValue = recover()
    }()
    _ = callback("/test/file.txt", mockDirEntry{name: "file.txt"}, nil)
}()

if panicValue != "test panic" {
    t.Errorf("Expected panic with 'test panic', got %v", panicValue)
}

call.InjectReturnValues(nil)
```

### Testing Named Function Types

**Example:** Named function types work the same as inline function types

```go
// Given: type WalkFunc func(path string, d fs.DirEntry, err error) error

mock := MockTreeWalker(t)

go func() {
    _ = mock.Interface().WalkWithNamedType("/data", func(_ string, _ fs.DirEntry, _ error) error {
        return nil
    })
}()

call := mock.WalkWithNamedType.Eventually(time.Second).ExpectCalledWithMatches("/data", imptest.Any())

rawArgs := call.RawArgs()
callback := rawArgs[1].(visitor.WalkFunc)  // Cast to named type

if err := callback("/data/file.txt", mockDirEntry{name: "file.txt"}, nil); err != nil {
    t.Errorf("Expected nil, got %v", err)
}

call.InjectReturnValues(nil)
```

## Mixing Exact Values and Matchers

`ExpectCalledWithMatches()` accepts both exact values and matchers:

```go
// "/test" is matched with DeepEqual, imptest.Any() matches any callback
call := mock.Walk.ExpectCalledWithMatches("/test", imptest.Any())

// You can also use exact callback matching (rarely useful)
call := mock.Walk.ExpectCalledWithMatches("/test", specificCallback)
```

## Eventually() vs Direct Matching

### Use `Eventually(duration)` when:
- The call happens in a goroutine
- Order of calls is non-deterministic
- You need a timeout for concurrent code

```go
call := mock.Walk.Eventually(time.Second).ExpectCalledWithMatches("/test", imptest.Any())
rawArgs := call.RawArgs()  // Must use RawArgs() with Eventually()
```

### Use direct matching when:
- Calls happen synchronously
- Order is deterministic
- No concurrency involved

```go
call := mock.Walk.ExpectCalledWithMatches("/test", imptest.Any())
args := call.GetArgs()  // Can use type-safe GetArgs()
```

## Complete Example

See `UAT/15-callback-visitor/visitor_test.go` for complete working examples:
- `TestCallbackMatcherSupport`: Basic callback extraction and invocation
- `TestCallbackPanicSupport`: Testing callbacks that panic
- `TestCountFiles`: Multiple callback invocations
- `TestWalkWithNamedType`: Named function types

## Key Points

1. **Extract callbacks** from mock call arguments using `RawArgs()` or `GetArgs()`
2. **Invoke directly** - no special wrapper needed (though wrappers can be generated for additional verification)
3. **Use `imptest.Any()`** to match callback parameters without comparing function pointers
4. **Mix exact values and matchers** in `ExpectCalledWithMatches()`
5. **Handle panics** with `defer recover()` when testing panic behavior
6. **Use `Eventually()`** for goroutine synchronization, then `RawArgs()` to extract callbacks

## Future Improvements

The V2 generator currently has a minor issue with generating wrappers for named function types (creates unused imports). Direct invocation works for all callback types as shown in the examples above.
