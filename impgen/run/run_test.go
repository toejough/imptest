//nolint:testpackage // Tests internal functions
package run

import (
	"errors"
	"go/token"
	"go/types"
	"io"
	"os"
	"strings"
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := determineGeneratedTypeName(testCase.mode, testCase.interfaceName)
			if got != testCase.want {
				t.Errorf("determineGeneratedTypeName() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestFindSymbol_Error(t *testing.T) {
	t.Parallel()

	// Test that findSymbol returns error when symbol not found
	info := generate.GeneratorInfo{
		LocalInterfaceName: "NonExistentSymbol",
		PkgName:            "testpkg",
	}

	// Empty files means no symbols will be found
	loader := &mockPkgLoader{
		files: []*dst.File{},
		fset:  token.NewFileSet(),
	}

	_, err := findSymbol(info, []*dst.File{}, token.NewFileSet(), ".", loader)
	if err == nil {
		t.Error("findSymbol() expected error, got nil")
	}
}

func TestGetOutputFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		impName string
		pkgName string
		goFile  string
		want    string
	}{
		{
			name:    "test package adds _test suffix",
			impName: "MockFoo",
			pkgName: "pkg_test",
			goFile:  "foo.go",
			want:    "generated_MockFoo_test.go",
		},
		{
			name:    "test file adds _test suffix",
			impName: "MockBar",
			pkgName: "pkg",
			goFile:  "bar_test.go",
			want:    "generated_MockBar_test.go",
		},
		{
			name:    "non-test adds .go suffix",
			impName: "WrapBaz",
			pkgName: "pkg",
			goFile:  "baz.go",
			want:    "generated_WrapBaz.go",
		},
		{
			name:    "impName already has .go suffix",
			impName: "Custom.go",
			pkgName: "pkg",
			goFile:  "custom.go",
			want:    "generated_Custom.go",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			getEnv := func(key string) string {
				if key == goFileEnvVar {
					return testCase.goFile
				}

				return ""
			}

			got := getOutputFilename(testCase.impName, testCase.pkgName, getEnv)
			if got != testCase.want {
				t.Errorf("getOutputFilename() = %q, want %q", got, testCase.want)
			}
		})
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

	// Pass stub values to satisfy nil checks, even though they won't be used
	// because the function returns an error before accessing them.
	_, err := routeFunctionGenerator(nil, info, nil, "", &mockPkgLoader{}, &dst.FuncDecl{})
	if !errors.Is(err, ErrFunctionModeRequired) {
		t.Errorf("expected ErrFunctionModeRequired, got %v", err)
	}
}

func TestRouteFunctionTypeGenerator_DefaultModeError(t *testing.T) {
	t.Parallel()

	info := generate.GeneratorInfo{
		Mode: generate.NamingModeDefault,
	}

	_, err := routeFunctionTypeGenerator(
		nil,
		info,
		nil,
		"",
		&mockPkgLoader{},
		detect.FuncTypeWithDetails{},
	)
	if !errors.Is(err, ErrFunctionModeRequired) {
		t.Errorf("expected ErrFunctionModeRequired, got %v", err)
	}
}

func TestRouteInterfaceGenerator_DefaultModeError(t *testing.T) {
	t.Parallel()

	info := generate.GeneratorInfo{
		Mode: generate.NamingModeDefault,
	}

	_, err := routeInterfaceGenerator(
		nil,
		info,
		nil,
		"",
		&mockPkgLoader{},
		detect.IfaceWithDetails{},
	)
	if !errors.Is(err, ErrInterfaceModeRequired) {
		t.Errorf("expected ErrInterfaceModeRequired, got %v", err)
	}
}

func TestRouteStructGenerator_DefaultModeError(t *testing.T) {
	t.Parallel()

	info := generate.GeneratorInfo{
		Mode: generate.NamingModeDefault,
	}

	_, err := routeStructGenerator(nil, info, nil, "", &mockPkgLoader{}, detect.StructWithDetails{})
	if !errors.Is(err, ErrFunctionModeRequired) {
		t.Errorf("expected ErrFunctionModeRequired, got %v", err)
	}
}

