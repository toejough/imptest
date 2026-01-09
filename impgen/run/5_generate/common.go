package generate

import (
	"bytes"
	"fmt"
	"go/token"
	"sort"
	"strings"
	"unicode"

	"github.com/dave/dst"
	astutil "github.com/toejough/imptest/impgen/run/0_util"
	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

// NamingMode represents the different modes for generating type names.
type NamingMode int

// NamingMode values.
const (
	NamingModeDefault NamingMode = iota
	NamingModeTarget
	NamingModeDependency
)

// GeneratorInfo holds information gathered for generation.
type GeneratorInfo struct {
	PkgName            string
	InterfaceName      string
	LocalInterfaceName string
	ImpName            string
	Mode               NamingMode
	ImportPathFlag     string
	NameProvided       bool // true if --name was explicitly provided
}

// ResultData contains all result-related template data.
type ResultData struct {
	Vars           string        // "r0, r1" or "ret0, ret1"
	Assignments    string        // "Result0: r0, Result1: r1"
	ExpectedParams string        // "v0 int, v1 error"
	MatcherParams  string        // "v0 any, v1 any"
	Fields         []resultField // resultField{Name: "Result0", Type: "int"}
	Checks         []resultCheck // resultCheck{Field: "Result0", Expected: "v0", Index: 0}
	ReturnList     string        // "int" or "(int, error)"
	HasResults     bool
	WaitMethodName string // "WaitForCompletion" or "WaitForResponse"
}

// ResultDataBuilder builds result-related template data from result types.
// This consolidates the duplicated result building logic across generators.
type ResultDataBuilder struct {
	ResultTypes []string
	VarPrefix   string // e.g., "r", "ret", "result"
}

// Build constructs ResultData from the builder configuration.
func (b *ResultDataBuilder) Build() ResultData {
	hasResults := len(b.ResultTypes) > 0

	data := ResultData{
		HasResults:     hasResults,
		WaitMethodName: "WaitForCompletion",
	}

	if !hasResults {
		return data
	}

	data.WaitMethodName = "WaitForResponse"

	var vars, assignments, expectedParams, matcherParams strings.Builder

	for idx, resultType := range b.ResultTypes {
		if idx > 0 {
			vars.WriteString(", ")
			assignments.WriteString(", ")
			expectedParams.WriteString(", ")
			matcherParams.WriteString(", ")
		}

		// Result variable name (e.g., "r0", "ret0", "result0")
		fmt.Fprintf(&vars, "%s%d", b.VarPrefix, idx)

		// Assignment for Returns struct (e.g., "Result0: r0")
		fmt.Fprintf(&assignments, "Result%d: %s%d", idx, b.VarPrefix, idx)

		// Expected params for ExpectReturnsEqual (e.g., "v0 int")
		fmt.Fprintf(&expectedParams, "v%d %s", idx, resultType)

		// Matcher params for ExpectReturnsMatch (e.g., "v0 any")
		fmt.Fprintf(&matcherParams, "v%d any", idx)

		// Result field for struct definition
		data.Fields = append(data.Fields, resultField{
			Name: fmt.Sprintf("Result%d", idx),
			Type: resultType,
		})

		// Result check for verification
		data.Checks = append(data.Checks, resultCheck{
			Field:    fmt.Sprintf("Result%d", idx),
			Expected: fmt.Sprintf("v%d", idx),
			Index:    idx,
		})
	}

	data.Vars = vars.String()
	data.Assignments = assignments.String()
	data.ExpectedParams = expectedParams.String()
	data.MatcherParams = matcherParams.String()
	data.ReturnList = buildResultReturnList(b.ResultTypes)

	return data
}

// GetPackageInfo extracts package info for a given target name (e.g., "pkg.Interface").
func GetPackageInfo(
	targetName string,
	pkgLoader detect.PackageLoader,
	currentPkgName string,
) (pkgPath, pkgName string, err error) {
	before, _, ok := strings.Cut(targetName, ".")
	if !ok {
		return "", "", nil
	}

	pkgName = before
	if pkgName == "" || pkgName == "." {
		return "", "", nil
	}

	// If it matches the package we're generating into, it's local.
	if currentPkgName == pkgName {
		return "", "", nil
	}

	astFiles, _, _, err := pkgLoader.Load(".")
	if err != nil {
		// If we can't load the local package, we can't find aliases,
		// but we might still be able to resolve the package path directly.
		files, _, _, err := pkgLoader.Load(pkgName)
		if err == nil && len(files) > 0 {
			return pkgName, pkgName, nil
		}

		return "", "", nil
	}

	pkgPath, err = detect.FindImportPath(astFiles, pkgName, pkgLoader)
	if err != nil {
		// If it's not a package we know about, assume it's a local reference (e.g. MyType.MyMethod)
		return "", "", nil //nolint:nilerr
	}

	return pkgPath, pkgName, nil
}

// unexported constants.
const (
	anyTypeString = "any"
	pkgFmt        = "_fmt"
	pkgImptest    = "_imptest"
	pkgReflect    = "_reflect"
	pkgTesting    = "_testing"
	pkgTime       = "_time"
)

// baseGenerator holds common state and methods for code generation.
type baseGenerator struct {
	codeWriter
	typeFormatter

	pkgName        string
	impName        string
	pkgPath        string
	qualifier      string
	typeParams     *dst.FieldList
	needsFmt       bool
	needsImptest   bool
	needsReflect   bool
	needsQualifier bool
}

// buildParamStrings builds the parameter string and collects parameter names from a function type.
//
//nolint:nestif,wsl_v5 // Complex logic for handling named/unnamed params; whitespace for readability
func (baseGen *baseGenerator) buildParamStrings(
	ftype *dst.FuncType,
) (paramsStr string, paramNames []string) {
	var builder strings.Builder
	first := true

	if ftype.Params != nil {
		for _, field := range ftype.Params.List {
			fieldType := baseGen.typeWithQualifier(field.Type)

			if len(field.Names) > 0 {
				for _, name := range field.Names {
					if !first {
						builder.WriteString(", ")
					}
					first = false
					builder.WriteString(name.Name)
					builder.WriteString(" ")
					builder.WriteString(fieldType)
					paramNames = append(paramNames, name.Name)
				}
			} else {
				paramName := fmt.Sprintf("arg%d", len(paramNames)+1)
				if !first {
					builder.WriteString(", ")
				}
				first = false
				builder.WriteString(paramName)
				builder.WriteString(" ")
				builder.WriteString(fieldType)
				paramNames = append(paramNames, paramName)
			}
		}
	}

	return builder.String(), paramNames
}

// buildResultStrings builds the result string and collects result types from a function type.
//
//nolint:cyclop,nestif,intrange,wsl_v5 // Complex logic for formatting results; whitespace for readability
func (baseGen *baseGenerator) buildResultStrings(
	ftype *dst.FuncType,
) (resultsStr string, resultTypes []string) {
	var builder strings.Builder

	if ftype.Results != nil && len(ftype.Results.List) > 0 {
		hasMultipleResults := len(ftype.Results.List) > 1 ||
			(len(ftype.Results.List) == 1 && len(ftype.Results.List[0].Names) > 1)

		if hasMultipleResults {
			builder.WriteString(" (")
		} else {
			builder.WriteString(" ")
		}

		first := true
		for _, field := range ftype.Results.List {
			fieldType := baseGen.typeWithQualifier(field.Type)

			count := len(field.Names)
			if count == 0 {
				count = 1
			}

			for i := 0; i < count; i++ {
				if !first {
					builder.WriteString(", ")
				}
				first = false
				builder.WriteString(fieldType)
				resultTypes = append(resultTypes, fieldType)
			}
		}

		if hasMultipleResults {
			builder.WriteString(")")
		}
	}

	return builder.String(), resultTypes
}

// checkIfQualifierNeeded pre-scans to determine if the package qualifier is needed.
func (baseGen *baseGenerator) checkIfQualifierNeeded(expr dst.Expr) {
	if baseGen.qualifier == "" {
		return
	}

	if hasExportedIdent(expr, baseGen.isTypeParam) {
		baseGen.needsQualifier = true
	}
}

// collectAdditionalImportsFromInterface collects imports needed for interface method signatures.
// This is a helper for generators that need to collect imports from all methods of an interface.
func (baseGen *baseGenerator) collectAdditionalImportsFromInterface(
	iface *dst.InterfaceType,
	astFiles []*dst.File,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	sourceImports []*dst.ImportSpec,
) []importInfo {
	if len(astFiles) == 0 {
		return nil
	}

	allImports := make(map[string]importInfo) // Deduplicate by path

	// Iterate over all interface methods to collect imports from their signatures
	_ = forEachInterfaceMethod(
		iface, astFiles, baseGen.fset, pkgImportPath, pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			// Collect from parameters
			if ftype.Params != nil {
				for _, field := range ftype.Params.List {
					imports := collectExternalImports(field.Type, sourceImports)
					for _, imp := range imports {
						allImports[imp.Path] = imp
					}
				}
			}

			// Collect from return types
			if ftype.Results != nil {
				for _, field := range ftype.Results.List {
					imports := collectExternalImports(field.Type, sourceImports)
					for _, imp := range imports {
						allImports[imp.Path] = imp
					}
				}
			}
		},
	)

	// Convert map to slice and sort for deterministic output
	result := make([]importInfo, 0, len(allImports))
	for _, imp := range allImports {
		result = append(result, imp)
	}

	// Sort by import path for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// formatTypeParamsDecl formats type parameters for declaration.
func (baseGen *baseGenerator) formatTypeParamsDecl() string {
	return formatTypeParamsDecl(baseGen.fset, baseGen.typeParams)
}

// formatTypeParamsUse formats type parameters for instantiation.
func (baseGen *baseGenerator) formatTypeParamsUse() string {
	return formatTypeParamsUse(baseGen.typeParams)
}

// isTypeParameter checks if a name is one of the type parameters.
func (baseGen *baseGenerator) isTypeParameter(name string) bool {
	if baseGen.typeParams == nil {
		return false
	}

	for _, field := range baseGen.typeParams.List {
		for _, paramName := range field.Names {
			if paramName.Name == name {
				return true
			}
		}
	}

	return false
}

// codeWriter provides common buffer writing functionality for code generators.
// ... (omitting some lines for brevity, but I must match exactly).
type codeWriter struct {
	buf bytes.Buffer
}

// bytes returns the buffer contents.
func (w *codeWriter) bytes() []byte {
	return w.buf.Bytes()
}

// pf writes a formatted string to the buffer.

// fieldInfo represents extracted information about a single field entry.
type fieldInfo struct {
	Name  string     // The name (explicit or generated)
	Index int        // The overall index across all fields
	Field *dst.Field // The original AST field (use Field.Type with typeWithQualifier)
}

// typeExprWalker traverses AST type expressions with a generic return type.
// It provides a unified way to walk type expressions, handling all AST node types
// while allowing custom logic for leaf nodes (Ident, SelectorExpr) and result combining.
type typeExprWalker[T any] struct {
	visitIdent    func(*dst.Ident) T
	visitSelector func(*dst.SelectorExpr) T
	combine       func(T, T) T
	zero          T
}

// walk traverses an AST expression and returns the combined result.
//
//nolint:cyclop // Type-switch dispatcher handling all AST node types; complexity is inherent
func (w *typeExprWalker[T]) walk(expr dst.Expr) T {
	switch typeExpr := expr.(type) {
	case *dst.Ident:
		return w.visitIdent(typeExpr)
	case *dst.SelectorExpr:
		return w.visitSelector(typeExpr)
	case *dst.StarExpr:
		return w.walk(typeExpr.X)
	case *dst.ArrayType:
		return w.walk(typeExpr.Elt)
	case *dst.MapType:
		return w.combine(w.walk(typeExpr.Key), w.walk(typeExpr.Value))
	case *dst.ChanType:
		return w.walk(typeExpr.Value)
	case *dst.FuncType:
		return w.walkFieldList(typeExpr.Params, typeExpr.Results)
	case *dst.StructType:
		return w.walkFieldList(typeExpr.Fields)
	case *dst.IndexExpr:
		return w.combine(w.walk(typeExpr.X), w.walk(typeExpr.Index))
	case *dst.IndexListExpr:
		result := w.walk(typeExpr.X)
		for _, idx := range typeExpr.Indices {
			result = w.combine(result, w.walk(idx))
		}

		return result
	}

	return w.zero
}

// walkFieldList traverses field lists (for FuncType params/results or StructType fields).
func (w *typeExprWalker[T]) walkFieldList(lists ...*dst.FieldList) T {
	result := w.zero

	for _, list := range lists {
		if list != nil {
			for _, field := range list.List {
				result = w.combine(result, w.walk(field.Type))
			}
		}
	}

	return result
}

// typeFormatter handles formatting AST types into strings with package qualifiers.
type typeFormatter struct {
	fset        *token.FileSet
	qualifier   string
	isTypeParam func(string) bool
}

// typeWithQualifier returns a type expression as a string with package qualifier if needed.
func (tf *typeFormatter) typeWithQualifier(expr dst.Expr) string {
	switch typeExpr := expr.(type) {
	case *dst.Ident:
		return tf.typeWithQualifierIdent(typeExpr)
	case *dst.StarExpr:
		return tf.typeWithQualifierStar(typeExpr)
	case *dst.SelectorExpr:
		return exprToString(tf.fset, typeExpr)
	default:
		return tf.typeWithQualifierComposite(expr)
	}
}

// typeWithQualifierArray handles array/slice types.
func (tf *typeFormatter) typeWithQualifierArray(arrType *dst.ArrayType) string {
	var buf strings.Builder
	buf.WriteString("[")

	if arrType.Len != nil {
		buf.WriteString(exprToString(tf.fset, arrType.Len))
	}

	buf.WriteString("]")
	buf.WriteString(tf.typeWithQualifier(arrType.Elt))

	return buf.String()
}

// typeWithQualifierChan handles channel types.
func (tf *typeFormatter) typeWithQualifierChan(chanType *dst.ChanType) string {
	var buf strings.Builder

	switch chanType.Dir {
	case dst.SEND:
		buf.WriteString("chan<- ")
	case dst.RECV:
		buf.WriteString("<-chan ")
	default:
		buf.WriteString("chan ")
	}

	buf.WriteString(tf.typeWithQualifier(chanType.Value))

	return buf.String()
}

// typeWithQualifierComposite handles composite types like arrays, maps, and funcs.
func (tf *typeFormatter) typeWithQualifierComposite(expr dst.Expr) string {
	switch typeExpr := expr.(type) {
	case *dst.ArrayType:
		return tf.typeWithQualifierArray(typeExpr)
	case *dst.MapType:
		return tf.typeWithQualifierMap(typeExpr)
	case *dst.ChanType:
		return tf.typeWithQualifierChan(typeExpr)
	case *dst.FuncType:
		return tf.typeWithQualifierFunc(typeExpr)
	case *dst.IndexExpr:
		return tf.typeWithQualifierIndex(typeExpr)
	case *dst.IndexListExpr:
		return tf.typeWithQualifierIndexList(typeExpr)
	default:
		return exprToString(tf.fset, expr)
	}
}

// typeWithQualifierFunc handles function types.
func (tf *typeFormatter) typeWithQualifierFunc(funcType *dst.FuncType) string {
	return typeWithQualifierFunc(tf.fset, funcType, tf.typeWithQualifier)
}

// typeWithQualifierIdent handles simple identifier types.
func (tf *typeFormatter) typeWithQualifierIdent(ident *dst.Ident) string {
	var buf strings.Builder

	// Don't qualify type parameters
	if !tf.isTypeParam(ident.Name) && tf.qualifier != "" &&
		len(ident.Name) > 0 && unicode.IsUpper(rune(ident.Name[0])) {
		buf.WriteString(tf.qualifier)
		buf.WriteString(".")
	}

	buf.WriteString(ident.Name)

	return buf.String()
}

// typeWithQualifierIndex handles generic type instantiation with single type parameter.
func (tf *typeFormatter) typeWithQualifierIndex(indexExpr *dst.IndexExpr) string {
	var buf strings.Builder

	buf.WriteString(tf.typeWithQualifier(indexExpr.X))
	buf.WriteString("[")
	buf.WriteString(tf.typeWithQualifier(indexExpr.Index))
	buf.WriteString("]")

	return buf.String()
}

// typeWithQualifierIndexList handles generic type instantiation with multiple type parameters.
func (tf *typeFormatter) typeWithQualifierIndexList(indexListExpr *dst.IndexListExpr) string {
	var buf strings.Builder

	buf.WriteString(tf.typeWithQualifier(indexListExpr.X))
	buf.WriteString("[")
	buf.WriteString(joinWith(indexListExpr.Indices, tf.typeWithQualifier, ", "))
	buf.WriteString("]")

	return buf.String()
}

// typeWithQualifierMap handles map types.
func (tf *typeFormatter) typeWithQualifierMap(mapType *dst.MapType) string {
	var buf strings.Builder

	buf.WriteString("map[")
	buf.WriteString(tf.typeWithQualifier(mapType.Key))
	buf.WriteString("]")
	buf.WriteString(tf.typeWithQualifier(mapType.Value))

	return buf.String()
}

// typeWithQualifierStar handles pointer types.
func (tf *typeFormatter) typeWithQualifierStar(t *dst.StarExpr) string {
	var buf strings.Builder

	buf.WriteString("*")
	buf.WriteString(tf.typeWithQualifier(t.X))

	return buf.String()
}

// variadicArgsResult holds the result of buildVariadicArgs.
type variadicArgsResult struct {
	hasVariadic     bool
	nonVariadicArgs string
	variadicArg     string
	allArgs         string
}

// buildResultReturnList builds the return type list from result types.
func buildResultReturnList(resultTypes []string) string {
	if len(resultTypes) == 0 {
		return ""
	}

	if len(resultTypes) == 1 {
		return resultTypes[0]
	}

	return "(" + strings.Join(resultTypes, ", ") + ")"
}

// visitParams iterates over function parameters and calls the visitor for each.
// The visitor receives each parameter with its type string and current indices,
// and returns the updated indices for the next iteration.

// buildVariadicArgs checks for variadic parameters and builds argument strings.
//
//nolint:wsl_v5,cyclop // Whitespace styling in nested loop logic; complexity from variadic handling
func buildVariadicArgs(ftype *dst.FuncType, paramNames []string) variadicArgsResult {
	var hasVariadic bool
	var nonVariadicArgs, variadicArg, allArgs strings.Builder

	if ftype.Params != nil && len(ftype.Params.List) > 0 {
		lastField := ftype.Params.List[len(ftype.Params.List)-1]
		_, hasVariadic = lastField.Type.(*dst.Ellipsis)
	}

	if hasVariadic && len(paramNames) > 0 {
		for i := range len(paramNames) - 1 {
			if i > 0 {
				nonVariadicArgs.WriteString(", ")
			}
			nonVariadicArgs.WriteString(paramNames[i])
		}
		variadicArg.WriteString(paramNames[len(paramNames)-1])

		for i, name := range paramNames {
			if i > 0 {
				allArgs.WriteString(", ")
			}
			allArgs.WriteString(name)
		}
	} else {
		for i, name := range paramNames {
			if i > 0 {
				allArgs.WriteString(", ")
			}
			allArgs.WriteString(name)
		}
	}

	return variadicArgsResult{
		hasVariadic:     hasVariadic,
		nonVariadicArgs: nonVariadicArgs.String(),
		variadicArg:     variadicArg.String(),
		allArgs:         allArgs.String(),
	}
}

// collectExternalImports walks a type expression and collects package references
// from SelectorExpr nodes (e.g., "io.Reader", "os.FileMode").
// It resolves each package reference to its full import path using the source imports.
// For stdlib packages, it adds a "_" prefix when there's a naming conflict with non-stdlib imports.
//
//nolint:cyclop,nestif // Conflict detection requires nested checks; complexity is inherent
func collectExternalImports(expr dst.Expr, sourceImports []*dst.ImportSpec) []importInfo {
	var imports []importInfo

	seen := make(map[string]bool) // Deduplicate by import path

	// Build a map of package names to their paths from source imports
	// This helps us detect conflicts between stdlib and non-stdlib packages with the same name
	sourcePackageNames := make(map[string]string) // name -> path

	for _, imp := range sourceImports {
		path := strings.Trim(imp.Path.Value, `"`)

		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			// Extract package name from path
			lastSlash := strings.LastIndex(path, "/")
			if lastSlash >= 0 {
				pkgName = path[lastSlash+1:]
			} else {
				pkgName = path
			}
		}

		sourcePackageNames[pkgName] = path
	}

	walker := &typeExprWalker[struct{}]{
		visitIdent: func(*dst.Ident) struct{} {
			return struct{}{}
		},
		visitSelector: func(sel *dst.SelectorExpr) struct{} {
			// Check if X is an identifier (package reference)
			if ident, ok := sel.X.(*dst.Ident); ok {
				pkgAlias := ident.Name
				// Find the import path for this package alias
				path := resolveImportPath(pkgAlias, sourceImports)
				if path != "" && !seen[path] {
					seen[path] = true

					// Determine the alias to use
					alias := pkgAlias
					// If this is a stdlib package and there's a non-stdlib source import with the same name,
					// prefix the stdlib package with "_" to avoid the conflict
					if detect.IsStdlibPackage(path) && path == pkgAlias {
						if existingPath, exists := sourcePackageNames[pkgAlias]; exists && !detect.IsStdlibPackage(existingPath) {
							// There's a non-stdlib package with the same name - prefix the stdlib one
							alias = "_" + pkgAlias
						}
					}

					imports = append(imports, importInfo{
						Alias: alias,
						Path:  path,
					})
				}
			}

			return struct{}{}
		},
		combine: func(_, _ struct{}) struct{} {
			return struct{}{}
		},
		zero: struct{}{},
	}

	walker.walk(expr)

	return imports
}

