package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	go_types "go/types"
	"strings"
	"text/template"
	"unicode"
)

const anyTypeString = "any"

var (
	errGOPACKAGENotSet = errors.New("GOPACKAGE environment variable not set")
	errUnexportedType  = errors.New("unexported type is not accessible from external packages")
)

// typeExprWalker traverses AST type expressions with a generic return type.
// It provides a unified way to walk type expressions, handling all AST node types
// while allowing custom logic for leaf nodes (Ident, SelectorExpr) and result combining.
type typeExprWalker[T any] struct {
	visitIdent    func(*ast.Ident) T
	visitSelector func(*ast.SelectorExpr) T
	combine       func(T, T) T
	zero          T
}

// walk traverses an AST expression and returns the combined result.
//
//nolint:cyclop // Type-switch dispatcher handling all AST node types; complexity is inherent
func (w *typeExprWalker[T]) walk(expr ast.Expr) T {
	switch typeExpr := expr.(type) {
	case *ast.Ident:
		return w.visitIdent(typeExpr)
	case *ast.SelectorExpr:
		return w.visitSelector(typeExpr)
	case *ast.StarExpr:
		return w.walk(typeExpr.X)
	case *ast.ArrayType:
		return w.walk(typeExpr.Elt)
	case *ast.MapType:
		return w.combine(w.walk(typeExpr.Key), w.walk(typeExpr.Value))
	case *ast.ChanType:
		return w.walk(typeExpr.Value)
	case *ast.FuncType:
		return w.walkFieldList(typeExpr.Params, typeExpr.Results)
	case *ast.StructType:
		return w.walkFieldList(typeExpr.Fields)
	case *ast.IndexExpr:
		return w.combine(w.walk(typeExpr.X), w.walk(typeExpr.Index))
	case *ast.IndexListExpr:
		result := w.walk(typeExpr.X)
		for _, idx := range typeExpr.Indices {
			result = w.combine(result, w.walk(idx))
		}

		return result
	}

	return w.zero
}

// walkFieldList traverses field lists (for FuncType params/results or StructType fields).
func (w *typeExprWalker[T]) walkFieldList(lists ...*ast.FieldList) T {
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

// joinWith joins a slice of items into a string using a format function and separator.
// This eliminates repetitive comma-separator loop patterns throughout the codebase.
func joinWith[T any](items []T, format func(T) string, sep string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = format(item)
	}

	return strings.Join(parts, sep)
}

// codeWriter provides common buffer writing functionality for code generators.
// ... (omitting some lines for brevity, but I must match exactly).
type codeWriter struct {
	buf bytes.Buffer
}

// pf writes a formatted string to the buffer.
func (w *codeWriter) pf(format string, args ...any) {
	fmt.Fprintf(&w.buf, format, args...)
}

// ps writes a string to the buffer.
func (w *codeWriter) ps(s string) {
	w.buf.WriteString(s)
}

// bytes returns the buffer contents.
func (w *codeWriter) bytes() []byte {
	return w.buf.Bytes()
}

// normalizeVariadicType converts a variadic type string ("...T") to slice syntax ("[]T").
func normalizeVariadicType(typeStr string) string {
	if strings.HasPrefix(typeStr, "...") {
		return "[]" + typeStr[3:]
	}

	return typeStr
}

// exprToString renders an ast.Expr to Go code.
func exprToString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, expr)

	return buf.String()
}

// hasParams returns true if the function type has parameters.
func hasParams(ftype *ast.FuncType) bool {
	return ftype.Params != nil && len(ftype.Params.List) > 0
}

// hasResults returns true if the function type has return values.
func hasResults(ftype *ast.FuncType) bool {
	return ftype.Results != nil && len(ftype.Results.List) > 0
}

// countFields counts the total number of individual fields in a field list.
func countFields(fields *ast.FieldList) int {
	total := 0

	for _, field := range fields.List {
		total += fieldNameCount(field)
	}

	return total
}

// fieldNameCount returns the number of names in a field (at least 1 for unnamed fields).
func fieldNameCount(field *ast.Field) int {
	if len(field.Names) > 0 {
		return len(field.Names)
	}

	return 1
}

// hasFieldNames returns true if the field has explicitly named parameters/results.
// Returns false for unnamed/anonymous fields (e.g., embedded interfaces).
func hasFieldNames(field *ast.Field) bool {
	return len(field.Names) > 0
}

// fieldInfo represents extracted information about a single field entry.
type fieldInfo struct {
	Name  string     // The name (explicit or generated)
	Type  string     // The type as a string
	Index int        // The overall index across all fields
	Field *ast.Field // The original AST field
}

