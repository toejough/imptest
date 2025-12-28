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

### Generated Wrapper Pattern
```go
// TODO: Code generation will create
fetcher := MockFetcher(t)
call := fetcher.ExpectCalledWithExactly(42)
call.InjectReturnValues("data", nil)

// Pass to code under test
ProcessData(42, fetcher.Func())

// Currently use manual generic wrapper
imp := imptest.NewImp(t)
fetcherMock := imptest.NewDependencyFunction[func(int) (string, error)](imp)
call := fetcherMock.ExpectCalledWithExactly(42)
call.InjectReturnValues("data", nil)
ProcessData(42, fetcherMock.Func())
```

The mock provides a `.Func()` method that returns the actual function to pass to code under test.
