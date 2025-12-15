# Gemini Code Assistant Context

This document provides context for the Gemini code assistant to understand and effectively contribute to the `imptest` project.

## Project Overview

`imptest` is a Go testing tool designed for "impure functions" – functions whose primary role is to coordinate calls to other functions (dependencies). It helps verify that the function under test interacts with its dependencies correctly, in the right order, and with the right data.

The core mechanism of `imptest` is code generation. It provides a tool called `impgen` that generates mock implementations of Go interfaces. These mocks are then used in tests to:

1.  Intercept calls to dependency methods.
2.  Assert that the calls were made with the expected arguments.
3.  Inject return values into the function under test at the time of the call.
4.  Verify the final return values of the function under test.

This approach allows for synchronous, interactive testing of function behavior without needing to write complex, stateful mock implementations by hand.

The project is structured into a few main components:
-   `impgen`: The code generation tool that creates the test implementations from interfaces.
-   `UAT`: User Acceptance Tests that demonstrate how to use the `imptest` library.
-   `magefiles`: Contains the build and development task automation scripts using `mage`.

## Detailed Architecture

### Core Components

1.  **`impgen`** (`impgen/`) - The code generator that produces mock implementations.
    *   `main.go` - Entry point, handles file system and package loading.
    *   `run/run.go` - Main generation logic and CLI argument parsing.
    *   `run/codegen_interface.go` - Generates interface mocks.
    *   `run/codegen_callable.go` - Generates callable wrappers for functions.
    *   `run/templates.go` - Go templates for generated code.
    *   `run/pkgparse.go` - AST parsing utilities.

2.  **`UAT`** (`UAT/`) - User Acceptance Tests demonstrating library usage.
    *   `run/run.go` - Contains example interfaces (e.g., `IntOps`).
    *   `run/run_test.go` - Contains tests showing both auto and manual validation patterns.
    *   Generated `*Imp_test.go` files are the mock implementations.

### Generation Modes

The generator supports two modes via `//go:generate` directives:

```go
// Interface mock (default): generates a mock with ExpectCallTo methods
//go:generate go run ../../impgen/main.go run.IntOps --name IntOpsImp

// Callable wrapper: wraps a function to allow for return/panic validation
//go:generate go run ../../impgen/main.go run.PrintSum --name PrintSumImp --call
```

### Testing Pattern

Tests use a synchronous, channel-based call-and-response pattern:
1.  Create the mock: `imp := NewIntOpsImp(t)`
2.  Start the function under test in a goroutine: `callable := NewPrintSumImp(t, run.PrintSum).Start(args...)`
3.  Expect calls on the mock and inject responses: `imp.ExpectCallTo.Add(a, b).InjectResult(result)`
4.  Verify the function's final return values: `callable.ExpectReturnedValues(expected...)`

## Building and Running

This project uses [Mage](https://magefile.org/) for build automation. Run `mage` to list all available commands (targets).

### Key Commands

```bash
# Run all checks (tidy, generate, test, lint, coverage, mutation, fuzz, etc.)
mage check

# Watch for file changes and re-run all checks automatically
mage watch

# --- Individual Commands ---
mage generate      # Run go generate to create/update mocks
mage test          # Run unit tests with race detection and coverage
mage lint          # Run golangci-lint for static analysis
mage mutate        # Run mutation tests to check test quality
mage fuzz          # Run fuzz tests to find edge cases
mage checknils     # Run nilaway to detect potential nil panics
mage deadcode      # Check for unused code
mage tidy          # Run go mod tidy to clean up dependencies
mage installtools  # Install all necessary development tools
```

To run a single test by name:
```bash
go test -v -run TestName ./path/to/package
```

## Development Conventions and Code Style

### Coding Philosophy
1.  **Test-Driven Development:** Identify desired functionality, write tests that fail, and then implement the code to make the tests pass.
2.  **Idiomatic Go:** Adhere to Go idioms and best practices. Use `gopls` and the provided linters to validate.
3.  **Simplicity Over Cleverness:** Favor simple, readable, and reusable code. Before writing new code, look for opportunities to reuse existing functions.
4.  **Separation of Concerns:** Separate orchestration logic from implementation details.
5.  **Performance:** Optimize performance only when justified by profiling.
6.  **Minimal Tests:** Write the minimum set of tests that adequately verify functionality, favoring clarity and maintainability.

### Code Organization

Use the following sort order within files:
1.  Package-level comments
2.  Imports
3.  Entry points (main, exported code, then internal code)
4.  Everything else.

Within those sections, sort by:
1.  Constants
2.  Variables
3.  Types
4.  Functions/Methods (grouped by receiver)
...and alphabetically within each of those.

### Simplicity and Readability
*   Avoid nesting loops or conditionals deeper than 3 levels.
*   Separate high-level orchestration from low-level implementation.
*   Use descriptive, full-word names for variables and functions.
*   Comment the "why," not the "what."
*   Add a doc comment to every function, even unexported ones.

### Linting
*   Linting configuration is defined in `dev/golangci.toml` and `dev/golangci-todos.toml`.
*   The project enforces a minimum of **80% function coverage**.
