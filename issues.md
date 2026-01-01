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

---

## Issue Template

Standard issue structure organized by category:

```markdown
### [number]. [Issue Title]

#### Universal

**Status**
[backlog/selected/in progress/review/done/cancelled/blocked/migrated]

**Description** (optional - recommended)
[What needs to be done - clear, concise explanation of the issue or feature]

#### Planning (for backlog/selected issues)

**Rationale** (optional)
[Why this is needed - business/technical justification]

**Acceptance** (optional - recommended)
[What defines completion - specific, measurable criteria]

**Effort** (optional)
[Trivial/Small/Medium/Large with optional time estimate like "2-4 hours"]

**Priority** (optional)
[Low/Medium/High/Critical]

**Dependencies** (optional)
[What this issue depends on - other issues, external factors, etc.]

**Linear** (optional)
[TOE-XXX - Linear issue tracking ID]

#### Work Tracking (for in-progress/done issues)

**Started**
[YYYY-MM-DD HH:MM TZ]

**Completed** (for done)
[YYYY-MM-DD HH:MM TZ or just YYYY-MM-DD]

**Commit** (for done)
[commit hash or multiple hashes for multi-commit work]

**Timeline**
- YYYY-MM-DD HH:MM TZ - [Phase/milestone]: [Activity description]
- YYYY-MM-DD HH:MM TZ - [Phase/milestone]: [Activity description]
- ...

#### Documentation (for done issues)

**Solution** (optional)
[How the issue was solved - implementation approach, key decisions]

**Files Modified** (optional)
[List of key files changed, especially useful for understanding impact]

#### Bug Details (for bug fixes)

**Discovered** (optional)
[When/how the bug was found - context about discovery]

**Root Cause** (optional)
[Technical explanation of what caused the issue]

**Current Behavior** (optional)
[What happens now - the problematic behavior]

**Expected Behavior** (optional)
[What should happen - the desired behavior]

#### Special Fields

**Combined With** (optional)
[Other issue numbers if this was implemented together with related work]

**Migrated To** (for migrated status)
[project/file #issue_number - where this issue was moved to]

**Taxonomy Gap** (optional)
[Reference to TAXONOMY.md location this addresses]

**Note** (optional)
[Any additional context, reminders, or important information]
```

**Template Usage Notes:**
- Required field: Status (Universal) - only field that's always needed
- Description is optional but recommended (skip for self-explanatory titles like "Fix typo in README")
- Use category sections (`#### Category`) as needed for the issue type
- Skip entire category sections if not applicable
- Backlog issues: Universal + Planning sections
- In-progress issues: Universal + Planning + Work Tracking sections
- Done issues: Universal + Planning + Work Tracking + Documentation sections
- Bug fixes: Add Bug Details section as needed
- Timeline format: `- YYYY-MM-DD HH:MM TZ - Phase: Activity`
- Keep descriptions concise but complete when used

---

## Backlog

Issues to choose from for future work.

### 39. Fix timeline entries added outside issue section

#### Universal

**Status**
backlog

**Description**


### 37. Explore mage replacement with subcommands and flags (go-arg syntax)

#### Universal

**Status**
backlog

**Description**


### 36. Split issue tracker into separate repository

#### Universal

**Status**
backlog

**Description**

#### Work Tracking

**Timeline**
- 2026-01-01 17:09 EST - Implementer fixed auditor issues + discovered and fixed 3 additional bugs (insertion point, horizontal rule boundary, section header orphan)
- 2026-01-01 17:09 EST - Auditor FAILED - found ineffectual assignment and test data bug
- 2026-01-01 17:09 EST - Implementer applied fix - changed boundary detection to 'first boundary wins' logic



### 1. remove the deprecation messages and any hint of V1/V2

#### Universal

**Status**
backlog

### 11. Prevent ExpectCalledWithExactly generation for function types

#### Universal

**Status**
backlog

**Description**
impgen generates ExpectCalledWithExactly() methods for function type parameters, but Go cannot compare functions with `==`, causing reflect.DeepEqual to hang. Since we perform code generation based on known types, we should detect function types and skip generating equality methods.

Workaround: Use ExpectCalledWithMatches() with imptest.Any() for function parameters

#### Planning

**Acceptance**
impgen detects function types and omits equality method generation

**Effort**
Medium (2-4 hours)

**Priority**
Medium - workaround exists, but better UX to prevent the footgun

#### Bug Details

