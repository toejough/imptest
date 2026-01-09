# Issue #22: impgen/run/ Flow-Oriented Reorganization

## Current Status (2025-01-09)

**Completed:**
- ✅ Stage 1: Numeric file prefixes
- ✅ Stage 2: Extract `2_load/` subpackage
- ✅ Stage 3: Extract `1_cache/` subpackage
- ✅ Stage 5: Extract `6_output/` subpackage

**Remaining (blocked by tight coupling):**
- ❌ Stage 4: Extract `3_detect/` - detection types used by generators
- ❌ Stage 6: Extract `5_generate/` - generators use `exprToString` from common, which detection also needs
- ❌ Stage 7: Extract `4_resolve/` - merged with detect concerns
- ⏸️ Stage 8: Partial - cleanup done, but `4_common.go` stays in place

**Current Structure:**
```
impgen/run/
├── 0_run.go              # Orchestrator
├── 1_cache/              # ✅ Subpackage
│   └── cache.go
├── 2_load/               # ✅ Subpackage
│   └── load.go
├── 3_pkgparse.go         # Symbol detection (not extracted)
├── 4_common.go           # Shared utilities (not extracted)
├── 5_*.go                # Generators (not extracted)
└── 6_output/             # ✅ Subpackage
    └── output.go
```

**Blocker:** Circular dependency between `3_pkgparse.go` and generators:
- `3_pkgparse.go` defines types (`symbolDetails`, `ifaceWithDetails`, etc.) consumed by generators
- `3_pkgparse.go` uses `exprToString()` from `4_common.go`
- Generators need both the types AND the utilities

**Options to complete:**
1. Create a shared `types/` package for the detection types
2. Duplicate `exprToString` into `3_pkgparse.go`
3. Accept current hybrid structure as final

---

## Goal

Transform impgen/run/ from a flat structure to numbered directories reflecting execution flow:

```
Current (16 files flat):          Target:
impgen/run/                       impgen/run/
├── run.go                        ├── 1_input/
├── pkgload.go                    ├── 2_load/
├── pkgparse.go                   ├── 3_detect/
├── cache.go                      ├── 4_resolve/
├── codegen_common.go             ├── 5_generate/
├── codegen_dependency.go         │   └── templates/
├── codegen_target.go             └── 6_output/
├── codegen_interface.go
├── codegen_interface_target.go
├── codegen_function_dependency.go
├── codegen_struct_target.go
├── templates.go
├── text_templates.go
└── *_test.go files
```

## Challenge

Go subdirectories are **separate packages**. Moving files to subdirs requires:
- Exporting functions (capitalize names)
- Updating all imports
- Avoiding circular dependencies

The current code has tight coupling, especially around `codegen_common.go` (36KB of shared utilities).

## Staged Migration Strategy

Each stage passes `mage check`. Stages are ordered by independence (least coupled first).

---

### Stage 1: Numeric File Prefixes (No Subdirs)

**Goal:** Visual ordering without breaking anything.

Rename files with numeric prefixes matching execution flow:

```
run.go                    → 0_run.go
cache.go                  → 1_cache.go
pkgload.go                → 2_pkgload.go
pkgparse.go               → 3_pkgparse.go
codegen_common.go         → 4_common.go
codegen_interface.go      → 5_interface.go
codegen_dependency.go     → 5_mock_interface.go
codegen_function_dependency.go → 5_mock_function.go
codegen_target.go         → 5_wrap_function.go
codegen_interface_target.go → 5_wrap_interface.go
codegen_struct_target.go  → 5_wrap_struct.go
templates.go              → 5_templates.go
text_templates.go         → 5_text_templates.go
```

Test files follow their source file.

**Commands:**
```bash
git mv run.go 0_run.go
git mv cache.go 1_cache.go
# ... etc
```

**Verify:** `mage check`

---

### Stage 2: Extract `2_load/` Subpackage

**Goal:** Extract the most independent code first.

`pkgload.go` contains only `LoadPackageDST()` which has minimal dependencies.

1. Create `impgen/run/2_load/` directory
2. Move and adapt:
   - `2_pkgload.go` → `2_load/load.go`
   - Change package to `load`
   - Export `LoadPackageDST` (already exported)
3. Update imports in `0_run.go` and other files

**New file:** `2_load/load.go`
```go
package load

import (...)

func LoadPackageDST(pkgPath string) ([]*dst.File, *token.FileSet, error) {
    // existing implementation
}
```

**Verify:** `mage check`

---

### Stage 3: Extract `1_cache/` Subpackage

**Goal:** Extract cache constants and interfaces.

`cache.go` contains constants and interfaces, minimal coupling.

1. Create `impgen/run/1_cache/` directory
2. Move:
   - `1_cache.go` → `1_cache/cache.go`
   - Change package to `cache`