// collectImportsFromFuncDecl collects additional imports needed for a function declaration's parameters and returns.
// This is shared logic used by both callableGenerator and targetGenerator.
//
//nolint:cyclop // Complexity from iterating params and results is unavoidable
func collectImportsFromFuncDecl(funcDecl *dst.FuncDecl, astFiles []*dst.File) []importInfo {
	if len(astFiles) == 0 {
		return nil
	}

	// Get source imports from the first AST file
	var sourceImports []*dst.ImportSpec

	for _, file := range astFiles {
		if len(file.Imports) > 0 {
			sourceImports = file.Imports
			break
		}
	}

	allImports := make(map[string]importInfo) // Deduplicate by path

	// Collect from parameters
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			imports := collectExternalImports(field.Type, sourceImports)
			for _, imp := range imports {
				allImports[imp.Path] = imp
			}
		}
	}

	// Collect from return types
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			imports := collectExternalImports(field.Type, sourceImports)
			for _, imp := range imports {
				allImports[imp.Path] = imp
			}
		}
	}

	// Convert map to slice and sort for deterministic output
	result := make([]importInfo, 0, len(allImports))
	for _, imp := range allImports {
		result = append(result, imp)
	}

	// Sort by import path for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// expandFieldListTypes expands a field list into individual type strings.
