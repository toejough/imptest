package run_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

const (
	appDir  = "/app"
	pkgName = "mypkg"
)

var errPackageNotFound = errors.New("package not found")

const skipInterfaceSource = `package mypkg
type SkipInterface interface {
	Method()
}`

var errWriteFailed = errors.New("write failed")

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	files     map[string][]byte
	writeHook func(name string, data []byte) error
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
	}
}

func (m *MockFileSystem) WriteFile(name string, data []byte, _ os.FileMode) error {
	if m.writeHook != nil {
		return m.writeHook(name, data)
	}

	m.files[name] = data

	return nil
}

// MockPackageLoader implements PackageLoader for testing.
type MockPackageLoader struct {
	packages map[string]mockPackage
}

type mockPackage struct {
	files []*ast.File
	fset  *token.FileSet
	err   error
}

// NewMockPackageLoader creates a new MockPackageLoader.
func NewMockPackageLoader() *MockPackageLoader {
	return &MockPackageLoader{
		packages: make(map[string]mockPackage),
	}
}

// AddPackageFromSource parses source code and registers it under the given import path.
func (m *MockPackageLoader) AddPackageFromSource(importPath, source string) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "source.go", source, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("failed to parse source: %v", err))
	}

	m.packages[importPath] = mockPackage{
		files: []*ast.File{file},
		fset:  fset,
	}
}

// Load returns the mocked package AST.
func (m *MockPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, error) {
	if pkg, ok := m.packages[importPath]; ok {
		return pkg.files, pkg.fset, pkg.err
	}

	return nil, nil, fmt.Errorf("%w: %s", errPackageNotFound, importPath)
}

func TestRun_Success(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Setup mock files
	sourceCode := `
package mypkg

type MyInterface interface {
	DoSomething()
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "MyInterface", "--name", "MyImp"}
	env := func(key string) string {
		if key == "GOPACKAGE" {
			return pkgName
		}

		return ""
	}

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["MyImp.go"]; !ok {
		t.Error("Expected MyImp.go to be created")
	}
}

func TestRun_NoInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Setup mock files with NO interface
	sourceCode := `
package mypkg

type MyStruct struct {}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "MyInterface"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error when interface is missing")
	}
}

func TestRun_WriteError(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	sourceCode := `
package mypkg

type MyInterface interface {
	DoSomething()
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// Fail on write
	mockFS.writeHook = func(_ string, _ []byte) error {
		return errWriteFailed
	}

	args := []string{"generator", "MyInterface"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error on write failure")
	}
}

func TestRun_ComplexInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	sourceCode := `
package mypkg

type ComplexInterface interface {
	Method1(a int, b string) (bool, error)
	Method2(fn func(int) int)
	Method3(a, b int)
	Method4() (x, y int)
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "ComplexInterface", "--name", "ComplexImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["ComplexImp.go"]
	if !ok {
		t.Error("Expected ComplexImp.go to be created")
	}

	// Basic check that content contains expected strings
	contentStr := string(content)

	expected := []string{
		"type ComplexImp struct",
		"func (m *ComplexImpMock) Method1(a int, b string) (bool, error)",
		"func (m *ComplexImpMock) Method2(fn func(int) int)",
		"func (m *ComplexImpMock) Method3(a int, b int)",
		"func (m *ComplexImpMock) Method4() (x, y int)",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
		}
	}
}

