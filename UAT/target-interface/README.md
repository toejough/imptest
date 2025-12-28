# Target Interface Tests

This directory demonstrates **wrapping interface implementations under test** with the imptest v2 API.

## Test Coverage

- **Interface method calls**: Wrapping specific methods of an interface
- **Ordered mode**: Sequential method calls with immediate expectations
- **Exact matching**: Verifying exact return values from methods
- **Matcher validation**: Using predicates to validate returns
- **Panic verification**: Asserting that methods panic correctly
- **Multiple return values**: Handling methods with multiple returns (including errors)
- **Value access**: Getting actual return values for custom assertions

## Key Concepts

### When to Use Interface Targets
Use interface target wrappers when testing code that implements an interface. This lets you verify that methods are called correctly and return expected values.

### Usage Pattern
```go
// Wrap an interface implementation
calc := &BasicCalculator{}
WrapCalculator(t, calc).Add.Start(2, 3).ExpectReturnsEqual(5)

// Or wrap a struct type directly
WrapBasicCalculator(t, calc).Subtract.Start(10, 3).ExpectReturnsMatch(
    And(BeNumerically(">", 0), BeNumerically("<", 10)),
)
```

Each interface method becomes a field on the wrapper (`.Add`, `.Subtract`, `.Divide`), providing type-safe access to call and verify specific methods. The `.Start()` method runs the method asynchronously in a goroutine, enabling conversational testing patterns where the method can interact with mocks.
