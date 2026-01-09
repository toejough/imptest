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

---

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
Audit current file organization andhannel message rather than return value

#### Planning

**Acceptance**
Can mock/test channel-based interactions

**Effort**
Large

**Priority**
Low

**Linear**
TOE-80

### 51. Investigate generated file consolidation in UATs

#### Universal

**Status**
backlog

**Description**
Each UAT directory contains its own generated mock/wrapper files (35+ files total). While this is correct behavior for impgen (each go:generate produces its own file), the organizational overhead may be worth investigating for consolidation.

#### Planning

**Rationale**
The coverage matrix analysis identified that generated files constitute organizational redundancy. Options to explore:
1. Keep as-is (current behavior is correct and simple)
2. Add impgen flag to consolidate multiple symbols into one file
3. Create a UAT-specific helper that runs multiple generations and combines output

**Acceptance**
- Investigation completed with clear recommendation
- If consolidation chosen: implementation plan documented
- If kept as-is: rationale documented for why current approach is preferred

**Effort**
Small (investigation) / Medium-Large (if implementing consolidation)

**Priority**
Low

**Note**
This emerged from Issue #24 analysis but was deemed lower priority than actual test redundancy removal. The current per-file approach has benefits (clear ownership, simple regeneration) that may outweigh consolidation benefits.

---

## Done

Completed issues.




### 25. Consolidate API and usage documentation (TOE-112)

#### Universal

**Status**
done

**Description**
Multiple documents (README.md, docs/V1_TO_V2_MIGRATION.md, etc.) describe API/usage with redundancy

#### Work Tracking

**Completed**
2026-01-09

**Solution**
Removed 12 outdated/redundant documentation files. Kept: README.md (user-facing), TAXONOMY.md (reference), REORGANIZATION_PROPOSAL.md (design), mutation-task.md. Historical docs available in git history.

**Files Removed**
- CALLBACKS.md, GEMINI.md, GOPLS-MCP.md, thoughts.md
- docs/API_REDESIGN.md, docs/audit-notes.md, docs/ISSUE-48-*.md
- docs/archive/PHASE2_TYPESAFE_ARGS_DESIGN.md
- imptest/RACE_*.md, imptest/REGRESSION_TEST_SUMMARY.md

---### 24. Identify and remove redundant non-taxonomy-specific UAT tests (TOE-114)

#### Universal

**Status**
done

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
### 48. Make Eventually truly async with mock.Wait()

#### Universal

**Status**
done

**Description**
Currently Eventually() expectations block until a matching call arrives. Make Eventually truly async by running expectations in goroutines, and add a `mock.Wait()` method to wait for all outstanding goroutines to complete.

#### Planning

**Rationale**
The current Eventually() pattern requires setting up expectations before the call happens, which works but isn't truly async. A fully async pattern would:
1. Allow expectations to be set up at any time
2. Run expectation matching in background goroutines
3. Provide `mock.Wait()` to block until all expectations are satisfied

**Acceptance**
- Eventually() expectations run in goroutines (non-blocking)
- `mock.Wait()` or similar blocks until all outstanding expectations complete
- Timeout handling moves to Wait() rather than per-expectation
- Existing tests continue to work (backward compatible or clear migration path)

**Effort**
Large

**Priority**
Medium

---
### 46. Add UAT for embedded structs

#### Universal

**Status**
done

**Description**
Verify wrapping/mocking structs that embed other structs. Similar to how UAT-08 tests embedded interfaces, we need to verify that struct embedding is handled correctly - the embedded struct's methods should be accessible on the outer struct.

#### Work Tracking

**Completed**
2026-01-08

**Commit**
cbf305d

**Solution**
Modified `collectStructMethods` in `pkgparse.go` to recursively collect methods from embedded structs using new `findEmbeddedStructTypes` helper. Created UAT at `UAT/variations/behavior/embedded-structs/` with Logger, Counter (base structs), and TimedLogger (embeds both). Generated mock includes all 5 methods: promoted Log, SetPrefix (from Logger), Inc, Value (from Counter), plus direct LogWithCount.

---

### 45. Add UAT for struct method as dependency (mock single method by signature)

#### Universal

**Status**
done

**Description**
Enable mocking a single struct method by extracting its signature and generating a mock for that function type. For example, `impgen mypackage.Calculator.Add --dependency` should generate `MockCalculatorAdd` with the method's signature.

#### Planning

**Rationale**
Taxonomy Layer 2 shows "Struct method as Mock" as a gap. This is similar to mocking a function - we extract the method's signature and treat it as a function type. Useful when you only need to mock one method from a struct.

