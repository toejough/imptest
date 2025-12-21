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
	"unicode"
)

var (
	errGOPACKAGENotSet = errors.New("GOPACKAGE environment variable not set")
	errUnexportedType  = errors.New("unexported type is not accessible from external packages")
)

// codeWriter provides common buffer writing functionality for code generators.
// ... (omitting some lines for brevity, but I must match exactly).
type codeWriter struct {
	buf  bytes.Buffer
	fset *token.FileSet
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

		// Normalize variadic type for struct fields
		structType := typeStr
		if strings.HasPrefix(structType, "...") {
			structType = "[]" + structType[3:]
		}

		if len(field.Names) > 0 {
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
//
//nolint:cyclop // Simple type-switch dispatcher with no nested logic; complexity is inherent to AST node types
func hasExportedIdent(expr ast.Expr, isTypeParam func(string) bool) bool {
	switch typeExpr := expr.(type) {
	case *ast.Ident:
		return hasExportedIdentInIdent(typeExpr, isTypeParam)
	case *ast.StarExpr:
		return hasExportedIdentInStar(typeExpr, isTypeParam)
	case *ast.ArrayType:
		return hasExportedIdentInArray(typeExpr, isTypeParam)
	case *ast.MapType:
		return hasExportedIdentInMap(typeExpr, isTypeParam)
	case *ast.ChanType:
		return hasExportedIdentInChan(typeExpr, isTypeParam)
	case *ast.FuncType:
		return hasExportedIdentInFunc(typeExpr, isTypeParam)
	case *ast.StructType:
		return hasExportedIdentInStruct(typeExpr, isTypeParam)
	case *ast.SelectorExpr:
		return hasExportedIdentInSelector(typeExpr, isTypeParam)
	case *ast.IndexExpr:
		return hasExportedIdentInIndex(typeExpr, isTypeParam)
	case *ast.IndexListExpr:
		return hasExportedIdentInIndexList(typeExpr, isTypeParam)
	}

	return false
}

// hasExportedIdentInArray checks if an array type contains exported identifiers.
func hasExportedIdentInArray(t *ast.ArrayType, isTypeParam func(string) bool) bool {
	return hasExportedIdent(t.Elt, isTypeParam)
}

// hasExportedIdentInChan checks if a channel type contains exported identifiers.
func hasExportedIdentInChan(t *ast.ChanType, isTypeParam func(string) bool) bool {
	return hasExportedIdent(t.Value, isTypeParam)
}

// hasExportedIdentInFunc checks if a function type contains exported identifiers.
func hasExportedIdentInFunc(funcType *ast.FuncType, isTypeParam func(string) bool) bool {
	// Check parameters for exported types
	if funcType.Params != nil {
		for _, field := range funcType.Params.List {
			if hasExportedIdent(field.Type, isTypeParam) {
				return true
			}
		}
	}

	// Check results for exported types
	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			if hasExportedIdent(field.Type, isTypeParam) {
				return true
			}
		}
	}

	return false
}

// hasExportedIdentInIdent checks if an identifier is exported.
func hasExportedIdentInIdent(t *ast.Ident, isTypeParam func(string) bool) bool {
	return len(t.Name) > 0 && unicode.IsUpper(rune(t.Name[0])) && !isBuiltinType(t.Name) && !isTypeParam(t.Name)
}

// hasExportedIdentInIndex checks if a generic type instantiation contains exported identifiers.
func hasExportedIdentInIndex(t *ast.IndexExpr, isTypeParam func(string) bool) bool {
	return hasExportedIdent(t.X, isTypeParam) || hasExportedIdent(t.Index, isTypeParam)
}

// hasExportedIdentInIndexList checks if a multi-parameter generic type contains exported identifiers.
func hasExportedIdentInIndexList(indexList *ast.IndexListExpr, isTypeParam func(string) bool) bool {
	return hasExportedIdent(indexList.X, isTypeParam)
}

// hasExportedIdentInMap checks if a map type contains exported identifiers.
func hasExportedIdentInMap(t *ast.MapType, isTypeParam func(string) bool) bool {
	return hasExportedIdent(t.Key, isTypeParam) || hasExportedIdent(t.Value, isTypeParam)
}

// hasExportedIdentInSelector checks if a selector expression is exported.
func hasExportedIdentInSelector(_ *ast.SelectorExpr, _ func(string) bool) bool {
	return true
}

// hasExportedIdentInStar checks if a pointer type contains exported identifiers.
func hasExportedIdentInStar(t *ast.StarExpr, isTypeParam func(string) bool) bool {
	return hasExportedIdent(t.X, isTypeParam)
}

// hasExportedIdentInStruct checks if a struct type contains exported identifiers.
func hasExportedIdentInStruct(t *ast.StructType, isTypeParam func(string) bool) bool {
	if t.Fields != nil {
		for _, field := range t.Fields.List {
			if hasExportedIdent(field.Type, isTypeParam) {
				return true
			}
		}
	}

	return false
}

