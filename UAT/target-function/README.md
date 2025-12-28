# Target Function Tests

This directory demonstrates **wrapping functions under test** with the imptest v2 API.

## Test Coverage

- **Ordered mode** (default): Sequential execution, expects next interaction immediately
- **Unordered mode** (Eventually): Concurrent/async code, waits for matching interactions
- **Exact matching**: `ExpectReturnsEqual()` for precise value verification
- **Matcher validation**: `ExpectReturnsMatch()` with custom predicates
- **Panic verification**: `ExpectPanicEquals()` and `ExpectPanicMatches()`
- **Value access**: `GetReturns()` for custom assertions

## Key Concepts

### When to Use Targets
Use target wrappers when you want to **verify the behavior** of a function under test. The wrapper calls the actual function, captures what happens (return or panic), and lets you assert on the results.

### Usage Pattern
```go
// Wrap the function, call it with args, and verify the result
WrapAdd(t, Add).CallWith(2, 3).ExpectReturnsEqual(5)
```

The generated `WrapAdd` function creates a wrapper that calls `Add(2, 3)`, captures the return value, and verifies it equals 5.