**Acceptance**
- UAT demonstrates `impgen pkg.Struct.Method --dependency` generating a mock
- Generated mock has same signature as the method (minus receiver)
- Mock can be used to verify calls and inject return values

**Effort**
Small

**Priority**
Medium

**Taxonomy Gap**
Capability Matrix - "Struct method" row, "As Dependency" column

**Note**
Implementation is essentially the same as #43 (function as dependency) - extract signature, mock as function type. The difference is just where the signature comes from (method vs package-level function).

---
### 44. Add UAT for struct as dependency (mock a struct's methods)

#### Universal

**Status**
done

**Description**
Enable mocking a struct type by generating a mock that implements all its methods, similar to how interfaces are mocked. For example, `impgen mypackage.Calculator --dependency` should generate `MockCalculator` with mock versions of all Calculator's methods.

#### Planning

**Rationale**
Taxonomy Layer 2 shows "Struct (whole) as Mock" as a gap. In Go, you can't inject a struct like an interface, but you can extract its methods and create an interface-like mock. This enables testing code that depends on struct method behavior.

**Acceptance**
- UAT demonstrates `impgen pkg.SomeStruct --dependency` generating a mock
- Generated mock has all methods from the original struct
- Each method can be expected/verified like interface mocks

**Effort**
Medium

**Priority**
Medium

**Taxonomy Gap**
Capability Matrix - "Struct (whole)" row, "As Dependency" column

---
### 49. Audit codebase for remaining V1 references

#### Universal

**Status**
done

**Description**

### 50. Add function type as dependency mock support

#### Universal

**Status**
done

**Description**

### 43. Add UAT for function as dependency (mock a function by signature)

#### Universal

**Status**
done

**Description**
Enable mocking a package-level function by extracting its signature and generating a mock for that function type. For example, `impgen mypackage.ProcessOrder --dependency` should generate `MockProcessOrder` with the same signature.

#### Planning

**Rationale**
Taxonomy Layer 2 shows "Function as Mock" as a gap. Users may want to mock a function without first defining a named function type. The implementation should extract the function's signature and treat it as an implicit function type.

**Acceptance**
- UAT demonstrates `impgen pkg.SomeFunction --dependency` generating a mock
- Generated mock has same signature as original function
- Mock can be used to verify calls and inject return values

**Effort**
Medium

**Priority**
Medium

**Taxonomy Gap**
Capability Matrix - "Function" row, "As Dependency" column

---
### 47. Restructure TAXONOMY.md around three-layer mental model

#### Universal

**Status**
done

**Description**
Restructure TAXONOMY.md to lead with "what are you trying to do" (testing patterns) rather than "what does imptest support" (types). This aligns documentation with how users think and makes navigation clearer.

#### Work Tracking

**Started**
2026-01-04 20:49 EST

**Completed**
2026-01-04 20:53 EST

**Timeline**
- 2026-01-04 20:49 EST - Started: Beginning docs-first restructure of TAXONOMY.md
- 2026-01-04 20:53 EST - Complete: Restructured TAXONOMY.md with three-layer model

#### Documentation

**Solution**
Restructured TAXONOMY.md following the three-layer mental model from REORGANIZATION_PROPOSAL.md:
- Layer 1: Testing Pattern (wrapper vs mock) as primary organization
- Layer 2: Symbol Type (function, method, struct, interface, functype)
- Layer 3: Variations (package, signature, behavior, concurrency)

