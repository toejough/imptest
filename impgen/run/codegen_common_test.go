package run //nolint:testpackage

import (
	"go/token"
	"testing"

	"github.com/dave/dst"
)

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

		expr := &dst.Ident{Name: "Exported"}
		baseGen.checkIfQualifierNeeded(expr)

		if !baseGen.needsQualifier {
			t.Error("expected needsQualifier to be true")
		}
	})

	t.Run("checkIfValidForExternalUsage", func(t *testing.T) {
		t.Parallel()

		ftype := &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{
					{Type: &dst.Ident{Name: "unexported"}},
				},
			},
		}

		err := baseGen.checkIfValidForExternalUsage(ftype)
		if err == nil {
			t.Error("expected error for unexported type in external usage")
		}
	})
}

func TestBaseGeneratorMultipleTypeParams(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	typeParams := &dst.FieldList{
		List: []*dst.Field{
			{
				Names: []*dst.Ident{{Name: "T"}, {Name: "U"}},
				Type:  &dst.Ident{Name: "any"},
			},
			{
				Names: []*dst.Ident{{Name: "V"}},
				Type:  &dst.Ident{Name: "comparable"},
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

//nolint:funlen // Table-driven test with comprehensive edge cases
func TestCountFields_mutant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fields   *dst.FieldList
		expected int
	}{
		{
			name:     "nil field list",
			fields:   nil,
			expected: 0,
		},
		{
			name:     "empty field list",
			fields:   &dst.FieldList{List: []*dst.Field{}},
			expected: 0,
		},
		{
			name: "single named field",
			fields: &dst.FieldList{
				List: []*dst.Field{
					{Names: []*dst.Ident{{Name: "x"}}, Type: &dst.Ident{Name: "int"}},
				},
			},
			expected: 1,
		},
		{
			name: "multiple names in one field",
			fields: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{{Name: "x"}, {Name: "y"}, {Name: "z"}},
						Type:  &dst.Ident{Name: "int"},
					},
				},
			},
			expected: 3,
		},
		{
			name: "unnamed field",
			fields: &dst.FieldList{
				List: []*dst.Field{
					{Names: nil, Type: &dst.Ident{Name: "int"}},
				},
			},
			expected: 1,
		},
		{
			name: "mixed named and unnamed fields",
			fields: &dst.FieldList{
				List: []*dst.Field{
					{Names: []*dst.Ident{{Name: "x"}}, Type: &dst.Ident{Name: "int"}},
					{Names: nil, Type: &dst.Ident{Name: "string"}},
					{Names: []*dst.Ident{{Name: "a"}, {Name: "b"}}, Type: &dst.Ident{Name: "bool"}},
				},
			},
			expected: 4,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := countFields(testCase.fields)
			if got != testCase.expected {
				t.Errorf("countFields() = %v, want %v", got, testCase.expected)
			}
		})
	}
}

func TestFieldNameCount_mutant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    *dst.Field
		expected int
	}{
		{
			name:     "unnamed field",
			field:    &dst.Field{Names: nil, Type: &dst.Ident{Name: "int"}},
			expected: 1,
		},
		{
			name:     "empty names",
			field:    &dst.Field{Names: []*dst.Ident{}, Type: &dst.Ident{Name: "int"}},
			expected: 1,
		},
		{
			name: "single named field",
			field: &dst.Field{
				Names: []*dst.Ident{{Name: "x"}},
				Type:  &dst.Ident{Name: "int"},
			},
			expected: 1,
		},
		{
			name: "multiple names",
			field: &dst.Field{
				Names: []*dst.Ident{{Name: "x"}, {Name: "y"}, {Name: "z"}},
				Type:  &dst.Ident{Name: "int"},
			},
			expected: 3,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := fieldNameCount(testCase.field)
			if got != testCase.expected {
				t.Errorf("fieldNameCount() = %v, want %v", got, testCase.expected)
			}
		})
	}
}