// For fields with multiple names (e.g., "a, b int"), outputs the type multiple times.
// For fields with no names (e.g., unnamed params), outputs the type once.
// expandFieldListTypes delegates to astutil.ExpandFieldListTypes.
func expandFieldListTypes(fields []*dst.Field, typeFormatter func(dst.Expr) string) []string {
	return astutil.ExpandFieldListTypes(fields, typeFormatter)
}

// exprToString renders a dst.Expr to Go code.
// This function converts DST expressions back to their string representation.
func exprToString(fset *token.FileSet, expr dst.Expr) string {
	return astutil.ExprToString(fset, expr)
}

// extractFields extracts all individual fields from a field list.
// For unnamed fields, generates names using the provided prefix and index.
// For named fields with multiple names, creates separate entries for each.
func extractFields(fields *dst.FieldList, prefix string) []fieldInfo {
	if fields == nil {
		return nil
	}

	var result []fieldInfo

	index := 0

	for _, field := range fields.List {
		if hasFieldNames(field) {
			for _, name := range field.Names {
				result = append(result, fieldInfo{
					Name:  name.Name,
					Index: index,
					Field: field,
				})

				index++
			}
		} else {
			result = append(result, fieldInfo{
				Name:  fmt.Sprintf("%s%d", prefix, index),
				Index: index,
				Field: field,
			})

			index++
		}
	}

	return result
}

