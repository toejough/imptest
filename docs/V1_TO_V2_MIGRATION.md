# V1 to V2 Migration Guide

## Overview

The V2 generator introduces a clearer, more explicit API that requires you to specify whether you're wrapping a function under test (`--target`) or mocking an interface dependency (`--dependency`). This guide will help you migrate your existing V1 `//go:generate` directives to V2 syntax.

## Breaking Changes

### V1 Syntax No Longer Works

The V1 implicit detection syntax has been removed. Directives like:

```go
//go:generate impgen MyInterface
//go:generate impgen MyFunction
```

Will now fail with helpful error messages directing you to use the V2 API.

### Required Flags

You must now explicitly specify:
- `--target` when wrapping a function under test
- `--dependency` when mocking an interface

## Side-by-Side Comparison

### Mocking an Interface Dependency

**V1 (deprecated):**
```go
//go:generate impgen run.IntOps
```

**V2 (current):**
```go
//go:generate impgen run.IntOps --dependency
```

### Wrapping a Function Under Test

**V1 (deprecated):**
```go
//go:generate impgen run.PrintSum
```

**V2 (current):**
```go
//go:generate impgen run.PrintSum --target
```

### Complete Test Example

**V1 (deprecated):**
```go
package mypackage_test

import (
    "testing"
    "github.com/toejough/imptest/UAT/run"
)

//go:generate impgen run.PrintSum
//go:generate impgen run.IntOps

func Test_PrintSum(t *testing.T) {
    t.Parallel()

    imp := NewIntOpsImp(t)
    printSumImp := NewPrintSumImp(t, run.PrintSum).Start(10, 32, imp.Mock)

    imp.ExpectCallIs.Add().ExpectArgsAre(10, 32).InjectResult(42)
    imp.ExpectCallIs.Format().ExpectArgsAre(42).InjectResult("42")
    imp.ExpectCallIs.Print().ExpectArgsAre("42").Resolve()

    printSumImp.ExpectReturnedValuesAre(10, 32, "42")
}
```

**V2 (current):**
```go
package mypackage_test

import (
    "testing"
    "github.com/toejough/imptest/UAT/run"
)

//go:generate impgen run.PrintSum --target
//go:generate impgen run.IntOps --dependency

func Test_PrintSum(t *testing.T) {
    t.Parallel()

    imp := NewIntOpsImp(t)
    printSumImp := NewPrintSumImp(t, run.PrintSum).Start(10, 32, imp.Mock)

    imp.ExpectCallIs.Add().ExpectArgsAre(10, 32).InjectResult(42)
    imp.ExpectCallIs.Format().ExpectArgsAre(42).InjectResult("42")
    imp.ExpectCallIs.Print().ExpectArgsAre("42").Resolve()

    printSumImp.ExpectReturnedValuesAre(10, 32, "42")
}
```

**Note:** Only the `//go:generate` directives changed. The test code remains identical.

## Migration Steps

1. **Update `//go:generate` directives:**
   - For interfaces: Add `--dependency` flag
   - For functions/callables: Add `--target` flag

2. **Regenerate mocks:**
   ```bash
   go generate ./...
   ```

3. **Verify tests still pass:**
   ```bash
   go test ./...
   ```

## Error Messages

If you forget to add the required flags, you'll see helpful error messages:

```
Error: symbol 'MyInterface' is an interface. Use --dependency flag to generate a mock.
Example: //go:generate impgen MyInterface --dependency
```

```
Error: symbol 'MyFunction' is a function. Use --target flag to generate a wrapper.
Example: //go:generate impgen MyFunction --target
```

## Why the Change?

The V2 API makes your intent explicit:
- **Clarity:** Reading `--target` immediately tells you this is a function under test
- **Clarity:** Reading `--dependency` immediately tells you this is a mocked dependency
- **Safety:** The generator can validate you're using the right flag for the right symbol type
- **Maintainability:** Future readers understand your test structure at a glance

## New Features in V2

Along with the explicit flag syntax, V2 includes:

### Type-Safe GetArgs()

Dependency mocks now have type-safe argument access:

```go
call := calc.Add.Eventually().ExpectCalledWithExactly(10, 20)
args := call.GetArgs()
a := args.A  // Type-safe int ✅ (no type assertion needed)
b := args.B  // Type-safe int ✅
```

### Generic Support

Both targets and dependencies now support Go generics:

```go
//go:generate impgen Calculator --dependency

type Calculator[T Numeric] interface {
    Add(a, b T) T
    Multiply(a, b T) T
}
```

## Need Help?

- **Documentation:** See [README.md](../README.md) for full API reference
- **Examples:** Browse [UAT](../UAT) directory for comprehensive examples
- **Callbacks:** See [CALLBACKS.md](../CALLBACKS.md) for testing callback patterns

## Quick Reference

| Use Case | V1 (deprecated) | V2 (current) |
|----------|-----------------|--------------|
| Mock an interface | `impgen MyInterface` | `impgen MyInterface --dependency` |
| Wrap a function | `impgen MyFunc` | `impgen MyFunc --target` |
| Mock generic interface | Not supported | `impgen MyInterface[T] --dependency` |
| Wrap generic function | Not supported | `impgen MyFunc[T] --target` |
