# Dependency Interface Tests

This directory demonstrates **mocking interface dependencies** with the imptest v2 API.

## Test Coverage

- **Interface method expectations**: Setting up expectations per method
- **Exact argument matching**: Verifying precise arguments to methods
- **Matcher validation**: Using predicates for flexible argument matching
- **Error injection**: `InjectReturnValues()` with error returns
- **Panic injection**: Simulating panics in dependencies
- **Argument access**: Getting actual arguments passed to mocked methods
- **Multiple method calls**: Ordered expectations across different methods
- **Service integration**: Testing services that depend on interfaces

## Key Concepts

### When to Use Interface Dependencies
Use interface dependency mocks when code under test accepts an **interface parameter** or has an interface field. The mock implements the interface, intercepts calls, and lets you control behavior.

### Usage Pattern - Conversational Flow
```go
// Create shared coordinator
imp := imptest.NewImp(t)

// Wrap the target method that uses the dependency
service := &Service{}
target := WrapService(imp, service)

// Create mock for the interface dependency
store := MockDataStore(imp)
service.store = store.Interface()

// Start target execution (runs in goroutine)
result := target.LoadAndProcess.Start(42)

// Interactively verify the call and inject response
call := store.Get.ExpectCalledWithExactly(42)
call.InjectReturnValues("data", nil)

// Verify the target result
result.ExpectReturnsEqual("processed: data", nil)
```

The **conversational flow** pattern:
1. Target method starts execution asynchronously via `.Start()`
2. When the target calls a mock method, the mock blocks waiting for the test
3. Test verifies arguments with `ExpectCalledWithExactly()` and injects response
4. Mock unblocks and target continues execution
5. Test verifies final result with `ExpectReturnsEqual()`

Each interface method (`.Get`, `.Save`, `.Delete`) is exposed as a field on the mock for setting up expectations. The `.Interface()` method returns the interface implementation to pass to code under test.
