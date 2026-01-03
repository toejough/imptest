# UAT-42: Call Handle Pattern for Goroutine Lifecycle Management

## Overview

This UAT defines the new call handle pattern for `--target` wrappers that focuses on goroutine lifecycle management rather than call observation/mocking.

## Problem

Current `--target` wrappers are designed for call observation (GetCalls(), call history, parameter tracking), but the actual use case is goroutine lifecycle management:
- Run target method in a goroutine
- Capture return values and panics
- Coordinate with imptest controller for ordering
- NO need for call history or parameter recording

## New Pattern: Call Handles

Instead of `wrapper.Start()` returning the wrapper itself (which can only manage one goroutine), it should return a **unique call handle** for each invocation.

### Old Pattern (Current)

```go
wrapper := WrapAdd(t, add)
wrapper.Start(10, 20).ExpectReturnsEqual(30)
wrapper.Start(30, 40).ExpectReturnsEqual(70)  // Problem: returns SAME wrapper
```

Multiple `Start()` calls return the same wrapper object, making concurrent goroutine management impossible.

### New Pattern (Target)

```go
wrapper := WrapAdd(t, add)
call1 := wrapper.Start(10, 20)  // Returns unique call handle
call2 := wrapper.Start(30, 40)  // Returns different call handle

call1.ExpectReturnsEqual(30)   // Independent verification
call2.ExpectReturnsEqual(70)   // Independent verification
```

Each `Start()` returns a **new call handle** that independently tracks its goroutine's lifecycle.

## Call Handle API

### Required Types

```go
// Wrapper - created by WrapX(t, fn)
type AddWrapper struct { ... }

// Call handle - returned by wrapper.Start(...)
type AddCallHandle struct {
    Returned *AddReturned  // Set when function returns
    Panic    any           // Set when function panics
}

// Return values struct
type AddReturned struct {
    Result0 int
    Result1 string
    ...
}
```

### Required Methods on Call Handle

1. **ExpectReturnsEqual(v0, v1, ...)** - Verify function returned expected values
2. **ExpectReturnsMatch(m0, m1, ...)** - Verify using matchers
3. **ExpectPanicEquals(expected)** - Verify function panicked with exact value
4. **ExpectPanicMatches(matcher)** - Verify panic value using matcher
5. **WaitForResponse()** - Explicitly wait for goroutine to complete (called internally by Expect methods)

## Test Coverage

The test file covers:

1. **Uniqueness**: Each `Start()` returns a different call handle
2. **Concurrent calls**: Multiple goroutines managed independently
3. **Return verification**: ExpectReturnsEqual and ExpectReturnsMatch work
4. **Panic verification**: ExpectPanicEquals and ExpectPanicMatches work
5. **Manual field access**: Can use WaitForResponse() and access Returned/Panic fields directly
6. **Internal waiting**: Expect methods call WaitForResponse internally
7. **Multiple returns**: Works with 0, 1, or many return values
8. **Interleaved execution**: Order of Start() vs finish order doesn't matter

## Expected Test Results (RED Phase)

All tests should **FAIL** because:
- Current implementation: `Start()` returns `*Wrapper` (self), not a unique call handle
- Stub implementation: Returns nil, causing nil pointer dereferences

When implementation is complete:
- `Start()` will create and return a new call handle each time
- Each handle will independently manage its goroutine lifecycle
- All tests will pass

## Files

- `callhandle_test.go` - Comprehensive test suite (14 test cases)
- `README.md` - This documentation
