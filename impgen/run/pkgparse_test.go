//nolint:testpackage // Need same package to test unexported isComparableExpr
package run

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

var errUnexpectedLoad = errors.New("unexpected load")

// mockPackageLoader implements PackageLoader for testing.
type mockPackageLoader struct {
	loadFunc func(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error)
}

func (m *mockPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
	return m.loadFunc(importPath)
}

func TestGetInterfacePackagePath_MismatchedDirectoryName(t *testing.T) {
	t.Parallel()

	// This test simulates the case where:
	// Directory on disk: "UAT/01-basic-interface-mocking"
	// Package name in code: "package basic"
	// The user provides: "basic.Returner"
	// The current file imports: "github.com/toejough/imptest/UAT/01-basic-interface-mocking" (no alias)

	targetImportPath := "github.com/toejough/imptest/UAT/01-basic-interface-mocking"
	targetPackageName := "basic"

	mockLoader := &mockPackageLoader{
		loadFunc: func(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
			fset := token.NewFileSet()

			if importPath == "." {
				// Return a file that imports the mismatched package WITHOUT an alias.
				file := &ast.File{
					Name: &ast.Ident{Name: "main"},
					Imports: []*ast.ImportSpec{
						{
							Path: &ast.BasicLit{
								Kind:  token.STRING,
								Value: fmt.Sprintf("%q", targetImportPath),
							},
						},
					},
				}

				return []*ast.File{file}, fset, nil, nil
			}

			if importPath == targetImportPath {
				// Return a file that defines the package with its internal name.
				file := &ast.File{
					Name: &ast.Ident{Name: targetPackageName},
				}

				return []*ast.File{file}, fset, nil, nil
			}

			return nil, nil, nil, fmt.Errorf("%w: %s", errUnexpectedLoad, importPath)
		},
	}

	qualifiedName := targetPackageName + ".Returner"

	// ACTION: Attempt to resolve "basic.Returner"
	path, err := getInterfacePackagePath(qualifiedName, mockLoader)
	// ASSERTION: This is expected to fail or return the wrong path
	// until we improve the package loader logic.
	if err != nil {
		t.Fatalf("Failed to resolve %q: %v", qualifiedName, err)
	}

	if path != targetImportPath {
		t.Errorf("Expected path %q, but got %q", targetImportPath, path)
	}
}

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

func TestExtractPackageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"pkg.Name", "pkg"},
		{"Name", ""},
		{"a.b.c", "a"},
	}

	for _, tt := range tests {
		got := extractPackageName(tt.input)
		if got != tt.want {
			t.Errorf("extractPackageName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