// extractParams extracts parameter info from a function type.
func extractParams(_ *token.FileSet, ftype *dst.FuncType) []fieldInfo {
	return extractFields(ftype.Params, "param")
}

// extractPkgNameFromPath extracts the package name from an import path.
// E.g., "net/http" -> "http", "encoding/json" -> "json".
func extractPkgNameFromPath(importPath string) string {
	parts := strings.Split(importPath, "/")
	return parts[len(parts)-1]
}

// extractResults extracts result info from a function type.

// findExternalTypeAlias resolves an external type alias like fs.WalkDirFunc.
//
// Get the package identifier (e.g., "fs" in "fs.WalkDirFunc")

// Find the import path for this package

// Explicit alias: import foo "bar/baz"

// No alias - check if the path ends with this package name

// Load the external package

// Search for the type definition in the external package

// findLocalTypeAlias searches for a type alias definition in the local AST files.

// Check if the underlying type is a function type

// Stop searching

// formatResultParameters formats result parameters as "prefix0 type, prefix1 type, ...".
// namePrefix: variable name prefix ("v" or "r")
// startIndex: starting index (0 for r0-based, 1 for v1-based)
// typeFormatter: function to format each result's type.

// formatTypeParamsDecl formats type parameters for declaration (e.g., "[T any, U comparable]").
// Returns empty string if there are no type parameters.
func formatTypeParamsDecl(fset *token.FileSet, typeParams *dst.FieldList) string {
	if typeParams == nil || len(typeParams.List) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("[")

	for i, field := range typeParams.List {
		if i > 0 {
			buf.WriteString(", ")
		}

		// Write parameter names
		for j, name := range field.Names {
			if j > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(name.Name)
		}

		// Write constraint
		if field.Type != nil {
			buf.WriteString(" ")
			buf.WriteString(exprToString(fset, field.Type))
		}
	}

	buf.WriteString("]")

	return buf.String()
}

