//nolint:testpackage,thelper,varnamelen,wsl_v5,funlen,gocognit,gocyclo,cyclop,maintidx // Tests internals
package generate

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	"github.com/dave/dst"

	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

func TestExtractFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fields   *dst.FieldList
		prefix   string
		wantLen  int
		wantName string // First field name to check
	}{
		{
			name:    "nil fields returns nil",
			fields:  nil,
			prefix:  "arg",
			wantLen: 0,
		},
		{
			name: "named field",
			fields: &dst.FieldList{
				List: []*dst.Field{
					{Names: []*dst.Ident{{Name: "foo"}}, Type: &dst.Ident{Name: "int"}},
				},
			},
			prefix:   "arg",
			wantLen:  1,
			wantName: "foo",
		},
		{
			name: "unnamed field gets generated name",
			fields: &dst.FieldList{
				List: []*dst.Field{
					{Type: &dst.Ident{Name: "int"}},
				},
			},
			prefix:   "arg",
			wantLen:  1,
			wantName: "arg0",
		},
		{
			name: "multiple names in one field",
			fields: &dst.FieldList{
				List: []*dst.Field{
					{Names: []*dst.Ident{{Name: "a"}, {Name: "b"}}, Type: &dst.Ident{Name: "int"}},
				},
			},
			prefix:   "arg",
			wantLen:  2,
			wantName: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := extractFields(tt.fields, tt.prefix)
			if len(result) != tt.wantLen {
				t.Errorf("extractFields() len = %v, want %v", len(result), tt.wantLen)
				return
			}

			if tt.wantLen > 0 && result[0].Name != tt.wantName {
				t.Errorf("extractFields()[0].Name = %v, want %v", result[0].Name, tt.wantName)
			}
		})
	}
}

// TestFieldParamNames_BlankIdentifier tests that blank identifiers get synthetic names.
func TestFieldParamNames_BlankIdentifier(t *testing.T) {
	t.Parallel()

	// Test with blank identifier
	field := &dst.Field{
		Names: []*dst.Ident{{Name: "_"}},
		Type:  &dst.Ident{Name: "string"},
	}
	names := fieldParamNames(field, 0)
	if len(names) != 1 {
		t.Fatalf("expected 1 name, got %d", len(names))
	}
	if names[0] != "arg1" {
		t.Errorf("expected arg1 for blank identifier, got %s", names[0])
	}
}

func TestGetPackageInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		targetName     string
		currentPkgName string
		loader         detect.PackageLoader
		wantPkgPath    string
		wantPkgName    string
		wantErr        error
	}{
		{
			name:           "no dot in target name",
			targetName:     "SomeType",
			currentPkgName: "mypkg",
			loader:         nil,
			wantPkgPath:    "",
			wantPkgName:    "",
			wantErr:        nil,
		},
		{
			name:           "empty package name before dot",
			targetName:     ".Method",
			currentPkgName: "mypkg",
			loader:         nil,
			wantPkgPath:    "",
			wantPkgName:    "",
			wantErr:        nil,
		},
		{
			name:           "current package name matches",
			targetName:     "mypkg.SomeType",
			currentPkgName: "mypkg",
			loader:         nil,
			wantPkgPath:    "",
			wantPkgName:    "",
			wantErr:        nil,
		},
		{
			name:           "load dot fails, fallback succeeds",
			targetName:     "otherpkg.SomeType",
			currentPkgName: "mypkg",
			loader: &mockPackageLoader{
				loadFunc: func(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error) {
					if importPath == "." {
						return nil, nil, nil, errors.New("load failed")
					}
					// Fallback to loading the package directly
					if importPath == "otherpkg" {
						return []*dst.File{{Name: &dst.Ident{Name: "otherpkg"}}}, nil, nil, nil
					}

					return nil, nil, nil, errors.New("not found")
				},
			},
			wantPkgPath: "otherpkg",
			wantPkgName: "otherpkg",
			wantErr:     nil,
		},
		{
			name:           "load dot fails, fallback also fails",
			targetName:     "nonexistent.SomeType",
			currentPkgName: "mypkg",
			loader: &mockPackageLoader{
				loadFunc: func(_ string) ([]*dst.File, *token.FileSet, *types.Info, error) {
					return nil, nil, nil, errors.New("load failed")
				},
			},
			wantPkgPath: "",
			wantPkgName: "",
			wantErr:     nil,
		},
		{
			name:           "type method reference returns error",
			targetName:     "Counter.Inc",
			currentPkgName: "mypkg",
			loader: &mockPackageLoader{
				loadFunc: func(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error) {
					// Return files only for "." (current package), error for "Counter"
					// This simulates a scenario where Counter is a type, not a package
					fset := token.NewFileSet()

					if importPath == "." {
						return []*dst.File{
							{
								Name:    &dst.Ident{Name: "mypkg"},
								Imports: nil, // No imports for Counter
							},
						}, fset, nil, nil
					}

					// For "Counter" or any other package, return an error
					// This triggers the "not found as package" path
					return nil, nil, nil, errors.New("package not found")
				},
			},
			wantPkgPath: "",
			wantPkgName: "",
			wantErr:     ErrNotPackageReference,
		},
	}

	for _, tt := range tests { //nolint:varnamelen // tt is idiomatic in Go tests
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkgPath, pkgName, err := GetPackageInfo(tt.targetName, tt.loader, tt.currentPkgName)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetPackageInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if pkgPath != tt.wantPkgPath {
				t.Errorf("GetPackageInfo() pkgPath = %v, want %v", pkgPath, tt.wantPkgPath)
			}

			if pkgName != tt.wantPkgName {
				t.Errorf("GetPackageInfo() pkgName = %v, want %v", pkgName, tt.wantPkgName)
			}
		})
	}
}

func TestQualifyExternalTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expr      dst.Expr
		qualifier string
		check     func(*testing.T, dst.Expr)
	}{
		{
			name:      "nil expression",
			expr:      nil,
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				if result != nil {
					t.Errorf("expected nil, got %T", result)
				}
			},
		},
		{
			name:      "exported ident gets qualified",
			expr:      &dst.Ident{Name: testTypeSomeType},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				sel, ok := result.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr, got %T", result)
					return
				}
				x, ok := sel.X.(*dst.Ident)
				if !ok || x.Name != "pkg" {
					t.Errorf("expected X to be pkg, got %v", sel.X)
				}
				if sel.Sel.Name != testTypeSomeType {
					t.Errorf("expected Sel to be SomeType, got %s", sel.Sel.Name)
				}
			},
		},
		{
			name:      "unexported ident unchanged",
			expr:      &dst.Ident{Name: "someType"},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				ident, ok := result.(*dst.Ident)
				if !ok {
					t.Errorf("expected Ident, got %T", result)
					return
				}
				if ident.Name != "someType" {
					t.Errorf("expected someType, got %s", ident.Name)
				}
			},
		},
		{
			name:      "star expr qualifies inner type",
			expr:      &dst.StarExpr{X: &dst.Ident{Name: testTypeSomeType}},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				star, ok := result.(*dst.StarExpr)
				if !ok {
					t.Errorf("expected StarExpr, got %T", result)
					return
				}
				sel, ok := star.X.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr inside StarExpr, got %T", star.X)
				}
				if sel.Sel.Name != testTypeSomeType {
					t.Errorf("expected SomeType, got %s", sel.Sel.Name)
				}
			},
		},
		{
			name:      "array type qualifies element",
			expr:      &dst.ArrayType{Elt: &dst.Ident{Name: testTypeSomeType}},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				arr, ok := result.(*dst.ArrayType)
				if !ok {
					t.Errorf("expected ArrayType, got %T", result)
					return
				}
				sel, ok := arr.Elt.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr in array element, got %T", arr.Elt)
				}
				if sel.Sel.Name != testTypeSomeType {
					t.Errorf("expected SomeType, got %s", sel.Sel.Name)
				}
			},
		},
		{
			name: "map type qualifies key and value",
			expr: &dst.MapType{
				Key:   &dst.Ident{Name: "KeyType"},
				Value: &dst.Ident{Name: "ValueType"},
			},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				m, ok := result.(*dst.MapType)
				if !ok {
					t.Errorf("expected MapType, got %T", result)
					return
				}
				keySel, ok := m.Key.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr for key, got %T", m.Key)
				} else if keySel.Sel.Name != "KeyType" {
					t.Errorf("expected KeyType, got %s", keySel.Sel.Name)
				}
				valSel, ok := m.Value.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr for value, got %T", m.Value)
				} else if valSel.Sel.Name != "ValueType" {
					t.Errorf("expected ValueType, got %s", valSel.Sel.Name)
				}
			},
		},
		{
			name: "chan type qualifies value",
			expr: &dst.ChanType{
				Dir:   dst.SEND | dst.RECV,
				Value: &dst.Ident{Name: testTypeSomeType},
			},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				ch, ok := result.(*dst.ChanType)
				if !ok {
					t.Errorf("expected ChanType, got %T", result)
					return
				}
				sel, ok := ch.Value.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr for channel value, got %T", ch.Value)
				}
				if sel.Sel.Name != testTypeSomeType {
					t.Errorf("expected SomeType, got %s", sel.Sel.Name)
				}
			},
		},
		{
			name: "selector expr unchanged",
			expr: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "other"},
				Sel: &dst.Ident{Name: "Type"},
			},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				sel, ok := result.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr, got %T", result)
					return
				}
				x, ok := sel.X.(*dst.Ident)
				if !ok || x.Name != "other" {
					t.Errorf("expected X to remain 'other', got %v", sel.X)
				}
			},
		},
		{
			name: "func type qualifies params and results",
			expr: &dst.FuncType{
				Params: &dst.FieldList{
					List: []*dst.Field{{Type: &dst.Ident{Name: "ParamType"}}},
				},
				Results: &dst.FieldList{
					List: []*dst.Field{{Type: &dst.Ident{Name: "ResultType"}}},
				},
			},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				fn, ok := result.(*dst.FuncType)
				if !ok {
					t.Errorf("expected FuncType, got %T", result)
					return
				}
				if len(fn.Params.List) == 0 {
					t.Error("expected params")
					return
				}
				paramSel, ok := fn.Params.List[0].Type.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr for param, got %T", fn.Params.List[0].Type)
				} else if paramSel.Sel.Name != "ParamType" {
					t.Errorf("expected ParamType, got %s", paramSel.Sel.Name)
				}
				if fn.Results == nil || len(fn.Results.List) == 0 {
					t.Error("expected results")
					return
				}
				resultSel, ok := fn.Results.List[0].Type.(*dst.SelectorExpr)
				if !ok {
					t.Errorf("expected SelectorExpr for result, got %T", fn.Results.List[0].Type)
				} else if resultSel.Sel.Name != "ResultType" {
					t.Errorf("expected ResultType, got %s", resultSel.Sel.Name)
				}
			},
		},
		{
			name:      "basic lit unchanged",
			expr:      &dst.BasicLit{Value: "123"},
			qualifier: "pkg",
			check: func(t *testing.T, result dst.Expr) {
				lit, ok := result.(*dst.BasicLit)
				if !ok {
					t.Errorf("expected BasicLit, got %T", result)
					return
				}
				if lit.Value != "123" {
					t.Errorf("expected 123, got %s", lit.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := qualifyExternalTypes(tt.expr, tt.qualifier)
			tt.check(t, result)
		})
	}
}

// TestWalkIndexType_DefaultCase tests the defensive default case in walkIndexType.
func TestWalkIndexType_DefaultCase(t *testing.T) {
	t.Parallel()

	// Create a walker with simple identity functions
	walker := &typeExprWalker[dst.Expr]{
		visitIdent:    func(e *dst.Ident) dst.Expr { return e },
		visitSelector: func(e *dst.SelectorExpr) dst.Expr { return e },
		combine:       func(a, _ dst.Expr) dst.Expr { return a },
		zero:          nil,
	}

	// Call walkIndexType with an expression that's neither IndexExpr nor IndexListExpr
	// This tests the "should never happen" default case
	result := walker.walkIndexType(&dst.Ident{Name: "unexpected"})
	if result != nil {
		t.Errorf("expected nil (zero value) for unexpected type, got %v", result)
	}
}

// unexported constants.
const (
	testTypeSomeType = "SomeType"
)

type mockPackageLoader struct {
	loadFunc func(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error)
}

func (m *mockPackageLoader) Load(
	importPath string,
) ([]*dst.File, *token.FileSet, *types.Info, error) {
	if m.loadFunc != nil {
		return m.loadFunc(importPath)
	}

	return nil, nil, nil, errors.New("not implemented")
}
