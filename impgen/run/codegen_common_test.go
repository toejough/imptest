//nolint:testpackage
package run

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

func TestIsBuiltinType(t *testing.T) {
	t.Parallel()

	builtins := []string{
		"bool", "byte", "complex64", "complex128", "error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64", "rune", "string", "uint",
		"uint8", "uint16", "uint32", "uint64", "uintptr", "any",
	}

	for _, builtin := range builtins {
		t.Run(builtin, func(t *testing.T) {
			t.Parallel()

			if !isBuiltinType(builtin) {
				t.Errorf("isBuiltinType(%q) = false, want true", builtin)
			}
		})
	}

	t.Run("non-builtin", func(t *testing.T) {
		t.Parallel()

		if isBuiltinType("MyCustomType") {
			t.Error("isBuiltinType(\"MyCustomType\") = true, want false")
		}
	})
}

func TestHasExportedIdent_Ident(t *testing.T) {
	t.Parallel()

	isTypeParam := func(name string) bool { return name == "T" }

	if !hasExportedIdent(&ast.Ident{Name: "MyType"}, isTypeParam) {
		t.Error("expected true for exported Ident")
	}

	if hasExportedIdent(&ast.Ident{Name: "myType"}, isTypeParam) {
		t.Error("expected false for unexported Ident")
	}

	if hasExportedIdent(&ast.Ident{Name: "int"}, isTypeParam) {
		t.Error("expected false for builtin Ident")
	}

	if hasExportedIdent(&ast.Ident{Name: "T"}, isTypeParam) {
		t.Error("expected false for type parameter Ident")
	}
}

func TestHasExportedIdent_Selector(t *testing.T) {
	t.Parallel()

	expr := &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Type"}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for SelectorExpr")
	}
}

func TestHasExportedIdent_Star(t *testing.T) {
	t.Parallel()

	expr := &ast.StarExpr{X: &ast.Ident{Name: "MyType"}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for StarExpr with exported base")
	}
}

func TestHasExportedIdent_Array(t *testing.T) {
	t.Parallel()

	expr := &ast.ArrayType{Elt: &ast.Ident{Name: "MyType"}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for ArrayType with exported element")
	}
}

func TestHasExportedIdent_Map(t *testing.T) {
	t.Parallel()

	expr1 := &ast.MapType{Key: &ast.Ident{Name: "MyType"}, Value: &ast.Ident{Name: "int"}}
	if !hasExportedIdent(expr1, func(string) bool { return false }) {
		t.Error("expected true for MapType with exported key")
	}

	expr2 := &ast.MapType{Key: &ast.Ident{Name: "int"}, Value: &ast.Ident{Name: "MyType"}}
	if !hasExportedIdent(expr2, func(string) bool { return false }) {
		t.Error("expected true for MapType with exported value")
	}
}

func TestHasExportedIdent_Chan(t *testing.T) {
	t.Parallel()

	expr := &ast.ChanType{Value: &ast.Ident{Name: "MyType"}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for ChanType with exported value")
	}
}

func TestHasExportedIdent_Func(t *testing.T) {
	t.Parallel()

	expr1 := &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "MyType"}}}}}
	if !hasExportedIdent(expr1, func(string) bool { return false }) {
		t.Error("expected true for FuncType with exported param")
	}

	expr2 := &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "MyType"}}}}}
	if !hasExportedIdent(expr2, func(string) bool { return false }) {
		t.Error("expected true for FuncType with exported result")
	}
}

func TestHasExportedIdent_Struct(t *testing.T) {
	t.Parallel()

	expr := &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "MyType"}}}}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for StructType with exported field")
	}
}

func TestHasExportedIdent_Index(t *testing.T) {
	t.Parallel()

	expr := &ast.IndexExpr{X: &ast.Ident{Name: "List"}, Index: &ast.Ident{Name: "MyType"}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for IndexExpr with exported index")
	}
}

func TestHasExportedIdent_IndexList(t *testing.T) {
	t.Parallel()

	expr := &ast.IndexListExpr{X: &ast.Ident{Name: "Map"}, Indices: []ast.Expr{&ast.Ident{Name: "MyType"}}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for IndexListExpr with exported index")
	}
}

