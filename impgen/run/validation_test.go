package run_test

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

func TestValidateExportedTypes(t *testing.T) {
	t.Parallel()

	isTypeParam := func(name string) bool {
		return name == "T"
	}

	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"exported ident", "string", false},
		{"exported custom ident", "MyType", false},
		{"unexported ident", "secret", true},
		{"type parameter", "T", false},
		{"pointer to exported", "*string", false},
		{"pointer to unexported", "*secret", true},
		{"slice of exported", "[]string", false},
		{"slice of unexported", "[]secret", true},
		{"map of exported", "map[string]int", false},
		{"map of unexported key", "map[secret]int", true},
		{"map of unexported value", "map[string]secret", true},
		{"chan of exported", "chan string", false},
		{"chan of unexported", "chan secret", true},
		{"func with unexported param", "func(secret)", true},
		{"func with unexported result", "func() secret", true},
		{"exported selector", "fmt.Stringer", false},
		{"unexported selector", "fmt.secret", true},
		{"struct with unexported field", "struct{ s secret }", true},
		{"generic with unexported base", "secret[T]", true},
		{"generic with unexported param", "MyType[secret]", true},
		{"generic list with unexported param", "MyType[T, secret]", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			expr, err := parser.ParseExpr(testCase.expr)
			if err != nil {
				t.Fatalf("failed to parse expr: %v", err)
			}

			err = run.ValidateExportedTypes(expr, isTypeParam)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ValidateExportedTypes() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestIsExportedIdent(t *testing.T) {
	t.Parallel()

	isTypeParam := func(name string) bool {
		return name == "T"
	}

	tests := []struct {
		name  string
		ident string
		want  bool
	}{
		{"exported", "Exported", true},
		{"unexported", "unexported", false},
		{"builtin", "int", true},
		{"type param", "T", true},
		{"empty", "", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ident := &ast.Ident{Name: testCase.ident}
			if got := run.IsExportedIdent(ident, isTypeParam); got != testCase.want {
				t.Errorf("IsExportedIdent() = %v, want %v", got, testCase.want)
			}
		})
	}
}
