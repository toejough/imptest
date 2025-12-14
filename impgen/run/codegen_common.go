package run

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
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
		if len(field.Names) > 0 {
			total += len(field.Names)
		} else {
			total++
		}
	}

	return total
}