// formatTypeParamsUse formats type parameters for instantiation (e.g., "[T, U]").
// Returns empty string if there are no type parameters.
func formatTypeParamsUse(typeParams *dst.FieldList) string {
	if typeParams == nil || len(typeParams.List) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("[")

	first := true

	for _, field := range typeParams.List {
		for _, name := range field.Names {
			if !first {
				buf.WriteString(", ")
			}

			buf.WriteString(name.Name)

			first = false
		}
	}

	buf.WriteString("]")

	return buf.String()
}

// generateResultVarNames creates variable names for results (e.g., "r0", "r1" or "ret0", "ret1").

// hasExportedIdent checks if an expression contains an exported identifier.
func hasExportedIdent(expr dst.Expr, isTypeParam func(string) bool) bool {
	walker := &typeExprWalker[bool]{
		visitIdent: func(ident *dst.Ident) bool {
			return len(ident.Name) > 0 &&
				unicode.IsUpper(rune(ident.Name[0])) &&
				!isBuiltinType(ident.Name) &&
				!isTypeParam(ident.Name)
		},
		visitSelector: func(*dst.SelectorExpr) bool {
			return true
		},
		combine: func(a, b bool) bool {
			return a || b
		},
		zero: false,
	}

	return walker.walk(expr)
}