func TestHasExportedIdent_Default(t *testing.T) {
	t.Parallel()

	if hasExportedIdent(&ast.BasicLit{}, func(string) bool { return false }) {
		t.Error("expected false for unsupported expression type")
	}
}

func TestGetPackageInfo_Simple(t *testing.T) {
	t.Parallel()

	t.Run("no dot", func(t *testing.T) {
		t.Parallel()

		path, name, err := GetPackageInfo("MyInterface", nil, "current")
		if err != nil || path != "" || name != "" {
			t.Errorf("expected empty results, got path=%q, name=%q, err=%v", path, name, err)
		}
	})

	t.Run("empty package", func(t *testing.T) {
		t.Parallel()

		path, name, err := GetPackageInfo(".MyInterface", nil, "current")
		if err != nil || path != "" || name != "" {
			t.Errorf("expected empty results, got path=%q, name=%q, err=%v", path, name, err)
		}
	})

	t.Run("current package", func(t *testing.T) {
		t.Parallel()

		path, name, err := GetPackageInfo("current.MyInterface", nil, "current")
		if err != nil || path != "" || name != "" {
			t.Errorf("expected empty results, got path=%q, name=%q, err=%v", path, name, err)
		}
	})
}

func TestGetPackageInfo_LoadLocalError(t *testing.T) {
	t.Parallel()

	const otherPkg = "other"

	t.Run("success resolving pkg directly", func(t *testing.T) {
		t.Parallel()

		mockLoader := &mockPackageLoader{
			loadFunc: func(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
				if importPath == "." {
					return nil, nil, nil, errors.New("load error")
				}

				if importPath == otherPkg {
					return []*ast.File{{Name: &ast.Ident{Name: otherPkg}}}, nil, nil, nil
				}

				return nil, nil, nil, errors.New("not found")
			},
		}

		path, name, err := GetPackageInfo(otherPkg+".MyInterface", mockLoader, "current")
		if err != nil || path != otherPkg || name != otherPkg {
			t.Errorf("expected path=other, name=other, got path=%q, name=%q, err=%v", path, name, err)
		}
	})

	t.Run("fail resolving pkg directly", func(t *testing.T) {
		t.Parallel()

		mockLoader := &mockPackageLoader{
			loadFunc: func(_ string) ([]*ast.File, *token.FileSet, *types.Info, error) {
				return nil, nil, nil, errors.New("load error")
			},
		}

		path, name, err := GetPackageInfo(otherPkg+".MyInterface", mockLoader, "current")
		if err != nil || path != "" || name != "" {
			t.Errorf("expected empty results, got path=%q, name=%q, err=%v", path, name, err)
		}
	})
}

func TestGetPackageInfo_FindImportPathError(t *testing.T) {
	t.Parallel()

	mockLoader := &mockPackageLoader{
		loadFunc: func(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
			if importPath == "." {
				return []*ast.File{{Name: &ast.Ident{Name: "current"}}}, nil, nil, nil
			}

			return nil, nil, nil, errors.New("not found")
		},
	}

	path, name, err := GetPackageInfo("other.MyInterface", mockLoader, "current")
	if err != nil || path != "" || name != "" {
		t.Errorf("expected empty results, got path=%q, name=%q, err=%v", path, name, err)
	}
}

func TestNormalizeVariadicType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"variadic type", "...int", "[]int"},
		{"variadic custom type", "...MyType", "[]MyType"},
		{"non-variadic type", "int", "int"},
		{"slice type", "[]string", "[]string"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeVariadicType(testCase.input)
			if got != testCase.expected {
				t.Errorf("normalizeVariadicType(%q) = %q, want %q", testCase.input, got, testCase.expected)
			}
		})
	}
}

// newTestBaseGenerator creates a baseGenerator for testing.
func newTestBaseGenerator() baseGenerator {
	fset := token.NewFileSet()
	typeParams := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "T"}},
				Type:  &ast.Ident{Name: "any"},
			},
		},
	}

	return newBaseGenerator(fset, "mypkg", "MyImp", "path", "qual", typeParams, nil)
}

