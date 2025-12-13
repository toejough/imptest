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

const pkgName = "mypkg"

var errPackageNotFound = errors.New("package not found")

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
		return pkg.files, pkg.fset, nil
	}

	return nil, nil, fmt.Errorf("%w: %s", errPackageNotFound, importPath)
}

func envWithPkgName(_ string) string { return pkgName }

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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
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
		// Call struct uses I
		"I:            param0",

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

func TestRun_ForeignInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Local package source that imports "fmt" so getPackageAndMatchName can resolve the import.
	localSource := `package mypkg
import "fmt"
var _ fmt.Stringer
`

	// Fake fmt.Stringer interface
	fmtSource := `package fmt

type Stringer interface {
	String() string
}
`

	// Create a mock package loader that returns both local and fmt packages
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localSource)
	mockPkgLoader.AddPackageFromSource("fmt", fmtSource)

	args := []string{"generator", "fmt.Stringer", "--name", "StringerImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error from package loader, got nil")
	}
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
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
	env := func(_ string) string { return "mypkg_test" }

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

func TestRun_ForeignPackageLoadError(t *testing.T) {
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error loading nonsense package path, got nil")
	}
}

func TestRun_InvalidArgs(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	// No interface argument provided - should fail argument parsing
	args := []string{"generator"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error from invalid arguments, got nil")
	}
}

func TestRun_LocalPackageLoadErrorForForeignInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()
	// Don't register any packages - Load will fail when trying to load "." to resolve imports

	// Use a qualified interface name (pkg.Interface) which requires loading local package
	args := []string{"generator", "pkg.SomeInterface", "--name", "TestImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error loading local package for foreign interface, got nil")
	}
}

func TestRun_MalformedImportPath(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	// Create a file with a malformed import (missing closing quote)
	fset := token.NewFileSet()
	file := &ast.File{
		Name: &ast.Ident{Name: "mypkg"},
		Imports: []*ast.ImportSpec{
			{
				Path: &ast.BasicLit{
					Value: `"valid/import"`,
				},
			},
			{
				Path: &ast.BasicLit{
					Value: `malformed`, // Invalid - not quoted properly
				},
			},
		},
	}
	mockPkgLoader.packages["."] = mockPackage{
		files: []*ast.File{file},
		fset:  fset,
	}

	// Try to reference a package - should skip the malformed import and fail to find it
	args := []string{"generator", "pkg.SomeInterface", "--name", "TestImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error when package not found (due to malformed import), got nil")
	}
}
