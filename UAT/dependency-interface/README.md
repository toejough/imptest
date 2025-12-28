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

### Generated Wrapper Pattern
```go
// TODO: Code generation will create
store := MockDataStore(t)
call := store.Get.ExpectCalledWithExactly(42)
call.InjectReturnValues("data", nil)

// Pass to code under test
service := &Service{store: store.Interface()}
service.LoadAndProcess(42)

// Currently use manual generic wrapper
imp := imptest.NewImp(t)
storeMock := imptest.NewDependencyInterface[DataStore](imp)
call := storeMock.Get.ExpectCalledWithExactly(42)  // .Get will be code-generated
call.InjectReturnValues("data", nil)
service := &Service{store: storeMock.Interface()}
```

The mock provides an `.Interface()` method that returns the interface implementation to pass to code under test.
