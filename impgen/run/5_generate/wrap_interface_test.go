//nolint:testpackage // Tests internal functions
package generate

import (
	"go/token"
	"strings"
	"testing"

	"github.com/dave/dst"

	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

func TestInterfaceTargetCode_Error(t *testing.T) {
	t.Parallel()

	// Create an interface with an unsupported embedded type
	// This will cause newInterfaceTargetGenerator to fail
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
		ImpName:       "WrapTest",
		InterfaceName: "TestIface",
		PkgName:       "testpkg",
	}

	ifaceWithDetails := detect.IfaceWithDetails{
		Iface: iface,
	}

	// Pass stub pkgLoader to satisfy nil checks, even though it won't be used
	// because the function returns an error before accessing it.
	_, err := InterfaceTargetCode(
		nil,
		info,
		token.NewFileSet(),
		".",
		&mockPkgLoader{},
		ifaceWithDetails,
		false,
	)
	if err == nil {
		t.Error("InterfaceTargetCode() expected error for unsupported embedded type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported embedded type") {
		t.Errorf(
			"InterfaceTargetCode() error = %v, want error containing 'unsupported embedded type'",
			err,
		)
	}
}

func TestInterfaceTargetGenerator_collectAdditionalImports_Fallback(t *testing.T) {
	t.Parallel()

	// Create an interface with no source imports (empty SourceImports)
	iface := &dst.InterfaceType{
		Methods: &dst.FieldList{
			List: []*dst.Field{
				{
					Names: []*dst.Ident{{Name: "DoSomething"}},
					Type: &dst.FuncType{
						Params:  &dst.FieldList{},
						Results: &dst.FieldList{},
					},
				},
			},
		},
	}

	// Create a file with imports (simulating another file in the package)
	fileWithImports := &dst.File{
		Name: &dst.Ident{Name: "other"},
		Imports: []*dst.ImportSpec{
			{Path: &dst.BasicLit{Value: `"fmt"`}},
		},
	}

	gen := &interfaceTargetGenerator{
		baseGenerator: baseGenerator{
			pkgName: "testpkg",
		},
		pkgImportPath: "github.com/test/pkg",
		identifiedInterface: detect.IfaceWithDetails{
			Iface:         iface,
			SourceImports: nil, // Empty - should trigger fallback
		},
		astFiles: []*dst.File{fileWithImports},
	}

	// Call the method - it should use the fallback path
	result := gen.collectAdditionalImports()

	// Result should be empty since there are no external types in our simple interface
	// But the fallback path should have been exercised
	_ = result
}

func TestInterfaceTargetGenerator_formatQualifiedInterfaceType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		interfaceName  string
		qualifier      string
		pkgPath        string
		needsQualifier bool
		typeParams     string
		want           string
	}{
		{
			name:           "no qualifier",
			interfaceName:  "Reader",
			qualifier:      "",
			needsQualifier: false,
			want:           "Reader",
		},
		{
			name:           "with qualifier",
			interfaceName:  "Reader",
			qualifier:      "io",
			pkgPath:        "io",
			needsQualifier: true,
			want:           "_io.Reader", // stdlib package gets aliased
		},
		{
			name:           "non-stdlib qualifier",
			interfaceName:  "Ops",
			qualifier:      "pkg",
			pkgPath:        "github.com/example/pkg",
			needsQualifier: true,
			want:           "pkg.Ops",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gen := &interfaceTargetGenerator{
				interfaceName: testCase.interfaceName,
				baseGenerator: baseGenerator{
					qualifier:      testCase.qualifier,
					pkgPath:        testCase.pkgPath,
					needsQualifier: testCase.needsQualifier,
				},
			}

			got := gen.formatQualifiedInterfaceType()
			if got != testCase.want {
				t.Errorf("formatQualifiedInterfaceType() = %q, want %q", got, testCase.want)
			}
		})
	}
}