3. Update imports

**Verify:** `mage check`

---

### Stage 4: Extract `3_detect/` Subpackage

**Goal:** Extract symbol detection logic.

`pkgparse.go` has symbol finding functions. This is larger and more coupled.

1. Create `impgen/run/3_detect/` directory
2. Move key functions:
   - `findSymbol()` → exported `FindSymbol()`
   - `findImportPath()` → exported `FindImportPath()`
   - `getMatchingInterfaceFromAST()` → exported
   - Related helper functions
3. Keep some utilities in common if needed for generators

**Key exports needed:**
- `FindSymbol()`
- `SymbolDetails` struct
- Symbol type constants

**Verify:** `mage check`

---

### Stage 5: Extract `6_output/` Subpackage

**Goal:** Extract file writing logic.

Parts of `run.go` handle output formatting and file writing.

1. Create `impgen/run/6_output/` directory
2. Extract:
   - `writeGeneratedCodeToFile()` → `output.WriteFile()`
   - File naming logic
3. Update orchestration in `0_run.go`

**Verify:** `mage check`

---

### Stage 6: Extract `5_generate/` Subpackage

**Goal:** Consolidate all code generation.

This is the largest change - move all codegen files.

1. Create `impgen/run/5_generate/` directory
2. Move generator files:
   - `5_mock_interface.go` → `5_generate/mock_interface.go`
   - `5_mock_function.go` → `5_generate/mock_function.go`
   - `5_wrap_function.go` → `5_generate/wrap_function.go`
   - `5_wrap_interface.go` → `5_generate/wrap_interface.go`
   - `5_wrap_struct.go` → `5_generate/wrap_struct.go`
   - `5_interface.go` → `5_generate/interface.go`
   - `5_templates.go` → `5_generate/templates.go`
   - `5_text_templates.go` → `5_generate/text_templates.go`
3. Create `5_generate/templates/` subdirectory
4. Export generator entry points:
   - `GenerateDependencyCode()`
   - `GenerateTargetCode()`
   - etc.

**Verify:** `mage check`

---

### Stage 7: Extract `4_resolve/` Subpackage (if needed)

**Goal:** Extract package/import resolution.

Some resolution logic from `codegen_common.go` could move here.

1. Create `impgen/run/4_resolve/` directory
2. Extract import resolution functions
3. This may stay in `5_generate/` if too coupled

**Verify:** `mage check`

---

### Stage 8: Final Cleanup

**Goal:** Clean orchestration layer.

1. `0_run.go` becomes pure orchestration:
   - Parse args
   - Call load
   - Call detect
   - Call generate
   - Call output
2. Rename `0_run.go` → `run.go` (orchestrator stays at root)
3. Move `4_common.go` into `5_generate/` as shared utilities

**Verify:** `mage check`

---

## File Mapping Summary

| Current | Stage 1 | Final Location |
|---------|---------|----------------|
| run.go | 0_run.go | run.go (orchestrator) |
| cache.go | 1_cache.go | 1_cache/cache.go |
| pkgload.go | 2_pkgload.go | 2_load/load.go |
| pkgparse.go | 3_pkgparse.go | 3_detect/detect.go |
| codegen_common.go | 4_common.go | 5_generate/common.go |
| codegen_interface.go | 5_interface.go | 5_generate/interface.go |
| codegen_dependency.go | 5_mock_interface.go | 5_generate/mock_interface.go |
| codegen_function_dependency.go | 5_mock_function.go | 5_generate/mock_function.go |
| codegen_target.go | 5_wrap_function.go | 5_generate/wrap_function.go |
| codegen_interface_target.go | 5_wrap_interface.go | 5_generate/wrap_interface.go |
| codegen_struct_target.go | 5_wrap_struct.go | 5_generate/wrap_struct.go |
| templates.go | 5_templates.go | 5_generate/templates.go |
| text_templates.go | 5_text_templates.go | 5_generate/text_templates.go |

---

## Risk Mitigation

- **Circular imports:** Stages ordered to avoid. If encountered, keep shared code at higher level.
- **Test files:** Move with their source files, update package names.
- **Golden tests:** May need path updates if test data location changes.
- **Each stage is reversible:** If `mage check` fails, revert and adjust approach.

---

## Time Estimates

- Stage 1: ~2 min (just renames)
- Stage 2: ~3 min (small, independent)
- Stage 3: ~2 min (constants only)
- Stage 4: ~5 min (larger, more exports needed)
- Stage 5: ~3 min (small extraction)
- Stage 6: ~10 min (largest change)
- Stage 7: ~5 min (may skip if too coupled)
- Stage 8: ~3 min (cleanup)

**Total: ~30-35 minutes** with verification between each stage.
