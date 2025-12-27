package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	go_types "go/types"
	"strings"
	"unicode"

	"github.com/dave/dst"
)

// GetPackageInfo extracts package info for a given target name (e.g., "pkg.Interface").
func GetPackageInfo(
	targetName string,
	pkgLoader PackageLoader,
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

	pkgPath, err = findImportPath(astFiles, pkgName, pkgLoader)
	if err != nil {
		// If it's not a package we know about, assume it's a local reference (e.g. MyType.MyMethod)
		return "", "", nil //nolint:nilerr
	}

	return pkgPath, pkgName, nil
}

// IsExportedIdent checks if an identifier is exported and not a builtin or type parameter.
func IsExportedIdent(ident *dst.Ident, isTypeParam func(string) bool) bool {
	if len(ident.Name) == 0 {
		return true
	}

	if unicode.IsUpper(rune(ident.Name[0])) {
		return true
	}

	return isBuiltinType(ident.Name) || isTypeParam(ident.Name)
}

// ValidateExportedTypes checks if an expression contains any unexported identifiers that would be inaccessible
// from another package. Returns an error if found.
func ValidateExportedTypes(expr dst.Expr, isTypeParam func(string) bool) error {
	walker := &typeExprWalker[error]{
		visitIdent: func(ident *dst.Ident) error {
			if !IsExportedIdent(ident, isTypeParam) {
				return fmt.Errorf("type '%s': %w", ident.Name, errUnexportedType)
			}

			return nil
		},
		visitSelector: func(sel *dst.SelectorExpr) error {
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

// unexported constants.
const (
	anyTypeString       = "any"
	goPackageEnvVarName = "GOPACKAGE"
	pkgImptest          = "_imptest"
	pkgReflect          = "_reflect"
	pkgTesting          = "_testing"
	pkgTime             = "_time"
)

// unexported variables.
var (
	errGOPACKAGENotSet = errors.New(goPackageEnvVarName + " environment variable not set")
	errUnexportedType  = errors.New("unexported type is not accessible from external packages")
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
	typesInfo      *go_types.Info
	needsImptest   bool
	needsReflect   bool
	needsQualifier bool
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

// checkIfValidForExternalUsage checks if the symbol can be used from an external package.
func (baseGen *baseGenerator) checkIfValidForExternalUsage(funcType *dst.FuncType) error {
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
func (w *codeWriter) pf(format string, args ...any) {
	fmt.Fprintf(&w.buf, format, args...)
}

// fieldInfo represents extracted information about a single field entry.
type fieldInfo struct {
	Name  string     // The name (explicit or generated)
	Type  string     // The type as a string
	Index int        // The overall index across all fields
	Field *dst.Field // The original AST field
}

// paramVisitor is called for each parameter during iteration.
// Returns the next (paramNameIndex, unnamedIndex).
type paramVisitor func(
	param *dst.Field,
	paramType string,
	paramNameIndex, unnamedIndex, totalParams int,
) (nextParamNameIndex, nextUnnamedIndex int)

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

// countFields counts the total number of individual fields in a field list.
func countFields(fields *dst.FieldList) int {
	if fields == nil {
		return 0
	}

	total := 0

	for _, field := range fields.List {
		total += fieldNameCount(field)
	}

	return total
}

// exprToString renders a dst.Expr to Go code.
// This function converts DST expressions back to their string representation.
func exprToString(_ *token.FileSet, expr dst.Expr) string {
	// We use a custom stringify function since decorator.Restorer.Fprint
	// only works with *dst.File, not individual expressions.
	return stringifyDSTExpr(expr)
}

// extractFields extracts all individual fields from a field list.
// For unnamed fields, generates names using the provided prefix and index.
// For named fields with multiple names, creates separate entries for each.
func extractFields(fset *token.FileSet, fields *dst.FieldList, prefix string) []fieldInfo {
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

// extractParams extracts parameter info from a function type.
func extractParams(fset *token.FileSet, ftype *dst.FuncType) []fieldInfo {
	return extractFields(fset, ftype.Params, "param")
}

// extractResults extracts result info from a function type.
func extractResults(fset *token.FileSet, ftype *dst.FuncType) []fieldInfo {
	return extractFields(fset, ftype.Results, "Result")
}

// fieldNameCount returns the number of names in a field (at least 1 for unnamed fields).
func fieldNameCount(field *dst.Field) int {
	if len(field.Names) > 0 {
		return len(field.Names)
	}

	return 1
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
func generateResultVarNames(count int, prefix string) []string {
	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("%s%d", prefix, i)
	}

	return names
}

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

// hasParams returns true if the function type has parameters.
func hasParams(ftype *dst.FuncType) bool {
	return ftype.Params != nil && len(ftype.Params.List) > 0
}

// hasResults returns true if the function type has return values.
func hasResults(ftype *dst.FuncType) bool {
	return ftype.Results != nil && len(ftype.Results.List) > 0
}

// isBasicComparableType determines if an expression is a basic type that supports == comparison.
// This works from syntax alone without requiring full type checking.
// Returns true for: bool, int*, uint*, float*, complex*, string, and pointers.
// Returns false for everything else (slices, maps, funcs, custom types) which should use reflect.DeepEqual.
//

func isBasicComparableType(expr dst.Expr) bool {
	switch t := expr.(type) {
	case *dst.Ident:
		// Basic built-in types
		switch t.Name {
		case "bool",
			"int", "int8", "int16", "int32", "int64", //nolint:goconst // Go type keywords
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"float32", "float64",
			"complex64", "complex128",
			"string", "byte", "rune": //nolint:goconst // Go type keywords
			return true
		}
		// Everything else: custom types, interfaces, etc. → use DeepEqual
		return false

	case *dst.StarExpr:
		// Pointers are always comparable
		return true

	case *dst.SelectorExpr:
		// Qualified types (e.g., pkg.Type) → conservatively use DeepEqual
		return false

	case *dst.ArrayType:
		// Arrays are comparable if their elements are comparable
		// But we'd need to recursively check, so conservatively return false
		return false

	case *dst.SliceExpr, *dst.MapType, *dst.FuncType, *dst.InterfaceType, *dst.StructType:
		// Definitely not comparable with ==
		return false

	default:
		// Unknown type → use DeepEqual to be safe
		return false
	}
}

// isBuiltinType checks if a type name is a Go builtin.
//

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

// isComparableExpr checks if an expression represents a comparable type.
func isComparableExpr(expr dst.Expr, _ *go_types.Info) bool {
	// For DST, we need to convert to AST to look up in types.Info
	// Since we don't have access to the original AST, we use syntax-based detection
	return isBasicComparableType(expr)
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

// lowerFirst lowercases the first character of a string.
// Used to create unexported field names from method names.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]

	return string(runes)
}

// newBaseGenerator initializes a baseGenerator.
func newBaseGenerator(
	fset *token.FileSet,
	pkgName, impName, pkgPath, qualifier string,
	typeParams *dst.FieldList,
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

// normalizeVariadicType converts a variadic type string ("...T") to slice syntax ("[]T").
func normalizeVariadicType(typeStr string) string {
	if strings.HasPrefix(typeStr, "...") {
		return "[]" + typeStr[3:]
	}

	return typeStr
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

// stringifyDSTExpr converts a DST expression to its string representation.
//
//nolint:cyclop,funlen // Type-switch dispatcher handling all DST expression types; complexity is inherent
func stringifyDSTExpr(expr dst.Expr) string {
	if expr == nil {
		return ""
	}

	switch typedExpr := expr.(type) {
	case *dst.Ident:
		return typedExpr.Name
	case *dst.BasicLit:
		return typedExpr.Value
	case *dst.SelectorExpr:
		return stringifyDSTExpr(typedExpr.X) + "." + typedExpr.Sel.Name
	case *dst.StarExpr:
		return "*" + stringifyDSTExpr(typedExpr.X)
	case *dst.ArrayType:
		if typedExpr.Len != nil {
			return "[" + stringifyDSTExpr(typedExpr.Len) + "]" + stringifyDSTExpr(typedExpr.Elt)
		}

		return "[]" + stringifyDSTExpr(typedExpr.Elt)
	case *dst.MapType:
		return "map[" + stringifyDSTExpr(typedExpr.Key) + "]" + stringifyDSTExpr(typedExpr.Value)
	case *dst.ChanType:
		switch typedExpr.Dir {
		case dst.SEND:
			return "chan<- " + stringifyDSTExpr(typedExpr.Value)
		case dst.RECV:
			return "<-chan " + stringifyDSTExpr(typedExpr.Value)
		default:
			return "chan " + stringifyDSTExpr(typedExpr.Value)
		}
	case *dst.FuncType:
		return stringifyFuncType(typedExpr)
	case *dst.InterfaceType:
		return "interface{}"
	case *dst.StructType:
		return "struct{}"
	case *dst.Ellipsis:
		return "..." + stringifyDSTExpr(typedExpr.Elt)
	case *dst.IndexExpr:
		return stringifyDSTExpr(typedExpr.X) + "[" + stringifyDSTExpr(typedExpr.Index) + "]"
	case *dst.IndexListExpr:
		indices := make([]string, len(typedExpr.Indices))
		for i, idx := range typedExpr.Indices {
			indices[i] = stringifyDSTExpr(idx)
		}

		return stringifyDSTExpr(typedExpr.X) + "[" + strings.Join(indices, ", ") + "]"
	case *dst.ParenExpr:
		return "(" + stringifyDSTExpr(typedExpr.X) + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// stringifyFieldList converts a field list to string.
func stringifyFieldList(fieldList *dst.FieldList) string {
	if fieldList == nil || len(fieldList.List) == 0 {
		return ""
	}

	parts := make([]string, 0, len(fieldList.List))
	for _, field := range fieldList.List {
		parts = append(parts, stringifyDSTExpr(field.Type))
	}

	return strings.Join(parts, ", ")
}

// stringifyFuncType converts a function type to string.
func stringifyFuncType(funcType *dst.FuncType) string {
	var buf strings.Builder
	buf.WriteString("func")

	if funcType.Params != nil {
		buf.WriteString("(")
		buf.WriteString(stringifyFieldList(funcType.Params))
		buf.WriteString(")")
	}

	if funcType.Results != nil {
		if len(funcType.Results.List) > 1 {
			buf.WriteString(" (")
			buf.WriteString(stringifyFieldList(funcType.Results))
			buf.WriteString(")")
		} else if len(funcType.Results.List) == 1 {
			buf.WriteString(" ")
			buf.WriteString(stringifyFieldList(funcType.Results))
		}
	}

	return buf.String()
}

// typeWithQualifierFunc handles function types.
func typeWithQualifierFunc(_ *token.FileSet, funcType *dst.FuncType, typeFormatter func(dst.Expr) string) string {
	var buf strings.Builder
	buf.WriteString("func")

	if funcType.Params != nil {
		buf.WriteString("(")
		buf.WriteString(joinWith(funcType.Params.List, func(f *dst.Field) string {
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

		buf.WriteString(joinWith(funcType.Results.List, func(f *dst.Field) string {
			return typeFormatter(f.Type)
		}, ", "))

		if len(funcType.Results.List) > 1 {
			buf.WriteString(")")
		}
	}

	return buf.String()
}

// visitParams iterates over function parameters and calls the visitor for each.
// The visitor receives each parameter with its type string and current indices,
// and returns the updated indices for the next iteration.
func visitParams(ftype *dst.FuncType, typeFormatter func(dst.Expr) string, visit paramVisitor) {
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