Key changes:
- Added Quick Start with decision tree
- Reorganized around Testing Patterns (not type capabilities)
- Updated all examples to current call handle API (Issue #42)
- Reorganized UAT index by pattern, symbol type, and variation

---

### 1. remove the deprecation messages and any hint of V1/V2

#### Universal

**Status**
done

#### Work Tracking

**Completed**
2026-01-04 14:38 EST

**Timeline**
- 2026-01-01 18:06 EST - test entry
- 2026-01-04 14:12 EST - GREEN: Starting V1/V2 cleanup (combined with #16)
- 2026-01-04 14:38 EST - Complete: V1/V2 cleanup done

**Combined With**
#16

### 16. Remove 'v2' from file and package names (TOE-109)

#### Universal

**Status**
done

**Description**
Now that V2 is the only implementation, remove "v2" suffix from filenames (codegen_v2_dependency.go → codegen_dependency.go, etc.)

#### Work Tracking

**Completed**
2026-01-04 14:38 EST

**Timeline**
- 2026-01-04 14:38 EST - Complete: File renames and internal reference updates

**Combined With**
#1

### 15. Consolidate duplicate Tester and TestReporter interfaces (TOE-105)

#### Universal

**Status**
done

**Description**
`Tester` and `TestReporter` interfaces in imptest/controller.go and imptest/imp.go were identical, causing code confusion.

#### Work Tracking

**Completed**
2026-01-04 15:12 EST

**Solution**
Consolidated to single `TestReporter` interface in controller.go. Removed duplicate interface definition from imp.go along with unnecessary `testerAdapter`. Deleted old generated mock files (`generated_MockTester_test.go`, `generated_TesterImp_test.go`) and updated tests to use `MockTestReporter`.

**Files Modified**
- imptest/controller.go: Renamed Tester → TestReporter throughout
- imptest/imp.go: Removed duplicate TestReporter, removed testerAdapter
- imptest/controller_test.go: Updated to use MockTestReporter
- imptest/race_regression_test.go: Updated tests to use simplified mock pattern
- Deleted: generated_MockTester_test.go, generated_TesterImp_test.go

**Linear**
TOE-105

---

### 40. Extract shared generator methods to reduce code duplication

#### Universal

**Status**
done

**Description**
Extract duplicated methods from generators to `codegen_common.go`.

#### Work Tracking

**Completed**
2026-01-04 14:56 EST

**Solution**
Main consolidation already complete: `buildParamStrings()`, `buildResultStrings()`, `checkIfQualifierNeeded()`, and `collectAdditionalImportsFromInterface()` are on `baseGenerator` in `codegen_common.go`. Remaining ~46 lines of thin adapter code in dependency/interface-target generators is minor and not worth further consolidation.

### 42. Redesign --target wrapper pattern for goroutine lifecycle management

#### Universal

**Status**
done

**Description**
Current --target wrappers are built for call observation/mocking (with GetCalls(), call history, parameter tracking) when the actual need is goroutine lifecycle management (run method in goroutine, capture returns/panics, coordinate with imptest controller).

#### Planning

**Rationale**
Discovered during Issue #33: The wrapper pattern implements the wrong abstraction. All --target wrappers (functions, interfaces, structs) have unnecessary complexity that doesn't match the actual use case.

**Actual Requirements**:
- One goroutine per method call
- Start() launches goroutine running the target method
- Signal to imptest controller when method returns (for ordering verification with mocked dependencies)
- GetResult() retrieves return values (blocking wait)
- ExpectPanicEquals() verifies panics
- NO call history tracking (GetCalls())
- NO parameter recording
- NO timeouts/cancellation

**Current Problems**:
- GetCalls() returns all call history - never needed
- Parameters tracked and recorded - never needed
- Built for observation pattern instead of goroutine coordination

**Acceptance**
- Wrapper API focused on goroutine lifecycle: Start() -> goroutine -> GetResult()/ExpectPanic()
- Integrates with imptest controller for event ordering
- No call history or parameter tracking
- Simpler generated code
- All existing UATs updated to new API

**Effort**
Large

**Priority**
Medium

**Dependencies**
None - can proceed immediately

**Timeline**

- 2026-01-03 09:55 EST - Created issue after discovering abstraction mismatch in Issue #33
- 2026-01-03 13:02 EST - RED: Writing test for function wrapper call handles
- 2026-01-03 13:06 EST - GREEN: Implementing call handle pattern for function wrappers
- 2026-01-03 13:48 EST - REFACTOR: Auditing call handle implementation
- 2026-01-03 14:01 EST - GREEN: Updating 5 UATs to new call handle API
- 2026-01-03 14:06 EST - REFACTOR: Final audit of UAT updates
- 2026-01-03 14:29 EST - Complete: Function wrapper call handle pattern (Step 1 of 5)
- 2026-01-03 14:29 EST - COMMIT: Step 1 complete - call handle pattern for functions
- 2026-01-03 14:51 EST - RED: Writing tests for interface/struct wrapper call handles
- 2026-01-03 15:12 EST - GREEN: Implementing call handle pattern for interface/struct wrappers (Step 3 of 5)
- 2026-01-03 15:24 EST - REFACTOR: Auditing interface/struct wrapper implementation
- 2026-01-03 15:32 EST - GREEN: Fixing UAT-32, UAT-33, and race condition
- 2026-01-03 15:40 EST - REFACTOR: Final audit after UAT fixes
- 2026-01-03 15:45 EST - Complete: Interface/struct wrapper call handle pattern (Step 3 of 5)
- 2026-01-03 15:46 EST - COMMIT: Step 3 complete - interface/struct call handle pattern
- 2026-01-04 13:46 EST - Marked done

**Completed**
2026-01-04

**Commits**
- Step 1: Function wrapper call handles
- Step 3: Interface/struct wrapper call handles (3895fea)

---

### 17. Remove stale .out and .test files (TOE-110)

#### Universal

**Status**
done

**Description**
Audit repository for stale build artifacts (_.out, _.test) and add to .gitignore

#### Planning

**Acceptance**
No build artifacts in version control, .gitignore updated

**Effort**
Trivial

**Priority**
Low

**Linear**
TOE-110

**Completed**
2026-01-04

**Solution**
Already resolved - .gitignore already had `*.test` (line 9) and `*.out` (line 12) patterns. Verified no such files are tracked in git. Local build artifacts (4 .out, 5 .test files) are correctly ignored.
### 33. Add UAT for struct type as target (comprehensive)

#### Universal

**Status**
done

**Description**
Verify wrapping struct types with --target flag comprehensively (beyond just methods)

#### Planning

**Rationale**
Taxonomy matrix shows "?" for "Struct type as Target", partially covered by UAT-32

**Acceptance**
UAT demonstrates wrapping struct types with --target flag

**Effort**
Small

**Priority**
Medium

**Commit**
ae017ce

- timeline:
  - 2026-01-03 09:32 EST - Complete: Struct type wrapping with --target flag (ae017ce)
  - 2026-01-03 09:28 EST - REFACTOR: Auditor PASS - ready for commit
  - 2026-01-03 09:26 EST - REFACTOR: Auditor review of all linter fixes
  - 2026-01-03 09:19 EST - REFACTOR: Fixing 4 pre-existing linter violations to get mage check clean
  - 2026-01-03 09:14 EST - REFACTOR: Auditor passed UAT-33 (0 violations), found 4 pre-existing issues
  - 2026-01-03 09:11 EST - REFACTOR: Fixed all 7 linter violations in UAT-33
  - 2026-01-03 09:07 EST - REFACTOR: Auditor found 7 linter issues in UAT-33 files
  - 2026-01-03 08:37 EST - REFACTOR: Removed Interface() method from struct and interface wrapping
  - 2026-01-02 21:24 EST - REFACTOR: User clarified - Interface() not needed, only method wrappers
  - 2026-01-02 21:18 EST - REFACTOR: Hit Go type system limitation with Interface() for structs
  - 2026-01-02 20:56 EST - GREEN: Implementation complete - 7/8 tests pass
  - 2026-01-02 20:53 EST - RED: Created UAT-33 with 8 comprehensive test cases
  - 2026-01-02 20:38 EST - PLAN MODE: Understanding struct type wrapping requirements

---

### 41. Fix issuesstatus command file corruption bug

#### Universal

**Status**
done

**Description**
The `mage issuesstatus` command has a severe bug that corrupts issues.md when moving issues between statuses. It truncates issue content, creates malformed headers, and can cause issues to disappear. Need to investigate root cause and fix or replace the command.

#### Planning

**Acceptance**
Command can move issues between statuses without corrupting the file

**Effort**
Small

**Priority**
High

- timeline:
  - 2026-01-02 20:26 EST - Complete: Fixed issuesstatus - simplified to update status field + call IssuesFix()
  - 2026-01-02 20:24 EST - GREEN: Tested bidirectional movement (in progress ↔ backlog), all tests pass
  - 2026-01-02 20:20 EST - GREEN: Implemented simplified version, eliminated fragile boundary detection
  - 2026-01-02 20:20 EST - PLAN MODE: Investigating issuesstatus corruption bug

#### Documentation

**Root Cause**
Original implementation used complex boundary detection to extract/move issues manually. This was fragile and prone to corruption when calculating offsets.

**Solution**
Simplified IssuesStatus() to:
1. Validate status value
2. Update ONLY the **Status** field in place
3. Delegate section movement to battle-tested IssuesFix()

**Result**
- 164 lines → 80 lines (51% reduction)
- No fragile boundary detection
- Leverages proven IssuesFix() logic
- Tested: bidirectional movement works flawlessly
- mage check: 0 issues

---

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
- 2026-01-01 16:12 EST - Used test-writer agent - wrote 10 comprehensive failing tests (RED phasnference, and ambiguity detection
- 2025-12-31 ~14:40 - Testing: All UAT-11 tests passing
- 2025-12-31 ~14:45 - Refactor: Fixed linter errors (err113, lll, nlreturn, wsl_v5, staticcheck)
- 2025-12-31 ~00:50 - Committed: fix(impgen): resolve stdlib package shadowing ambiguity

#### Documentation

**Solution**
Implemented 4-tier package resolution strategy:

1. Explicit --import-path flag lds

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

**Acceptan-31 01:35 EST - REFACTOR: Fixed linter errors (err113, nlreturn, wrapcheck)
- 2025-12-31 01:40 EST - Complete: All tests passing, mage check clean

#### Special Fields

**Note**
After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Named" parameters/returns as "Yes" with UAT reference

### 5. Add UAT for interface literal parameters

#### Universal

**Status**
done

**Description**
Verify interface literals in signatures are handled (e.g., `func Process(obj interface{ Get() strine failing tests
- 2025-12-31 10:30 EST - Generated mocks with impgen, discovered critical bug
- 2025-12-31 10:45 EST - BUG FIX: Implemented `stringifyInterfaceType` to preserve interface literal method signatures
- 2025-12-31 11:00 EST - GREEN: All 5 test cases passing (single-method, multi-method, error returns, return types, matchers)
- 2025-12-31 11:15 EST - REFACTOR: Fixed 10 linter errors (funlen, cyclop, nestif, inamedparam, nlreturn, revivption**
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

**Comctions with struct literal params/returns)
- 2026-01-01 03:06 EST - BUG DISCOVERED: impgen strips struct literal fields during generation (Process(cfg struct{ Timeout int }) → Process(cfg struct{}))
- 2026-01-01 03:06 EST - Created Issue #34 to track struct literal bug (blocking Issue #6 per EXP-005)
- 2026-01-01 03:06 EST - RED: Test-writer creating codegen_common_struct_test.go with 18 comprehensive unit  4 (code quality review)
- 2026-01-01 11:05 EST - REFACTOR: Auditor found 15 linter violations (5 wsl_v5 + 10 others), routing to implementer
- 2026-01-01 11:05 EST - REFACTOR: Routing to implementer to fix linter violations
- 2026-01-01 11:14 EST - REFACTOR: All 15 linter violations fixed, mage check passes with 0 issues
- 2026-0 - Routing to git-workflow to commit TAXONOMY.md update

#### Special Fields

**Note**
After completing UAT, update Signature Variations Matrix in TAXONOMY.md to mark "Struct literal" as "Yes" with UAT reference OR add to "Cannot Do" section if unsupported

### 7. Support dot imports for mocking

#### Universal

**Status**
done

**Description**
Enable impgen UAT-26 demonstrates mocking of dot-imported symbols (Storage, Processor interfaces)

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

**DiscT-25, compiler errors showed stripped signatures
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
UAT-2larified: no timeout at all, ordered mode should fail-fast - 2025-12-31 14:20 EST: DESIGN - Solution-architect redesigned with ordered vs eventually modes - 2025-12-31 14:25 EST: DESIGN complete - User approved: ordered (fail-fast) vs eventually (queue) - 2025-12-31 14:30 EST: PLANNING phase - Routed to solution-planner - 2025-12-31 14:35 EST: PLANNING - Architectural decisions confirmed (hard break, wait forever, per-expectation, FIFO) - 2025-12-31 14:45 EST: PLANNING complete - 13-step plan created (~10 hours estimated) - 2025-12-31 14:50 EST: RED phase - Step 1: test-writer writing tests for failOnMismatch - 2025-12-31 14:55 EST: RED complete - 2 failing tests written (GetCallOrdered, GetCallEventually) - 2025-12-31 15:00 EST: GREEN phase - Step 1: implementer adding failOnMismatch to waiter struct - 2025-12-31 15:05 EST: GREEN complete - Step 1 done: waiter struct extended, GetCallOrdered implemented, tests passing - 2025-12-31 15:10 EST: AUDIT phase - Step 1: auditor reviewing code quality - 2025-12-31 15:12 EST: AUDIT PASS - Step 1: Clean implementation, thr12-31 15:43 EST: AUDIT phase - Step 3: auditor reviewing dispatcher logic - 2025-12-31 15:46 EST: AUDIT PASS - Step 3: Correct logic, thread-safe, 100% coverage, ready for next phase - 2025-12-31 15:48 EST: Phase 1 (Controller Layer) COMPLETE - Steps 1-3 done, moving to Phase 2 (Imp Layer) - 2025-12-31 15:50 EST: RED phase - Step 5: test-writer writing tests for Imp layer methods - 2025-12-31 15:53 EST: RED complete - Step 5: 3 failing tests written (GetCallOrdered/Eventually on Imp) - 2025-12-31 15:55 EST: GREEN phase - Step 5: implementer adding GetCallOrdered/Eventually to Imp - 2025-12-31 15:58 EST: GREEN complete - Step 5: Imp layer methods implemented, all tests passing - 2025-12-31 16:00 EST: AUDIT phase - Step 5: auditor reviewing Imp layer - 2025-12-31 16:02 EST: AUDIT NOTE - Step 5: Tests pass, some linter warnings (may be pre-existing), continuing - 2025-12-31 16:05 EST: Phases 1-2 CHECKPOINT - Controller & Imp layers functional, committing progress - 2025-12-31 16:08 EST: COMMITTED - feat(imptest): implement ordered vs eventually call matching modes (3629eeb) - 2025-12-31 16:10 EST: Phase 3 START - DependencyMethod layer: removing timeout, adding eventually flag - 2025-12-31 16:12 EST: RED phase - Step 6: test-writer writing tests for DependencyMethod transformation - 2025-12-31 16:15 EST: RED complete - Step 6: 3 failing tests written (Ev25-12-31 17:49 EST: Steps 9-10 SKIP - UAT test updates already completed in Step 7 - 2025-12-31 17:50 EST: Step 11 START - Create new UAT test demonstrating ordered vs eventually modes - 2025-12-31 17:52 EST: GREEN - implementer: Created UAT/28 directory structure - 2025-12-31 17:53 EST: GREEN - implementer: Created service.go interface - 2025-12-31 17:55 EST: GREEN - implementer: Created modes_test.go with 6 comprehensive tests - 2025-12-31 17:58 EST: GREEN - implementer: Generated mock, fixed test issues - 2025-12-31 18:00 EST: GREEN - All 5 UAT-28 tests passing with race detector - 2025-12-31 18:01 EST: Step 11 complete - UAT-28 demonstrates ordered vs eventually modes - 2025-12-31 18:02 EST: Step 12 START - Update documentation and comments - 2025-12-31 18:05 EST: Step 12 - Updated README.md (3 Eventually() examples) - 2025-12-31 18:06 EST: Step 12 - Updated V1_TO_V2_MIGRATION.md - 2025-12-31 18:07 EST: Step 12 complete - All documentation updated - 2025-12-31 18:08 EST: Step 13 START - Run full test suite and verify - 2025-12-31 18:20 EST: BLOCKER FOUND - Regression in commit 3629eeb: fail-fast logic has bugs - 2025-12-31 18:21 EST: Bug details: TestImpGetCallOrdered_WrongMethod, TestDispatchLoop_OrderedFailsEventuallyWaits failing - 2025-12-31 18:22 EST: Root cause: matched=true fix reveals deeper dispatcher logic issues - 2025-12-31 18:23 EST: STATUS: Steps 7-12 COMPLETE and w22:20 EST: FIX - All 3 fixed tests pass with -race (0 races detected) - 2025-12-31 22:22 EST: VERIFIED - All production tests pass with -race (excluding intentional regression tests) - 2025-12-31 22:23 EST: STATUS: Race conditions FIXED, all tests passing - 2025-12-31 22:25 EST: MAGE CHECK - Updated magefile to skip TestRaceRegression tests in CI (using -skip flag) - 2025-12-31 22:30 EST: CLEANUP - Removed deprecated GetCallWithTimeout (0% coverage, no callers) - 2025-12-31 22:35 EST: LINTING - Fixed 14 new linting issues in race_regression_test.go (wsl whitespace + revive) - 2025-12-31 22:40 EST: STATUS: Race fix work COMPLETE - 35 pre-existing linting issues remain (tracked separately) - 2025-12-31 22:45 EST: LINTING - Fixed all 35 pre-existing linting issues (err113, wsl, nolintlint, staticcheck, nlreturn, noinlineerr, funlen, revive, errcheck)

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
AudFACTOR: Step 4 COMPLETE ✓ - Added comprehensive "How imptest Tests Itself (Dogfooding)" section to README.md with 3 patterns, API comparison, and race detection guidance
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
- 2026-01-01 01:02 EST - RED: Generated mocks FAIL - missing `os` and `time` impks Issue #6, causes compiler errors in generated code

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

- impgen/run/codegen_common.go: Added stringifyStructType() and stringifyFuncType(), updated linddress in next refactoring opportunity or when extending interface wrapper functionality

**Note**
Identified by project-health-auditor. This inconsistency was introduced in Issue #32 to ship the feature quickly. Aligning with the established template pattern will make future maintenance easier.
#### Work Tracking

**Timeline**
- 2026-01-02 18:24 EST - Complete: Creating git commit for all linter fixes
- 2026-01-02 18:18 EST - Complete: Fixed all linter issues (ours + pre-existing + dupl violations)
- 2026-01-02 16:47 EST - Complete: Auditor passed, ready for commit
- 2026-01-02 16:45 EST - REFACTOR: Implementation complete, routing to auditor
- 2026-01-02 15:41 EST - GREEN: Implementing Option 1 (Full Template Migration)
- 2026-01-02 15:32 EST - PLAN MODE: Designing template migration approach



### 31. Add UAT for function type as dependency

#### Universal

**Status**
done

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

#### Work Tracking

**Completed**
2026-01-01

**Commit**
ad7a465, 96f7104

**Timeline**
- 2026-01-01 22:34 EST - Complete: TAXONOMY.md updated (96f7104), function type as dependency marked as supported
- 2026-01-01 22:33 EST - Committed: UAT-31 test files (ad7a465) - all 8 tests passing
- 2026-01-01 22:14 EST - REFACTOR: Auditor PASS - all tests passing, 0 violations in UAT-31, ready to commit
- 2026-01-0ot be reproduced with current code - appears to have been inadvertently fixed
- 2026-01-01 18:05 EST - Testing: Multiple test cases all work correctly, timeline entries added in proper location
- 2026-01-01 18:02 EST - RED: Unable to reproduce bug - existing code works correctly in tests
- 2026-01-01 17:53 EST - RED: Creating regression tests to demonstrate orphaning bug
- 2026-01-01 17:52 EST - Started: Investigating IssuesTimeline function to reproduce and fix bug

#### Documentation

**Solution**
Bug could not be reproduced with current code. Multiple test cases (issues with and without existing Work Tracking sections) all correctly placed timeline entries within the issue's Work Tracking section.

Likely inadvertently fixed by recent boundary detection improvements in IssuesStatus function (Issue #38), which uses similar insertion point calculation logic.

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
- 2026-01-01 17:18 EST - failure-debugger identified root cause andl

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

### 19. Fix Eventually() type loss in concurrent call matching (TOE-104)

#### Universal

**Status**
done

**Description**
Eventually() returns base \*DependencyCall instead of typed wrapper, losing type-safe GetArgs() access

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

**Timeline** - 2025-12-31 14:00 EST: DESIGN phase - Routed to solution-architect - 2025-12-31 14:15 EST: DESIGN - User clarified: no timeout at all, ordered mode should fail-fast - 2025-12-31 14:20 EST: DESIGN - Solution-architect redesigned with ordered vs eventually modes - 2025-12-31 14:25 EST: DESIGN complete - User approved: ordered (fail-fast) vs eventually (queue) - 2025-12-31 14:30 EST: PLANNING phase - Routed to solution-planner - 2025-12-31 14:35 EST: PLANNING - Architectural decisions confirmed (hard break, wait forever, per-expectation, FIFO) - 2025-12-31 14:45 EST: PLANNING complete - 13-step plan created (~10 hours estimated) - 2025-12-31 14:50 EST:tep 3: 3 tests written, 2 failing (exposed dispatcher missing fail-fast priority) - 2025-12-31 15:38 EST: GREEN phase - Step 3: implementer fixing dispatcher fail-fast priority logic - 2025-12-31 15:41 EST: GREEN complete - Step 3: Dispatcher fixed, all 19 tests passing, FIFO with fail-fast working - 2025-12-31 15:43 EST: AUDIT phase - Step 3: auditor reviewing dispatcher logic - 2025-12-31 15:46 EST: AUDIT PASS - Step 3: Correct logic, thread-safe, 100% coverage, ready for next phase - 2025-12-31 15:48 EST: Phase 1 (Controller Layer) COMPLETE - Steps 1-3 done, moving to Phase 2 (Imp Layer) - 2025-12-31 15:50 EST: RED phase - Step 5: test-writer writing tests for Imp layer methods - 2025-12-31 15:53 EST: RED complete - Step 5: 3 failing tests written (GetCallOrdered/Eventually on Imp) - 2025-12-31 15:55 EST: GREEN phase - Step 5: implementer adding GetCallOrdered/Eventually to Imp - 2025-12-31 15:58 EST: GREEN complete - Step 5: Imp layer methods implemented, all tests passing - 2025-12-31 16:00 EST: AUDIT phase - Step 5: auditor reviewing Imp layer - 2025-12-31 16:02 EST: AUDIT NOTE - Step 5: Tests pass, some linter warnings (may be pre-existing), continuing - 2025-12-31 16:05 EST: Phases 1-2 CHECKPOINT - Controller & Imp layers functional, committing progress - 2025-12-31 16:08 EST: COMMITTED - feat(imptest): implement ordered vs eventually call matching modes (3629eeb) - 2025-12-31 16:10 EST: Phase 3 START - DependencyMethod layer: removing timeout, adding eventually flag - 2025-12-31 16:12 EST: RED phase - Step 6: test-writer writing tests for DependencyMethod transformation - 2025-12-31 16:15 EST: RED complete - Step 6: 3 failing tests written (Eventually() API transformation) - 2025-12-31 16:18 EST: GREEN phase - Step 6: implementer transforming DependencyMethod to mode-based - 2025-12-31 16:22 EST: GREEN complete - Step 6: Timeout removed! Eventually() now no-param, mode-based API working - 2025-12-31 15:23 EST: MILESTONE - Issue #20 core complete: Eventually() has NO timeout parameter! - 2025-12-31 15:25 EST: COMMITTED - feat(imptest): remove timeout from Eventually() (46c6966) - 2025-12-31 15:27 EST: Phase 4 START - Code generation: Add typed Eventually() to templates for Issue #19 - 2025-12-31 15:27 EST: GREEN phase - Routed to implementer for template changes - 2025-12-31 15:28 EST: GREEN - implementer: Added Eventually() to v2DepMethodWrapperTmpl - 2025-12-31 15:29 EST: GREEN - implementer: Updated 5 UAT test files (removed timeout parameter) - 2025-12-31 15:30 EST: GREEN - implementer: Regenerated mocks, created eventually_test.go - 2025-12-31 15:31 EST: GREEN - All UAT tests passing - 2025-12-31 17:40 EST: Step 7 complete: Added typed Eventually() to templates, Issue #19 COMPLETE - 2025-12-31 17:45 EST: COMMITTED - feat(imptest): add type-safe Eventually() to generated mocks (d66a013) - 2025-12-31 17:47 EST: Step 8 START - Cleanup: Remove remaining timeout-related template code - 2025-12-31 17:49 EST: Step 8 complete - Verified no timeout references remain (all cleaned in Step 7) - 2025-12-31 17:49 EST: Steps 9-10 SKIP - UAT test updates already completed in Step 7 - 2025-12-31 17:50 EST: Step 11 START - Create new UAT test demonstrating ordered vs eventually modes - 2025-12-31 17:52 EST: GREEN - implementer: Created UAT/28 directory structure - 2025-12-31 17:53 EST: GREEN - implementer: Created service.go interface - 2025-12-31 17:55 EST: GREEN - implementer: Created modes_test.go with 6 comprehensive tests - 2025-12-31 17:58 EST: ve timeout parameter from Eventually() (TOE-107)

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




### 36. Split issue tracker into separate repository

#### Universal

**Status**
cancelled

**Description**

#### Work Tracking

**Timeline**

- 2026-01-01 18:03 EST - Testing timeline orphaning
- 2026-01-01 17:09 EST - Implementer fixed auditor issues + discovered and fixed 3 additional bugs (insertion point, horizontal rule boundary, section header orphan)
- 2026-01-01 17:09 EST - Auditor FAILED - found ineffectual assignment and test data bug
- 2026-01-01 17:09 EST - Implementer applied fix - changed boundary detection to 'first boundary wins' logic
### 11. Prevent ExpectCalledWithExactly generation for function types

#### Universal

**Status**
cancelled

**Description**
impgen generates ExpectCalledWithExactly() methods for function type parameters, but Go cannot compare functions with `==`, causing reflect.DeepEqual to hang. Since we perform code generation based on known types, we should detect function types and skip generating equality methods.

Workaround: Use ExpectCalledWithMatches() with imptest.Any() for function parameters

**Cancellation Reason**
Not worth the extra complexity right now. Workaround (using imptest.Any()) is sufficient.

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
### 37. Explore mage replacement with subcommands and flags (go-arg syntax)

#### Universal

**Status**
cancelled

**Description**

#### Work Tracking

**Timeline**

- 2026-01-01 18:03 EST - Testing without Work Tracking section
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


*No blocked issues*
Issues waiting on dependencies.

_No blocked issues_
  - 2026-01-03 09:14 EST - REFACTOR: Re-auditing after fixing all 7 linter violations
- 2026-01-03 15:41 EST - REFACTOR: Final audit after UAT fixes

