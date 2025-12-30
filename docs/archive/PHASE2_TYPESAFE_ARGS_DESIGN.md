# Phase 2: Type-Safe GetArgs() - Design Document

## Status: ✅ COMPLETED AND SHIPPED

This feature has been fully implemented and shipped as part of the V2 generator.
All dependency mocks now include type-safe `GetArgs()` methods.

### Overview
This document describes the design for adding type-safe `GetArgs()` methods to v2 dependency mocks, eliminating the need for type assertions when accessing method arguments.

### Current State (Problem)
```go
// Current v2 dependency API - requires type assertions
call := calc.Add.Eventually(time.Second).ExpectCalledWithExactly(10, 20)
args := call.GetArgs()
a := args.A1.(int)  // Type assertion required ❌
b := args.A2.(int)  // Type assertion required ❌
```

### Target State (Solution)
```go
// New typed API - no type assertions
call := calc.Add.Eventually(time.Second).ExpectCalledWithExactly(10, 20)
args := call.GetArgs()
a := args.A  // Type-safe int ✅
b := args.B  // Type-safe int ✅
```

### Implementation Approach

#### 1. Foundation (✅ Complete)
Added `RawArgs()` method to `DependencyCall` in `imptest/dependency.go`:
```go
func (dc *DependencyCall) RawArgs() []any {
    return dc.call.Args
}
```

#### 2. Generated Code Structure (Proof of Concept Complete)

For each method in the interface, generate:

**A. Args Struct**
```go
type CalculatorAddArgs struct {
    A int
    B int
}
```

**B. Call Wrapper**
```go
type CalculatorAddCall struct {
    *imptest.DependencyCall
}

func (c *CalculatorAddCall) GetArgs() CalculatorAddArgs {
    raw := c.RawArgs()
    return CalculatorAddArgs{
        A: raw[0].(int),
        B: raw[1].(int),
    }
}
```

**C. Method Wrapper**
```go
type CalculatorAddMethod struct {
    *imptest.DependencyMethod
}

func (m *CalculatorAddMethod) Eventually(d time.Duration) *CalculatorAddMethod {
    return &CalculatorAddMethod{
        DependencyMethod: m.DependencyMethod.Eventually(d),
    }
}

func (m *CalculatorAddMethod) ExpectCalledWithExactly(a, b int) *CalculatorAddCall {
    call := m.DependencyMethod.ExpectCalledWithExactly(a, b)
    return &CalculatorAddCall{DependencyCall: call}
}

func (m *CalculatorAddMethod) ExpectCalledWithMatches(matchers ...any) *CalculatorAddCall {
    call := m.DependencyMethod.ExpectCalledWithMatches(matchers...)
    return &CalculatorAddCall{DependencyCall: call}
}
```

**D. Updated Mock Struct**
```go
type CalculatorMock struct {
    imp      *imptest.Imp
    Add      *CalculatorAddMethod     // Typed wrapper
    Multiply *CalculatorMultiplyMethod
    Store    *CalculatorStoreMethod
}

func MockCalculator(t imptest.TestReporter) *CalculatorMock {
    imp := imptest.NewImp(t)
    return &CalculatorMock{
        imp:      imp,
        Add:      &CalculatorAddMethod{
            DependencyMethod: imptest.NewDependencyMethod(imp, "Add"),
        },
        // ... other methods
    }
}
```

### Testing

#### Manual Test (✅ Passes)
`UAT/17-typesafe-getargs/manual_test.go` demonstrates the concept works:
```bash
$ cd UAT/17-typesafe-getargs && go test -v -run TestManualTypeSafeGetArgs
=== RUN   TestManualTypeSafeGetArgs
--- PASS: TestManualTypeSafeGetArgs (0.00s)
PASS
```

### Generator Implementation Requirements

To complete the implementation, the v2 dependency generator needs to:

1. **Collect parameter information** for each method:
   - Parameter names (or generate names like `A`, `B` for unnamed params)
   - Parameter types
   - Parameter indices

2. **Generate args structs** with template data:
   ```go
   type v2DepMethodTemplateData struct {
       // ... existing fields ...
       ParamFields    []paramField
       ArgsTypeName   string  // e.g., "CalculatorAddArgs"
       CallTypeName   string  // e.g., "CalculatorAddCall"
       MethodTypeName string  // e.g., "CalculatorAddMethod"
       TypedParams    string  // e.g., "a int, b int"
   }
   ```

3. **Create templates** for:
   - Args struct generation
   - Call wrapper generation
   - Method wrapper generation
   - Updated mock struct field types

4. **Update generation flow** in `generateWithTemplates()` to:
   - Collect all method data during impl method generation
   - After impl methods, generate typed wrappers
   - Update mock struct to use typed method fields

### Backward Compatibility

This design maintains backward compatibility:
- Generic `*imptest.DependencyMethod` still works
- Generated wrappers embed the generic types
- Existing tests continue to work

### Benefits

1. **Type Safety**: No type assertions needed for args
2. **IDE Support**: Better autocomplete and type checking
3. **Compile-Time Safety**: Invalid field access caught at compile time
4. **Cleaner Tests**: More readable test code

### Next Steps

1. Implement parameter collection in `generateImplMethodWithTemplate()`
2. Create text templates for args/call/method wrappers
3. Integrate wrapper generation into `generateWithTemplates()`
4. Update UAT 17 to use generated (not manual) code
5. Migrate UAT 15 to new API
6. Run full test suite

### Files Modified

- `imptest/dependency.go` - Added `RawArgs()` method
- `impgen/run/templates.go` - Extended template data structures
- `UAT/17-typesafe-getargs/` - Test cases and proof of concept

### Related Issue

TOE-86: Add callback invocation support to v2 generators
