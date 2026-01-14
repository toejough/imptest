//nolint:testpackage // Tests internal functions
package generate

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/dave/dst"

	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

func TestDependencyCode_Error(t *testing.T) {
	t.Parallel()

	// Create an interface with an unsupported embedded type (e.g., *dst.BasicLit)
	// This will cause interfaceExpandEmbedded to fail with errUnsupportedEmbeddedType
	iface := &dst.InterfaceType{
		Methods: &dst.FieldList{
			List: []*dst.Field{
				{
					// Embedded type with no names - triggers interfaceExpandEmbedded
					Type: &dst.BasicLit{Value: "unsupported"}, // Not Ident or SelectorExpr
				},
			},
		},
	}

	info := GeneratorInfo{
		ImpName:       "MockTest",
		InterfaceName: "TestIface",
		PkgName:       "testpkg",
	}

	ifaceWithDetails := detect.IfaceWithDetails{
		Iface: iface,
	}

	// Pass stub pkgLoader to satisfy nil checks, even though it won't be used
	// because the function returns an error before accessing it.
	_, err := DependencyCode(nil, info, token.NewFileSet(), ".", &mockPkgLoader{}, ifaceWithDetails)
	if err == nil {
		t.Error("DependencyCode() expected error for unsupported embedded type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported embedded type") {
		t.Errorf(
			"DependencyCode() error = %v, want error containing 'unsupported embedded type'",
			err,
		)
	}
}

// mockPkgLoader is a test mock for detect.PackageLoader.
type mockPkgLoader struct {
	files []*dst.File
	fset  *token.FileSet
	info  *types.Info
	err   error
}

func (m *mockPkgLoader) Load(_ string) ([]*dst.File, *token.FileSet, *types.Info, error) {
	return m.files, m.fset, m.info, m.err
}
