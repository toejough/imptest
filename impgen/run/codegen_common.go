package run

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	go_types "go/types"
	"slices"
	"strings"
	"unicode"
)

// codeWriter provides common buffer writing functionality for code generators.
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

// getPackageInfo extracts package info for a given target name (e.g., "pkg.Interface").
func getPackageInfo(
	targetName string,
	pkgLoader PackageLoader,
	usesExportedTypes func() bool,
) (pkgPath, pkgName string, err error) {
	if !strings.Contains(targetName, ".") {
		return "", "", nil
	}

	if !usesExportedTypes() {
		return "", "", nil
	}

	pkgName = extractPackageName(targetName)

	astFiles, _, _, err := pkgLoader.Load(".")
	if err != nil {
		return "", "", fmt.Errorf("failed to load local package: %w", err)
	}

	pkgPath, err = findImportPath(astFiles, pkgName, pkgLoader)
	if err != nil {
		return "", "", err
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
	if hasExportedIdent(indexList.X, isTypeParam) {
		return true
	}

	return slices.ContainsFunc(indexList.Indices, func(e ast.Expr) bool { return hasExportedIdent(e, isTypeParam) })
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
	builtins := map[string]bool{
		"bool": true, "byte": true, "complex64": true, "complex128": true,
		"error": true, "float32": true, "float64": true, "int": true,
		"int8": true, "int16": true, "int32": true, "int64": true,
		"rune": true, "string": true, "uint": true, "uint8": true,
		"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
		"any": true,
	}

	return builtins[name]
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
