# Gemini Code Assistant Context

This document provides context for the Gemini code assistant to understand and effectively contribute to the `imptest` project.

## Project Overview

`imptest` is a Go testing tool designed for "impure functions" – functions whose primary role is to coordinate calls to other functions (dependencies). It helps verify that the function under test interacts with its dependencies correctly, in the right order, and with the right data.

**Core Philosophy:** Zero mock code. Full control.

The library generates type-safe mocks from interfaces. Tests interactively control the mock—expecting calls, injecting responses, and validating behavior—all with compile-time safety or flexible matchers.

## Architecture

### Core Components

1.  **`impgen`** (`impgen/`) - The code generator.
    *   `main.go`: Entry point.
    *   `run/run.go`: Main generation logic.
    *   `run/codegen_interface.go`: Generates interface mocks.
    *   `run/codegen_callable.go`: Generates callable wrappers.
    *   `run/templates.go`: Go templates.
    *   `run/pkgparse.go`: AST parsing.

2.  **`UAT`** (`UAT/`) - User Acceptance Tests.
    *   `run/run.go`: Example interfaces (e.g., `IntOps`).
    *   `run/run_test.go`: Tests demonstrating usage patterns.

### Generation Modes

Controlled via `//go:generate` directives:

```go
// 1. Interface Mock (Default): Generates ExpectCallTo methods
//go:generate go run ../../impgen/main.go run.IntOps --name IntOpsImp

// 2. Callable Wrapper: Wraps functions for return/panic validation
//go:generate go run ../../impgen/main.go run.PrintSum --name PrintSumImp --call
```

## Build and Automation

This project uses [Mage](https://magefile.org/).

### Commands
*   `mage check`: Run **all** checks (tidy, generate, test, lint, coverage, mutation, fuzz, etc.). **Must pass before work is complete.**
*   `mage watch`: Watch for changes and re-run checks.
*   `mage generate`: Run `go generate`.
*   `mage test`: Run unit tests with race detection and coverage.
*   `mage lint`: Run `golangci-lint`.
*   `mage mutate`: Run mutation tests.
*   `mage fuzz`: Run fuzz tests.
*   `mage checknils`: Run `nilaway`.
*   `mage deadcode`: Check for dead code.
*   `mage tidy`: Run `go mod tidy`.
*   `mage installtools`: Install dev tools.

## Coding Standards & Preferences

### Priority Checklist
1.  **Test First:** Identify functionality, write failing tests, then implement.
2.  **Idiomatic Go:** Validate with `gopls` and linters.
3.  **Simplicity:** Favor readability and reuse. Avoid cleverness.
4.  **Separation of Concerns:** Separate orchestration from implementation.
5.  **Performance:** Optimize only if profiled.
6.  **Minimal Tests:** Favor clarity and maintainability over exhaustive coverage.

### Code Organization & Sorting
**Strict Alphabetical Sorting** applies.

**File Layout Order:**
1.  Package comments
2.  Imports
3.  Entry points (`main`, exported code)
4.  Everything else

**Within Sections (Sort Order):**
1.  Constants
2.  Variables
3.  Types
4.  Functions/Methods (Grouped by receiver type, then sorted alphabetically)

*Note: "Sort alphabetically" means by name. Helper functions should be sorted by their name, not placed adjacent to the functions they support unless they share a receiver.*

### Linting & Refactoring Patterns

*   **Reduce Cognitive Complexity:** Extract `switch` cases into separate helper functions. The main function should be a simple dispatcher.
*   **Static Errors:** Use `var errNotFound = errors.New("...")` instead of dynamic `errors.New` inline. Wraps with `%w`.
*   **Parameter Naming:** Use descriptive names. `func process(funcType *ast.FuncType)` instead of `func process(f ...)`

### Go Generics in Code Generation
*   Type parameters belong on **declarations** (`type Foo[T any] struct`).
*   Function **fields** use parameters as concrete types (`callable func(T) T`).
*   Do **not** package-qualify type parameter names (e.g., `T`, `U`).

### Coverage
*   **80% Minimum** function coverage is required for **ALL** functions individually.

## Testing Patterns

### Synchronous Control
Tests use a channel-based call/response pattern:
1.  **Create Mock:** `imp := NewIntOpsImp(t)`
2.  **Start System:** `callable := NewPrintSumImp(t, run.PrintSum).Start(...)`
3.  **Expect & Inject:** `imp.ExpectCallIs.Add().ExpectArgsAre(1, 2).InjectResult(3)`
4.  **Verify:** `callable.ExpectReturnedValues(...)`

### Flexible Matching (Gomega)
Imptest supports Gomega matchers via duck typing.

```go
// Exact match
imp.ExpectCallIs.Add().ExpectArgsAre(1, 2)

// Flexible match
imp.ExpectCallIs.Add().ExpectArgsShould(BeNumerically(">", 0), Any())
```

### Concurrency
Use `.Within(duration)` for concurrent/out-of-order calls.

```go
imp.Within(time.Second).ExpectCallIs.Add().ExpectArgsAre(5, 6).InjectResult(11)
```

## Problem Solving Strategy
1.  **Analyze constraints:** If something seems impossible, explain the reasoning first.
2.  **Propose options:** List multiple approaches before implementing.
3.  **Fix comprehensively:** Resolve all related issues, not just the symptom.