func TestRun_CacheHit(t *testing.T) {
	t.Parallel()

	loader, fileSystem := createTestInterfaceAST("TestIface")

	getEnv := func(key string) string {
		switch key {
		case "GOPACKAGE":
			return "testpkg_test"
		case goFileEnvVar:
			return "test_file_test.go"
		case "IMPGEN_NO_CACHE":
			return "" // Enable caching
		}

		return ""
	}

	var output1 strings.Builder

	// First run: generates the file (cache miss)
	err := Run(
		[]string{"impgen", "TestIface", "--dependency"},
		getEnv,
		fileSystem,
		loader,
		&output1,
	)
	if err != nil {
		t.Fatalf("First Run() error = %v", err)
	}

	// Verify file was written
	if _, ok := fileSystem.files["generated_MockTestIface_test.go"]; !ok {
		t.Fatal("Expected file to be written on first run")
	}

	// Verify first run was NOT cached
	if strings.Contains(output1.String(), "unchanged (cached)") {
		t.Error("First run should not be cached")
	}

	var output2 strings.Builder

	// Second run: should hit cache
	err = Run([]string{"impgen", "TestIface", "--dependency"}, getEnv, fileSystem, loader, &output2)
	if err != nil {
		t.Fatalf("Second Run() error = %v", err)
	}

	// Verify second run hit cache
	if !strings.Contains(output2.String(), "unchanged (cached)") {
		t.Errorf("Second run should be cached, got output: %s", output2.String())
	}
}

func TestRun_GOPACKAGENotSet(t *testing.T) {
	t.Parallel()

	// Test that Run returns error when GOPACKAGE is not set
	getEnv := func(_ string) string { return "" }

	// Pass stub values to satisfy nil checks, even though they won't be used
	// because the function returns an error before accessing them.
	err := Run(
		[]string{"-target", "SomeInterface"},
		getEnv,
		&mockCachingFileSystem{},
		&mockPkgLoader{},
		io.Discard,
	)
	if !errors.Is(err, errGOPACKAGENotSet) {
		t.Errorf("Run() error = %v, want %v", err, errGOPACKAGENotSet)
	}
}

func TestRun_WithTiming(t *testing.T) {
	t.Parallel()

	loader, fileSystem := createTestInterfaceAST("TimingIface")

	getEnv := func(key string) string {
		switch key {
		case "GOPACKAGE":
			return "testpkg_test"
		case goFileEnvVar:
			return "test_file_test.go"
		case "IMPGEN_NO_CACHE":
			return "" // Enable caching
		case "IMPGEN_TIMING":
			return "1" // Enable timing output
		}

		return ""
	}

	var output strings.Builder

	// Run with timing enabled
	err := Run(
		[]string{"impgen", "TimingIface", "--dependency"},
		getEnv,
		fileSystem,
		loader,
		&output,
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify timing output was produced
	outputStr := output.String()
	if !strings.Contains(outputStr, "Args/resolve:") {
		t.Errorf("Expected timing output, got: %s", outputStr)
	}
}

// unexported constants.
const (
	goFileEnvVar = "GOFILE"
)

// mockCachingFileSystem implements FileSystem for testing cache behavior.
type mockCachingFileSystem struct {
	files map[string][]byte
}

func (m *mockCachingFileSystem) Glob(pattern string) ([]string, error) {
	var matches []string

	for name := range m.files {
		if strings.Contains(name, strings.TrimPrefix(strings.TrimSuffix(pattern, "*"), "*")) {
			matches = append(matches, name)
		}
	}

	return matches, nil
}

func (m *mockCachingFileSystem) ReadFile(name string) ([]byte, error) {
	if data, ok := m.files[name]; ok {
		return data, nil
	}

	return nil, errors.New("file not found")
}

func (m *mockCachingFileSystem) WriteFile(name string, data []byte, _ os.FileMode) error {
	m.files[name] = data

	return nil
}

// mockFileReader is a test mock for FileReader.
type mockFileReader struct {
	files map[string][]byte
}

func (m *mockFileReader) Glob(_ string) ([]string, error) {
	return nil, nil
}

func (m *mockFileReader) ReadFile(name string) ([]byte, error) {
	if data, ok := m.files[name]; ok {
		return data, nil
	}

	return nil, errors.New("file not found")
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

// createTestInterfaceAST creates a minimal interface AST for testing.
func createTestInterfaceAST(ifaceName string) (*mockPkgLoader, *mockCachingFileSystem) {
	ifaceDecl := &dst.GenDecl{
		Tok: token.TYPE,
		Specs: []dst.Spec{
			&dst.TypeSpec{
				Name: &dst.Ident{Name: ifaceName},
				Type: &dst.InterfaceType{
					Methods: &dst.FieldList{
						List: []*dst.Field{
							{
								Names: []*dst.Ident{{Name: "Method"}},
								Type: &dst.FuncType{
									Params:  &dst.FieldList{},
									Results: nil,
								},
							},
						},
					},
				},
			},
		},
	}
	testFile := &dst.File{
		Name:  &dst.Ident{Name: "testpkg"},
		Decls: []dst.Decl{ifaceDecl},
	}

	return &mockPkgLoader{
			files: []*dst.File{testFile},
			fset:  token.NewFileSet(),
		}, &mockCachingFileSystem{
			files: make(map[string][]byte),
		}
}