// hasFieldNames returns true if the field has explicitly named parameters/results.
// Returns false for unnamed/anonymous fields (e.g., embedded interfaces).
func hasFieldNames(field *dst.Field) bool {
	return len(field.Names) > 0
}

// isBasicComparableType determines if an expression is a basic type that supports == comparison.
// This works from syntax alone without requiring full type checking.
// Returns true for: bool, int*, uint*, float*, complex*, string, and pointers.
// Returns false for everything else (slices, maps, funcs, custom types) which should use reflect.DeepEqual.
//

// Basic built-in types
// Everything else: custom types, interfaces, etc. → use DeepEqual

// Pointers are always comparable

// Qualified types (e.g., pkg.Type) → conservatively use DeepEqual

// Arrays are comparable if their elements are comparable
// But we'd need to recursively check, so conservatively return false

// Definitely not comparable with ==

// Unknown type → use DeepEqual to be safe

// isBuiltinType checks if a type name is a Go builtin.
func isBuiltinType(name string) bool {
	switch name {
	case "bool", "byte", "complex64", "complex128",
		"error", "float32", "float64", "int",
		"int8", "int16", "int32", "int64",
		"rune", "string", "uint", "uint8",
		"uint16", "uint32", "uint64", "uintptr",
		"comparable", anyTypeString:
		return true
	}

	return false
}

