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

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)

	// Should get an error because external embedded interfaces aren't supported yet
	if err == nil {
		t.Fatal("Expected error for external embedded interface, got nil")
	}

	if !strings.Contains(err.Error(), "embedded interface from external package is not yet supported") {
		t.Errorf("Expected error about unsupported external embedded interface, got: %v", err)
	}
}