// extractFields extracts all individual fields from a field list.
// For unnamed fields, generates names using the provided prefix and index.
// For named fields with multiple names, creates separate entries for each.
func extractFields(fset *token.FileSet, fields *ast.FieldList, prefix string) []fieldInfo {
	if fields == nil {
		return nil
	}

	var result []fieldInfo

	index := 0

	for _, field := range fields.List {
		typeStr := exprToString(fset, field.Type)
		structType := normalizeVariadicType(typeStr)

		if hasFieldNames(field) {
			for _, name := range field.Names {
				result = append(result, fieldInfo{
					Name:  name.Name,
					Type:  structType,
					Index: index,
					Field: field,
				})

				index++
			}
		} else {
			result = append(result, fieldInfo{
				Name:  fmt.Sprintf("%s%d", prefix, index),
				Type:  structType,
				Index: index,
				Field: field,
			})

			index++
		}
	}

	return result
}

// extractResults extracts result info from a function type.
func extractResults(fset *token.FileSet, ftype *ast.FuncType) []fieldInfo {
	return extractFields(fset, ftype.Results, "Result")
}

// extractParams extracts parameter info from a function type.
func extractParams(fset *token.FileSet, ftype *ast.FuncType) []fieldInfo {
	return extractFields(fset, ftype.Params, "param")
}

// paramNamesToString returns a comma-separated string of parameter names.
func paramNamesToString(params []fieldInfo) string {
	if len(params) == 0 {
		return ""
	}

	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}

	return strings.Join(names, ", ")
}

// formatResultParameters formats result parameters as "prefix0 type, prefix1 type, ...".
// namePrefix: variable name prefix ("v" or "r")
// startIndex: starting index (0 for r0-based, 1 for v1-based)
// typeFormatter: function to format each result's type.
func formatResultParameters(
	results []fieldInfo,
	namePrefix string,
	startIndex int,
	typeFormatter func(fieldInfo) string,
) string {
	return joinWith(results, func(r fieldInfo) string {
		idx := r.Index + startIndex
		typePart := typeFormatter(r)

		return fmt.Sprintf("%s%d %s", namePrefix, idx, typePart)
	}, ", ")
}

// generateResultVarNames creates variable names for results (e.g., "r0", "r1" or "ret0", "ret1").
func generateResultVarNames(count int, prefix string) []string {
	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("%s%d", prefix, i)
	}

	return names
}

// paramVisitor is called for each parameter during iteration.
// Returns the next (paramNameIndex, unnamedIndex).
type paramVisitor func(
	param *ast.Field,
	paramType string,
	paramNameIndex, unnamedIndex, totalParams int,
) (nextParamNameIndex, nextUnnamedIndex int)

// visitParams iterates over function parameters and calls the visitor for each.
// The visitor receives each parameter with its type string and current indices,
// and returns the updated indices for the next iteration.
func visitParams(ftype *ast.FuncType, typeFormatter func(ast.Expr) string, visit paramVisitor) {
	if !hasParams(ftype) {
		return
	}

	totalParams := countFields(ftype.Params)
	paramNameIndex := 0
	unnamedIndex := 0

	for _, param := range ftype.Params.List {
		paramType := typeFormatter(param.Type)
		paramNameIndex, unnamedIndex = visit(param, paramType, paramNameIndex, unnamedIndex, totalParams)
	}
}

// GetPackageInfo extracts package info for a given target name (e.g., "pkg.Interface").
func GetPackageInfo(
	targetName string,
	pkgLoader PackageLoader,
	currentPkgName string,
) (pkgPath, pkgName string, err error) {
	dotIdx := strings.Index(targetName, ".")
	if dotIdx == -1 {
		return "", "", nil
	}

	pkgName = targetName[:dotIdx]
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

	pkgPath, err = findImportPath(astFiles, pkgName, pkgLoader)
	if err != nil {
		// If it's not a package we know about, assume it's a local reference (e.g. MyType.MyMethod)
		return "", "", nil //nolint:nilerr
	}

	return pkgPath, pkgName, nil
}

// hasExportedIdent checks if an expression contains an exported identifier.
func hasExportedIdent(expr ast.Expr, isTypeParam func(string) bool) bool {
	walker := &typeExprWalker[bool]{
		visitIdent: func(ident *ast.Ident) bool {
			return len(ident.Name) > 0 &&
				unicode.IsUpper(rune(ident.Name[0])) &&
				!isBuiltinType(ident.Name) &&
				!isTypeParam(ident.Name)
		},
		visitSelector: func(*ast.SelectorExpr) bool {
			return true
		},
		combine: func(a, b bool) bool {
			return a || b
		},
		zero: false,
	}

	return walker.walk(expr)
}

