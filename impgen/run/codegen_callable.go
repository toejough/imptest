package run

import (
	"github.com/dave/dst"
)

// callableExtendedTemplateData extends callableTemplateData with dynamic signature info.
type callableExtendedTemplateData struct {
	callableTemplateData //nolint:unused // Embedded fields accessed via promotion

	CallableSignature string
	CallableReturns   string
	ParamNames        string   // comma-separated parameter names for calling
	ReturnVars        string   // comma-separated return variable names (ret0, ret1, ...)
	ReturnVarsList    []string // slice of return variable names
	ReturnFields      []returnFieldData
	ResultParams      string // parameters for ExpectReturnedValues (v1 Type1, v2 Type2, ...)
	ResultParamsAny   string // parameters for ExpectReturnedValuesShould (v1 any, v2 any, ...)
	ResultComparisons string // comparisons for ExpectReturnedValues
	ResultMatchers    string // matcher-based comparisons for ExpectReturnedValuesShould
}

// Types

// callableGenerator holds state for generating callable wrapper code.
type callableGenerator struct {
	baseGenerator

	templates                  *TemplateRegistry
	funcDecl                   *dst.FuncDecl
	astFiles                   []*dst.File                   // Source AST files for import resolution
	cachedTemplateData         *callableTemplateData         // Cache to avoid redundant templateData() calls
	cachedExtendedTemplateData *callableExtendedTemplateData // Cache to avoid redundant extendedTemplateData() calls
}

// buildReturnFieldData builds return field data with types for templates.

// checkIfReflectNeeded scans return types and sets needsReflect if any are non-comparable.
// This must be called before generating templates to ensure the reflect import is included.

// collectAdditionalImports collects all external type imports needed for the callable function signature.

// extendedTemplateData returns template data with dynamic signature info.
// The result is cached after the first call to avoid redundant struct construction.

// generateCallableTemplates executes all templates to generate the callable wrapper code.

// Generate header

// Generate structs and methods

// Generate ExpectReturnedValues methods

// Generate panic and response methods

// numReturns returns the total number of return values.
// This should only be called when hasResults(g.funcDecl.Type) is true.

// paramNamesString returns comma-separated parameter names for function calls.

// paramsString returns the parameter list as a string.

// resultComparisonsString returns comparison code for return values.

// resultMatchersString returns matcher-based comparison code for return values.

// resultParamsAnyString returns parameters for ExpectReturnedValuesShould (v1 any, v2 any, ...).

// resultParamsString returns parameters for ExpectReturnedValues (v1 Type1, v2 Type2, ...).

// returnTypeName returns the appropriate type name for return channels and fields.
// Returns "{impName}Return{TypeParams}" if the function has returns, otherwise "struct{}".

// returnVarNames generates return variable names for the function call.

// returnsString returns the return type list as a string.

// templateData returns the base template data for this generator.
// The result is cached after the first call to avoid redundant struct construction.

// writeParamsWithQualifiersTo writes function parameters with package qualifiers to a buffer.

// writeResultChecks generates comparison code for return values.
// When useMatcher is true, uses imptest.MatchValue for flexible matching.
// When useMatcher is false, uses == or reflect.DeepEqual for equality checks.

// writeResultTypesWithQualifiersTo writes function return types to a buffer.

// Functions

// returnFieldData holds data for a single return field.
type returnFieldData struct {
	Index int
	Name  string
	Type  string // Type name for struct field definitions
}

// Entry Point

// generateCallableWrapperCode generates a type-safe wrapper for a callable function.
//
//nolint:cyclop,funlen,nestif // Code generation requires conditional logic for package resolution

// For callables, GetPackageInfo won't work since function names don't have package qualifiers
// Instead, if we're in a test package, we need to import the source package

// We're in a test package (e.g., visitor_test) and the callable is from the source package (e.g., visitor)
// We need to import the source package

// Get the actual package name by loading the package

// Extract package name from the first file

// Fallback: extract from import path (last component)

// Special case: when pkgImportPath is "." and we're in a test package,
// the callable is from the non-test version of this package

// Initialize template registry
