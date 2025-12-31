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
   - status: backlog
   - description: Verify interface literals in signatures are handled (e.g., `func Process(obj interface{ Get() string })`)
   - rationale: Common ad-hoc interface pattern, currently untested
   - acceptance: UAT demonstrating interface literals in method signatures
   - effort: Small (1-2 hours)
   - **NOTE**: After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Interface literal" as "Yes" with UAT reference
6. Add UAT for struct literal parameters
   - status: backlog
   - description: Verify struct literals in signatures work (e.g., `func Accept(cfg struct{ Timeout int })`)
   - rationale: Valid Go pattern, should verify support or document limitation
   - acceptance: UAT or documented limitation with workaround
   - effort: Small (1-2 hours)
   - **NOTE**: After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Struct literal" as "Yes" with UAT reference OR add to "Cannot Do" section if unsupported
7. Document dot import behavior
   - status: backlog
   - description: Verify and document whether dot imports (`import . "pkg"`) work or fail gracefully
   - rationale: Currently marked "?" in Package Matrix, should know definitive answer
   - acceptance: UAT demonstrating support OR updated taxonomy documenting limitation
   - effort: Small (1 hour)
   - **NOTE**: After testing, update Package Variations Matrix in TAXONOMY.md to mark "Dot import" as "Yes" with UAT reference OR add to "Cannot Do" section with workaround
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
   - status: backlog
   - description: In dependency.go, the generated wrapper functions do not use typesafe return values, leading to potential runtime panics when type assertions fail or someone passes the wrong number of return values. This can be improved by generating typesafe return values in the mocks.
