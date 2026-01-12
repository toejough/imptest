// Package astutil provides shared utilities for AST manipulation and string formatting.
package astutil

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/dave/dst"
)

// ExpandFieldListTypes expands a field list into individual type strings.
// For fields with multiple names (e.g., "a, b int"), outputs the type once per name.
// For unnamed fields, outputs the type once.
func ExpandFieldListTypes(fields []*dst.Field, typeFormatter func(dst.Expr) string) []string {
	var parts []string

	for _, f := range fields {
		typeStr := typeFormatter(f.Type)
		// If field has names (e.g., "a, b int"), output type once per name
		// If field has no names (e.g., unnamed "int, int"), output type once
		count := len(f.Names)
		if count == 0 {
			count = 1
		}

		for range count {
			parts = append(parts, typeStr)
		}
	}

	return parts
}

// ExprToString renders a dst.Expr to Go code.
// This function converts DST expressions back to their string representation.
func ExprToString(_ *token.FileSet, expr dst.Expr) string {
	// We use a custom stringify function since decorator.Restorer.Fprint
	// only works with *dst.File, not individual expressions.
	return StringifyExpr(expr)
}

// StringifyExpr converts a DST expression to its string representation.
//
//nolint:cyclop,funlen // Type-switch dispatcher handling all DST expression types; complexity is inherent
func StringifyExpr(expr dst.Expr) string {
	if expr == nil {
		return ""
	}

	switch typedExpr := expr.(type) {
	case *dst.Ident:
		return typedExpr.Name
	case *dst.BasicLit:
		return typedExpr.Value
	case *dst.SelectorExpr:
		return StringifyExpr(typedExpr.X) + "." + typedExpr.Sel.Name
	case *dst.StarExpr:
		return "*" + StringifyExpr(typedExpr.X)
	case *dst.ArrayType:
		if typedExpr.Len != nil {
			return "[" + StringifyExpr(typedExpr.Len) + "]" + StringifyExpr(typedExpr.Elt)
		}

		return "[]" + StringifyExpr(typedExpr.Elt)
	case *dst.MapType:
		return "map[" + StringifyExpr(typedExpr.Key) + "]" + StringifyExpr(typedExpr.Value)
	case *dst.ChanType:
		switch typedExpr.Dir {
		case dst.SEND:
			return "chan<- " + StringifyExpr(typedExpr.Value)
		case dst.RECV:
			return "<-chan " + StringifyExpr(typedExpr.Value)
		default:
			return "chan " + StringifyExpr(typedExpr.Value)
		}
	case *dst.InterfaceType:
		return stringifyInterfaceType(typedExpr)
	case *dst.StructType:
		return stringifyStructType(typedExpr)
	case *dst.FuncType:
		return stringifyFuncType(typedExpr)
	case *dst.Ellipsis:
		return "..." + StringifyExpr(typedExpr.Elt)
	case *dst.IndexExpr:
		return StringifyExpr(typedExpr.X) + "[" + StringifyExpr(typedExpr.Index) + "]"
	case *dst.IndexListExpr:
		indices := make([]string, len(typedExpr.Indices))
		for i, idx := range typedExpr.Indices {
			indices[i] = StringifyExpr(idx)
		}

		return StringifyExpr(typedExpr.X) + "[" + strings.Join(indices, ", ") + "]"
	case *dst.ParenExpr:
		return "(" + StringifyExpr(typedExpr.X) + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// stringifyFuncType converts a DST FuncType to its string representation.
func stringifyFuncType(funcType *dst.FuncType) string {
	var buf strings.Builder
	buf.WriteString("func")

	// Parameters
	if funcType.Params != nil {
		buf.WriteString("(")

		paramParts := ExpandFieldListTypes(funcType.Params.List, StringifyExpr)
		buf.WriteString(strings.Join(paramParts, ", "))
		buf.WriteString(")")
	}

	// Results
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		buf.WriteString(" ")

		resultParts := ExpandFieldListTypes(funcType.Results.List, StringifyExpr)
		if len(resultParts) > 1 {
			buf.WriteString("(")
			buf.WriteString(strings.Join(resultParts, ", "))
			buf.WriteString(")")
		} else if len(resultParts) == 1 {
			buf.WriteString(resultParts[0])
		}
	}

	return buf.String()
}

// stringifyInterfaceType converts an interface type to its string representation,
// preserving method signatures for interface literals.
//
//nolint:cyclop,nestif // Complexity inherent to building interface string representation
func stringifyInterfaceType(interfaceType *dst.InterfaceType) string {
	// Empty interface
	if interfaceType.Methods == nil || len(interfaceType.Methods.List) == 0 {
		return "interface{}"
	}

	var buf strings.Builder
	buf.WriteString("interface{")

	// For single method, use compact format: interface{ MethodName(...) ... }
	// For multiple methods, use multi-line format
	methodCount := len(interfaceType.Methods.List)

	for _, method := range interfaceType.Methods.List {
		if methodCount > 1 {
			buf.WriteString("\n\t")
		} else {
			buf.WriteString(" ")
		}

		// Method name (if any - embedded interfaces have no name)
		if len(method.Names) > 0 {
			buf.WriteString(method.Names[0].Name)
		}

		// Method signature (function type)
		if funcType, ok := method.Type.(*dst.FuncType); ok {
			// Don't write "func" prefix for interface methods
			if funcType.Params != nil {
				buf.WriteString("(")

				paramParts := ExpandFieldListTypes(funcType.Params.List, StringifyExpr)
				buf.WriteString(strings.Join(paramParts, ", "))
				buf.WriteString(")")
			}

			if funcType.Results != nil && len(funcType.Results.List) > 0 {
				buf.WriteString(" ")

				resultParts := ExpandFieldListTypes(funcType.Results.List, StringifyExpr)
				if len(resultParts) > 1 {
					buf.WriteString("(")
					buf.WriteString(strings.Join(resultParts, ", "))
					buf.WriteString(")")
				} else if len(resultParts) == 1 {
					buf.WriteString(resultParts[0])
				}
			}
		} else {
			// Embedded interface - just the type
			buf.WriteString(StringifyExpr(method.Type))
		}
	}

	if methodCount > 1 {
		buf.WriteString("\n}")
	} else {
		buf.WriteString(" }")
	}

	return buf.String()
}

// stringifyStructType converts a DST StructType to its string representation,
// preserving all field information including names, types, and tags.
func stringifyStructType(structType *dst.StructType) string {
	// Handle nil/empty cases
	if structType.Fields == nil || structType.Fields.List == nil ||
		len(structType.Fields.List) == 0 {
		return "struct{}"
	}

	// Build field list
	fields := make([]string, 0, len(structType.Fields.List))

	for _, field := range structType.Fields.List {
		var fieldStr strings.Builder

		// Handle field names (can have multiple names OR be embedded with no names)
		if len(field.Names) > 0 {
			// Named field(s) - e.g., "Host, Port string"
			nameStrs := make([]string, len(field.Names))
			for i, name := range field.Names {
				nameStrs[i] = name.Name
			}

			fieldStr.WriteString(strings.Join(nameStrs, ", "))
			fieldStr.WriteString(" ")
		}

		// Get type string recursively
		fieldStr.WriteString(StringifyExpr(field.Type))

		// Add tag if present
		if field.Tag != nil {
			fieldStr.WriteString(" ")
			fieldStr.WriteString(field.Tag.Value)
		}

		fields = append(fields, fieldStr.String())
	}

	// Return formatted struct literal
	return fmt.Sprintf("struct{ %s }", strings.Join(fields, "; "))
}
