package generate

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"sort"
	"strings"
	"unicode"

	"github.com/dave/dst"

	astutil "github.com/toejough/imptest/impgen/run/0_util"
	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

type NamingMode int

// NamingMode values.
const (
	NamingModeDefault NamingMode = iota
	NamingModeTarget
	NamingModeDependency
)

// Exported variables.
var (
	ErrNotPackageReference = errors.New("not a package reference")
)

type GeneratorInfo struct {
	PkgName            string
	InterfaceName      string
	LocalInterfaceName string
	ImpName            string
	Mode               NamingMode
	ImportPathFlag     string
	NameProvided       bool // true if --name was explicitly provided
}

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
		// Not found as a package - likely a type method reference like Counter.Inc
		return "", "", ErrNotPackageReference
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
func (baseGen *baseGenerator) buildParamStrings(
	ftype *dst.FuncType,
) (paramsStr string, paramNames []string) {
	if ftype.Params == nil {
		return "", nil
	}

	var parts []string

	for _, field := range ftype.Params.List {
		fieldType := baseGen.typeWithQualifier(field.Type)
		names := fieldParamNames(field, len(paramNames))

		for _, name := range names {
			parts = append(parts, name+" "+fieldType)
			paramNames = append(paramNames, name)
		}
	}

	return strings.Join(parts, ", "), paramNames
}

