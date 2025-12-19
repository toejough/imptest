package run_test

import (
	"strings"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

func TestRun_EmbeddedInterface_ExternalPackage(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Interface with embedded interface from external package
	sourceCode := `
package mypkg

import "fmt"

type MyInterface interface {
	fmt.Stringer  // External embedded interface
	OwnMethod()
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// Add fmt package with Stringer interface
	fmtSource := `
package fmt

type Stringer interface {
	String() string
}
`
	mockPkgLoader.AddPackageFromSource("fmt", fmtSource)

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	// External embedded interfaces are now supported, so this should succeed
	if err != nil {
		t.Fatalf("Expected success for external embedded interface, got error: %v", err)
	}

	// Verify the generated code includes both String() from fmt.Stringer and OwnMethod()
	content := mockFS.files["MyImp.go"]

	contentStr := string(content)
	if !strings.Contains(contentStr, "String()") {
		t.Error("Generated code should include String() method from embedded fmt.Stringer")
	}

	if !strings.Contains(contentStr, "OwnMethod()") {
		t.Error("Generated code should include OwnMethod()")
	}
}

func TestRun_EmbeddedInterface_Unsupported(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	// In Go, only Ident and SelectorExpr are valid for embedded interfaces.
	// We can't easily trigger the 'default' case via normal parsing of valid Go code,
	// but we can test that it returns an error if we somehow get there.
	// For now, let's just test a failing external package load to increase coverage.

	sourceCode := `
package mypkg
import nonexistent "some/path"
type MyInterface interface {
	nonexistent.Interface
}
`
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Fatal("Expected error for nonexistent external embedded interface package, got nil")
	}

	if !strings.Contains(err.Error(), "failed to load external embedded interface package") {
		t.Errorf("Expected load error, got: %v", err)
	}
}
