//nolint:testpackage,varnamelen // Tests internal functions
package generate

import (
	"errors"
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

	_, err := DependencyCode(nil, info, token.NewFileSet(), ".", nil, ifaceWithDetails)
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

func TestResolveTestPackageImport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		loader        *mockPkgLoader
		pkgName       string
		wantPkgPath   string
		wantQualifier string
	}{
		{
			name: "load error returns empty",
			loader: &mockPkgLoader{
				err: errors.New("load failed"),
			},
			pkgName:       "mypkg_test",
			wantPkgPath:   "",
			wantQualifier: "",
		},
		{
			name: "empty files returns empty",
			loader: &mockPkgLoader{
				files: []*dst.File{},
				fset:  token.NewFileSet(),
			},
			pkgName:       "mypkg_test",
			wantPkgPath:   "",
			wantQualifier: "",
		},
		{
			name: "no go.mod returns empty path",
			loader: &mockPkgLoader{
				files: []*dst.File{{Name: &dst.Ident{Name: "mypkg"}}},
				fset:  token.NewFileSet(),
			},
			pkgName:       "mypkg_test",
			wantPkgPath:   "",
			wantQualifier: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkgPath, qualifier := resolveTestPackageImport(tt.loader, tt.pkgName)
			if pkgPath != tt.wantPkgPath {
				t.Errorf(
					"resolveTestPackageImport() pkgPath = %v, want %v",
					pkgPath,
					tt.wantPkgPath,
				)
			}

			if qualifier != tt.wantQualifier {
				t.Errorf(
					"resolveTestPackageImport() qualifier = %v, want %v",
					qualifier,
					tt.wantQualifier,
				)
			}
		})
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
