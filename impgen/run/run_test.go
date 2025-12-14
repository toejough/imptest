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

const localPackageSource = `package mypkg
import "github.com/toejough/imptest/UAT/run"
`

var errPackageNotFound = errors.New("package not found")

var errWriteFailed = errors.New("write failed")

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	files     map[string][]byte
	writeHook func(name string, data []byte) error
}

// NewMockFileSystem creates a new MockFileSystem for testing file operations.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
	}
}

// WriteFile implements FileSystem.WriteFile for testing.
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

// envWithPkgName returns the test package name, ignoring the provided cwd parameter.
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

	// Try to reference a package - should fail with error about malformed import
	args := []string{"generator", "pkg.SomeInterface", "--name", "TestImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error for malformed import path, got nil")
	}
}

func TestRun_PackageNotInImports(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	// Local package with valid imports, but not the one we're looking for
	sourceCode := `package mypkg
import "fmt"
import "strings"
`
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// Try to reference a package that isn't imported
	args := []string{"generator", "nothere.SomeInterface", "--name", "TestImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err == nil {
		t.Error("Expected error when package not in imports, got nil")
	}
}

func TestRun_CallableWrapper_Simple(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Package with a simple function
	sourceCode := `package run

func PrintSum(a, b int) int {
	return a + b
}
`
	// Create a mock package loader that returns both local and run packages
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	// Generate test helper for this function
	args := []string{"impgen", "run.PrintSum", "--name", "PrintSumImp", "--call"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["PrintSumImp.go"]
	if !ok {
		t.Fatal("Expected PrintSumImp.go to be created")
	}

	contentStr := string(content)

	expected := []string{
		"type PrintSumImp struct",
		"type PrintSumImpReturn struct",
		"func NewPrintSumImp",
		"func (s *PrintSumImp) Start(a, b int)",
		"func (s *PrintSumImp) ExpectReturnedValues(v1 int)",
		"returnChan",
		"panicChan",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRun_CallableWrapper_ComplexTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Package with a function that uses complex types
	sourceCode := `package run

type MyType struct {
	Value int
}

func ProcessData(data []string, callback func(string) error) (*MyType, error) {
	return &MyType{Value: 42}, nil
}
`
	// Create a mock package loader
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	// Generate test helper
	args := []string{"impgen", "run.ProcessData", "--name", "ProcessDataImp", "--call"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["ProcessDataImp.go"]
	if !ok {
		t.Fatal("Expected ProcessDataImp.go to be created")
	}

	contentStr := string(content)

	// Verify complex types are handled correctly
	expected := []string{
		"type ProcessDataImp struct",
		"type ProcessDataImpReturn struct",
		"func NewProcessDataImp",
		"func (s *ProcessDataImp) Start",
		"func (s *ProcessDataImp) ExpectReturnedValues",
		"[]string",           // slice type
		"func(string) error", // function type
		"*run.MyType",        // pointer to custom type (with qualifier since it's from different package)
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRun_CallableWrapper_MapAndChannelTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Package with a function that uses map and channel types
	sourceCode := `package run

func ProcessMap(data map[string]int, ch chan<- string) map[int][]string {
	return make(map[int][]string)
}
`
	// Create a mock package loader
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	// Generate test helper
	args := []string{"impgen", "run.ProcessMap", "--name", "ProcessMapImp", "--call"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["ProcessMapImp.go"]
	if !ok {
		t.Fatal("Expected ProcessMapImp.go to be created")
	}

	contentStr := string(content)

	// Verify map and channel types are handled correctly
	expected := []string{
		"type ProcessMapImp struct",
		"type ProcessMapImpReturn struct",
		"func NewProcessMapImp",
		"func (s *ProcessMapImp) Start",
		"func (s *ProcessMapImp) ExpectReturnedValues",
		"map[string]int",   // map type
		"chan<- string",    // send-only channel
		"map[int][]string", // return map type with slice value
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func Test_Run_CallableWrapper_MethodReference(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Source includes: a standalone function, a method with the name we want,
	// and another method with a different name - this exercises all branches
	sourceCode := `package run

func StandaloneFunc() {}

type Player struct{ name string }

func (p *Player) OtherMethod() {}
func (p *Player) Play() {}

type OtherType struct{}
func (o OtherType) Play() {}
`
	localPackageSource := `package run_test

import "github.com/toejough/imptest/UAT/run"

var _ = run.Player{}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.Player.Play", "--name", "PlayerPlayImp", "--call"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["PlayerPlayImp.go"]
	if !ok {
		t.Fatal("Expected PlayerPlayImp.go to be created")
	}

	contentStr := string(content)

	expected := []string{
		"type PlayerPlayImp struct",
		"func NewPlayerPlayImp",
		"func (s *PlayerPlayImp) Start",
		"func (s *PlayerPlayImp) ExpectReturnedValues",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
		}
	}

	// Should not import the run package since Play() has no params or returns
	if strings.Contains(contentStr, `"github.com/toejough/imptest/UAT/run"`) {
		t.Error("Should not import run package for method with no params or returns")
	}
}

func Test_Run_CallableWrapper_MethodReferenceWithParams(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Source with a struct and method that uses a VALUE receiver (not pointer)
	sourceCode := `package run

type Calculator struct{}

func (c Calculator) Add(a, b int) int {
return a + b
}
`
	localPackageSource := `package run_test

import "github.com/toejough/imptest/UAT/run"

var _ = run.Calculator{}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.Calculator.Add", "--name", "CalculatorAddImp", "--call"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["CalculatorAddImp.go"]
	if !ok {
		t.Fatal("Expected CalculatorAddImp.go to be created")
	}

	contentStr := string(content)

	// Verify structure
	expected := []string{
		"type CalculatorAddImp struct",
		"type CalculatorAddImpReturn struct",
		"func NewCalculatorAddImp",
		"func (s *CalculatorAddImp) Start(a, b int)",
		"func (s *CalculatorAddImp) ExpectReturnedValues",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRunCallable_NamedReturns(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Source with a function that has named return values
	sourceCode := `package run

func Divide(a, b int) (quotient, remainder int) {
	return a / b, a % b
}
`
	localPackageSource := `package run_test

import "github.com/toejough/imptest/UAT/run"

var _ = run.Divide
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.Divide", "--name", "DivideImp", "--call"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["DivideImp.go"]
	if !ok {
		t.Fatal("Expected DivideImp.go to be created")
	}

	contentStr := string(content)

	// Verify structure for named returns
	expected := []string{
		"type DivideImp struct",
		"type DivideImpReturn struct",
		"func NewDivideImp",
		"func (s *DivideImp) Start(a, b int)",
		"func (s *DivideImp) ExpectReturnedValues(v1 int, v2 int)",
		"Result0 int",
		"Result1 int",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}
