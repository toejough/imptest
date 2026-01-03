# UAT-33: Struct Type as Target

## Purpose

This UAT tests whether `impgen` can wrap entire struct types with the `--target` flag, similar to how UAT-32 wraps interface types.

## Current Status

**FAILING** - Feature not implemented

Running `go generate` produces:
```
Error: symbol (interface or function) not found: Calculator in package github.com/toejough/imptest/UAT/33-struct-as-target
```

## Expected Behavior

When the feature is implemented:

1. `impgen calculator.Calculator --target` should generate `WrapCalculator` function
2. `WrapCalculator(t, calc)` should return a wrapper with interceptors for ALL methods on the struct
3. Each method should have its own interceptor (Add, Multiply, Divide, Process)
4. Calls through the wrapper should be intercepted and recorded
5. The wrapper should provide access to a wrapped instance via `Interface()` method

## Comparison with Related UATs

- **UAT-02 (Method as Target)**: Wraps individual methods one at a time
  - `impgen Calculator.Add --target` → generates `WrapCalculatorAdd`
  - Each method wrapped separately with individual directives

- **UAT-32 (Interface as Target)**: Wraps interface types
  - `impgen Logger --target` → generates `WrapLogger`
  - All interface methods wrapped together

- **UAT-33 (Struct as Target)**: Should wrap struct types (NOT YET IMPLEMENTED)
  - `impgen Calculator --target` → should generate `WrapCalculator`
  - All struct methods wrapped together (like interfaces)

## Test Structure

The test file includes:

1. **TestWrapCalculator_BasicWrapping**: Basic struct wrapping with single method call
2. **TestWrapCalculator_MultipleMethodCalls**: Intercepting multiple different methods
3. **TestWrapCalculator_MultipleReturnValues**: Methods with multiple return values
4. **TestWrapCalculator_ErrorHandling**: Methods that return errors
5. **TestWrapCalculator_RepeatedCalls**: Multiple calls to same method
6. **TestWrapCalculator_StatePreservation**: Wrapped struct maintains internal state
7. **TestWrapCalculator_MethodInteraction**: Method that calls other methods
8. **TestWrapCalculator_Interface**: Using wrapped struct in place of original

## Implementation Notes

When implementing this feature, the wrapper should:

- Generate a wrapper struct with interceptors for each public method
- Provide an `Interface()` method that returns a usable struct instance
- Route method calls through interceptors while maintaining struct state
- Record all calls with parameters and return values
- Support methods with various signatures (multiple params, multiple returns, errors)

## Files

- `calculator.go`: Defines Calculator struct with multiple methods
- `structtarget_test.go`: Comprehensive tests for struct wrapping
- `README.md`: This file