// isBuiltinType checks if a type name is a Go builtin.
func isBuiltinType(name string) bool {
	switch name {
	case "bool", "byte", "complex64", "complex128",
		"error", "float32", "float64", "int",
		"int8", "int16", "int32", "int64",
		"rune", "string", "uint", "uint8",
		"uint16", "uint32", "uint64", "uintptr",
		anyTypeString:
		return true
	}

	return false
}

// typeWithQualifierFunc handles function types.
func typeWithQualifierFunc(_ *token.FileSet, funcType *ast.FuncType, typeFormatter func(ast.Expr) string) string {
	var buf strings.Builder
	buf.WriteString("func")

	if funcType.Params != nil {
		buf.WriteString("(")
		buf.WriteString(joinWith(funcType.Params.List, func(f *ast.Field) string {
			return typeFormatter(f.Type)
		}, ", "))
		buf.WriteString(")")
	}

	if funcType.Results != nil {
		if len(funcType.Results.List) > 1 {
			buf.WriteString(" (")
		} else {
			buf.WriteString(" ")
		}

		buf.WriteString(joinWith(funcType.Results.List, func(f *ast.Field) string {
			return typeFormatter(f.Type)
		}, ", "))

		if len(funcType.Results.List) > 1 {
			buf.WriteString(")")
		}
	}

	return buf.String()
}

// ValidateExportedTypes checks if an expression contains any unexported identifiers that would be inaccessible
// from another package. Returns an error if found.
func ValidateExportedTypes(expr ast.Expr, isTypeParam func(string) bool) error {
	walker := &typeExprWalker[error]{
		visitIdent: func(ident *ast.Ident) error {
			if !IsExportedIdent(ident, isTypeParam) {
				return fmt.Errorf("type '%s': %w", ident.Name, errUnexportedType)
			}

			return nil
		},
		visitSelector: func(sel *ast.SelectorExpr) error {
			if !unicode.IsUpper(rune(sel.Sel.Name[0])) {
				return fmt.Errorf("type '%s': %w", sel.Sel.Name, errUnexportedType)
			}

			return nil
		},
		combine: func(a, b error) error {
			if a != nil {
				return a
			}

			return b
		},
		zero: nil,
	}

	return walker.walk(expr)
}

// IsExportedIdent checks if an identifier is exported and not a builtin or type parameter.
func IsExportedIdent(ident *ast.Ident, isTypeParam func(string) bool) bool {
	if len(ident.Name) == 0 {
		return true
	}

	if unicode.IsUpper(rune(ident.Name[0])) {
		return true
	}

	return isBuiltinType(ident.Name) || isTypeParam(ident.Name)
}

// isComparableExpr checks if an expression represents a comparable type.
func isComparableExpr(expr ast.Expr, typesInfo *go_types.Info) bool {
	if typesInfo == nil {
		return false // Conservatively assume not comparable if type info is unavailable
	}

	t := typesInfo.Types[expr].Type
	if t == nil {
		return false // Type not found, assume not comparable
	}

	return go_types.Comparable(t)
}

// typeFormatter handles formatting AST types into strings with package qualifiers.
type typeFormatter struct {
	fset        *token.FileSet
	qualifier   string
	isTypeParam func(string) bool
}

// typeWithQualifier returns a type expression as a string with package qualifier if needed.
func (tf *typeFormatter) typeWithQualifier(expr ast.Expr) string {
	switch typeExpr := expr.(type) {
	case *ast.Ident:
		return tf.typeWithQualifierIdent(typeExpr)
	case *ast.StarExpr:
		return tf.typeWithQualifierStar(typeExpr)
	case *ast.SelectorExpr:
		return exprToString(tf.fset, typeExpr)
	default:
		return tf.typeWithQualifierComposite(expr)
	}
}

// typeWithQualifierComposite handles composite types like arrays, maps, and funcs.
func (tf *typeFormatter) typeWithQualifierComposite(expr ast.Expr) string {
	switch typeExpr := expr.(type) {
	case *ast.ArrayType:
		return tf.typeWithQualifierArray(typeExpr)
	case *ast.MapType:
		return tf.typeWithQualifierMap(typeExpr)
	case *ast.ChanType:
		return tf.typeWithQualifierChan(typeExpr)
	case *ast.FuncType:
		return tf.typeWithQualifierFunc(typeExpr)
	case *ast.IndexExpr:
		return tf.typeWithQualifierIndex(typeExpr)
	case *ast.IndexListExpr:
		return tf.typeWithQualifierIndexList(typeExpr)
	default:
		return exprToString(tf.fset, expr)
	}
}

