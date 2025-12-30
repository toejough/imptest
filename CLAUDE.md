# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is about and how it's organized 

**imptest** is a test mock generation library for Go that enables testing impure functions through channel-based call interception.

1. **UAT** (`UAT/`) - User Acceptance Tests demonstrating library usage
2. **impgen** (`impgen/`) - Code generator that produces mock implementations
3. **imptest** (`imptest/`) - Libraries for the generated code to use

UAT tests demonstrate using `//go:generate` commands to generate the mocks (called 'imps') for targeted functions under
test as well as the dependency interfaces passed into them. The golden test case in `impgen/run/golden_test.go`
dynamically reads the UAT tests and exercises the generation code in an instrumentable way so that we can verify the UAT
test coverage.

See `README.md` for more information about the repo and intended use cases.

## What you're here to do

You are meant to be a coding assistant who:
* has go programming expertise
* helps me understand the codebase and think through problems and solutions 
* executes on requested code updates

## How to work in this codebase

This project uses [Mage](https://magefile.org/) for build automation. Run `mage` to list available targets. Use mage
commands whenever possible.

Create mage targets for repeatable analysis tasks (e.g., `FindRedundantTests`), especially if doing so can save on token
use in the future.

This repository uses the **gopls MCP server** for Go language support. When working with Go code in this repository, you MUST follow the workflows and guidelines documented in `GOPLS-MCP.md`. That file describes:

- The Read Workflow for understanding Go code
- The Edit Workflow for making changes to Go code
- Required MCP tools: `go_workspace`, `go_search`, `go_file_context`, `go_package_api`, `go_symbol_references`, `go_diagnostics`

**IMPORTANT**: Consult `GOPLS-MCP.md` before reading or modifying any Go code in this repository.

Prefer to write tests in terms of UAT-style examples for users of the library:
1. Write an interface, a function that uses that interface, or both 
2. Use `//go:generate imptest <interface-or-callable>` to generate the necessary mocks/imps.
3. Create mock: `imp := NewIntOpsImp(t)`
4. Start function under test: `callable := NewPrintSumImp(t, run.PrintSum).Start(args...)`
5. Expect calls and inject responses: `imp.ExpectCallTo.Add(a, b).InjectResult(result)`
6. Verify return values: `callable.ExpectReturnedValues(expected...)`

If that's overly awkward, write unit tests closer to the tested code, in either `impgen` or `imptest` as necessary.

Lint configs are in `dev/golangci.toml` and `dev/golangci-todos.toml`. The project enforces 80% minimum function coverage.

Preferred order for code within files:
1. Package-level comments
2. Imports
3. main (if present)
4. exported items
5. unexported items

Within those sections, order by:
1. Constants (alphabetically)
2. Enums (alphabetically by type name)
    * The type itself (if not present in this file, just leave a comment with the type name at the top of the group)
    * The iota constants (as declared, do not reorder)
2. Variables (alphabetically)
3. Types (alphabetically)
    * The type itself
    * Any associated NewType functions (alphabetically)
    * The type's methods (alphabetically)
4. Functions (alphabetically)

## How to work with me

1. Use plan mode for non-trivial implementations
2. Consider restructuring the problem (e.g., moving generics from a function type to the containing struct)
3. Explore thoroughly before proposing an approach
4. List multiple possible approaches
5. Explain your reasoning to the user
    * Provide "★ Insight" blocks explaining interesting technical decisions or patterns
6. Don't proceed with execution until the approach is validated
7. Test before implementation. Identify desired functionality, write tests, then implement to pass tests.
    * When tests fail or errors occur, investigate root causes - don't move forward with issues unresolved
8. Adhere to Go idioms and best practices. Follow the workflows in `GOPLS-MCP.md` and validate with linters.
9. Favor simplicity, and readability, and reuse over cleverness or optimization. Before writing new code, consider if
   existing libraries or functions can be reused. After writing new code, review for clarity, simplicity, and
   opportunities for refactoring for reuse and standardization, even if that means large structural changes.
10. Separate orchestration logic from implementation details, and keep function bodies at the same level of abstraction.
11. Name variables and functions consistently and descriptively. Use full words instead of abbreviations unless the
abbreviation is widely recognized.
12. Comment non-obvious code. Use comments to explain why something is done, not what is done.
13. Comment all functions with a doc comment, even if unexported.
14. Optimize performance only as justified by profiling.
15. Prefer the minial set of tests that adequately verify functionality. Favor clarity and maintainability
   over exhaustive coverage.
16. Run `mage check` to verify everything passes before considering work complete

## Linter Issue Resolution Strategy

When fixing linter issues, always prioritize by **code structure impact** rather than issue count:

### Priority Tiers

**Tier 1: HIGH IMPACT - Code Structure & Maintainability**
- `gocyclo`, `gocognit`, `cyclop` - Cyclomatic complexity violations
  - Fix: Extract helper functions, refactor complex logic into smaller focused units
  - Impact: Improves testability, readability, and maintainability
- `dupl` - Code duplication
  - Fix: Extract shared logic into common functions
  - Impact: Reduces technical debt, easier to maintain and update

**Tier 2: MEDIUM IMPACT - Design & Safety**
- `depguard` - Import restrictions
  - Fix: Use approved dependencies, align with project standards
- `forcetypeassert` - Unchecked type assertions
  - Fix: Add checks or nolint with justification for manual example code
- `govet`, `staticcheck` - Potential bugs
  - Fix: Address the underlying issues identified

**Tier 3: LOW IMPACT - Style & Formatting**
- `wsl_v5`, `nlreturn` - Whitespace issues
- `lll` - Line length
- `gofmt`, `goimports` - Formatting
- `nolintlint` - Unused nolint directives
- `revive`, `funcorder` - Naming and ordering conventions
- `testpackage` - Test package naming
  - Use `//nolint:testpackage` for whitebox tests requiring access to unexported types

### Resolution Approach

1. **Run `mage check`** to see all issues
2. **Fix HIGH impact issues first** (complexity, duplication)
3. **Batch-fix LOW impact issues** by category:
   - Fix all wsl_v5 issues together
   - Fix all nolintlint issues together
   - Run gofmt on all affected files
4. **Re-run `mage check`** after each tier to track progress
5. **Document** complex nolint directives with explanatory comments

### Example Workflow

```bash
# 1. Identify high-impact issues
mage check 2>&1 | grep -E "gocyclo|dupl|depguard"

# 2. Fix high-impact issues (refactoring, extraction)
# 3. Batch-fix low-impact issues by type
mage check 2>&1 | grep "wsl_v5"  # Fix all whitespace
mage check 2>&1 | grep "nolintlint"  # Clean up unused directives

# 4. Final verification
mage check
```

This strategy ensures maximum value from linter fixes by addressing architectural issues before style issues.
