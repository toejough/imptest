# UAT-30: Struct Literal Parameters and Returns

## Status: FAILING - Bug Discovered

## Overview
This UAT tests whether impgen correctly handles struct literal (anonymous struct) types in parameters and return values. Struct literals are a common Go pattern for configuration options and API responses where creating a full named type is unnecessary overhead.

## Test Scenarios

### Interface with Struct Literals (DataProcessor)
1. **Single-field struct literal parameter**: `Process(cfg struct{ Timeout int }) error`
2. **Multi-field struct literal parameter**: `Transform(opts struct{ Debug bool; Level int }) (string, error)`
3. **Struct literal return type**: `GetConfig() struct{ Host string; Port int }`
4. **Struct literal in both**: `Apply(req struct{ Method string }) struct{ Status int }`

### Functions/Methods with Struct Literals
1. **Standalone function**: `ValidateRequest(req struct{ APIKey string; Timeout int }) error`
2. **Method with struct literal return**: `ConfigManager.Load(path string) struct{ Host string; Port int; TLS bool }`
3. **Function with struct literal return**: `GetDefaults() struct{ MaxRetries int; Timeout int }`

## Execution Results (Steps 1-5)

### Step 1: Directory Structure - SUCCESS
Created `/Users/joe/repos/personal/imptest/UAT/30-struct-literal-params/` with:
- `structlit.go` - interface and function definitions
- `structlit_test.go` - test file with generate directives

### Step 2: Test Interface Definition - SUCCESS
Defined `DataProcessor` interface with comprehensive struct literal scenarios covering all combinations of parameters and returns.

### Step 3: Test Functions/Methods - SUCCESS
Added standalone function, struct method, and function with struct literal returns.

### Step 4: Interface Mock Generation - PARTIAL SUCCESS
**Generate directive**: `//go:generate ../../bin/impgen DataProcessor --dependency`

**Result**: Generation succeeded but produced INCORRECT code.

**Bug discovered**: Struct literal field definitions are stripped.

**Expected**:
```go
func (impl *mockDataProcessorImpl) Process(cfg struct{ Timeout int }) error
```

**Generated**:
```go
func (impl *mockDataProcessorImpl) Process(cfg struct{}) error
```

All struct literals become empty `struct{}` - field definitions are lost.

### Step 5: Function Target Wrapping - PARTIAL SUCCESS
**Generate directives**:
- `//go:generate ../../bin/impgen ValidateRequest --target`
- `//go:generate ../../bin/impgen ConfigManager.Load --target`
- `//go:generate ../../bin/impgen GetDefaults --target`

**Results**: All generations succeeded but produced INCORRECT code.

**Bugs discovered**:
1. Same field-stripping bug affects target wrappers
2. Method naming issue: `ConfigManager.Load` generated `WrapLoad` instead of `WrapConfigManagerLoad`

## Build/Test Results

### Compilation
```bash
$ go build ./...
# SUCCESS - but only because empty struct{} types are compatible
```

The code compiles because Go allows assignment between any empty struct types. However, this masks the underlying bug.

### Test Execution
```bash
$ go test -v
FAIL - Multiple compilation errors:

1. Type mismatch in Interface() return:
   have Apply(struct{}) struct{}
   want Apply(struct{Method string}) struct{Status int}

2. Function type mismatch for wrappers:
   cannot use func(req struct{APIKey string; Timeout int}) error
   as func(struct{}) error

3. Missing method: WrapConfigManagerLoad undefined
   (generated WrapLoad instead)

4. Missing method: ExpectReturnsMatches on WrapGetDefaultsWrapper
   (likely related to empty struct issue)
```

## Root Cause Analysis

**Primary Bug**: impgen's type parsing logic strips field definitions from struct literals, converting all anonymous structs to empty `struct{}`.

**Impact**:
- Mock interfaces don't implement the actual interface (signature mismatch)
- Target wrappers can't accept the real function (signature mismatch)
- Generated code compiles in isolation but fails when integrated
- Type safety is completely lost for struct literal parameters/returns

**Secondary Bug**: Method wrapper naming doesn't include the receiver type, causing naming collisions for common method names like `Load`.

## Expected Behavior

Impgen should preserve the complete struct literal type definition:

```go
// Source interface
Process(cfg struct{ Timeout int }) error

// Generated mock should match exactly
func (impl *mockDataProcessorImpl) Process(cfg struct{ Timeout int }) error
```

## Files Generated (with bugs)
- `generated_MockDataProcessor_test.go` - All methods have empty struct{} instead of fielded structs
- `generated_WrapValidateRequest_test.go` - Empty struct{} instead of {APIKey string; Timeout int}
- `generated_WrapLoad_test.go` - Empty struct{} instead of {Host string; Port int; TLS bool}, wrong name
- `generated_WrapGetDefaults_test.go` - Empty struct{} instead of {MaxRetries int; Timeout int}

## Recommendation

This is a **critical bug** that completely breaks struct literal support. Two fixes needed:

1. **Type preservation**: Update impgen's AST parsing to preserve field definitions from struct literal types
2. **Method naming**: Include receiver type in wrapper names: `WrapConfigManagerLoad` not `WrapLoad`

Without these fixes, any interface or function using struct literals cannot be properly mocked or wrapped.

## Related UATs
- UAT-24: Function literal parameters (PASSING - different type pattern)
- UAT-25: Interface literal parameters (status unknown)
- This demonstrates that Go's various "literal" type patterns have different parsing requirements