**Discovered**
During UAT-24 (Issue #4), tests with ExpectCalledWithExactly() on function params timeout

**Current Behavior**
Generates equality methods for all parameter types including functions

**Expected Behavior**
Detect function types in parameters and only generate ExpectCalledWithMatches(), not ExpectCalledWithExactly()

### 15. Consolidate duplicate Tester and TestReporter interfaces (TOE-105)

#### Universal

**Status**
backlog

**Description**
`Tester` and `TestReporter` interfaces in imptest/controller.go and imptest/imp.go are identical, causing code confusion

#### Planning

**Acceptance**
Single interface used consistently throughout codebase

**Effort**
Low

**Priority**
Low

**Linear**
TOE-105

### 16. Remove 'v2' from file and package names (TOE-109)

#### Universal

**Status**
backlog

**Description**
Now that V2 is the only implementation, remove "v2" suffix from filenames (codegen_v2_dependency.go → codegen_dependency.go, etc.)

#### Planning

**Acceptance**
All V2 references removed from file and package names, functionality unchanged

**Effort**
Small

**Priority**
Low

**Linear**
TOE-109

### 17. Remove stale .out and .test files (TOE-110)

#### Universal

**Status**
backlog

**Description**
Audit repository for stale build artifacts (*.out, *.test) and add to .gitignore

#### Planning

**Acceptance**
No build artifacts in version control, .gitignore updated

**Effort**
Trivial

**Priority**
Low

**Linear**
TOE-110

### 18. Reduce blanket nolint directives in V2 generators (TOE-106)

#### Universal

**Status**
backlog

**Description**
codegen_v2_dependency.go and codegen_v2_target.go suppress many linters with blanket directives, may hide code quality issues

#### Planning

**Acceptance**
Targeted nolint directives or refactored code to reduce suppression needs

**Effort**
Medium

**Priority**
Low

**Linear**
TOE-106


### 22. Re-evaluate and reorganize file structure for clarity (TOE-115)

#### Universal

**Status**
backlog

**Description**
Audit current file organization and evaluate whether it clearly communicates imptest/impgen architecture

#### Planning

**Acceptance**
File structure clearly separates concerns (templates, codegen, runtime, etc.)

**Effort**
Large

**Priority**
Low

**Linear**
TOE-115

### 24. Identify and remove redundant non-taxonomy-specific UAT tests (TOE-114)

#### Universal

**Status**
backlog

**Description**
Once taxonomy is clear, audit existing UATs to remove tests that duplicate coverage without adding value

#### Planning

**Acceptance**
Each UAT demonstrates unique taxonomy aspect, redundant tests removed

**Effort**
Medium

**Priority**
Low

**Dependencies**
Requires clear taxonomy definition

**Linear**
TOE-114

### 25. Consolidate API and usage documentation (TOE-112)

#### Universal

**Status**
backlog

**Description**
Multiple documents (README.md, docs/V1_TO_V2_MIGRATION.md, etc.) describe API/usage with redundancy

#### Planning

**Acceptance**
Consolidated documentation with clear organization, no redundancy

**Effort**
Medium

**Priority**
Medium

**Linear**
TOE-112

### 26. Document architecture with diagrams (TOE-116)

#### Universal

**Status**
backlog

**Description**
Create architectural documentation with diagrams: high-level flow, runtime architecture, code generation pipeline, test execution lifecycle

#### Planning

**Acceptance**
Clear diagrams explaining imptest architecture for users and contributors

**Effort**
Large

**Priority**
Medium

**Linear**
TOE-116

### 27. Better support for channel interactions (TOE-80)

#### Universal

**Status**
backlog

**Description**
Support testing patterns where submission results in channel message rather than return value

#### Planning

**Acceptance**
Can mock/test channel-based interactions

**Effort**
Large

**Priority**
Low

**Linear**
TOE-80

### 28. Investigate SSA format for codebase restructuring (TOE-78)

#### Universal

**Status**
backlog

**Description**
Investigate using SSA format to understand and possibly restructure codebase to simplify data/control flow

#### Planning

**Acceptance**
Analysis complete, recommendation documented

**Effort**
Large

**Priority**
Low

**Linear**
TOE-78


### 31. Add UAT for function type as dependency

#### Universal

**Status**
backlog

**Description**
Verify mocking function types (e.g., http.HandlerFunc) with --dependency flag

#### Planning

**Rationale**
Taxonomy matrix shows "?" for "Function type as Dependency", capability untested

**Acceptance**
UAT demonstrating function type mocking with --dependency, or documented limitation with workaround

**Effort**
Small (1-2 hours)

**Priority**
Medium

**Note**
After completing UAT, update Capability Matrix in TAXONOMY.md to mark "Function type as Dependency" as "Yes" with UAT reference OR add to "Cannot Do" section if unsupported

#### Special Fields

**Taxonomy Gap**
Capability Matrix - "Function type" row, "As Dependency" column

### 32. Add UAT for interface as target

#### Universal

**Status**
backlog

**Description**
Verify wrapping interfaces with --target flag (not mocking)

#### Planning

**Rationale**
Taxonomy matrix shows "?" for "Interface type as Target", capability untested

**Acceptance**
UAT demonstrating interface wrapping with --target, or documented limitation with workaround

**Effort**
Small (1-2 hours)

**Priority**
Medium

**Note**
After completing UAT, update Capability Matrix in TAXONOMY.md to mark "Interface type as Target" as "Yes" with UAT reference OR add to "Cannot Do" section if unsupported

#### Special Fields

**Taxonomy Gap**
Capability Matrix - "Interface type" row, "As Target" column

### 33. Add UAT for struct type as target (comprehensive)

#### Universal

**Status**
backlog

**Description**
Verify wrapping struct types with --target flag comprehensively (beyond just methods)

#### Planning

**Rationale**
Taxonomy matrix shows "?" for "Struct type as Target", partially covered in UAT-02 but needs comprehensive coverage

**Acceptance**
UAT demonstrating full struct type wrapping capabilities with --target

**Effort**
Small (1-2 hours)

**Priority**
Low

**Note**
After completing UAT, update Capability Matrix in TAXONOMY.md to mark "Struct type as Target" as "Yes" with comprehensive UAT reference

#### Special Fields

**Taxonomy Gap**
Capability Matrix - "Struct type" row, "As Target" column

---
## Selected

Issues selected for upcoming work.


### 30. Add UAT for function as dependency

#### Universal

**Status**
selected

**Description**
Verify mocking bare package-level functions with --dependency flag (not interface methods)

#### Planning

**Rationale**
Taxonomy matrix shows "?" for "Function as Dependency", capability untested

**Acceptance**
UAT demonstrating function mocking with --dependency, or documented limitation with workaround

**Effort**
Small (1-2 hours)

**Priority**
Medium

**Note**
After completing UAT, update Capability Matrix in TAXONOMY.md to mark "Function as Dependency" as "Yes" with UAT reference OR add to "Cannot Do" section if unsupported

#### Special Fields

**Taxonomy Gap**
Capability Matrix - "Function" row, "As Dependency" column

---

## In Progress

Issues currently being worked on.

*No issues currently in progress*

---

---

## Done

Completed issues.

---

### 38. Fix issuesStatus section not found error when moving last issue

#### Universal

**Status**
done

**Description**


#### Work Tracking

**Timeline**
- 2026-01-01 17:23 EST - Git-workflow created 2 commits: 6c4739b (tests), f1f773f (fix)
- 2026-01-01 17:20 EST - Auditor PASSED - all tests passing, no new linter violations, ready for commit
- 2026-01-01 17:18 EST - Implementer applied fix and discovered 3 additional bugs - all tests passing
- 2026-01-01 17:18 EST - failure-debugger identified root cause and created regression test
- 2026-01-01 17:18 EST - User reported: mage issuesstatus 33 selected fails with 'section not found: ## Selected'

## Review

Issues ready for review/testing.

*No issues currently in review*

### 35. Fix issuesfix to move issues to correct sections

#### Universal

**Status**
done

**Description**


#### Work Tracking

**Timeline**
- 2026-01-01 16:31 EST - Git-workflow created 3 commits: 90509ad (tests), 6d02fc3 (feat), 52483cd (docs)
- 2026-01-01 16:31 EST - Auditor re-audit PASSED - all tests passing, code quality verified
- 2026-01-01 16:19 EST - Implementer fixed integration: moved code to magefile.go, integrated into IssuesFix(), all tests pass
- 2026-01-01 16:16 EST - Auditor FAILED: Function not integrated into IssuesFix(), production code in test file
- 2026-01-01 16:12 EST - Used implementer agent - implemented moveIssuesToCorrectSections, all tests pass (GREEN phase)
- 2026-01-01 16:12 EST - Used test-writer agent - wrote 10 comprehensive failing tests (RED phase)
- 2026-01-01 16:12 EST - Used problem-clarifier agent - identified 5 root causes of file corruption
- 2026-01-01 16:12 EST - Added EXP-018 to patterns.md about orchestrator doing implementation instead of routing
- 2026-01-01 16:12 EST - Started: User reported issuesfix doesn't move issues to correct sections

### 2. Fix stdlib package shadowing ambiguity (UAT-11)

#### Universal

**Status**
done

**Description**
When a local package shadows a stdlib package name (e.g., local `time` package shadowing stdlib `time`), and the test file doesn't import the shadowed package, impgen cannot determine which package to use. Currently accepts syntax like `impgen time.Timer --dependency` but makes incorrect assumptions about which `time` package is intended.

#### Work Tracking

**Completed**
2025-12-31

**Commit**
cae3385

**Timeline**
- 2025-12-31 ~14:15 - Started: Design phase with solution-architect
- 2025-12-31 ~14:20 - Planning: Broke down into 7 steps with solution-planner
- 2025-12-31 ~14:25 - Implementation: Added --import-path flag, inference, and ambiguity detection
- 2025-12-31 ~14:40 - Testing: All UAT-11 tests passing
- 2025-12-31 ~14:45 - Refactor: Fixed linter errors (err113, lll, nlreturn, wsl_v5, staticcheck)
- 2025-12-31 ~00:50 - Committed: fix(impgen): resolve stdlib package shadowing ambiguity

#### Documentation

**Solution**
Implemented 4-tier package resolution strategy:
1. Explicit --import-path flag (highest priority)
2. Infer from test file imports (automatic)
3. Detect ambiguity and error with helpful message
4. Fallback to existing logic

#### Bug Details

**Current Behavior**
Accepts ambiguous syntax, generates incorrect code

**Expected Behavior**
Either require explicit package qualification (e.g., `--package=stdlib` flag) or detect and report ambiguity error

#### Special Fields

**Note**
Affected: UAT-11 demonstrates this issue

### 3. Add UAT for named parameters and returns

#### Universal

**Status**
done

**Description**
Verify impgen handles named parameters and returns correctly (e.g., `func Process(ctx context.Context, id int) (user User, err error)`)

#### Planning

**Rationale**
Common Go pattern for readability, currently untested ("?" in Signature Matrix)

**Acceptance**
UAT demonstrating named params/returns in both target and dependency modes

**Effort**
Small (1-2 hours) - actual: ~11 minutes

#### Work Tracking

**Started**
2025-12-31 01:29 EST

**Completed**
2025-12-31 01:40 EST

**Timeline**
- 2025-12-31 01:29 EST - RED: Wrote test file with named param/return expectations
- 2025-12-31 01:30 EST - GREEN: Generated mocks/wrappers, fixed invocation errors
- 2025-12-31 01:35 EST - REFACTOR: Fixed linter errors (err113, nlreturn, wrapcheck)
- 2025-12-31 01:40 EST - Complete: All tests passing, mage check clean

#### Special Fields

**Note**
After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Named" parameters/returns as "Yes" with UAT reference

### 4. Add UAT for function literal parameters

#### Universal

**Status**
done

**Description**
Verify functions accepting function literals as parameters work correctly (e.g., `func Map(items []int, fn func(int) int) []int`)

#### Planning

**Rationale**
Extremely common pattern for callbacks/middleware, currently untested

**Acceptance**
UAT with function literal params in both interface methods and wrapped functions

**Effort**
Small (1-2 hours) - actual: ~9 minutes

#### Work Tracking

**Started**
2025-12-31 01:50 EST

**Completed**
2025-12-31 01:59 EST

**Timeline**
- 2025-12-31 01:50 EST - RED: Wrote test file with function literal expectations
- 2025-12-31 01:52 EST - GREEN: Generated mocks/wrappers, discovered impgen bug with multi-param function literals
- 2025-12-31 01:55 EST - GREEN: Fixed tests to use matchers for function literal params
- 2025-12-31 01:58 EST - REFACTOR: Fixed linter errors (noinlineerr, wsl_v5)
- 2025-12-31 01:59 EST - Complete: All tests passing, mage check clean

#### Special Fields

**Note**
- After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Function literal" as "Yes" with UAT reference
- DISCOVERY: Found impgen bug - multi-parameter function literals (e.g., `func(int, int) int`) have parameters dropped in code generation. Workaround: use single-param function literals and matchers

### 5. Add UAT for interface literal parameters

#### Universal

**Status**
done

**Description**
Verify interface literals in signatures are handled (e.g., `func Process(obj interface{ Get() string })`)

#### Planning

**Rationale**
Common ad-hoc interface pattern, currently untested

**Acceptance**
UAT demonstrating interface literals in method signatures

**Effort**
Small (1-2 hours) - actual: ~65 minutes

#### Work Tracking

**Started**
2025-12-31 10:25 EST

**Completed**
2025-12-31 11:30 EST

**Timeline**
- 2025-12-31 10:25 EST - RED: Created UAT structure, defined interface, wrote failing tests
- 2025-12-31 10:30 EST - Generated mocks with impgen, discovered critical bug
- 2025-12-31 10:45 EST - BUG FIX: Implemented `stringifyInterfaceType` to preserve interface literal method signatures
- 2025-12-31 11:00 EST - GREEN: All 5 test cases passing (single-method, multi-method, error returns, return types, matchers)
- 2025-12-31 11:15 EST - REFACTOR: Fixed 10 linter errors (funlen, cyclop, nestif, inamedparam, nlreturn, revive, wsl_v5)
- 2025-12-31 11:20 EST - Updated TAXONOMY.md, mage check clean

#### Special Fields

**Note**
- TAXONOMY.md updated - Interface literal marked as "Yes" with UAT-25 reference
- Discovered critical bug during test creation (tracked separately as Issue #12) - impgen was stripping interface literal method signatures, had to fix before tests could pass

### 6. Add UAT for struct literal parameters

#### Universal

**Status**
done

**Description**
Verify struct literals in signatures work (e.g., `func Accept(cfg struct{ Timeout int })`)

#### Planning

**Rationale**
Valid Go pattern, should verify support or document limitation

**Acceptance**
UAT or documented limitation with workaround

**Effort**
Small (1-2 hours)

#### Work Tracking

**Started**
2026-01-01 01:40 EST

**Completed**
2026-01-01

**Commit**
bb59a33

**Timeline**
- 2026-01-01 01:40 EST - Started: Planning phase with solution-planner
- 2026-01-01 03:04 EST - Planning complete: 10-step plan created, bug location identified (line 942 in codegen_common.go)
- 2026-01-01 03:04 EST - RED: Test-writer creating UAT-30 structure
- 2026-01-01 03:06 EST - RED: UAT-30 created (DataProcessor interface, test functions with struct literal params/returns)
- 2026-01-01 03:06 EST - BUG DISCOVERED: impgen strips struct literal fields during generation (Process(cfg struct{ Timeout int }) → Process(cfg struct{}))
- 2026-01-01 03:06 EST - Created Issue #34 to track struct literal bug (blocking Issue #6 per EXP-005)
- 2026-01-01 03:06 EST - RED: Test-writer creating codegen_common_struct_test.go with 18 comprehensive unit tests
- 2026-01-01 03:10 EST - RED: 18 unit tests created, 16 failing (expected - defines spec for stringifyStructType)
- 2026-01-01 03:10 EST - GREEN: Implementer adding stringifyStructType() and stringifyFuncType() to codegen_common.go
- 2026-01-01 03:12 EST - GREEN complete: stringifyStructType implemented, all 18 unit tests pass, UAT-30 passes
- 2026-01-01 03:12 EST - REFACTOR: Routing to auditor for Step 4 (code quality review)
- 2026-01-01 11:05 EST - REFACTOR: Auditor found 15 linter violations (5 wsl_v5 + 10 others), routing to implementer
- 2026-01-01 11:05 EST - REFACTOR: Routing to implementer to fix linter violations
- 2026-01-01 11:14 EST - REFACTOR: All 15 linter violations fixed, mage check passes with 0 issues
- 2026-01-01 11:14 EST - REFACTOR: Routing to auditor for re-verification
- 2026-01-01 11:15 EST - REFACTOR: Auditor PASS - ready for commit
- 2026-01-01 11:15 EST - Routing to git-workflow to commit struct literal fix
- 2026-01-01 11:45 EST - Complete: TAXONOMY.md updated, struct literal marked as "Yes" with UAT-30
- 2026-01-01 11:45 EST - Routing to git-workflow to commit TAXONOMY.md update

#### Special Fields

**Note**
After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Struct literal" as "Yes" with UAT reference OR add to "Cannot Do" section if unsupported

### 7. Support dot imports for mocking

#### Universal

**Status**
done

**Description**
Enable impgen to generate mocks for types/functions available via dot imports (`import . "pkg"`). If someone wants to mock something that is available in a dot import, we need to be able to do that.

#### Planning

**Rationale**
Currently marked "?" in Package Matrix. Dot imports are a valid Go pattern and users should be able to mock dot-imported symbols.

**Acceptance**
UAT-26 demonstrates mocking of dot-imported symbols (Storage, Processor interfaces)

**Effort**
Medium (~55 minutes actual, 2-3 hours estimated)

#### Work Tracking

**Started**
2025-12-31 11:19 EST

**Completed**
2025-12-31 12:14 EST

**Timeline**
- 2025-12-31 11:19 EST - RED: Created UAT-26 structure, helpers package, comprehensive test file
- 2025-12-31 11:40 EST - GREEN: Discovered symbol resolution bug, fixed `findSymbol()` to check dot imports
- 2025-12-31 11:50 EST - GREEN: Fixed package import path issue in generated mocks
- 2025-12-31 12:05 EST - REFACTOR: Fixed all linter errors, added unit test for `getDotImportPaths()`
- 2025-12-31 12:10 EST - REFACTOR: mage check passes with 0 issues
- 2025-12-31 12:14 EST - Complete: TAXONOMY.md updated, all tests passing

#### Documentation

**Solution**
Modified symbol resolution to check dot-imported packages when symbol not found in current package:
1. Added `getDotImportPaths()` to collect dot import paths from AST
2. Modified `findSymbol()` to recursively search dot-imported packages when symbol not found
3. Added `pkgPath` field to `symbolDetails` to track which package symbol was found in
4. Updated `generateCode()` to reload package AST when symbol found via dot import
5. Modified `newV2DependencyGenerator()` to use actual package path for unqualified names from dot imports

#### Special Fields

**Note**
TAXONOMY.md updated - Dot import marked as "Yes" with UAT-26 reference

### 8. Update taxonomy for resolved stdlib shadowing

#### Universal

**Status**
done

**Description**
Update TAXONOMY.md to reflect that issue #2 (stdlib shadowing) is now fixed with 4-tier resolution strategy

#### Planning

**Rationale**
Taxonomy still lists this as a "**Hole**" but implementation is complete (commit cae3385)

**Acceptance**
TAXONOMY.md updated in following sections:
- Package Variations Matrix: Change "Standard library shadowing" from "**Hole**" to "Yes" with UAT-11 reference
- Known Issues section: Remove or update to indicate this is resolved
- Cannot Do section: Remove stdlib shadowing entry or update to show it's now supported

**Effort**
Trivial (15-30 min)

#### Work Tracking

**Started**
2025-12-31 01:14 EST

**Completed**
2025-12-31 01:15 EST

#### Documentation

**Solution**
The 4-tier resolution strategy is: (1) Explicit --import-path flag, (2) Infer from test file imports, (3) Detect ambiguity and error with helpful message, (4) Fallback to existing logic

Changes made:
- Package Variations Matrix: Changed "**Hole**" to "Yes" with note "4-tier resolution (see below)"
- Renamed "Known Issues" to "Standard Library Shadowing Resolution" with full explanation of 4-tier strategy
- Cannot Do section: Changed "Known Hole" to "Now Supported ✓" with usage examples

### 9. dependency.go does not use typesafe return values

#### Universal

**Status**
done

**Description**
In dependency.go, the generated wrapper functions do not use typesafe return values, leading to potential runtime panics when type assertions fail or someone passes the wrong number of return values. This can be improved by generating typesafe return values in the mocks.

#### Planning

**Priority**
CRITICAL - breaks fundamental type-safety expectation of the library

**Acceptance**
Dependency mocks have same type-safe return value guarantees as target wrappers

#### Work Tracking

**Started**
2025-12-31 02:42 EST

**Completed**
2025-12-31 03:25 EST

**Timeline**
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

#### Documentation

**Solution**
Generated typed InjectReturnValues(result0 T1, result1 T2, ...) methods on call wrappers that shadow the untyped base method, providing:
- Compile-time type safety (wrong types = compiler error)
- Compile-time arity checking (wrong number of params = compiler error)
- Zero migration impact (same method name, calling pattern unchanged)
- Full backward compatibility (all existing tests pass)

**Files Modified**
- impgen/run/templates.go: Added TypedReturnParams and ReturnParamNames fields
- impgen/run/text_templates.go: Modified v2DepCallWrapperTmpl to generate typed method
- impgen/run/codegen_v2_dependency.go: Added buildTypedReturnParams helper and updated buildMethodTemplateData
- UAT/01-basic-interface-mocking/typesafety_test.go: Added test demonstrating typed API
- UAT/01-basic-interface-mocking/typesafety_compile_errors_test.go.disabled: Examples of compile errors

#### Bug Details

**Current Behavior**
InjectReturnValues(...any) with silent type assertion failures returning zero values

**Expected Behavior**
Typed InjectReturnValues methods that shadow base method with compile-time type safety

### 10. Fix multi-parameter function literal code generation bug

#### Universal

**Status**
done

**Description**
impgen incorrectly parses multi-parameter function literal types (e.g., `func(int, int) int`), dropping all but the first parameter in generated code. This prevented testing functions with multi-param callbacks/reducers.

#### Planning

**Acceptance**
UAT-24 tests pass with original multi-parameter function literal signatures (e.g., `Reduce(items []int, initial int, reducer func(acc, item int) int) int`)

**Effort**
Medium (2-4 hours) - actual: ~8 minutes

**Priority**
High - blocks testing of common Go patterns (reducers, accumulators, callbacks with context)

#### Work Tracking

**Started**
2025-12-31 02:12 EST

**Completed**
2025-12-31 02:20 EST

**Timeline**
- 2025-12-31 02:12 EST - Started: Created issue and began investigation
- 2025-12-31 02:13 EST - Analysis: Wrote test to understand AST structure for multi-param function literals
- 2025-12-31 02:15 EST - Bug found: typeWithQualifierFunc was not expanding fields with multiple names
- 2025-12-31 02:17 EST - Fix implemented: Created expandFieldListTypes helper to expand multi-name fields
- 2025-12-31 02:18 EST - Verification: All UAT-24 tests passing with original multi-param interface
- 2025-12-31 02:20 EST - Refactor: Fixed linter errors (cyclop, intrange, wsl_v5), mage check clean

#### Documentation

**Solution**
Created expandFieldListTypes helper function that checks field.Names length and repeats the type string for each name, properly expanding multi-name fields into multiple type occurrences.

#### Bug Details

**Discovered**
During UAT-24 (Issue #4), discovered impgen generates `func(int) int` when interface specifies `func(int, int) int`

**Root Cause**
In Go AST, `func(a, b int)` is represented as ONE field with Names=[a,b] and Type=int. The typeWithQualifierFunc function only called typeFormatter once per field, resulting in `func(int)` instead of `func(int, int)`.

### 12. Fix interface literal method signature stripping

#### Universal

**Status**
done

**Description**
impgen strips interface literal method signatures during code generation, converting `interface{ Get() string }` to plain `interface{}`, causing compiler errors in generated mocks

#### Planning

**Acceptance**
Generated code preserves interface literal signatures for both parameters and return types

**Effort**
Small (1 hour) - actual: ~30 minutes

**Priority**
Critical - causes compiler errors, blocks any use of interface literals in mocked interfaces

#### Work Tracking

**Started**
2025-12-31 10:30 EST

**Completed**
2025-12-31 11:00 EST

**Commit**
68a312c

**Timeline**
- 2025-12-31 10:30 EST - Discovered: Generated mocks for UAT-25, compiler errors showed stripped signatures
- 2025-12-31 10:35 EST - Root cause: `stringifyDSTExpr` returns `"interface{}"` for all `*dst.InterfaceType` nodes
- 2025-12-31 10:40 EST - Implementation: Added `stringifyInterfaceType` helper function
- 2025-12-31 10:50 EST - Verification: Rebuilt impgen, regenerated mocks, all tests passing
- 2025-12-31 11:00 EST - Complete: Single-line and multi-line interface literals working correctly

#### Documentation

**Solution**
Added `stringifyInterfaceType` helper function that checks `Methods` field, iterates over method list, and builds proper signature strings (e.g., `interface{ Get() string }` for single methods, multi-line format for multiple methods)

#### Bug Details

**Discovered**
During UAT-25 (Issue #5) test creation, when running `go generate` on interface with interface literal parameters

**Root Cause**
In `codegen_common.go` line 949, `case *dst.InterfaceType: return "interface{}"` unconditionally returns empty interface instead of preserving method signatures from `typedExpr.Methods` field list

### 13. Add UAT for business logic dot imports

#### Universal

**Status**
done

**Description**
Verify impgen works when business logic (not test code) uses dot imports. Service package imports storage via `import . "storage"` and uses Repository interface. Test package imports service normally and mocks the dot-imported Repository interface.

#### Planning

**Rationale**
Tests real-world pattern where production code uses dot imports and test code needs to mock those types

**Acceptance**
UAT-27 demonstrates mocking interfaces that are dot-imported by production code (not test code)

**Effort**
Small (~20 minutes actual)

#### Work Tracking

**Started**
2025-12-31 15:30 EST

**Completed**
2025-12-31 15:50 EST

**Timeline**
- 2025-12-31 15:30 EST - RED: Created UAT-27 structure (storage, service, test packages)
- 2025-12-31 15:35 EST - RED: Wrote service package with dot import of storage
- 2025-12-31 15:40 EST - RED: Wrote comprehensive test file mocking dot-imported Repository
- 2025-12-31 15:42 EST - GREEN: Generated mocks successfully with `go generate`
- 2025-12-31 15:43 EST - GREEN: All 4 test cases passing
- 2025-12-31 15:45 EST - REFACTOR: Fixed linter errors (depguard, noinlineerr, staticcheck)
- 2025-12-31 15:50 EST - Complete: mage check clean, all tests passing

#### Special Fields

**Note**
This complements UAT-26 which tests dot imports in test code itself

### 19. Fix Eventually() type loss in concurrent call matching (TOE-104)

#### Universal

**Status**
done

**Description**
Eventually() returns base *DependencyCall instead of typed wrapper, losing type-safe GetArgs() access

#### Planning

**Acceptance**
Eventually() returns same typed wrapper as synchronous call matching

**Effort**
High - requires changes to controller/dependency interaction

**Priority**
Medium

**Linear**
TOE-104

#### Work Tracking

**Started**
2025-12-31 14:00 EST

**Completed**
2025-12-31

**Commit**
3629eeb, 46c6966, d66a013

**Timeline**
      - 2025-12-31 14:00 EST: DESIGN phase - Routed to solution-architect
      - 2025-12-31 14:15 EST: DESIGN - User clarified: no timeout at all, ordered mode should fail-fast
      - 2025-12-31 14:20 EST: DESIGN - Solution-architect redesigned with ordered vs eventually modes
      - 2025-12-31 14:25 EST: DESIGN complete - User approved: ordered (fail-fast) vs eventually (queue)
      - 2025-12-31 14:30 EST: PLANNING phase - Routed to solution-planner
      - 2025-12-31 14:35 EST: PLANNING - Architectural decisions confirmed (hard break, wait forever, per-expectation, FIFO)
      - 2025-12-31 14:45 EST: PLANNING complete - 13-step plan created (~10 hours estimated)
      - 2025-12-31 14:50 EST: RED phase - Step 1: test-writer writing tests for failOnMismatch
      - 2025-12-31 14:55 EST: RED complete - 2 failing tests written (GetCallOrdered, GetCallEventually)
      - 2025-12-31 15:00 EST: GREEN phase - Step 1: implementer adding failOnMismatch to waiter struct
      - 2025-12-31 15:05 EST: GREEN complete - Step 1 done: waiter struct extended, GetCallOrdered implemented, tests passing
      - 2025-12-31 15:10 EST: AUDIT phase - Step 1: auditor reviewing code quality
      - 2025-12-31 15:12 EST: AUDIT PASS - Step 1: Clean implementation, thread-safe, comprehensive tests, ready for Step 2
      - 2025-12-31 15:15 EST: RED phase - Step 2: test-writer writing tests for GetCallEventually
      - 2025-12-31 15:18 EST: RED complete - Step 2: 2 failing tests written (WaitsForMatch, ChecksQueueFirst)
      - 2025-12-31 15:20 EST: GREEN phase - Step 2: implementer creating GetCallEventually method
      - 2025-12-31 15:23 EST: GREEN complete - Step 2: GetCallEventually implemented, all tests passing
      - 2025-12-31 15:25 EST: AUDIT phase - Step 2: auditor reviewing code quality
      - 2025-12-31 15:27 EST: AUDIT PASS - Step 2: Functional and safe, minor code duplication (temp), ready for Step 3
      - 2025-12-31 15:30 EST: RED phase - Step 3: test-writer writing tests for mixed ordered/eventually waiters
      - 2025-12-31 15:35 EST: RED complete - Step 3: 3 tests written, 2 failing (exposed dispatcher missing fail-fast priority)
      - 2025-12-31 15:38 EST: GREEN phase - Step 3: implementer fixing dispatcher fail-fast priority logic
      - 2025-12-31 15:41 EST: GREEN complete - Step 3: Dispatcher fixed, all 19 tests passing, FIFO with fail-fast working
      - 2025-12-31 15:43 EST: AUDIT phase - Step 3: auditor reviewing dispatcher logic
      - 2025-12-31 15:46 EST: AUDIT PASS - Step 3: Correct logic, thread-safe, 100% coverage, ready for next phase
      - 2025-12-31 15:48 EST: Phase 1 (Controller Layer) COMPLETE - Steps 1-3 done, moving to Phase 2 (Imp Layer)
      - 2025-12-31 15:50 EST: RED phase - Step 5: test-writer writing tests for Imp layer methods
      - 2025-12-31 15:53 EST: RED complete - Step 5: 3 failing tests written (GetCallOrdered/Eventually on Imp)
      - 2025-12-31 15:55 EST: GREEN phase - Step 5: implementer adding GetCallOrdered/Eventually to Imp
      - 2025-12-31 15:58 EST: GREEN complete - Step 5: Imp layer methods implemented, all tests passing
      - 2025-12-31 16:00 EST: AUDIT phase - Step 5: auditor reviewing Imp layer
      - 2025-12-31 16:02 EST: AUDIT NOTE - Step 5: Tests pass, some linter warnings (may be pre-existing), continuing
      - 2025-12-31 16:05 EST: Phases 1-2 CHECKPOINT - Controller & Imp layers functional, committing progress
      - 2025-12-31 16:08 EST: COMMITTED - feat(imptest): implement ordered vs eventually call matching modes (3629eeb)
      - 2025-12-31 16:10 EST: Phase 3 START - DependencyMethod layer: removing timeout, adding eventually flag
      - 2025-12-31 16:12 EST: RED phase - Step 6: test-writer writing tests for DependencyMethod transformation
      - 2025-12-31 16:15 EST: RED complete - Step 6: 3 failing tests written (Eventually() API transformation)
      - 2025-12-31 16:18 EST: GREEN phase - Step 6: implementer transforming DependencyMethod to mode-based
      - 2025-12-31 16:22 EST: GREEN complete - Step 6: Timeout removed! Eventually() now no-param, mode-based API working
      - 2025-12-31 15:23 EST: MILESTONE - Issue #20 core complete: Eventually() has NO timeout parameter!
      - 2025-12-31 15:25 EST: COMMITTED - feat(imptest): remove timeout from Eventually() (46c6966)
      - 2025-12-31 15:27 EST: Phase 4 START - Code generation: Add typed Eventually() to templates for Issue #19
      - 2025-12-31 15:27 EST: GREEN phase - Routed to implementer for template changes
      - 2025-12-31 15:28 EST: GREEN - implementer: Added Eventually() to v2DepMethodWrapperTmpl
      - 2025-12-31 15:29 EST: GREEN - implementer: Updated 5 UAT test files (removed timeout parameter)
      - 2025-12-31 15:30 EST: GREEN - implementer: Regenerated mocks, created eventually_test.go
      - 2025-12-31 15:31 EST: GREEN - All UAT tests passing
      - 2025-12-31 17:40 EST: Step 7 complete: Added typed Eventually() to templates, Issue #19 COMPLETE
      - 2025-12-31 17:45 EST: COMMITTED - feat(imptest): add type-safe Eventually() to generated mocks (d66a013)
      - 2025-12-31 17:47 EST: Step 8 START - Cleanup: Remove remaining timeout-related template code
      - 2025-12-31 17:49 EST: Step 8 complete - Verified no timeout references remain (all cleaned in Step 7)
      - 2025-12-31 17:49 EST: Steps 9-10 SKIP - UAT test updates already completed in Step 7
      - 2025-12-31 17:50 EST: Step 11 START - Create new UAT test demonstrating ordered vs eventually modes
      - 2025-12-31 17:52 EST: GREEN - implementer: Created UAT/28 directory structure
      - 2025-12-31 17:53 EST: GREEN - implementer: Created service.go interface
      - 2025-12-31 17:55 EST: GREEN - implementer: Created modes_test.go with 6 comprehensive tests
      - 2025-12-31 17:58 EST: GREEN - implementer: Generated mock, fixed test issues
      - 2025-12-31 18:00 EST: GREEN - All 5 UAT-28 tests passing with race detector
      - 2025-12-31 18:01 EST: Step 11 complete - UAT-28 demonstrates ordered vs eventually modes
      - 2025-12-31 18:02 EST: Step 12 START - Update documentation and comments
      - 2025-12-31 18:05 EST: Step 12 - Updated README.md (3 Eventually() examples)
      - 2025-12-31 18:06 EST: Step 12 - Updated V1_TO_V2_MIGRATION.md
      - 2025-12-31 18:07 EST: Step 12 complete - All documentation updated
      - 2025-12-31 18:08 EST: Step 13 START - Run full test suite and verify
      - 2025-12-31 18:20 EST: BLOCKER FOUND - Regression in commit 3629eeb: fail-fast logic has bugs
      - 2025-12-31 18:21 EST: Bug details: TestImpGetCallOrdered_WrongMethod, TestDispatchLoop_OrderedFailsEventuallyWaits failing
      - 2025-12-31 18:22 EST: Root cause: matched=true fix reveals deeper dispatcher logic issues
      - 2025-12-31 18:23 EST: STATUS: Steps 7-12 COMPLETE and working, but blocked by earlier commit bug
      - 2025-12-31 21:00 EST: REFACTOR - User identified: violated TDD by updating library before tests
      - 2025-12-31 21:05 EST: REFACTOR - Systematically updated all validators: bool→error-based (controller, imp, dependency, tests)
      - 2025-12-31 21:25 EST: REFACTOR - Fixed hanging test: TestDispatchLoop_FIFOPriority (Eventually no longer times out)
      - 2025-12-31 21:30 EST: REFACTOR PASS - All tests passing (0.463s)
      - 2025-12-31 21:35 EST: BLOCKER - Data races detected by race detector
      - 2025-12-31 21:40 EST: INVESTIGATION - Routed to failure-debugger agent
      - 2025-12-31 21:50 EST: INVESTIGATION - Agent identified 3 racy tests using mockTester pattern (unsynchronized variable access)
      - 2025-12-31 21:52 EST: INVESTIGATION - Root cause: test goroutine reads variables while dispatcher goroutine writes them
      - 2025-12-31 21:52 EST: STATUS: Regression tests created in race_regression_test.go, ready for TDD fix cycle
      - 2025-12-31 22:05 EST: FIX - User insight: use imptest to test imptest (TesterImp/MockTester generated mocks)
      - 2025-12-31 22:10 EST: FIX - Updated controller_test.go: 2 racy tests now use TesterImp (proper sync via Controller)
      - 2025-12-31 22:12 EST: FIX - Hit import cycle for imp_test.go (package imptest can't import itself)
      - 2025-12-31 22:15 EST: FIX - Used channel-based sync for imp_test.go (mockTestReporter with fatalfChan)
      - 2025-12-31 22:20 EST: FIX - All 3 fixed tests pass with -race (0 races detected)
      - 2025-12-31 22:22 EST: VERIFIED - All production tests pass with -race (excluding intentional regression tests)
      - 2025-12-31 22:23 EST: STATUS: Race conditions FIXED, all tests passing
      - 2025-12-31 22:25 EST: MAGE CHECK - Updated magefile to skip TestRaceRegression tests in CI (using -skip flag)
      - 2025-12-31 22:30 EST: CLEANUP - Removed deprecated GetCallWithTimeout (0% coverage, no callers)
      - 2025-12-31 22:35 EST: LINTING - Fixed 14 new linting issues in race_regression_test.go (wsl whitespace + revive)
      - 2025-12-31 22:40 EST: STATUS: Race fix work COMPLETE - 35 pre-existing linting issues remain (tracked separately)
      - 2025-12-31 22:45 EST: LINTING - Fixed all 35 pre-existing linting issues (err113, wsl, nolintlint, staticcheck, nlreturn, noinlineerr, funlen, revive, errcheck)
- 2025-12-31 22:50 EST: ✅ VERIFIED - mage check PASSES with 0 issues
- 2025-12-31 22:50 EST: STATUS: Issues #19 and #20 COMPLETE - All race conditions fixed, all tests passing, mage check clean

#### Special Fields

**Combined With**
Issue #20 (both related to Eventually() API)

### 20. Remove timeout parameter from Eventually() (TOE-107)

#### Universal

**Status**
done

**Description**
Eventually(time.Second) requires timeout, but timeout is only for handling unordered concurrent code, not real delays. Should use sensible default instead.

#### Planning

**Acceptance**
Eventually() has no timeout parameter, waits indefinitely (test framework handles timeout)

**Effort**
Medium

**Priority**
Medium

**Linear**
TOE-107

#### Work Tracking

**Started**
2025-12-31 14:00 EST

**Completed**
2025-12-31

**Commit**
3629eeb, 46c6966, d66a013

**Timeline**
- 2025-12-31 14:00 EST: Combined with issue #19 for unified solution
- 2025-12-31 14:20 EST: DESIGN - Decision: NO timeout parameter, wait indefinitely
- 2025-12-31 14:45 EST: PLANNING - Timeout removed entirely from design
- 2025-12-31 15:05 EST: GREEN phase - Step 1/13 complete (waiter struct foundation)

#### Documentation

**Solution**
Implemented ordered vs eventually modes - ordered mode fails fast on mismatch, eventually mode queues mismatches and waits indefinitely. Eventually() no longer takes timeout parameter. Added type-safe Eventually() methods to generated mock templates. Created UAT-28 demonstrating both modes.

#### Special Fields

**Combined With**
Issue #19 (both related to Eventually() API)

### 23. Verify dogfooding: use imptest in all applicable tests (TOE-111)

#### Universal

**Status**
done

**Description**
Audit all *_test.go files to ensure we use imptest's mocking where applicable - serves as dogfooding and improves test quality

#### Planning

**Acceptance**
All applicable tests use imptest patterns, gaps documented

**Effort**
Medium

**Priority**
Medium

**Linear**
TOE-111

#### Work Tracking

**Completed**
2026-01-01

**Commit**
32443a6

**Timeline**
- 2026-01-01 00:05 EST - Started: Routing to Explore agent to audit test files for imptest usage opportunities
- 2026-01-01 00:08 EST - Audit complete: 90 test files analyzed, ~75 already using imptest correctly (controller_test.go is gold standard, all 78 UATs correct). Found 2 high-priority opportunities: (1) imp_test.go should use generated MockTestReporter instead of manual mockTestReporter, (2) race_regression_test.go "proper sync" tests should use TesterImp. Medium priority: generate MockTimer for timeout testing, document dogfooding pattern.
- 2026-01-01 00:14 EST - REFACTOR: Starting Step 1 of 4 - Generate MockTestReporter and refactor imp_test.go (5 tests affected)
- 2026-01-01 00:44 EST - REFACTOR: Step 1 auditor found linter issue - generate directive missing --dependency flag, fixing now
- 2026-01-01 01:00 EST - CRITICAL: Step 1 implementer fix caused test deadlock - TestImpFatalf hangs/times out after 2m, external package approach broke mock expectations, routing back to implementer to fix or revert
- 2026-01-01 01:05 EST - REFACTOR: Implementer reverted to manual mock (wrong direction) - controller_test.go shows external package DOES work with generated mocks, routing back with correct instructions to follow controller_test.go pattern exactly
- 2026-01-01 01:13 EST - REFACTOR: Step 1 implementation complete using controller_test.go pattern (external package + generated MockTestReporter), routing to auditor for verification
- 2026-01-01 01:16 EST - REFACTOR: Step 1 COMPLETE ✓ - imp_test.go now uses generated MockTestReporter, no manual mocks, all tests pass, mage check clean. Starting Step 2 of 4 - Update race_regression_test.go proper sync tests to use TesterImp
- 2026-01-01 01:21 EST - REFACTOR: Step 2 COMPLETE ✓ - race_regression_test.go proper sync tests now use TesterImp, regression tests preserved with manual mocks (intentional), all tests pass, mage check clean. Starting Step 3 of 4 - Generate MockTimer
- 2026-01-01 01:25 EST - REFACTOR: Step 3 COMPLETE ✓ - MockTimer generated successfully, infrastructure ready for deterministic timeout testing, mage check clean. Starting Step 4 of 4 - Document dogfooding pattern in README
- 2026-01-01 01:30 EST - REFACTOR: Step 4 COMPLETE ✓ - Added comprehensive "How imptest Tests Itself (Dogfooding)" section to README.md with 3 patterns, API comparison, and race detection guidance
- 2026-01-01 01:35 EST - Committed: refactor(test): use generated mocks to test imptest (dogfooding) - 7 files changed (+492/-124), 2 new generated mocks, all tests pass with -race

#### Documentation

**Solution**
Completed 4-step dogfooding implementation:
1. Generated MockTestReporter and refactored imp_test.go (5 tests) to external package with --dependency flag
2. Refactored race_regression_test.go proper sync tests to use TesterImp (preserved regression tests as anti-pattern documentation)
3. Generated MockTimer infrastructure for deterministic timeout testing
4. Documented dogfooding patterns in README.md with real examples

### 29. Fix missing imports for interface parameters/returns with external types

#### Universal

**Status**
done

**Description**
When generating mocks for an interface, `collectExternalImports()` uses imports from the wrong source file. It takes imports from the first file with imports in the AST file list, instead of from the file that defines the interface. This causes generated mocks to reference external types (e.g., `time.Time`, `os.FileMode`) without importing the required packages.

#### Planning

**Acceptance**
UAT-29 demonstrates cross-file external imports, generated mocks include correct imports

**Effort**
Small (2-3 hours) - actual: ~35 minutes

**Priority**
High - causes compilation errors in generated mocks

#### Work Tracking

**Started**
2026-01-01 00:43 EST

**Completed**
2026-01-01 01:18 EST

**Commit**
abd895e

**Timeline**
- 2026-01-01 00:43 EST - RED: Planning complete, starting UAT-29 creation
- 2026-01-01 01:02 EST - RED: Created UAT-29 structure (aaa_dummy.go, types.go, types_test.go)
- 2026-01-01 01:02 EST - RED: Generated mocks FAIL - missing `os` and `time` imports (as expected)
- 2026-01-01 01:13 EST - GREEN: Fixed bug - added sourceImports to ifaceWithDetails struct
- 2026-01-01 01:13 EST - GREEN: UAT-29 now passes, all 29 UAT suites passing, 0 linter errors
- 2026-01-01 01:16 EST - REFACTOR: Auditor PASS - clean implementation, 0 auditor failures
- 2026-01-01 01:18 EST - Complete: Committed abd895e, mage check clean, all tests passing

#### Documentation

**Solution**
Track sourceImports in ifaceWithDetails struct, capture imports from interface's file in getMatchingInterfaceFromAST(), use those imports in collectAdditionalImports()

**Files Modified**
- impgen/run/pkgparse.go: Added sourceImports field to ifaceWithDetails struct
- impgen/run/codegen_v2_dependency.go: Updated to use interface file's imports
- UAT/29-cross-file-external-imports/: New UAT demonstrating bug scenario
- docs/TAXONOMY.md: Added UAT-29 to directory index

#### Bug Details

**Discovered**
Real-world example in glowsync codebase - FileSystem interface uses `os.FileMode`, `time.Time` but generated mocks missing `import "os"` and `import "time"`

**Root Cause**
`codegen_v2_dependency.go:267-271` iterates through all AST files and takes the first file's imports, not the interface's file imports

#### Special Fields

**Note**
Classification: Both implementation bug AND taxonomy gap (no UAT for test-package → main-package with external types)

### 34. Fix struct literal field stripping in code generation

#### Universal

**Status**
done

**Description**
impgen strips struct literal field definitions during code generation, converting `Process(cfg struct{ Timeout int })` to `Process(cfg struct{})`, causing compiler errors in generated mocks

#### Planning

**Acceptance**
UAT-30 demonstrates struct literal params/returns work correctly, 18 unit tests verify all edge cases

**Effort**
Small (1 hour) - actual: ~6 minutes (bug discovery to fix)

**Priority**
Critical - blocks Issue #6, causes compiler errors in generated code

#### Work Tracking

**Started**
2026-01-01 03:06 EST

**Completed**
2026-01-01 03:12 EST

**Commit**
bb59a33

**Timeline**
- 2026-01-01 03:06 EST - BUG DISCOVERED: UAT-30 generation revealed struct literal fields being stripped
- 2026-01-01 03:06 EST - Created Issue #34, decided to fix NOW per EXP-005 (blocks Issue #6)
- 2026-01-01 03:06 EST - RED: Test-writer creating codegen_common_struct_test.go with 18 unit tests
- 2026-01-01 03:10 EST - RED: 18 unit tests created, 16 failing (defines stringifyStructType spec)
- 2026-01-01 03:10 EST - GREEN: Implementer adding stringifyStructType() and stringifyFuncType()
- 2026-01-01 03:12 EST - GREEN complete: All 18 unit tests pass, UAT-30 passes, bug fixed

#### Documentation

**Solution**
Implemented `stringifyStructType()` function following same pattern as `stringifyInterfaceType()` from Issue #12:
- Iterates through struct fields preserving names, types (via recursive stringifyDSTExpr), and tags
- Handles multiple names per field (e.g., `Host, Port string`)
- Handles embedded fields (no names)
- Preserves struct tags (backtick-wrapped)

**Files Modified**
- impgen/run/codegen_common.go: Added stringifyStructType() and stringifyFuncType(), updated line 942
- impgen/run/codegen_common_struct_test.go: 18 comprehensive unit tests

#### Bug Details

**Discovered**
During UAT-30 creation for Issue #6 - running `go generate` revealed all struct literal fields were lost

**Root Cause**
Line 942 in codegen_common.go had hardcoded `return "struct{}"` for all `*dst.StructType` nodes instead of iterating through fields

#### Special Fields

**Note**
Handled per EXP-005 framework (bug discovered during TDD, fixed immediately before continuing)

---

## Migrated

Issues moved to other projects.

### 14. Fix stale 'copy-files' reference in documentation (TOE-87)

#### Universal

**Status**
migrated

**Description**
Remaining reference to old 'copy-files' repository name that needs updating to 'glowsync'

#### Planning

**Acceptance**
All documentation uses correct repository name

**Effort**
Trivial

**Priority**
Low

**Linear**
TOE-87

#### Special Fields

**Migrated To**
glowsync/issues.md #16

**Note**
Migrated to glowsync project as it was mistakenly assigned to imptest

---
## Cancelled

Issues that will not be completed.


### 21. Rename --target/--dependency flags to --wrap/--mock (TOE-108)

#### Universal

**Status**
cancelled

**Description**
Current flag names describe user intent rather than what generator does. Rename to focus on action: --wrap for WrapX, --mock for MockX

#### Planning

**Acceptance**
Flags renamed, documentation updated, backward compatibility considered

**Effort**
Medium

**Priority**
Medium

**Linear**
TOE-108

---

## Blocked

Issues waiting on dependencies.

*No blocked issues*