// isExportedIdent checks if an identifier name is exported (starts with uppercase).
func isExportedIdent(name string) bool {
	if name == "" {
		return false
	}

	return unicode.IsUpper(rune(name[0]))
}

// isLocalPackage checks if an import path is from the current module.
// For now, we use a simple heuristic: packages from the current module
// start with "github.com/toejough/imptest".
// TODO: Make this more robust by reading the module path from go.mod.
func isLocalPackage(importPath string) bool {
	// Packages from the current module
	const modulePrefix = "github.com/toejough/imptest"

	return strings.HasPrefix(importPath, modulePrefix)
}

// isComparableExpr checks if an expression represents a comparable type.

// For DST, we need to convert to AST to look up in types.Info
// Since we don't have access to the original AST, we use syntax-based detection

// joinWith joins a slice of items into a string using a format function and separator.
// This eliminates repetitive comma-separator loop patterns throughout the codebase.
func joinWith[T any](items []T, format func(T) string, sep string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = format(item)
	}

	return strings.Join(parts, sep)
}

// newBaseGenerator initializes a baseGenerator.
func newBaseGenerator(
	fset *token.FileSet,
	pkgName, impName, pkgPath, qualifier string,
	typeParams *dst.FieldList,
) baseGenerator {
	baseGen := baseGenerator{
		typeFormatter: typeFormatter{
			fset:      fset,
			qualifier: qualifier,
		},
		pkgName:    pkgName,
		impName:    impName,
		pkgPath:    pkgPath,
		qualifier:  qualifier,
		typeParams: typeParams,
	}
	baseGen.isTypeParam = baseGen.isTypeParameter

	return baseGen
}

// normalizeVariadicType converts a variadic type string ("...T") to slice syntax ("[]T").
func normalizeVariadicType(typeStr string) string {
	if strings.HasPrefix(typeStr, "...") {
		return "[]" + typeStr[3:]
	}

	return typeStr
}

// qualifyExternalTypes walks an expression and qualifies exported type identifiers
// from an external package with the given qualifier.
// E.g., transforms "ResponseWriter" to "http.ResponseWriter" when qualifier is "http".
//
//nolint:cyclop // Expression type switching inherently requires multiple cases
func qualifyExternalTypes(expr dst.Expr, qualifier string) dst.Expr {
	// Handle nil
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *dst.Ident:
		// Only qualify exported identifiers (starts with uppercase)
		// Skip builtins: error, string, int, bool, etc.
		if isExportedIdent(node.Name) && !isBuiltinType(node.Name) {
			return &dst.SelectorExpr{
				X:   &dst.Ident{Name: qualifier},
				Sel: node,
			}
		}

		return node

	case *dst.StarExpr:
		return &dst.StarExpr{X: qualifyExternalTypes(node.X, qualifier)}

	case *dst.ArrayType:
		return &dst.ArrayType{
			Len: node.Len,
			Elt: qualifyExternalTypes(node.Elt, qualifier),
		}

	case *dst.MapType:
		return &dst.MapType{
			Key:   qualifyExternalTypes(node.Key, qualifier),
			Value: qualifyExternalTypes(node.Value, qualifier),
		}

	case *dst.ChanType:
		return &dst.ChanType{
			Dir:   node.Dir,
			Value: qualifyExternalTypes(node.Value, qualifier),
		}

	case *dst.SelectorExpr:
		// Already qualified - leave as is
		return node

	case *dst.FuncType:
		// Recurse into function type params and results
		return qualifyFuncType(node, qualifier)

	default:
		return node
	}
}

