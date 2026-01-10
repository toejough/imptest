package run

import (
	"errors"
	"go/token"
	"go/types"
	"io"
	"testing"

	"github.com/dave/dst"
	detect "github.com/toejough/imptest/impgen/run/3_detect"
	generate "github.com/toejough/imptest/impgen/run/5_generate"
)

func TestDetermineGeneratedTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mode          generate.NamingMode
		interfaceName string
		want          string
	}{
		{
			name:          "target mode",
			mode:          generate.NamingModeTarget,
			interfaceName: "Calculator",
			want:          "WrapCalculator",
		},
		{
			name:          "dependency mode",
			mode:          generate.NamingModeDependency,
			interfaceName: "Calculator",
			want:          "MockCalculator",
		},
		{
			name:          "default mode interface",
			mode:          generate.NamingModeDefault,
			interfaceName: "Calculator",
			want:          "CalculatorImp",
		},
		{
			name:          "default mode method",
			mode:          generate.NamingModeDefault,
			interfaceName: "Calculator.Add",
			want:          "CalculatorAdd",
		},
		{
			name:          "target mode with dots",
			mode:          generate.NamingModeTarget,
			interfaceName: "Calculator.Add",
			want:          "WrapCalculatorAdd",
		},
		{
			name:          "dependency mode with dots",
			mode:          generate.NamingModeDependency,
			interfaceName: "Calculator.Add",
			want:          "MockCalculatorAdd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := determineGeneratedTypeName(tt.mode, tt.interfaceName)
			if got != tt.want {
				t.Errorf("determineGeneratedTypeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateCode_FindSymbolError(t *testing.T) {
	t.Parallel()

	// Test that generateCode returns error when FindSymbol fails
	info := generate.GeneratorInfo{
		LocalInterfaceName: "NonExistentSymbol",
		PkgName:            "testpkg",
	}

	// Empty files means no symbols will be found
	loader := &mockPkgLoader{
		files: []*dst.File{},
		fset:  token.NewFileSet(),
	}

	_, err := generateCode(info, []*dst.File{}, token.NewFileSet(), ".", loader)
	if err == nil {
		t.Error("generateCode() expected error, got nil")
	}
}

func TestLoadPackage_Error(t *testing.T) {
	t.Parallel()

	// Test that loadPackage returns error when Load fails
	loader := &mockPkgLoader{err: errors.New("load failed")}

	_, _, err := loadPackage(".", loader)
	if err == nil {
		t.Error("loadPackage() expected error, got nil")
	}
}

func TestRouteFunctionGenerator_DefaultModeError(t *testing.T) {
	t.Parallel()

	info := generate.GeneratorInfo{
		Mode: generate.NamingModeDefault,
	}

	_, err := routeFunctionGenerator(nil, info, nil, "", nil, nil)
	if !errors.Is(err, ErrFunctionModeRequired) {
		t.Errorf("expected ErrFunctionModeRequired, got %v", err)
	}
}

func TestRouteFunctionTypeGenerator_DefaultModeError(t *testing.T) {
	t.Parallel()

	info := generate.GeneratorInfo{
		Mode: generate.NamingModeDefault,
	}

	_, err := routeFunctionTypeGenerator(nil, info, nil, "", nil, detect.FuncTypeWithDetails{})
	if !errors.Is(err, ErrFunctionModeRequired) {
		t.Errorf("expected ErrFunctionModeRequired, got %v", err)
	}
}

func TestRouteInterfaceGenerator_DefaultModeError(t *testing.T) {
	t.Parallel()

	info := generate.GeneratorInfo{
		Mode: generate.NamingModeDefault,
	}

	_, err := routeInterfaceGenerator(nil, info, nil, "", nil, detect.IfaceWithDetails{})
	if !errors.Is(err, ErrInterfaceModeRequired) {
		t.Errorf("expected ErrInterfaceModeRequired, got %v", err)
	}
}

func TestRouteStructGenerator_DefaultModeError(t *testing.T) {
	t.Parallel()

	info := generate.GeneratorInfo{
		Mode: generate.NamingModeDefault,
	}

	_, err := routeStructGenerator(nil, info, nil, "", nil, detect.StructWithDetails{})
	if !errors.Is(err, ErrFunctionModeRequired) {
		t.Errorf("expected ErrFunctionModeRequired, got %v", err)
	}
}

func TestRun_GOPACKAGENotSet(t *testing.T) {
	t.Parallel()

	// Test that Run returns error when GOPACKAGE is not set
	getEnv := func(_ string) string { return "" }

	err := Run([]string{"-target", "SomeInterface"}, getEnv, nil, nil, io.Discard)
	if !errors.Is(err, errGOPACKAGENotSet) {
		t.Errorf("Run() error = %v, want %v", err, errGOPACKAGENotSet)
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
