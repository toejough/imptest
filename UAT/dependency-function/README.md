# Dependency Function Tests

This directory demonstrates **mocking function dependencies** with the imptest v2 API.

## Test Coverage

- **Exact argument matching**: `ExpectCalledWithExactly()` for precise args
- **Matcher validation**: `ExpectCalledWithMatches()` with custom predicates
- **Return value injection**: `InjectReturnValues()` to control mock behavior
- **Panic injection**: `InjectPanicValue()` to simulate errors
- **Argument access**: `GetArgs()` to verify actual arguments passed
- **Multiple return values**: Handling functions with multiple returns
- **Bool returns**: Mocking validator-style functions

## Key Concepts

### When to Use Function Dependencies
Use function dependency mocks when code under test accepts a **function parameter**. The mock intercepts calls, lets you verify arguments, and inject controlled responses.

### Usage Pattern - Conversational Flow
```go
// Create shared coordinator
imp := imptest.NewImp(t)

// Wrap the target function
target := WrapProcessData(imp, ProcessData)

// Create mock for the dependency
fetcher := MockFetcher(imp)

// Start target execution (runs in goroutine)
result := target.Start(42, fetcher.Func())

// Interactively verify the call and inject response
call := fetcher.ExpectCalledWithExactly(42)
call.InjectReturnValues("data", nil)

// Verify the target result
result.ExpectReturnsEqual("processed: data", nil)
```

The **conversational flow** pattern:
1. Target function starts execution asynchronously via `.Start()`
2. When the target calls the mock, the mock blocks waiting for the test
3. Test verifies arguments with `ExpectCalledWithExactly()` and injects response
4. Mock unblocks and target continues execution
5. Test verifies final result with `ExpectReturnsEqual()`

This channel-based coordination enables testing code as an interactive conversation between target and dependencies.