func TestBaseGenerator(t *testing.T) {
	t.Parallel()

	baseGen := newTestBaseGenerator()

	t.Run("formatTypeParams", func(t *testing.T) {
		t.Parallel()

		if got := baseGen.formatTypeParamsDecl(); got != "[T any]" {
			t.Errorf("expected [T any], got %q", got)
		}

		if got := baseGen.formatTypeParamsUse(); got != "[T]" {
			t.Errorf("expected [T], got %q", got)
		}
	})

	t.Run("isTypeParameter", func(t *testing.T) {
		t.Parallel()

		if !baseGen.isTypeParameter("T") {
			t.Error("expected T to be a type parameter")
		}

		if baseGen.isTypeParameter("U") {
			t.Error("expected U not to be a type parameter")
		}
	})

	t.Run("checkIfQualifierNeeded", func(t *testing.T) {
		t.Parallel()

		expr := &ast.Ident{Name: "Exported"}
		baseGen.checkIfQualifierNeeded(expr)

		if !baseGen.needsQualifier {
			t.Error("expected needsQualifier to be true")
		}
	})

	t.Run("checkIfValidForExternalUsage", func(t *testing.T) {
		t.Parallel()

		ftype := &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "unexported"}},
				},
			},
		}

		err := baseGen.checkIfValidForExternalUsage(ftype)
		if err == nil {
			t.Error("expected error for unexported type in external usage")
		}
	})
}

func TestBaseGeneratorNilTypeParams(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	baseGenNil := newBaseGenerator(fset, "pkg", "Imp", "path", "qual", nil, nil)

	if got := baseGenNil.formatTypeParamsDecl(); got != "" {
		t.Errorf("expected empty string for nil typeParams, got %q", got)
	}

	if got := baseGenNil.formatTypeParamsUse(); got != "" {
		t.Errorf("expected empty string for nil typeParams, got %q", got)
	}

	if baseGenNil.isTypeParameter("T") {
		t.Error("expected nil typeParams to return false for any name")
	}
}

func TestBaseGeneratorMultipleTypeParams(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	typeParams := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "T"}, {Name: "U"}},
				Type:  &ast.Ident{Name: "any"},
			},
			{
				Names: []*ast.Ident{{Name: "V"}},
				Type:  &ast.Ident{Name: "comparable"},
			},
		},
	}
	baseGen := newBaseGenerator(fset, "pkg", "Imp", "path", "qual", typeParams, nil)

	// Test all type parameters are recognized
	if !baseGen.isTypeParameter("T") {
		t.Error("expected T to be a type parameter")
	}

	if !baseGen.isTypeParameter("U") {
		t.Error("expected U to be a type parameter")
	}

	if !baseGen.isTypeParameter("V") {
		t.Error("expected V to be a type parameter")
	}

	// Test non-type parameter
	if baseGen.isTypeParameter("W") {
		t.Error("expected W not to be a type parameter")
	}
}

func TestIsComparableExpr(t *testing.T) {
	t.Parallel()

	t.Run("nil typesInfo", func(t *testing.T) {
		t.Parallel()

		expr := &ast.Ident{Name: "int"}
		if isComparableExpr(expr, nil) {
			t.Error("expected false for nil typesInfo")
		}
	})

	t.Run("type not in map", func(t *testing.T) {
		t.Parallel()

		typesInfo := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}
		expr := &ast.Ident{Name: "unknown"}

		if isComparableExpr(expr, typesInfo) {
			t.Error("expected false for type not in map")
		}
	})

	t.Run("comparable type", func(t *testing.T) {
		t.Parallel()

		typesInfo := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}
		expr := &ast.Ident{Name: "int"}
		typesInfo.Types[expr] = types.TypeAndValue{Type: types.Typ[types.Int]}

		if !isComparableExpr(expr, typesInfo) {
			t.Error("expected true for comparable type (int)")
		}
	})

	t.Run("non-comparable type", func(t *testing.T) {
		t.Parallel()

		typesInfo := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}
		expr := &ast.Ident{Name: "slice"}
		// Create a slice type which is not comparable
		sliceType := types.NewSlice(types.Typ[types.Int])
		typesInfo.Types[expr] = types.TypeAndValue{Type: sliceType}

		if isComparableExpr(expr, typesInfo) {
			t.Error("expected false for non-comparable type (slice)")
		}
	})
}
