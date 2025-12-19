package run

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
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
func visitParams(fset *token.FileSet, ftype *ast.FuncType, visit paramVisitor) {
	if !hasParams(ftype) {
		return
	}

	totalParams := countFields(ftype.Params)
	paramNameIndex := 0
	unnamedIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(fset, param.Type)
		paramNameIndex, unnamedIndex = visit(param, paramType, paramNameIndex, unnamedIndex, totalParams)
	}
}
