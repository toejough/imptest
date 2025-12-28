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
// Wrap the function, start execution with args, and verify the result
WrapAdd(t, Add).Start(2, 3).ExpectReturnsEqual(5)
```

The generated `WrapAdd` function creates a wrapper that:
1. `.Start(2, 3)` runs `Add(2, 3)` asynchronously in a goroutine and returns immediately
2. `.ExpectReturnsEqual(5)` blocks waiting for the function to complete, then verifies the result equals 5

This channel-based pattern enables conversational testing where target functions can call mocks that block waiting for the test to inject responses.