// buildResultStrings builds the result string and collects result types from a function type.
func (baseGen *baseGenerator) buildResultStrings(
	ftype *dst.FuncType,
) (resultsStr string, resultTypes []string) {
	if ftype.Results == nil || len(ftype.Results.List) == 0 {
		return "", nil
	}

	resultTypes = baseGen.collectResultTypes(ftype.Results)
	resultsStr = baseGen.formatResultTypes(ftype.Results, resultTypes)

	return resultsStr, resultTypes
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

// collectResultTypes extracts all result types from a field list.
func (baseGen *baseGenerator) collectResultTypes(results *dst.FieldList) []string {
	var types []string

	for _, field := range results.List {
		fieldType := baseGen.typeWithQualifier(field.Type)

		count := len(field.Names)
		if count == 0 {
			count = 1
		}

		for range count {
			types = append(types, fieldType)
		}
	}

	return types
}

// formatResultTypes formats result types into a return type string.
func (baseGen *baseGenerator) formatResultTypes(results *dst.FieldList, types []string) string {
	hasMultiple := len(results.List) > 1 ||
		(len(results.List) == 1 && len(results.List[0].Names) > 1)

	joined := strings.Join(types, ", ")

	if hasMultiple {
		return " (" + joined + ")"
	}

	return " " + joined
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

type codeWriter struct {
	buf bytes.Buffer
}

// bytes returns the buffer contents.
func (w *codeWriter) bytes() []byte {
	return w.buf.Bytes()
}

type fieldInfo struct {
	Name  string     // The name (explicit or generated)
	Index int        // The overall index across all fields
	Field *dst.Field // The original AST field (use Field.Type with typeWithQualifier)
}

type typeExprWalker[T any] struct {
	visitIdent    func(*dst.Ident) T
	visitSelector func(*dst.SelectorExpr) T
	combine       func(T, T) T
	zero          T
}

// walk traverses an AST expression and returns the combined result.
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
	case *dst.ChanType:
		return w.walk(typeExpr.Value)
	case *dst.MapType:
		return w.walkMapType(typeExpr)
	case *dst.FuncType:
		return w.walkFieldList(typeExpr.Params, typeExpr.Results)
	case *dst.StructType:
		return w.walkFieldList(typeExpr.Fields)
	case *dst.IndexExpr, *dst.IndexListExpr:
		return w.walkIndexType(expr)
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

func (w *typeExprWalker[T]) walkIndexListExpr(typeExpr *dst.IndexListExpr) T {
	result := w.walk(typeExpr.X)
	for _, idx := range typeExpr.Indices {
		result = w.combine(result, w.walk(idx))
	}

	return result
}

// walkIndexType handles generic type indexing (IndexExpr and IndexListExpr).
func (w *typeExprWalker[T]) walkIndexType(expr dst.Expr) T {
	switch typed := expr.(type) {
	case *dst.IndexExpr:
		return w.combine(w.walk(typed.X), w.walk(typed.Index))
	case *dst.IndexListExpr:
		return w.walkIndexListExpr(typed)
	default:
		// Should never happen - caller guarantees expr is IndexExpr or IndexListExpr
		return w.zero
	}
}

// walkMapType handles map type traversal.
func (w *typeExprWalker[T]) walkMapType(typeExpr *dst.MapType) T {
	return w.combine(w.walk(typeExpr.Key), w.walk(typeExpr.Value))
}

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

// buildSourcePackageNameMap builds a map of package names to their import paths.
func buildSourcePackageNameMap(sourceImports []*dst.ImportSpec) map[string]string {
	result := make(map[string]string)

	for _, imp := range sourceImports {
		path := strings.Trim(imp.Path.Value, `"`)
		pkgName := extractPackageNameFromImport(imp, path)
		result[pkgName] = path
	}

	return result
}

// visitParams iterates over function parameters and calls the visitor for each.
// The visitor receives each parameter with its type string and current indices,
// and returns the updated indices for the next iteration.

// buildVariadicArgs checks for variadic parameters and builds argument strings.
func buildVariadicArgs(ftype *dst.FuncType, paramNames []string) variadicArgsResult {
	hasVariadic := isVariadicFunc(ftype)
	allArgs := strings.Join(paramNames, ", ")

	if !hasVariadic || len(paramNames) == 0 {
		return variadicArgsResult{
			hasVariadic: hasVariadic,
			allArgs:     allArgs,
		}
	}

	return variadicArgsResult{
		hasVariadic:     true,
		nonVariadicArgs: strings.Join(paramNames[:len(paramNames)-1], ", "),
		variadicArg:     paramNames[len(paramNames)-1],
		allArgs:         allArgs,
	}
}

// collectExternalImports walks a type expression and collects package references
// from SelectorExpr nodes (e.g., "io.Reader", "os.FileMode").
// It resolves each package reference to its full import path using the source imports.
// For stdlib packages, it adds a "_" prefix when there's a naming conflict with non-stdlib imports.
func collectExternalImports(expr dst.Expr, sourceImports []*dst.ImportSpec) []importInfo {
	var imports []importInfo

	seen := make(map[string]bool)
	sourcePackageNames := buildSourcePackageNameMap(sourceImports)

	walker := &typeExprWalker[struct{}]{
		visitIdent: func(*dst.Ident) struct{} { return struct{}{} },
		visitSelector: func(sel *dst.SelectorExpr) struct{} {
			if imp := collectImportFromSelector(sel, sourceImports, sourcePackageNames, seen); imp != nil {
				imports = append(imports, *imp)
			}

			return struct{}{}
		},
		combine: func(_, _ struct{}) struct{} { return struct{}{} },
		zero:    struct{}{},
	}

	walker.walk(expr)

	return imports
}

// collectImportFromSelector extracts import info from a selector expression.
func collectImportFromSelector(
	sel *dst.SelectorExpr,
	sourceImports []*dst.ImportSpec,
	sourcePackageNames map[string]string,
	seen map[string]bool,
) *importInfo {
	ident, ok := sel.X.(*dst.Ident)
	if !ok {
		return nil
	}

	pkgAlias := ident.Name
	path := resolveImportPath(pkgAlias, sourceImports)

	if path == "" || seen[path] {
		return nil
	}

	seen[path] = true
	alias := resolveAliasForImport(pkgAlias, path, sourcePackageNames)

	return &importInfo{Alias: alias, Path: path}
}

// collectImportsFromFieldList collects imports from a field list into a map.
func collectImportsFromFieldList(
	fields *dst.FieldList,
	sourceImports []*dst.ImportSpec,
	allImports map[string]importInfo,
) {
	if fields == nil {
		return
	}

	for _, field := range fields.List {
		for _, imp := range collectExternalImports(field.Type, sourceImports) {
			allImports[imp.Path] = imp
		}
	}
}

// collectImportsFromFuncDecl collects additional imports needed for a function declaration's parameters and returns.
// This is shared logic used by both callableGenerator and targetGenerator.
func collectImportsFromFuncDecl(funcDecl *dst.FuncDecl, astFiles []*dst.File) []importInfo {
	if len(astFiles) == 0 {
		return nil
	}

	sourceImports := findSourceImports(astFiles)
	allImports := make(map[string]importInfo)

	collectImportsFromFieldList(funcDecl.Type.Params, sourceImports, allImports)
	collectImportsFromFieldList(funcDecl.Type.Results, sourceImports, allImports)

	return sortedImportSlice(allImports)
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

// extractPackageNameFromImport extracts the package name from an import spec.
func extractPackageNameFromImport(imp *dst.ImportSpec, path string) string {
	if imp.Name != nil {
		return imp.Name.Name
	}

	lastSlash := strings.LastIndex(path, "/")
	if lastSlash >= 0 {
		return path[lastSlash+1:]
	}

	return path
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

// fieldParamNames returns the parameter names for a field.
// For named parameters, returns the names from the AST.
// For unnamed parameters or blank identifiers, generates a name like "arg1".
func fieldParamNames(field *dst.Field, currentIndex int) []string {
	if len(field.Names) > 0 {
		names := make([]string, len(field.Names))
		for i, name := range field.Names {
			// Blank identifier can't be used as a value, generate synthetic name
			if name.Name == "_" {
				names[i] = fmt.Sprintf("arg%d", currentIndex+i+1)
			} else {
				names[i] = name.Name
			}
		}

		return names
	}

	return []string{fmt.Sprintf("arg%d", currentIndex+1)}
}

// findSourceImports finds the first non-empty imports from AST files.
func findSourceImports(astFiles []*dst.File) []*dst.ImportSpec {
	for _, file := range astFiles {
		if len(file.Imports) > 0 {
			return file.Imports
		}
	}

	return nil
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

// isVariadicFunc checks if the function has a variadic parameter.
func isVariadicFunc(ftype *dst.FuncType) bool {
	if ftype.Params == nil || len(ftype.Params.List) == 0 {
		return false
	}

	lastField := ftype.Params.List[len(ftype.Params.List)-1]
	_, isEllipsis := lastField.Type.(*dst.Ellipsis)

	return isEllipsis
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
func qualifyExternalTypes(expr dst.Expr, qualifier string) dst.Expr {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *dst.Ident:
		return qualifyIdent(node, qualifier)
	case *dst.StarExpr:
		return &dst.StarExpr{X: qualifyExternalTypes(node.X, qualifier)}
	case *dst.ArrayType:
		return &dst.ArrayType{Len: node.Len, Elt: qualifyExternalTypes(node.Elt, qualifier)}
	case *dst.MapType:
		return &dst.MapType{
			Key:   qualifyExternalTypes(node.Key, qualifier),
			Value: qualifyExternalTypes(node.Value, qualifier),
		}
	case *dst.ChanType:
		return &dst.ChanType{Dir: node.Dir, Value: qualifyExternalTypes(node.Value, qualifier)}
	case *dst.SelectorExpr:
		return node // Already qualified
	case *dst.FuncType:
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

// qualifyIdent qualifies an identifier if it's an exported non-builtin type.
func qualifyIdent(node *dst.Ident, qualifier string) dst.Expr {
	if isExportedIdent(node.Name) && !isBuiltinType(node.Name) {
		return &dst.SelectorExpr{
			X:   &dst.Ident{Name: qualifier},
			Sel: node,
		}
	}

	return node
}

// resolveAliasForImport determines the alias to use for an import.
// For stdlib packages with naming conflicts, adds a "_" prefix.
func resolveAliasForImport(pkgAlias, path string, sourcePackageNames map[string]string) string {
	if !detect.IsStdlibPackage(path) || path != pkgAlias {
		return pkgAlias
	}

	if existingPath, exists := sourcePackageNames[pkgAlias]; exists &&
		!detect.IsStdlibPackage(existingPath) {
		return "_" + pkgAlias
	}

	return pkgAlias
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
//
// Note: resolvePackageInfo converts all ErrNotPackageReference errors to empty returns,
// and GetPackageInfo only returns nil or ErrNotPackageReference, so this cannot fail.
func resolveInterfaceGeneratorPackage(
	info GeneratorInfo,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
) (pkgPath, qualifier string) {
	if pkgImportPath != "." {
		if strings.Contains(info.InterfaceName, ".") {
			// Qualified name - use normal resolution
			pkgPath, qualifier, _ = resolvePackageInfo(info, pkgLoader)
			return pkgPath, qualifier
		}
		// Unqualified name - must be from dot import, use pkgImportPath directly
		return pkgImportPath, extractPkgNameFromPath(pkgImportPath)
	}

	if strings.HasSuffix(info.PkgName, "_test") {
		// Test package referencing non-test version
		pkgPath, qualifier, _ = resolvePackageInfo(info, pkgLoader)
		return pkgPath, qualifier
	}

	return "", ""
}

// sortedImportSlice converts an import map to a sorted slice.
func sortedImportSlice(imports map[string]importInfo) []importInfo {
	result := make([]importInfo, 0, len(imports))

	for _, imp := range imports {
		result = append(result, imp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// stringifyDSTExpr delegates to astutil.StringifyExpr.

// stringifyFuncType delegates to astutil.StringifyExpr.

// stringifyInterfaceType delegates to astutil.StringifyExpr.

// stringifyStructType delegates to astutil.StringifyExpr.

// typeWithQualifierFunc handles function types.
func typeWithQualifierFunc(
	_ *token.FileSet,
	funcType *dst.FuncType,
	typeFormatter func(dst.Expr) string,
) string {
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