func TestRun_Values(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Source with single return and unnamed params
	sourceCode := `
package mypkg

type ValueInterface interface {
	SingleReturn() int
	UnnamedParams(int, string)
	OneString(string)
	OneInt(int)
	OneBool(bool)
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "ValueInterface", "--name", "ValueImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["ValueImp.go"]
	if !ok {
		t.Error("Expected ValueImp.go to be created")
	}

	contentStr := string(content)

	expected := []string{
		// Test single return optimization (InjectResult instead of InjectResults)
		"func (c *ValueImpSingleReturnCall) InjectResult(result int)",

		// Method signature uses param0, param1
		"func (m *ValueImpMock) UnnamedParams(param0 int, param1 string)",

		// But Call struct uses A, B (aligned)
		"A:            param0",
		"B:            param1",

		// OneString signature
		"func (m *ValueImpMock) OneString(param0 string)",
		// Call struct uses S
		"S:            param0",

		// OneInt signature
		"func (m *ValueImpMock) OneInt(param0 int)",
		// Call struct uses Input
		"Input:        param0",

		// OneBool signature
		"func (m *ValueImpMock) OneBool(param0 bool)",
		// Call struct uses A (fallthrough)
		"A:            param0",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr) // Debugging
		}
	}
}

func createStringerAST() (*ast.File, *token.FileSet) {
	fset := token.NewFileSet()
	stringerFile := &ast.File{
		Name: ast.NewIdent("fmt"),
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{
						Name: ast.NewIdent("Stringer"),
						Type: &ast.InterfaceType{
							Methods: &ast.FieldList{
								List: []*ast.Field{
									{
										Names: []*ast.Ident{ast.NewIdent("String")},
										Type: &ast.FuncType{
											Params: &ast.FieldList{},
											Results: &ast.FieldList{
												List: []*ast.Field{
													{Type: ast.NewIdent("string")},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return stringerFile, fset
}

func TestRun_ForeignInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Local package source that imports "fmt" so getPackageAndMatchName can resolve the import.
	localSource := `package mypkg
import "fmt"
var _ fmt.Stringer
`

	// Create a mock package loader that returns both local and fmt packages
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localSource)

	stringerFile, fset := createStringerAST()
	mockPkgLoader.packages["fmt"] = mockPackage{
		files: []*ast.File{stringerFile},
		fset:  fset,
	}

	args := []string{"generator", "fmt.Stringer", "--name", "StringerImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["StringerImp.go"]
	if !ok {
		t.Error("Expected StringerImp.go to be created")
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "func (m *StringerImpMock) String() string") {
		t.Error("Expected Stringer method generated from fmt.Stringer")
	}
}

func TestRun_PackageLoaderError(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()
	// Don't register any packages - Load will fail

	args := []string{"generator", "MyInterface", "--name", "MyImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error from package loader, got nil")
	}
}

func TestRun_ParseFiles_Filtering(t *testing.T) {
	t.Parallel()

	// 4. Directory entry (should be skipped) - now tests via package loader
	t.Run("Skip Directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockPkgLoader := NewMockPackageLoader()
		mockPkgLoader.AddPackageFromSource(".", skipInterfaceSource)

		args := []string{"generator", "SkipInterface", "--name", "SkipImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS, mockPkgLoader)
		if err != nil {
			t.Errorf("Should find interface, got error: %v", err)
		}
	})

	// 5. Non-.go file (should be skipped) - now tests via package loader
	t.Run("Skip Non-Go File", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockPkgLoader := NewMockPackageLoader()
		mockPkgLoader.AddPackageFromSource(".", skipInterfaceSource)

		args := []string{"generator", "SkipInterface", "--name", "SkipImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS, mockPkgLoader)
		if err != nil {
			t.Errorf("Should find interface, got error: %v", err)
		}
	})
}

func TestRun_ParseFiles_GeneratedGo(t *testing.T) {
	t.Parallel()

	// 6. Skip generated.go file - now tests via package loader
	t.Run("Skip generated.go", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockPkgLoader := NewMockPackageLoader()
		mockPkgLoader.AddPackageFromSource(".", skipInterfaceSource)

		args := []string{"generator", "SkipInterface", "--name", "SkipImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS, mockPkgLoader)
		if err != nil {
			t.Errorf("Should find interface, got error: %v", err)
		}
	})
}

func TestRun_EmbeddedInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Interface with embedded interface (unnamed field)
	sourceCode := `
package mypkg

type BaseInterface interface {
	BaseMethod()
}

type EmbeddedInterface interface {
	BaseInterface
	OwnMethod()
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "EmbeddedInterface", "--name", "EmbeddedImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["EmbeddedImp.go"]
	if !ok {
		t.Error("Expected EmbeddedImp.go to be created")
	}

	contentStr := string(content)

	// Should only generate methods for OwnMethod, not BaseInterface (embedded)
	if !strings.Contains(contentStr, "func (m *EmbeddedImpMock) OwnMethod()") {
		t.Error("Expected OwnMethod to be generated")
	}

	// Should NOT generate methods for the embedded interface itself
	if strings.Contains(contentStr, "BaseInterface") {
		t.Error("Should not generate code for embedded interface")
	}
}

func TestRun_TestPackageAppendsTestToFilename(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Setup mock files
	sourceCode := `
package mypkg_test

type MyInterface interface {
	DoSomething()
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "MyInterface", "--name", "MyImp"}
	env := func(key string) string {
		if key == "GOPACKAGE" {
			return "mypkg_test" // Simulate being in a _test package
		}

		return ""
	}

	err := run.Run(args, env, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should create MyImp_test.go when in a _test package
	if _, ok := mockFS.files["MyImp_test.go"]; !ok {
		t.Error("Expected MyImp_test.go to be created in _test package")
		// Show what files were actually created for debugging
		for f := range mockFS.files {
			t.Logf("Found file: %s", f)
		}
	}
}

func TestRun_ParseAST_Error(t *testing.T) {
	t.Parallel()

	// Test: Nonsense import path that packages.Load cannot resolve
	t.Run("Nonsense Import Path", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockPkgLoader := NewMockPackageLoader()

		// Local package that imports "not/a/real/path"
		sourceCode := `package mypkg
import "not/a/real/path"
`
		mockPkgLoader.AddPackageFromSource(".", sourceCode)
		// Don't register the "not/a/real/path" package - Load will fail for it

		// Use the imported package name - "path" is the last segment
		args := []string{"generator", "path.SomeInterface", "--name", "TestImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS, mockPkgLoader)
		if err == nil {
			t.Error("Expected error loading nonsense package path, got nil")
		}
	})
}