// isBuiltinType checks if a type name is a Go builtin.
func isBuiltinType(name string) bool {
	switch name {
	case "bool", "byte", "complex64", "complex128",
		"error", "float32", "float64", "int",
		"int8", "int16", "int32", "int64",
		"rune", "string", "uint", "uint8",
		"uint16", "uint32", "uint64", "uintptr",
		"any":
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

		for i, field := range funcType.Params.List {
			if i > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(typeFormatter(field.Type))
		}

		buf.WriteString(")")
	}

	if funcType.Results != nil {
		if len(funcType.Results.List) > 1 {
			buf.WriteString(" (")
		} else {
			buf.WriteString(" ")
		}

		for i, field := range funcType.Results.List {
			if i > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(typeFormatter(field.Type))
		}

		if len(funcType.Results.List) > 1 {
			buf.WriteString(")")
		}
	}

	return buf.String()
}

// ValidateExportedTypes checks if an expression contains any unexported identifiers that would be inaccessible
// from another package. Returns an error if found.
func ValidateExportedTypes(expr ast.Expr, isTypeParam func(string) bool) error {
	switch typeExpr := expr.(type) {
	case *ast.Ident, *ast.StarExpr, *ast.ArrayType, *ast.ChanType:
		return validateExportedSimpleTypes(expr, isTypeParam)
	case *ast.MapType:
		return validateExportedMapTypes(typeExpr, isTypeParam)
	case *ast.FuncType:
		return ValidateExportedTypesInFunc(typeExpr, isTypeParam)
	case *ast.IndexExpr:
		return validateExportedIndexTypes(typeExpr, isTypeParam)
	case *ast.IndexListExpr:
		return validateExportedIndexListTypes(typeExpr, isTypeParam)
	case *ast.SelectorExpr:
		if !unicode.IsUpper(rune(typeExpr.Sel.Name[0])) {
			return fmt.Errorf("type '%s': %w", typeExpr.Sel.Name, errUnexportedType)
		}

		return nil
	case *ast.StructType:
		return validateExportedStructTypes(typeExpr, isTypeParam)
	}

	return nil
}

// validateExportedSimpleTypes handles simple types (Ident, StarExpr, ArrayType, ChanType).
func validateExportedSimpleTypes(expr ast.Expr, isTypeParam func(string) bool) error {
	switch typeExpr := expr.(type) {
	case *ast.Ident:
		if !IsExportedIdent(typeExpr, isTypeParam) {
			return fmt.Errorf("type '%s': %w", typeExpr.Name, errUnexportedType)
		}

		return nil
	case *ast.StarExpr:
		return ValidateExportedTypes(typeExpr.X, isTypeParam)
	case *ast.ArrayType:
		return ValidateExportedTypes(typeExpr.Elt, isTypeParam)
	case *ast.ChanType:
		return ValidateExportedTypes(typeExpr.Value, isTypeParam)
	}

	return nil
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

// validateExportedIndexListTypes checks if a generic type instantiation with multiple
// parameters contains exported identifiers.
func validateExportedIndexListTypes(indexList *ast.IndexListExpr, isTypeParam func(string) bool) error {
	err := ValidateExportedTypes(indexList.X, isTypeParam)
	if err != nil {
		return err
	}

	for _, idx := range indexList.Indices {
		err := ValidateExportedTypes(idx, isTypeParam)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateExportedIndexTypes checks if a generic type instantiation contains exported identifiers.
func validateExportedIndexTypes(indexExpr *ast.IndexExpr, isTypeParam func(string) bool) error {
	err := ValidateExportedTypes(indexExpr.X, isTypeParam)
	if err != nil {
		return err
	}

	return ValidateExportedTypes(indexExpr.Index, isTypeParam)
}

// validateExportedMapTypes checks if a map type contains exported identifiers.
func validateExportedMapTypes(mapType *ast.MapType, isTypeParam func(string) bool) error {
	err := ValidateExportedTypes(mapType.Key, isTypeParam)
	if err != nil {
		return err
	}

	return ValidateExportedTypes(mapType.Value, isTypeParam)
}

// validateExportedStructTypes checks if a struct type contains exported identifiers.
func validateExportedStructTypes(structType *ast.StructType, isTypeParam func(string) bool) error {
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			err := ValidateExportedTypes(field.Type, isTypeParam)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateExportedTypesInFunc(funcType *ast.FuncType, isTypeParam func(string) bool) error {
	if funcType.Params != nil {
		for _, field := range funcType.Params.List {
			err := ValidateExportedTypes(field.Type, isTypeParam)
			if err != nil {
				return err
			}
		}
	}

	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			err := ValidateExportedTypes(field.Type, isTypeParam)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
