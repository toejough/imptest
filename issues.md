# Issue Tracker

A simple md issue tracker.

## Statuses

- backlog (to choose from)
- selected (to work on next)
- in progress (currently being worked on)
- review (ready for review/testing)
- done (completed)
- cancelled (not going to be done, for whatever reason, should have a reason)
- blocked (waiting on something else)

## Issues

1. remove the deprecation messages and any hint of V1/V2
   - status: backlog
2. Fix stdlib package shadowing ambiguity (UAT-11)
   - status: done
   - description: When a local package shadows a stdlib package name (e.g., local `time` package shadowing stdlib `time`), and the test file doesn't import the shadowed package, impgen cannot determine which package to use. Currently accepts syntax like `impgen time.Timer --dependency` but makes incorrect assumptions about which `time` package is intended.
   - current behavior: Accepts ambiguous syntax, generates incorrect code
   - expected behavior: Either require explicit package qualification (e.g., `--package=stdlib` flag) or detect and report ambiguity error
   - affected: UAT-11 demonstrates this issue
   - completed: 2025-12-31
   - commit: cae3385
   - timeline:
     - 2025-12-31 ~14:15 - Started: Design phase with solution-architect
     - 2025-12-31 ~14:20 - Planning: Broke down into 7 steps with solution-planner
     - 2025-12-31 ~14:25 - Implementation: Added --import-path flag, inference, and ambiguity detection
     - 2025-12-31 ~14:40 - Testing: All UAT-11 tests passing
     - 2025-12-31 ~14:45 - Refactor: Fixed linter errors (err113, lll, nlreturn, wsl_v5, staticcheck)
     - 2025-12-31 ~00:50 - Committed: fix(impgen): resolve stdlib package shadowing ambiguity
   - solution: Implemented 4-tier package resolution strategy:
     1. Explicit --import-path flag (highest priority)
     2. Infer from test file imports (automatic)
     3. Detect ambiguity and error with helpful message
     4. Fallback to existing logic
3. Add UAT for named parameters and returns
   - status: done
   - started: 2025-12-31 01:29 EST
   - completed: 2025-12-31 01:40 EST
   - timeline:
     - 2025-12-31 01:29 EST - RED: Wrote test file with named param/return expectations
     - 2025-12-31 01:30 EST - GREEN: Generated mocks/wrappers, fixed invocation errors
     - 2025-12-31 01:35 EST - REFACTOR: Fixed linter errors (err113, nlreturn, wrapcheck)
     - 2025-12-31 01:40 EST - Complete: All tests passing, mage check clean
   - description: Verify impgen handles named parameters and returns correctly (e.g., `func Process(ctx context.Context, id int) (user User, err error)`)
   - rationale: Common Go pattern for readability, currently untested ("?" in Signature Matrix)
   - acceptance: UAT demonstrating named params/returns in both target and dependency modes
   - effort: Small (1-2 hours) - actual: ~11 minutes
   - **NOTE**: After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Named" parameters/returns as "Yes" with UAT reference
4. Add UAT for function literal parameters
   - status: done
   - started: 2025-12-31 01:50 EST
   - completed: 2025-12-31 01:59 EST
   - timeline:
     - 2025-12-31 01:50 EST - RED: Wrote test file with function literal expectations
     - 2025-12-31 01:52 EST - GREEN: Generated mocks/wrappers, discovered impgen bug with multi-param function literals
     - 2025-12-31 01:55 EST - GREEN: Fixed tests to use matchers for function literal params
     - 2025-12-31 01:58 EST - REFACTOR: Fixed linter errors (noinlineerr, wsl_v5)
     - 2025-12-31 01:59 EST - Complete: All tests passing, mage check clean
   - description: Verify functions accepting function literals as parameters work correctly (e.g., `func Map(items []int, fn func(int) int) []int`)
   - rationale: Extremely common pattern for callbacks/middleware, currently untested
   - acceptance: UAT with function literal params in both interface methods and wrapped functions
   - effort: Small (1-2 hours) - actual: ~9 minutes
   - **NOTE**: After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Function literal" as "Yes" with UAT reference
   - **DISCOVERY**: Found impgen bug - multi-parameter function literals (e.g., `func(int, int) int`) have parameters dropped in code generation. Workaround: use single-param function literals and matchers
