# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

This project uses [Mage](https://magefile.org/) for build automation. Run `mage` to list available targets.

```bash
# Run all checks (tidy, generate, test, lint, coverage, mutation, fuzz, etc.)
mage check

# Watch for changes and re-run checks
mage watch

# Individual commands:
mage generate      # Run go generate
mage test          # Run unit tests with race detection and coverage
mage lint          # Run golangci-lint
mage mutate        # Run mutation tests
mage fuzz          # Run fuzz tests
mage checknils     # Run nilaway
mage deadcode      # Check for dead code
mage tidy          # Run go mod tidy
mage installtools  # Install development tools (golangci-lint, nilaway, deadcode, etc.)
```

Run a single test:
```bash
go test -v -run TestName ./path/to/package
```

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
2. Adhere to Go idioms and best practices. Validate with the gopls tools and linters.
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

### Quality Standards

- The 80% function coverage requirement applies to ALL functions individually
- `mage check` must pass fully before considering work complete
- Complete work to stated standards without asking permission to stop short
- When fixing issues, resolve all related problems comprehensively

### Code Organization Clarification

"Sort alphabetically" means **strictly alphabetically by name**—not "grouped by relationship then alphabetically." Helper functions are interspersed with other functions based on their names, not placed adjacent to the functions they support.
