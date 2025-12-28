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

### Generated Wrapper Pattern
```go
// TODO: Code generation will create
calc := &BasicCalculator{}
WrapCalculator(t, calc).Add.CallWith(2, 3).ExpectReturnsEqual(5)

// Currently use manual generic wrapper
imp := imptest.NewImp(t)
target := imptest.NewTargetInterface(imp, calc)
// Note: Methods like .Add will be code-generated
```

Each interface method becomes a field on the wrapper, providing type-safe access to wrap specific method calls.
