//nolint:testpackage // Needs access to unexported types for whitebox testing
package run

import (
	"go/token"
	"strings"
	"testing"

	"github.com/dave/dst"
)

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
	if data.ArgNames == "" {
		t.Fatal("ArgNames must be populated for non-variadic methods")
	}

	expectedArgNames := "a, b"
	if data.ArgNames != expectedArgNames {
		t.Errorf("ArgNames = %q, want %q", data.ArgNames, expectedArgNames)
	}

	// Verify variadic flags are false
	if data.HasVariadic {
		t.Error("HasVariadic should be false")
	}

	if data.NonVariadicArgs != "" {
		t.Errorf("NonVariadicArgs should be empty for non-variadic methods, got %q", data.NonVariadicArgs)
	}

	if data.VariadicArg != "" {
		t.Errorf("VariadicArg should be empty for non-variadic methods, got %q", data.VariadicArg)
	}
}

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
//
//nolint:funlen // Comprehensive regression test for TOE-86 - documents all aspects of the fix
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
	if !data.HasVariadic {
		t.Error("HasVariadic must be true for variadic methods")
	}

	if data.NonVariadicArgs == "" {
		t.Error("NonVariadicArgs must be populated for variadic methods with non-variadic params")
	}

	if data.VariadicArg == "" {
		t.Error("VariadicArg must be populated for variadic methods")
	}

	// Verify individual components have correct values
	if data.NonVariadicArgs != "message" {
		t.Errorf("NonVariadicArgs = %q, want %q", data.NonVariadicArgs, "message")
	}

	if data.VariadicArg != "ids" {
		t.Errorf("VariadicArg = %q, want %q", data.VariadicArg, "ids")
	}

	// Verify that ArgNames contains all parameters for consistency
	// (even though the template uses NonVariadicArgs and VariadicArg for variadic methods)
	if data.ArgNames == "" {
		t.Error("ArgNames should be populated for consistency")
	}

	if !strings.Contains(data.ArgNames, "message") {
		t.Errorf("ArgNames %q should include 'message'", data.ArgNames)
	}

	if !strings.Contains(data.ArgNames, "ids") {
		t.Errorf("ArgNames %q should include 'ids'", data.ArgNames)
	}

	// Verify the complete parameter list for the typed wrapper signature
	expectedParams := "message string, ids ...int"
	if data.TypedParams != expectedParams {
		t.Errorf("TypedParams = %q, want %q", data.TypedParams, expectedParams)
	}
}