5. Add UAT for interface literal parameters
   - status: done
   - started: 2025-12-31 10:25 EST
   - completed: 2025-12-31 11:30 EST
   - timeline:
     - 2025-12-31 10:25 EST - RED: Created UAT structure, defined interface, wrote failing tests
     - 2025-12-31 10:30 EST - Generated mocks with impgen, discovered critical bug
     - 2025-12-31 10:45 EST - BUG FIX: Implemented `stringifyInterfaceType` to preserve interface literal method signatures
     - 2025-12-31 11:00 EST - GREEN: All 5 test cases passing (single-method, multi-method, error returns, return types, matchers)
     - 2025-12-31 11:15 EST - REFACTOR: Fixed 10 linter errors (funlen, cyclop, nestif, inamedparam, nlreturn, revive, wsl_v5)
     - 2025-12-31 11:20 EST - Updated TAXONOMY.md, mage check clean
   - description: Verify interface literals in signatures are handled (e.g., `func Process(obj interface{ Get() string })`)
   - rationale: Common ad-hoc interface pattern, currently untested
   - acceptance: UAT demonstrating interface literals in method signatures
   - effort: Small (1-2 hours) - actual: ~65 minutes
   - **NOTE**: TAXONOMY.md updated - Interface literal marked as "Yes" with UAT-25 reference
   - **NOTE**: Discovered critical bug during test creation (tracked separately as Issue #12) - impgen was stripping interface literal method signatures, had to fix before tests could pass
6. Add UAT for struct literal parameters
   - status: backlog
   - description: Verify struct literals in signatures work (e.g., `func Accept(cfg struct{ Timeout int })`)
   - rationale: Valid Go pattern, should verify support or document limitation
   - acceptance: UAT or documented limitation with workaround
   - effort: Small (1-2 hours)
   - **NOTE**: After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Struct literal" as "Yes" with UAT reference OR add to "Cannot Do" section if unsupported
7. Support dot imports for mocking
   - status: done
   - started: 2025-12-31 11:19 EST
   - completed: 2025-12-31 12:14 EST
   - timeline:
     - 2025-12-31 11:19 EST - RED: Created UAT-26 structure, helpers package, comprehensive test file
     - 2025-12-31 11:40 EST - GREEN: Discovered symbol resolution bug, fixed `findSymbol()` to check dot imports
     - 2025-12-31 11:50 EST - GREEN: Fixed package import path issue in generated mocks
     - 2025-12-31 12:05 EST - REFACTOR: Fixed all linter errors, added unit test for `getDotImportPaths()`
     - 2025-12-31 12:10 EST - REFACTOR: mage check passes with 0 issues
     - 2025-12-31 12:14 EST - Complete: TAXONOMY.md updated, all tests passing
   - description: Enable impgen to generate mocks for types/functions available via dot imports (`import . "pkg"`). If someone wants to mock something that is available in a dot import, we need to be able to do that.
   - rationale: Currently marked "?" in Package Matrix. Dot imports are a valid Go pattern and users should be able to mock dot-imported symbols.
   - solution: Modified symbol resolution to check dot-imported packages when symbol not found in current package:
     1. Added `getDotImportPaths()` to collect dot import paths from AST
     2. Modified `findSymbol()` to recursively search dot-imported packages when symbol not found
     3. Added `pkgPath` field to `symbolDetails` to track which package symbol was found in
     4. Updated `generateCode()` to reload package AST when symbol found via dot import
     5. Modified `newV2DependencyGenerator()` to use actual package path for unqualified names from dot imports
   - acceptance: UAT-26 demonstrates mocking of dot-imported symbols (Storage, Processor interfaces)
   - effort: Medium (~55 minutes actual, 2-3 hours estimated)
   - **NOTE**: TAXONOMY.md updated - Dot import marked as "Yes" with UAT-26 reference
8. Update taxonomy for resolved stdlib shadowing
   - status: done
   - started: 2025-12-31 01:14 EST
   - completed: 2025-12-31 01:15 EST
   - description: Update TAXONOMY.md to reflect that issue #2 (stdlib shadowing) is now fixed with 4-tier resolution strategy
   - rationale: Taxonomy still lists this as a "**Hole**" but implementation is complete (commit cae3385)
   - acceptance: TAXONOMY.md updated in following sections:
     - Package Variations Matrix: Change "Standard library shadowing" from "**Hole**" to "Yes" with UAT-11 reference
     - Known Issues section: Remove or update to indicate this is resolved
     - Cannot Do section: Remove stdlib shadowing entry or update to show it's now supported
   - effort: Trivial (15-30 min)
   - details: The 4-tier resolution strategy is: (1) Explicit --import-path flag, (2) Infer from test file imports, (3) Detect ambiguity and error with helpful message, (4) Fallback to existing logic
   - changes made:
     - Package Variations Matrix: Changed "**Hole**" to "Yes" with note "4-tier resolution (see below)"
     - Renamed "Known Issues" to "Standard Library Shadowing Resolution" with full explanation of 4-tier strategy
     - Cannot Do section: Changed "Known Hole" to "Now Supported ✓" with usage examples
9. dependency.go does not use typesafe return values
   - status: done
   - started: 2025-12-31 02:42 EST
   - completed: 2025-12-31 03:25 EST
   - timeline:
     - 2025-12-31 02:42 EST - Started issue, entered PLAN MODE
     - 2025-12-31 02:43 EST - Launched 2 Explore agents in parallel:
       - Agent 1: Exploring target wrapper typesafe return pattern
       - Agent 2: Exploring dependency mock return handling
     - 2025-12-31 02:52 EST - Exploration complete, launched Plan agent to design implementation
     - 2025-12-31 03:05 EST - Plan revised based on user feedback: Use method shadowing instead of return structs
     - 2025-12-31 03:06 EST - Plan complete, exiting PLAN MODE for user approval
     - 2025-12-31 03:08 EST - Plan approved, starting TDD implementation
     - 2025-12-31 03:10 EST - RED: Wrote test demonstrating typed InjectReturnValues usage
     - 2025-12-31 03:12 EST - GREEN: Implemented code generation (templates, template data, helper function)
     - 2025-12-31 03:20 EST - REFACTOR: All tests passing, mage check clean (0 linter errors)
     - 2025-12-31 03:25 EST - Complete: Verified compile-time type safety, all UATs passing
   - description: In dependency.go, the generated wrapper functions do not use typesafe return values, leading to potential runtime panics when type assertions fail or someone passes the wrong number of return values. This can be improved by generating typesafe return values in the mocks.
   - priority: CRITICAL - breaks fundamental type-safety expectation of the library
   - current behavior: InjectReturnValues(...any) with silent type assertion failures returning zero values
   - expected behavior: Typed InjectReturnValues methods that shadow base method with compile-time type safety
   - acceptance: Dependency mocks have same type-safe return value guarantees as target wrappers
   - solution: Generated typed InjectReturnValues(result0 T1, result1 T2, ...) methods on call wrappers that shadow the untyped base method, providing:
     - Compile-time type safety (wrong types = compiler error)
     - Compile-time arity checking (wrong number of params = compiler error)
     - Zero migration impact (same method name, calling pattern unchanged)
     - Full backward compatibility (all existing tests pass)
   - files modified:
     - impgen/run/templates.go: Added TypedReturnParams and ReturnParamNames fields
     - impgen/run/text_templates.go: Modified v2DepCallWrapperTmpl to generate typed method
     - impgen/run/codegen_v2_dependency.go: Added buildTypedReturnParams helper and updated buildMethodTemplateData
     - UAT/01-basic-interface-mocking/typesafety_test.go: Added test demonstrating typed API
     - UAT/01-basic-interface-mocking/typesafety_compile_errors_test.go.disabled: Examples of compile errors
10. Fix multi-parameter function literal code generation bug
   - status: done
   - started: 2025-12-31 02:12 EST
   - completed: 2025-12-31 02:20 EST
   - timeline:
     - 2025-12-31 02:12 EST - Started: Created issue and began investigation
     - 2025-12-31 02:13 EST - Analysis: Wrote test to understand AST structure for multi-param function literals
     - 2025-12-31 02:15 EST - Bug found: typeWithQualifierFunc was not expanding fields with multiple names
     - 2025-12-31 02:17 EST - Fix implemented: Created expandFieldListTypes helper to expand multi-name fields
     - 2025-12-31 02:18 EST - Verification: All UAT-24 tests passing with original multi-param interface
     - 2025-12-31 02:20 EST - Refactor: Fixed linter errors (cyclop, intrange, wsl_v5), mage check clean
   - description: impgen incorrectly parses multi-parameter function literal types (e.g., `func(int, int) int`), dropping all but the first parameter in generated code. This prevented testing functions with multi-param callbacks/reducers.
   - discovered: During UAT-24 (Issue #4), discovered impgen generates `func(int) int` when interface specifies `func(int, int) int`
   - root cause: In Go AST, `func(a, b int)` is represented as ONE field with Names=[a,b] and Type=int. The typeWithQualifierFunc function only called typeFormatter once per field, resulting in `func(int)` instead of `func(int, int)`.
   - solution: Created expandFieldListTypes helper function that checks field.Names length and repeats the type string for each name, properly expanding multi-name fields into multiple type occurrences.
   - acceptance: UAT-24 tests pass with original multi-parameter function literal signatures (e.g., `Reduce(items []int, initial int, reducer func(acc, item int) int) int`)
   - effort: Medium (2-4 hours) - actual: ~8 minutes
   - priority: High - blocks testing of common Go patterns (reducers, accumulators, callbacks with context)
11. Prevent ExpectCalledWithExactly generation for function types
   - status: backlog
   - description: impgen generates ExpectCalledWithExactly() methods for function type parameters, but Go cannot compare functions with `==`, causing reflect.DeepEqual to hang. Since we perform code generation based on known types, we should detect function types and skip generating equality methods.
   - discovered: During UAT-24 (Issue #4), tests with ExpectCalledWithExactly() on function params timeout
   - current behavior: Generates equality methods for all parameter types including functions
   - expected behavior: Detect function types in parameters and only generate ExpectCalledWithMatches(), not ExpectCalledWithExactly()
   - workaround: Use ExpectCalledWithMatches() with imptest.Any() for function parameters
   - acceptance: impgen detects function types and omits equality method generation
   - effort: Medium (2-4 hours)
   - priority: Medium - workaround exists, but better UX to prevent the footgun
12. Fix interface literal method signature stripping
   - status: done
   - started: 2025-12-31 10:30 EST
   - completed: 2025-12-31 11:00 EST
   - timeline:
     - 2025-12-31 10:30 EST - Discovered: Generated mocks for UAT-25, compiler errors showed stripped signatures
     - 2025-12-31 10:35 EST - Root cause: `stringifyDSTExpr` returns `"interface{}"` for all `*dst.InterfaceType` nodes
     - 2025-12-31 10:40 EST - Implementation: Added `stringifyInterfaceType` helper function
     - 2025-12-31 10:50 EST - Verification: Rebuilt impgen, regenerated mocks, all tests passing
     - 2025-12-31 11:00 EST - Complete: Single-line and multi-line interface literals working correctly
   - description: impgen strips interface literal method signatures during code generation, converting `interface{ Get() string }` to plain `interface{}`, causing compiler errors in generated mocks
   - discovered: During UAT-25 (Issue #5) test creation, when running `go generate` on interface with interface literal parameters
   - root cause: In `codegen_common.go` line 949, `case *dst.InterfaceType: return "interface{}"` unconditionally returns empty interface instead of preserving method signatures from `typedExpr.Methods` field list
   - solution: Added `stringifyInterfaceType` helper function that checks `Methods` field, iterates over method list, and builds proper signature strings (e.g., `interface{ Get() string }` for single methods, multi-line format for multiple methods)
   - acceptance: Generated code preserves interface literal signatures for both parameters and return types
   - effort: Small (1 hour) - actual: ~30 minutes
   - priority: Critical - causes compiler errors, blocks any use of interface literals in mocked interfaces
   - commit: 68a312c