func TestHasExportedIdent_Func(t *testing.T) {
	t.Parallel()

	expr1 := &dst.FuncType{Params: &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "MyType"}}}}}
	if !hasExportedIdent(expr1, func(string) bool { return false }) {
		t.Error("expected true for FuncType with exported param")
	}

	expr2 := &dst.FuncType{Results: &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "MyType"}}}}}
	if !hasExportedIdent(expr2, func(string) bool { return false }) {
		t.Error("expected true for FuncType with exported result")
	}
}

func TestHasExportedIdent_Struct(t *testing.T) {
	t.Parallel()

	expr := &dst.StructType{Fields: &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "MyType"}}}}}
	if !hasExportedIdent(expr, func(string) bool { return false }) {
		t.Error("expected true for StructType with exported field")
	}
}

func TestHasParams_mutant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ftype    *dst.FuncType
		expected bool
	}{
		{
			name:     "nil params",
			ftype:    &dst.FuncType{Params: nil},
			expected: false,
		},
		{
			name:     "empty params list",
			ftype:    &dst.FuncType{Params: &dst.FieldList{List: []*dst.Field{}}},
			expected: false,
		},
		{
			name: "single param",
			ftype: &dst.FuncType{
				Params: &dst.FieldList{
					List: []*dst.Field{
						{Names: []*dst.Ident{{Name: "x"}}, Type: &dst.Ident{Name: "int"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple params",
			ftype: &dst.FuncType{
				Params: &dst.FieldList{
					List: []*dst.Field{
						{Names: []*dst.Ident{{Name: "x"}}, Type: &dst.Ident{Name: "int"}},
						{Names: []*dst.Ident{{Name: "y"}}, Type: &dst.Ident{Name: "string"}},
					},
				},
			},
			expected: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := hasParams(testCase.ftype)
			if got != testCase.expected {
				t.Errorf("hasParams() = %v, want %v", got, testCase.expected)
			}
		})
	}
}

func TestHasResults_mutant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ftype    *dst.FuncType
		expected bool
	}{
		{
			name:     "nil results",
			ftype:    &dst.FuncType{Results: nil},
			expected: false,
		},
		{
			name:     "empty results list",
			ftype:    &dst.FuncType{Results: &dst.FieldList{List: []*dst.Field{}}},
			expected: false,
		},
		{
			name: "single result",
			ftype: &dst.FuncType{
				Results: &dst.FieldList{
					List: []*dst.Field{
						{Type: &dst.Ident{Name: "int"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple results",
			ftype: &dst.FuncType{
				Results: &dst.FieldList{
					List: []*dst.Field{
						{Type: &dst.Ident{Name: "int"}},
						{Type: &dst.Ident{Name: "error"}},
					},
				},
			},
			expected: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := hasResults(testCase.ftype)
			if got != testCase.expected {
				t.Errorf("hasResults() = %v, want %v", got, testCase.expected)
			}
		})
	}
}

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

func TestParamNamesToString_mutant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   []fieldInfo
		expected string
	}{
		{
			name:     "empty params",
			params:   []fieldInfo{},
			expected: "",
		},
		{
			name:     "nil params",
			params:   nil,
			expected: "",
		},
		{
			name: "single param",
			params: []fieldInfo{
				{Name: "x", Index: 0},
			},
			expected: "x",
		},
		{
			name: "multiple params",
			params: []fieldInfo{
				{Name: "x", Index: 0},
				{Name: "y", Index: 1},
				{Name: "z", Index: 2},
			},
			expected: "x, y, z",
		},
		{
			name: "generated param names",
			params: []fieldInfo{
				{Name: "param0", Index: 0},
				{Name: "param1", Index: 1},
			},
			expected: "param0, param1",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := paramNamesToString(testCase.params)
			if got != testCase.expected {
				t.Errorf("paramNamesToString() = %q, want %q", got, testCase.expected)
			}
		})
	}
}

// newTestBaseGenerator creates a baseGenerator for testing.
func newTestBaseGenerator() baseGenerator {
	fset := token.NewFileSet()
	typeParams := &dst.FieldList{
		List: []*dst.Field{
			{
				Names: []*dst.Ident{{Name: "T"}},
				Type:  &dst.Ident{Name: "any"},
			},
		},
	}

	return newBaseGenerator(fset, "mypkg", "MyImp", "path", "qual", typeParams, nil)
}
