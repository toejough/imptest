# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

This project uses [Mage](https://magefile.org/) for build automation. Run `mage` to list available targets.

Create mage targets for repeatable analysis tasks (e.g., `FindRedundantTests`)

## Working with Go Code

This repository uses the **gopls MCP server** for Go language support. When working with Go code in this repository, you MUST follow the workflows and guidelines documented in `GOPLS-MCP.md`. That file describes:

- The Read Workflow for understanding Go code
- The Edit Workflow for making changes to Go code
- Required MCP tools: `go_workspace`, `go_search`, `go_file_context`, `go_package_api`, `go_symbol_references`, `go_diagnostics`

**IMPORTANT**: Consult `GOPLS-MCP.md` before reading or modifying any Go code in this repository.

## Architecture

**imptest** is a test mock generation library for Go that enables testing impure functions through channel-based call interception.

### Core Components

1. **impgen** (`impgen/`) - Code generator that produces mock implementations:
   - `main.go` - Entry point, handles file system and package loading
   - `run/run.go` - Main generation logic and CLI argument parsing
   - `run/codegen_interface.go` - Generates interface mocks
   - `run/codegen_callable.go` - Generates callable wrappers for functions
   - `run/templates.go` - Go templates for generated code
   - `run/pkgparse.go` - AST parsing utilities

2. **UAT** (`UAT/`) - User Acceptance Tests demonstrating library usage:
   - `run/run.go` - Example interfaces (e.g., `IntOps`)
   - `run/run_test.go` - Tests showing both auto and manual validation patterns
   - Generated `*Imp_test.go` files - Mock implementations

### Generation Modes

The generator supports two modes via `//go:generate` directives:

```go
// Interface mock (default): generates mock with ExpectCallTo methods
//go:generate go run ../../impgen/main.go run.IntOps --name IntOpsImp

// Callable wrapper: wraps functions for return/panic validation
//go:generate go run ../../impgen/main.go run.PrintSum --name PrintSumImp --call
```

### Testing Pattern

Tests use channel-based synchronous call/response:
1. Create mock: `imp := NewIntOpsImp(t)`
2. Start function under test: `callable := NewPrintSumImp(t, run.PrintSum).Start(args...)`
3. Expect calls and inject responses: `imp.ExpectCallTo.Add(a, b).InjectResult(result)`
4. Verify return values: `callable.ExpectReturnedValues(expected...)`

### Lint Configuration

Lint configs are in `dev/golangci.toml` and `dev/golangci-todos.toml`. The project enforces 80% minimum function coverage.

## Code Preferences

Order of preferred considerations when writing code for this repository:
1. Test before implementation. Identify desired functionality, write tests, then implement to pass tests.
2. Adhere to Go idioms and best practices. Follow the workflows in `GOPLS-MCP.md` and validate with linters.
3. Favor simplicity, and readability, and reuse over cleverness or optimization. Before writing new code, consider if
   existing libraries or functions can be reused. After writing new code, review for clarity, simplicity, and
   opportunities for refactoring for reuse and standardization.
4. Separate orchestration logic from implementation details.
5. Optimize performance only as justified by profiling.
6. Prefer the minial set of tests that adequately verify functionality. Favor clarity and maintainability
   over exhaustive coverage.

### Organization 

Preferred Sorting order for code within files:
1. Package-level comments
2. Imports
3. Entry points (in this order: main, exported code, code called by other files)
4. Everything else.

Within those sections, sort by:
1. Constants
2. Variables
3. Types
4. Functions/Methods (grouped by receiver type for methods)

Within those sections, sort alphabetically.

### Simplicity and Readability

Avoid functions with nested loops or conditionals deeper than 3 levels. Refactor complex logic into smaller functions.

Avoid mixing orchestration with implementation details. Separate high-level logic from low-level operations.

Name variables and functions consistently and descriptively. Use full words instead of abbreviations unless the
abbreviation is widely recognized.

Comment non-obvious code. Use comments to explain why something is done, not what is done.

Comment all functions with a doc comment, even if unexported.

### Linting Patterns

**Reducing cognitive complexity (gocognit/cyclop):**
Extract each case in a type switch into a separate helper function. The main function becomes a simple dispatcher:
```go
func handleType(expr ast.Expr) string {
    switch t := expr.(type) {
    case *ast.Ident:
        return handleIdent(t)
    case *ast.StarExpr:
        return handleStar(t)
    }
}
```

**Static error variables (err113):**
Prefer static error variables over dynamic `errors.New()` calls:
```go
// Good
var errNotFound = errors.New("not found")
return fmt.Errorf("%w: %s", errNotFound, name)

// Avoid
return errors.New("not found: " + name)
```

**Parameter naming (varnamelen):**
Use descriptive parameter names. Expand single-letter names to indicate the type:
```go
// Good
func process(funcType *ast.FuncType, arrType *ast.ArrayType)

// Avoid
func process(f *ast.FuncType, a *ast.ArrayType)
```

### Go Generics in Code Generation

When generating generic wrapper code:
- Type parameters belong on struct/function **declarations**: `type Foo[T any] struct`
- Function **field** types use those parameters as concrete types: `callable func(T) T`
- Type parameter names (T, U) should NOT be package-qualified—check if an identifier is a type parameter before adding a package prefix

### Problem-Solving Approach

**Before claiming something is impossible:**
1. Explain your reasoning to the user
2. Be open to correction—you may be misunderstanding the constraint
3. Consider restructuring the problem (e.g., moving generics from a function type to the containing struct)

**Before implementing fixes:**
1. Articulate WHY the problem is difficult
2. List multiple possible approaches
3. Consider if difficulty indicates a design issue rather than a testing issue
4. Present options before implementing

**When planning complex changes:**
1. Use plan mode for non-trivial implementations
2. Explore thoroughly before proposing an approach
3. Expect iteration on the plan - be open to corrections and refinements
4. Don't proceed with execution until the approach is validated
5. Create concrete, testable steps rather than vague descriptions

**During execution:**
1. Use TodoWrite to track multi-step tasks and show progress
2. Mark todos as in_progress BEFORE starting work, completed IMMEDIATELY after finishing
3. Provide "★ Insight" blocks explaining interesting technical decisions or patterns
4. When tests fail or errors occur, investigate root causes - don't move forward with issues unresolved
5. Run `mage check` to verify everything passes before considering work complete

### Quality Standards

- The 80% function coverage requirement applies to ALL functions individually
- `mage check` must pass fully before considering work complete
- Complete work to stated standards without asking permission to stop short
- When fixing issues, resolve all related problems comprehensively

### Coverage Measurement and Analysis

**Philosophy**: Fix root causes, not symptoms. When coverage measurements seem wrong, investigate and fix the underlying issue rather than adding workarounds or exclusions.

**Measuring Coverage Correctly**:
- Use `-coverpkg=./...` to measure coverage across all packages, not just the package being tested
- Clear test cache when coverage results seem inconsistent: `rm -f coverage.out && go clean -testcache`


**Running Baseline Coverage**:
```bash
# DON'T use a single command with -run that applies to all packages
# DO run tests separately and merge:
go test -coverprofile=golden.out -coverpkg=./... -run TestUATConsistency ./impgen/run
go test -coverprofile=uat.out -coverpkg=./... ./UAT/...
# Then merge with segment-based splitting
```

### Common Refactoring Patterns

When eliminating code duplication in this codebase:

**Callback-based iteration helpers**:
```go
// Extract common iteration patterns into helpers that accept callbacks
func forEachParamField(param *ast.Field, ..., action func(fieldName, paramName string)) {
    // Common iteration logic
    action(fieldName, paramName)
}
```

**Consolidating similar functions**:
- When multiple functions differ only in small details, extract the common logic into a base function
- Use parameters or callbacks to handle the variations
- See `hasExportedIdent` consolidation as an example

**Moving shared logic to base generators**:
- When both interface and callable generators need the same functionality, move it to `baseGenerator`
- See `execTemplate`, `formatTypeParams*`, etc.

### Code Organization Clarification

"Sort alphabetically" means **strictly alphabetically by name**—not "grouped by relationship then alphabetically." Helper functions are interspersed with other functions based on their names, not placed adjacent to the functions they support.