// qualifyFieldList qualifies types in a field list.
func qualifyFieldList(fl *dst.FieldList, qualifier string) *dst.FieldList {
	result := &dst.FieldList{}

	for _, field := range fl.List {
		newField := &dst.Field{
			Names: field.Names, // Keep parameter names as-is
			Type:  qualifyExternalTypes(field.Type, qualifier),
			Tag:   field.Tag,
		}
		result.List = append(result.List, newField)
	}

	return result
}

// qualifyFuncType qualifies types in a function signature.
func qualifyFuncType(funcType *dst.FuncType, qualifier string) *dst.FuncType {
	result := &dst.FuncType{
		TypeParams: funcType.TypeParams, // Don't modify type params
	}

	if funcType.Params != nil {
		result.Params = qualifyFieldList(funcType.Params, qualifier)
	}

	if funcType.Results != nil {
		result.Results = qualifyFieldList(funcType.Results, qualifier)
	}

	return result
}

// resolveImportPath finds the full import path for a package alias in the source imports.
// Returns empty string if the alias is not found.
func resolveImportPath(alias string, imports []*dst.ImportSpec) string {
	for _, imp := range imports {
		var pkgName string
		if imp.Name != nil {
			// Aliased import: `foo "github.com/example/bar"`
			pkgName = imp.Name.Name
		} else {
			// Standard import: extract package name from path
			// "github.com/example/bar" -> "bar"
			// "io" -> "io"
			path := strings.Trim(imp.Path.Value, `"`)

			lastSlash := strings.LastIndex(path, "/")
			if lastSlash >= 0 {
				pkgName = path[lastSlash+1:]
			} else {
				pkgName = path
			}
		}

		if pkgName == alias {
			return strings.Trim(imp.Path.Value, `"`)
		}
	}

	return ""
}

// resolveInterfaceGeneratorPackage resolves the package path and qualifier for interface-based
// generators (mock_interface, wrap_interface). It handles three scenarios:
//   - External qualified name (e.g., "basic.Ops") - resolves via pkgLoader
//   - External unqualified name from dot import - uses pkgImportPath directly
//   - Test package referencing non-test package - resolves via pkgLoader
func resolveInterfaceGeneratorPackage(
	info GeneratorInfo,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
) (pkgPath, qualifier string, err error) {
	if pkgImportPath != "." {
		if strings.Contains(info.InterfaceName, ".") {
			// Qualified name - use normal resolution
			return resolvePackageInfo(info, pkgLoader)
		}
		// Unqualified name - must be from dot import, use pkgImportPath directly
		return pkgImportPath, extractPkgNameFromPath(pkgImportPath), nil
	}

	if strings.HasSuffix(info.PkgName, "_test") {
		// Test package referencing non-test version
		return resolvePackageInfo(info, pkgLoader)
	}

	return "", "", nil
}

// stringifyDSTExpr delegates to astutil.StringifyExpr.

// stringifyFuncType delegates to astutil.StringifyExpr.

// stringifyInterfaceType delegates to astutil.StringifyExpr.

// stringifyStructType delegates to astutil.StringifyExpr.

// typeWithQualifierFunc handles function types.
func typeWithQualifierFunc(_ *token.FileSet, funcType *dst.FuncType, typeFormatter func(dst.Expr) string) string {
	var buf strings.Builder
	buf.WriteString("func")

	if funcType.Params != nil {
		buf.WriteString("(")

		paramParts := expandFieldListTypes(funcType.Params.List, typeFormatter)
		buf.WriteString(strings.Join(paramParts, ", "))
		buf.WriteString(")")
	}

	if funcType.Results != nil {
		if len(funcType.Results.List) > 1 {
			buf.WriteString(" (")
		} else {
			buf.WriteString(" ")
		}

		resultParts := expandFieldListTypes(funcType.Results.List, typeFormatter)
		buf.WriteString(strings.Join(resultParts, ", "))

		if len(funcType.Results.List) > 1 {
			buf.WriteString(")")
		}
	}

	return buf.String()
}
