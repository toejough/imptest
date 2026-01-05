# Codebase Reorganization Proposal

**Status:** Draft for discussion
**Created:** 2026-01-04
**Purpose:** Align codebase structure, documentation, and features around a cleaner mental model

---

## Table of Contents

- [Current State Analysis](#current-state-analysis)
- [The Conceptual Gap](#the-conceptual-gap)
- [Proposed Mental Model](#proposed-mental-model)
- [Taxonomy Gaps and Ambiguities](#taxonomy-gaps-and-ambiguities)
- [Proposed File Structure](#proposed-file-structure)
- [Proposed UAT Reorganization](#proposed-uat-reorganization)
- [Proposed Taxonomy Restructure](#proposed-taxonomy-restructure)
- [Migration Path](#migration-path)
- [Open Questions](#open-questions)
- [Decisions](#decisions)

---

## Current State Analysis

### What We Have

| Artifact       | Organization Principle                                       |
| -------------- | ------------------------------------------------------------ |
| TAXONOMY.md    | Type → Package → Signature → Concurrency                     |
| File structure | Flag + Type (codegen_dependency.go, codegen_target.go, etc.) |
| UATs           | Numbered 01-42, loosely by complexity                        |

### The Core Tension

These three don't align on the same primary dimension:

- Docs organize by **what Go construct** you're working with
- Code organizes by **what flag** you're using + type
- UATs organize by **when they were created** (roughly)

---

## The Conceptual Gap

### How Taxonomy Frames Things

Around **what Go construct** you're working with:

- Functions, function types, interfaces, structs

### How Users Think

In terms of **what they're trying to accomplish**:

- "I want to test this function"
- "I need to mock that dependency"
- "I want to verify these callbacks"

### The Result

The `--target` vs `--dependency` flags are actually about **testing patterns**, but they're presented as if they're just different output modes for the same thing.

---

## Proposed Mental Model

### Three Layers

**Layer 1: Testing Pattern** (primary - what are you trying to do?)

| Pattern     | Description                                | Flag           | Output       |
| ----------- | ------------------------------------------ | -------------- | ------------ |
| **Wrapper** | Wrap code under test to observe/control it | `--target`     | `Wrap{Name}` |
| **Mock**    | Mock a dependency the code uses            | `--dependency` | `Mock{Name}` |

**Layer 2: Symbol Type** (what Go construct?)

| Symbol         | As Wrapper | As Mock | Notes                                    |
| -------------- | ---------- | ------- | ---------------------------------------- |
| Function       | Yes        | Yes     | Mock extracts signature as function type |
| Function type  | Yes        | Yes     | Named type for callbacks                 |
| Interface      | Yes        | Yes     | Mocking is common; wrapping less common  |
| Struct (whole) | Yes        | Yes     | Wrap/mock all methods at once            |
| Struct method  | Yes        | Yes     | Mock extracts signature as function type |

**Key insight:** Mocking a function or method is equivalent to mocking a function type - we just derive the type from the symbol's signature. This means the underlying mechanism is the same; we're just giving users different entry points based on what they have in their code.

**Layer 3: Variations** (special handling)

| Category    | Variations                                      |
| ----------- | ----------------------------------------------- |
| Package     | same, different, stdlib, shadowed, dot-imported |
| Signature   | generics, variadic, callbacks, non-comparable   |
| Concurrency | ordered (default), Eventually()                 |

### Why This Model?

1. **Layer 1** answers: "What am I trying to do?"
2. **Layer 2** answers: "What Go thing am I working with?"
3. **Layer 3** answers: "Are there any special cases?"

Users can navigate: Pattern → Symbol → Variation

---

## Taxonomy Gaps and Ambiguities

### 1. The `?` Marks in Capability Matrix

Current:

```
| What | As Target | As Dependency |
| Function | Yes | ? |
| Interface | ? | Yes |
| Struct type | Yes | ? |
```

These `?` marks represent gaps in implementation or documentation, not fundamental limitations:

- **Function as dependency** → Issue #43: extract signature, mock as function type
- **Interface as target** → Works (UAT-32)
- **Struct as dependency** → Issue #44: mock all methods like an interface
- **Struct method as dependency** → Issue #45: extract signature, mock as function type
- **Struct as target (whole)** → Works (UAT-33)

**Issues Created:**

- #43: Function as dependency (mock a function by signature)
- #44: Struct as dependency (mock a struct's methods)
- #45: Struct method as dependency (mock single method by signature)

**Recommendation:** Implement these three issues to fill the matrix. The underlying mechanisms (function type mocking, method wrapping) can serve multiple entry points.

### 2. Missing: Why You'd Choose Each Pattern

The taxonomy explains _what_ works but not _when_ to use each approach.

**Recommendation:** Add decision guidance:

- Testing a `func ProcessOrder(...)` → Use `--target` wrapper
- Testing code that accepts an `OrderService` interface → Use `--dependency` mock
- Testing a function that takes a callback → Combine both

### 3. Call Handle Pattern Not Reflected

UAT-42 introduces significant API change but TAXONOMY.md shows old pattern:

```go
// Old (in current docs)
wrapper.ExpectReturnsEqual(expectedOutput)

// New (UAT-42 pattern)
call := wrapper.Start(input)
call.ExpectReturnsEqual(expectedOutput)
```

**Recommendation:** Update all examples to new pattern once it's fully implemented.

### 4. "Cannot Do" Section Mixes Current and Historical

Sections 3-5 are marked "Now Supported" but remain in "Cannot Do":

- Stdlib Package Shadowing (Now Supported)
- Dot Imports (Now Supported)
- Multi-Parameter Function Literals (Now Fixed)

**Recommendation:** Move to main matrices, keep only true limitations in "Cannot Do".

---

## Proposed File Structure

> **Note:** This section contains an earlier proposal organized by mental model layers. See [Decision: Flow-Oriented Structure](#decision-flow-oriented-structure) for the chosen approach, which organizes by execution flow instead.

### Current

```
impgen/run/
├── codegen_dependency.go       # Mock generator
├── codegen_target.go           # Function wrapper
├── codegen_interface_target.go # Interface wrapper
├── codegen_struct_target.go    # Struct wrapper
├── codegen_interface.go        # Interface discovery
├── codegen_common.go           # Shared utilities
├── templates.go                # Registry
├── text_templates.go           # Template definitions
├── pkgparse.go                 # Symbol detection
├── pkgload.go                  # Package loading
└── run.go                      # Entry point
```

### Proposed

```
impgen/
├── main.go
├── run/
│   ├── run.go                 # Entry point, routing
│   │
│   ├── symbols/               # Layer 2: Symbol detection
│   │   ├── detect.go          # Find symbol in package, route to source
│   │   └── sources/           # One file per symbol type (all 5)
│   │       ├── function.go    # Package-level function
│   │       ├── method.go      # Struct method
│   │       ├── struct.go      # Struct (all methods)
│   │       ├── interface.go   # Interface
│   │       └── functype.go    # Named function type
│   │
│   ├── packages/              # Layer 3: Package resolution
│   │   ├── resolve.go         # 4-tier resolution
│   │   └── imports.go         # Import management
│   │
│   ├── generators/            # Layer 1: Testing patterns
│   │   ├── wrapper.go         # --target pattern
│   │   ├── mock.go            # --dependency pattern
│   │   └── common.go          # Shared generation
│   │
│   └── templates/             # Output templates (by signature count, not type)
│       ├── wrapper/
│       │   ├── single.tmpl    # function, method, functype (one signature)
│       │   └── multi.tmpl     # struct, interface (multiple signatures)
│       └── mock/
│           ├── single.tmpl    # function, method, functype (one signature)
│           └── multi.tmpl     # struct, interface (multiple signatures)
```

**Template insight:** The real distinction isn't symbol type, it's signature count:

- **Single-signature:** function, method, function type → one callable to wrap/mock
- **Multi-signature:** struct, interface → multiple methods to wrap/mock

This means 2 templates per pattern, not 5.

### Benefits

- Top-level directories map to the three-layer mental model
- `generators/` contains the two patterns users choose between
- `symbols/` is internal (users don't care, but code needs it)
- `templates/` organized by pattern + type (easy to find/modify)

### Concerns / Trade-offs

- [ ] Significant refactor - is the alignment benefit worth the churn?
- [ ] Breaks familiarity for anyone who knows current structure
- [ ] May not be necessary if we're deprecating V1 patterns anyway

---

## Proposed UAT Reorganization

### Current

Numbered 01-42, loosely by complexity:

- 01-14: Core functionality
- 15-22: Advanced features
- 23-31: Type-specific features
- 32-33, 42: Newer patterns

### Proposed

Organized by testing pattern and feature:

```
UAT/
├── wrapper-pattern/              # Layer 1: --target (all 5 symbol types, basic only)
│   ├── function/                # was 02
│   ├── method/                  # was 02 (partial)
│   ├── struct/                  # was 33
│   ├── interface/               # was 32
│   └── functype/                # was 16
│
├── mock-pattern/                 # Layer 1: --dependency (all 5 symbol types, basic only)
│   ├── function/                # Issue #43
│   ├── method/                  # Issue #45
│   ├── struct/                  # Issue #44
│   ├── interface/               # was 01
│   └── functype/                # was 31
│
├── package-variations/           # Layer 3: where symbols come from
│   ├── same-package/            # was 12, 14
│   ├── stdlib/                  # was 08, 18
│   ├── shadowing/               # was 11
│   └── dot-import/              # was 26, 27
│
├── signature-variations/         # Layer 3: parameter/return complexity
│   ├── generics/                # was 07, 21
│   ├── variadic/                # was 01 (partial)
│   ├── named-params/            # was 23
│   ├── non-comparable/          # was 03
│   ├── function-literal/        # was 24
│   ├── struct-literal/          # was 30
│   ├── interface-literal/       # was 25
│   ├── channels/                # was 20
│   └── external-types/          # was 13, 19, 29
│
├── behavior-variations/          # Layer 3: runtime behavior
│   ├── panic-handling/          # was 04
│   ├── callbacks/               # was 15
│   ├── embedded-interfaces/     # was 08
│   └── embedded-structs/        # NEW - struct embedding another struct
│
└── concurrency/                  # Layer 3: call ordering
    ├── ordered/                 # was 28 (partial)
    └── eventually/              # was 06, 28
```

**Structure insight:** Pattern directories now contain only "basic" demonstrations of each symbol type. All variations live in dedicated Layer 3 sections:

- **package-variations:** where the symbol comes from
- **signature-variations:** parameter/return type complexity
- **behavior-variations:** runtime behavior (panics, callbacks, embedding)
- **concurrency:** call ordering modes

**Multi-method is inherent, not a variation:** The basic tests for `interface/` and `struct/` must use multiple methods - that's what distinguishes them from `functype/`. A single-method interface test wouldn't be testing interface support, it would be testing function type support with extra steps.

### Benefits

- Directory structure = Table of Contents for taxonomy
- Users can navigate: "I'm using --target on a function" → `wrapper-pattern/function/`
- Feature cross-cutting concerns (generics, packages) are in their own directories
- README in each directory becomes the detailed documentation

### Concerns / Trade-offs

- [ ] Many UATs span multiple categories - where do they live?
- [ ] Breaking change to any tooling that references UAT numbers
- [ ] Could use symlinks or a mapping file instead of moving

---

## Proposed Taxonomy Restructure

### Current Table of Contents

```markdown
- Introduction
- Capability Matrix (Types)
- Package Variations Matrix
- Signature Variations Matrix
- Concurrency Support
- Target Examples
- Dependency Examples
- Cannot Do (With Workarounds)
- UAT Directory Index
```

### Proposed Table of Contents

```markdown
# imptest Documentation

## Quick Start

- Decision tree: "What should I do?"
- When to use --target (wrapper pattern)
- When to use --dependency (mock pattern)

## Testing Patterns

### Wrapper Pattern (--target)

- What it does
- When to use it
- Symbol types that support it
- Examples by symbol type

### Mock Pattern (--dependency)

- What it does
- When to use it
- Symbol types that support it
- Examples by symbol type

## Variations Reference

### Package Handling

- Same package, different package, stdlib, shadowing, dot imports
- Resolution strategy

### Signature Handling

- Generics, variadic, callbacks, non-comparable types
- Special cases

### Concurrency

- Ordered (default)
- Eventually()

## Limitations

- What imptest cannot do
- Workarounds

## UAT Index

- By pattern
- By symbol type
- By variation
```

### Key Changes

1. **Lead with "what are you trying to do"** not "what does imptest support"
2. **Group by testing pattern first**, then symbol type
3. **Move "Cannot Do" items that are now fixed** to main sections
4. **Add decision tree** for newcomers

---

## Migration Path

### Option A: Big Bang

1. Reorganize everything at once
2. Update all references
3. Ship as major version

**Pros:** Clean break, consistent state
**Cons:** High risk, lots of churn

### Option B: Incremental

1. Start with taxonomy restructure (docs only)
2. Add new UAT organization as symlinks/aliases
3. Gradually move code to new structure
4. Deprecate old paths

**Pros:** Lower risk, can validate approach
**Cons:** Inconsistent state during transition

### Option C: Documentation First

1. Restructure TAXONOMY.md around new mental model
2. Keep file structure and UATs as-is
3. Add mapping/cross-references
4. Only restructure code if docs prove the model works

**Pros:** Tests the model with minimal code churn
**Cons:** Docs and code stay misaligned

### Recommendation

**Option C** - Documentation First. Reasons:

- The mental model is the hypothesis; prove it works before restructuring code
- Docs are user-facing; if the new structure helps users, that's the win
- Code structure can follow if we're confident in the model

---

## Architectural Insight: Signature Extraction

A key realization from this analysis:

**Mocking a function or method is equivalent to mocking a function type.**

The implementation path:

1. User says: `impgen mypackage.ProcessOrder --dependency`
2. impgen finds `func ProcessOrder(order Order) (Receipt, error)`
3. impgen extracts the signature as an implicit function type: `func(Order) (Receipt, error)`
4. impgen generates a mock for that function type
5. User gets `MockProcessOrder` with the same signature

This means:

- `--dependency` on a function → extract signature → mock as function type
- `--dependency` on a method → extract signature → mock as function type
- `--dependency` on a struct → extract all method signatures → mock like an interface
- `--dependency` on an interface → already have method signatures → mock
- `--dependency` on a function type → mock it directly

**The underlying generator is the same.** We're just providing different entry points based on what the user has in their code.

Similarly for wrappers:

- `--target` on a function → wrap it
- `--target` on a method → wrap that single method
- `--target` on a struct → wrap all its methods
- `--target` on an interface → wrap (delegates to concrete impl)
- `--target` on a function type → wrap it

**This suggests the code structure could be:**

```
generators/
├── mock.go      # Single mock generator that handles:
│                #   - functions (extract signature)
│                #   - methods (extract signature)
│                #   - structs (extract all method signatures)
│                #   - interfaces (already have signatures)
│                #   - function types (direct)
│
└── wrapper.go   # Single wrapper generator that handles:
                 #   - functions
                 #   - methods
                 #   - structs (all methods)
                 #   - interfaces (delegating)
                 #   - function types
```

The symbol detection layer figures out what you have; the generator layer doesn't care - it just needs a list of signatures to mock/wrap.

---

## Open Questions

- [ ] Is the three-layer model the right abstraction?
- [ ] Are there patterns we're missing?
- [ ] Should "interface as target" be promoted or de-emphasized?
- [ ] How do we handle UATs that span multiple categories?
- [ ] Is file structure alignment worth the refactor cost?
- [ ] What's the right scope for a first iteration?
- [ ] Does the "signature extraction" insight hold up under scrutiny?
- [ ] Are there cases where mocking a function differs fundamentally from mocking a function type?

---

## Notes / Discussion

_Add discussion notes here as we iterate_

### thoughts about proposal alignment

The three tiers are:

- Testing Pattern (Wrapper vs Mock)
- Symbol Type (Function, Method, Struct, Interface, Function Type)
- Variations (Package, Signature, Concurrency)

The UAT sections are:

```
├── wrapper-pattern/ # Layer 1: --target (all 5 symbol types, basic only)
├── mock-pattern/ # Layer 1: --dependency (all 5 symbol types, basic only)
├── package-variations/ # Layer 3: where symbols come from
├── signature-variations/ # Layer 3: parameter/return complexity
├── behavior-variations/ # Layer 3: runtime behavior
└── concurrency/ # Layer 3: call ordering
```

The file structure is:

```
impgen/
├── main.go
├── run/
│   ├── run.go                 # Entry point, routing
│   ├── symbols/               # Layer 2: Symbol detection
│   ├── packages/              # Layer 3: Package resolution
│   ├── generators/            # Layer 1: Testing patterns
│   └── templates/             # Output templates (by signature count, not type)
```

These things are not clearly aligned yet. There are three layers in the mental model, but the UATs and file structure
don't map cleanly to those layers (6 directories in UATs, 4 in file structure).

Can we improve this alignment further? For example, could the UATs be organized into three top-level directories? Could the code? Or does this mean the mental model needs adjustment?

### Alignment Analysis

**The mismatch:** Layer 3 (Variations) is exploded into 4 categories at the top level, breaking the 3-layer mapping.

**Observation:** Layer 1 (Pattern) and Layer 2 (Symbol) are actually combined in the UAT structure - `wrapper-pattern/function/` tests both "wrapper pattern" AND "function symbol type" together. This makes sense because you can't test a pattern without a symbol type.

**Possible cleaner structure (2 top-level directories):**

```
UAT/
├── core/                    # Layer 1 × Layer 2 (pattern + symbol)
│   ├── wrapper/
│   │   ├── function/
│   │   ├── method/
│   │   ├── struct/
│   │   ├── interface/
│   │   └── functype/
│   └── mock/
│       ├── function/
│       ├── method/
│       ├── struct/
│       ├── interface/
│       └── functype/
└── variations/              # Layer 3 (all special cases)
    ├── package/
    ├── signature/
    ├── behavior/
    └── concurrency/
```

**Why this might be better:**

- Only 2 top-level concepts: "core functionality" vs "special cases"
- Pattern and symbol are naturally combined (you always pick both)
- Variations are clearly secondary/additive
- Maps to user thinking: "Does the basic thing work?" → "What about edge cases?"

**Revised mental model?**

Maybe Layer 2 isn't really a "layer" users navigate - it's a constraint on Layer 1. Users don't think "I want to test a function symbol" - they think "I have a function and want to wrap/mock it."

This suggests:

- **Primary dimension:** Pattern (wrapper vs mock)
- **Input constraint:** Symbol type (what you have in your code)
- **Variations:** Everything else (package, signature, behavior, concurrency)

The "3 layers" might be better described as:

1. **Pattern** - what you want to do
2. **Core** - basic functionality for each pattern × symbol combination
3. **Variations** - special handling for edge cases

### Code Structure Alignment

The code structure doesn't need to match the user mental model exactly - it needs to be organized for maintainability. But it should be _explainable_ in terms of the mental model:

```
impgen/run/
├── generators/     # Layer 1: Pattern implementations
├── symbols/        # Input processing: figure out what symbol you have
├── packages/       # Variation handling: package resolution
└── templates/      # Output: what gets generated
```

The `symbols/` directory isn't a "layer" - it's the mechanism that translates user input into something the generators can work with. It's internal plumbing, not user-facing organization.

### Response

This resonates, but is there a way to clearly express the problem-solving flow for run? We have to go from user input to
package loading to symbol detection to variation handling to code generation with templates to formatting & saving output. Each of these steps could map to a
directory or a file, and I'd _really_ like it if that flow could be obvious from the file structure (allow the
files/dirs to be sortable).

### Flow-Oriented Code Structure

The execution flow is:

1. **Input** - Parse CLI args, validate flags
2. **Load** - Load Go packages
3. **Detect** - Find and classify the symbol
4. **Resolve** - Handle variations (package paths, imports)
5. **Generate** - Build code using templates
6. **Output** - Format and write files

**Option A: Numbered prefixes (most explicit)**

```
impgen/run/
├── 1_input.go      # CLI parsing, flag validation
├── 2_load.go       # Package loading
├── 3_detect.go     # Symbol detection and classification
├── 4_resolve.go    # Package/import resolution, variation handling
├── 5_generate.go   # Template execution, code generation
├── 6_output.go     # Formatting (gofmt), file writing
└── templates/      # Template definitions (not a flow step)
```

**Option B: Numbered directories (if files get large)**

```
impgen/run/
├── 1_input/
│   └── parse.go
├── 2_load/
│   └── packages.go
├── 3_detect/
│   ├── symbols.go
│   └── classify.go
├── 4_resolve/
│   ├── packages.go
│   └── imports.go
├── 5_generate/
│   ├── wrapper.go
│   └── mock.go
├── 6_output/
│   └── write.go
└── templates/
```

**Option C: Verb-based that sorts naturally**

If we pick verbs carefully, alphabetical sorting approximates the flow:

```
impgen/run/
├── a_args.go       # Input parsing
├── b_build.go      # Package loading (build system)
├── c_classify.go   # Symbol detection
├── d_derive.go     # Variation resolution
├── e_emit.go       # Code generation
├── f_finish.go     # Output formatting/writing
└── templates/
```

This is clever but fragile - adding new steps breaks the scheme.

**Recommendation: Option A or B**

Numbered prefixes are:
- Explicit about order
- Self-documenting (`ls` tells the story)
- Easy to extend (insert `2a_` or renumber)
- Common pattern in build systems, migrations, etc.

The trade-off is aesthetic - some find numbered prefixes ugly. But they're unambiguous.

**What about templates?**

Templates don't fit the flow - they're data, not a step. Keep them unnumbered at the end, or in a separate `templates/` directory that sorts last alphabetically.

---

## Decisions

### Decision: Flow-Oriented Structure

**Chosen approach: Option B with templates inside generate directory**

```
impgen/run/
├── 1_input/
│   └── parse.go
├── 2_load/
│   └── packages.go
├── 3_detect/
│   ├── symbols.go
│   └── classify.go
├── 4_resolve/
│   ├── packages.go
│   └── imports.go
├── 5_generate/
│   ├── wrapper.go
│   ├── mock.go
│   └── templates/      # Templates live with generation, not separate
└── 6_output/
    └── write.go
```

**Rationale:**
- Numbered directories make execution flow obvious from `ls` output
- Templates belong with code generation (step 5) since that's where they're used
- Each directory corresponds to one phase of the pipeline
- Easy to navigate: "where does symbol detection happen?" → `3_detect/`
