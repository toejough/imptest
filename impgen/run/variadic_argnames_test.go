package run

import (
	"go/token"
	"testing"

	"github.com/dave/dst"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVariadicArgNamesPopulated is a regression test for the bug where
// ArgNames was not populated for variadic methods, causing
// ExpectCalledWithExactly() to be called with no arguments in the generated
// method wrapper code. This resulted in test timeouts due to argument mismatch.
//
// Bug context: TOE-86 Phase 2 implementation added type-safe GetArgs() which
// required generating method wrappers. The template uses {{.ArgNames}} to
// forward arguments to the underlying DependencyMethod.ExpectCalledWithExactly().
// For variadic methods, allArgs (and thus ArgNames) was left empty, causing
// ExpectCalledWithExactly() to be called with zero arguments while the actual
// method received multiple arguments, creating a channel deadlock.
func TestVariadicArgNamesPopulated(t *testing.T) {
	t.Parallel()

	// Create a mock v2DependencyGenerator with minimal setup
	fset := token.NewFileSet()
	gen := &v2DependencyGenerator{
		baseGenerator: baseGenerator{
			typeFormatter: typeFormatter{
				fset: fset,
				isTypeParam: func(string) bool {
					return false // No type parameters in this test
				},
			},
			pkgName: "test",
		},
		mockTypeName: "TestMock",
		implName:     "testImpl",
	}

	// Create a variadic function type: func(message string, ids ...int) bool
	ftype := &dst.FuncType{
		Params: &dst.FieldList{
			List: []*dst.Field{
				{
					Names: []*dst.Ident{{Name: "message"}},
					Type:  &dst.Ident{Name: "string"},
				},
				{
					Names: []*dst.Ident{{Name: "ids"}},
					Type: &dst.Ellipsis{
						Elt: &dst.Ident{Name: "int"},
					},
				},
			},
		},
		Results: &dst.FieldList{
			List: []*dst.Field{
				{Type: &dst.Ident{Name: "bool"}},
			},
		},
	}

	// Build template data
	data := gen.buildMethodTemplateData("Notify", ftype, "TestInterface")

	// CRITICAL: Verify that variadic-specific fields are populated
	require.True(t, data.HasVariadic, "HasVariadic must be true for variadic methods")
	//nolint:lll // Test assertion message
	require.NotEmpty(t, data.NonVariadicArgs, "NonVariadicArgs must be populated for variadic methods with non-variadic params")
	require.NotEmpty(t, data.VariadicArg, "VariadicArg must be populated for variadic methods")

	// Verify individual components have correct values
	assert.Equal(t, "message", data.NonVariadicArgs, "NonVariadicArgs should contain 'message'")
	assert.Equal(t, "ids", data.VariadicArg, "VariadicArg should contain 'ids'")

	// Verify that ArgNames contains all parameters for consistency
	// (even though the template uses NonVariadicArgs and VariadicArg for variadic methods)
	assert.NotEmpty(t, data.ArgNames, "ArgNames should be populated for consistency")
	assert.Contains(t, data.ArgNames, "message", "ArgNames should include 'message'")
	assert.Contains(t, data.ArgNames, "ids", "ArgNames should include 'ids'")

	// Verify the complete parameter list for the typed wrapper signature
	assert.Equal(t, "message string, ids ...int", data.TypedParams,
		"TypedParams should include full parameter list with types")
}

// TestNonVariadicArgNamesPopulated verifies that non-variadic methods
// correctly populate ArgNames (baseline test to ensure fix doesn't break non-variadic case).
func TestNonVariadicArgNamesPopulated(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	gen := &v2DependencyGenerator{
		baseGenerator: baseGenerator{
			typeFormatter: typeFormatter{
				fset: fset,
				isTypeParam: func(string) bool {
					return false // No type parameters in this test
				},
			},
			pkgName: "test",
		},
		mockTypeName: "TestMock",
		implName:     "testImpl",
	}

	// Create a non-variadic function type: func(a int, b int) int
	ftype := &dst.FuncType{
		Params: &dst.FieldList{
			List: []*dst.Field{
				{
					Names: []*dst.Ident{{Name: "a"}},
					Type:  &dst.Ident{Name: "int"},
				},
				{
					Names: []*dst.Ident{{Name: "b"}},
					Type:  &dst.Ident{Name: "int"},
				},
			},
		},
		Results: &dst.FieldList{
			List: []*dst.Field{
				{Type: &dst.Ident{Name: "int"}},
			},
		},
	}

	data := gen.buildMethodTemplateData("Add", ftype, "TestInterface")

	// Verify ArgNames is populated
	require.NotEmpty(t, data.ArgNames, "ArgNames must be populated for non-variadic methods")
	assert.Equal(t, "a, b", data.ArgNames, "ArgNames should contain comma-separated parameter names")

	// Verify variadic flags are false
	assert.False(t, data.HasVariadic, "HasVariadic should be false")
	assert.Empty(t, data.NonVariadicArgs, "NonVariadicArgs should be empty for non-variadic methods")
	assert.Empty(t, data.VariadicArg, "VariadicArg should be empty for non-variadic methods")
}
