//nolint:testpackage // Need same package to test unexported isComparableExpr
package run

import (
	"go/ast"
	"go/types"
	"testing"
)

func TestIsComparableExpr_NilTypesInfo(t *testing.T) {
	t.Parallel()

	// Create a dummy expression
	expr := &ast.Ident{Name: "test"}

	// Call with nil typesInfo - should return false (conservative)
	result := isComparableExpr(expr, nil)

	if result {
		t.Error("Expected false when typesInfo is nil")
	}
}

func TestIsComparableExpr_MissingTypeInfo(t *testing.T) {
	t.Parallel()

	// Create a dummy expression
	expr := &ast.Ident{Name: "test"}

	// Create typesInfo but don't add this expression to it
	typesInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	// Call with expression not in typesInfo - should return false (conservative)
	result := isComparableExpr(expr, typesInfo)

	if result {
		t.Error("Expected false when expression not in typesInfo")
	}
}

func TestIsComparableExpr_ComparableType(t *testing.T) {
	t.Parallel()

	expr := &ast.Ident{Name: "test"}

	typesInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	// Add a comparable type (int)
	typesInfo.Types[expr] = types.TypeAndValue{
		Type: types.Typ[types.Int],
	}

	result := isComparableExpr(expr, typesInfo)

	if !result {
		t.Error("Expected true for comparable type (int)")
	}
}

func TestIsComparableExpr_NonComparableType(t *testing.T) {
	t.Parallel()

	expr := &ast.Ident{Name: "test"}

	typesInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	// Add a non-comparable type (slice)
	typesInfo.Types[expr] = types.TypeAndValue{
		Type: types.NewSlice(types.Typ[types.String]),
	}

	result := isComparableExpr(expr, typesInfo)

	if result {
		t.Error("Expected false for non-comparable type ([]string)")
	}
}