// typeWithQualifierArray handles array/slice types.
func (tf *typeFormatter) typeWithQualifierArray(arrType *ast.ArrayType) string {
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
func (tf *typeFormatter) typeWithQualifierChan(chanType *ast.ChanType) string {
	var buf strings.Builder

	switch chanType.Dir {
	case ast.SEND:
		buf.WriteString("chan<- ")
	case ast.RECV:
		buf.WriteString("<-chan ")
	default:
		buf.WriteString("chan ")
	}

	buf.WriteString(tf.typeWithQualifier(chanType.Value))

	return buf.String()
}

// typeWithQualifierFunc handles function types.
func (tf *typeFormatter) typeWithQualifierFunc(funcType *ast.FuncType) string {
	return typeWithQualifierFunc(tf.fset, funcType, tf.typeWithQualifier)
}

// typeWithQualifierIdent handles simple identifier types.
func (tf *typeFormatter) typeWithQualifierIdent(ident *ast.Ident) string {
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
func (tf *typeFormatter) typeWithQualifierIndex(indexExpr *ast.IndexExpr) string {
	var buf strings.Builder

	buf.WriteString(tf.typeWithQualifier(indexExpr.X))
	buf.WriteString("[")
	buf.WriteString(tf.typeWithQualifier(indexExpr.Index))
	buf.WriteString("]")

	return buf.String()
}

// typeWithQualifierIndexList handles generic type instantiation with multiple type parameters.
func (tf *typeFormatter) typeWithQualifierIndexList(indexListExpr *ast.IndexListExpr) string {
	var buf strings.Builder

	buf.WriteString(tf.typeWithQualifier(indexListExpr.X))
	buf.WriteString("[")
	buf.WriteString(joinWith(indexListExpr.Indices, tf.typeWithQualifier, ", "))
	buf.WriteString("]")

	return buf.String()
}

// typeWithQualifierMap handles map types.
func (tf *typeFormatter) typeWithQualifierMap(mapType *ast.MapType) string {
	var buf strings.Builder

	buf.WriteString("map[")
	buf.WriteString(tf.typeWithQualifier(mapType.Key))
	buf.WriteString("]")
	buf.WriteString(tf.typeWithQualifier(mapType.Value))

	return buf.String()
}

// typeWithQualifierStar handles pointer types.
func (tf *typeFormatter) typeWithQualifierStar(t *ast.StarExpr) string {
	var buf strings.Builder

	buf.WriteString("*")
	buf.WriteString(tf.typeWithQualifier(t.X))

	return buf.String()
}

// newBaseGenerator initializes a baseGenerator.
func newBaseGenerator(
	fset *token.FileSet,
	pkgName, impName, pkgPath, qualifier string,
	typeParams *ast.FieldList,
	typesInfo *go_types.Info,
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
		typesInfo:  typesInfo,
	}
	baseGen.isTypeParam = baseGen.isTypeParameter

	return baseGen
}

// baseGenerator holds common state and methods for code generation.
type baseGenerator struct {
	codeWriter
	typeFormatter

	pkgName        string
	impName        string
	pkgPath        string
	qualifier      string
	typeParams     *ast.FieldList
	typesInfo      *go_types.Info
	needsImptest   bool
	needsReflect   bool
	needsQualifier bool
}

// checkIfQualifierNeeded pre-scans to determine if the package qualifier is needed.
func (baseGen *baseGenerator) checkIfQualifierNeeded(expr ast.Expr) {
	if baseGen.qualifier == "" {
		return
	}

	if hasExportedIdent(expr, baseGen.isTypeParam) {
		baseGen.needsQualifier = true
	}
}

// checkIfValidForExternalUsage checks if the symbol can be used from an external package.
func (baseGen *baseGenerator) checkIfValidForExternalUsage(funcType *ast.FuncType) error {
	if baseGen.qualifier == "" {
		return nil
	}

	// Validate params
	if funcType.Params != nil {
		for _, field := range funcType.Params.List {
			err := ValidateExportedTypes(field.Type, baseGen.isTypeParam)
			if err != nil {
				return err
			}
		}
	}

	// Validate results
	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			err := ValidateExportedTypes(field.Type, baseGen.isTypeParam)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// execTemplate executes a template and writes the result to the buffer.
func (baseGen *baseGenerator) execTemplate(tmpl *template.Template, data any) {
	baseGen.ps(executeTemplate(tmpl, data))
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

// formatTypeParamsDecl formats type parameters for declaration (e.g., "[T any, U comparable]").
// Returns empty string if there are no type parameters.
func formatTypeParamsDecl(fset *token.FileSet, typeParams *ast.FieldList) string {
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
func formatTypeParamsUse(typeParams *ast.FieldList) string {
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
