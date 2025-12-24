package run_test

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

// MockCacheFileSystem implements CacheFileSystem for testing.
type MockCacheFileSystem struct {
	files  map[string][]byte
	cwd    string
	stats  map[string]os.FileInfo
	mkdirs []string
}

// NewMockCacheFileSystem creates a new MockCacheFileSystem.
func NewMockCacheFileSystem() *MockCacheFileSystem {
	return &MockCacheFileSystem{
		files:  make(map[string][]byte),
		cwd:    "/test/project",
		stats:  make(map[string]os.FileInfo),
		mkdirs: []string{},
	}
}

// Create implements CacheFileSystem.Create.
func (m *MockCacheFileSystem) Create(path string) (io.WriteCloser, error) {
	return &mockWriteCloser{fs: m, path: path}, nil
}

// Getwd implements CacheFileSystem.Getwd.
func (m *MockCacheFileSystem) Getwd() (string, error) {
	return m.cwd, nil
}

// MkdirAll implements CacheFileSystem.MkdirAll.
func (m *MockCacheFileSystem) MkdirAll(path string, _ os.FileMode) error {
	m.mkdirs = append(m.mkdirs, path)

	return nil
}

// Open implements CacheFileSystem.Open.
func (m *MockCacheFileSystem) Open(path string) (io.ReadCloser, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// SetCwd sets the current working directory for testing.
func (m *MockCacheFileSystem) SetCwd(cwd string) {
	m.cwd = cwd
}

// SetFile sets file content for testing.
func (m *MockCacheFileSystem) SetFile(path string, data []byte) {
	m.files[path] = data
}

// SetStat sets file info for a path.
func (m *MockCacheFileSystem) SetStat(path string, info os.FileInfo) {
	m.stats[path] = info
}

// Stat implements CacheFileSystem.Stat.
func (m *MockCacheFileSystem) Stat(path string) (os.FileInfo, error) {
	if info, ok := m.stats[path]; ok {
		return info, nil
	}

	return nil, os.ErrNotExist
}

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	files     map[string][]byte
	writeHook func(name string, data []byte) error
	globHook  func(pattern string) ([]string, error)
}

// NewMockFileSystem creates a new MockFileSystem for testing file operations.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
	}
}

// Glob implements FileSystem.Glob for testing.
func (m *MockFileSystem) Glob(pattern string) ([]string, error) {
	if m.globHook != nil {
		return m.globHook(pattern)
	}

	// Simple mock implementation - return files that match *.go pattern
	if pattern == "*.go" {
		var matches []string

		for name := range m.files {
			if strings.HasSuffix(name, ".go") {
				matches = append(matches, name)
			}
		}

		return matches, nil
	}

	return nil, nil
}

// ReadFile implements FileSystem.ReadFile for testing.
func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	return data, nil
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
	errors   map[string]error
}

// NewMockPackageLoader creates a new MockPackageLoader.
func NewMockPackageLoader() *MockPackageLoader {
	return &MockPackageLoader{
		packages: make(map[string]mockPackage),
		errors:   make(map[string]error),
	}
}

// AddError configures the mock to return an error for a specific import path.
func (m *MockPackageLoader) AddError(importPath string, err error) {
	m.errors[importPath] = err
}

// AddPackageFromSource parses source code and registers it under the given import path.
func (m *MockPackageLoader) AddPackageFromSource(importPath, source string) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "source.go", source, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("failed to parse source: %v", err))
	}

	// Type-check the package
	typesInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	conf := types.Config{
		Importer: importer.Default(),
		Error:    func(_ error) {}, // Ignore type errors in tests
	}

	_, _ = conf.Check(importPath, fset, []*ast.File{file}, typesInfo)

	m.packages[importPath] = mockPackage{
		files:     []*ast.File{file},
		fset:      fset,
		typesInfo: typesInfo,
	}
}

// Load returns the mocked package AST, FileSet, and type info.
func (m *MockPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
	if err, ok := m.errors[importPath]; ok {
		return nil, nil, nil, err
	}

	if pkg, ok := m.packages[importPath]; ok {
		return pkg.files, pkg.fset, pkg.typesInfo, nil
	}

	return nil, nil, nil, fmt.Errorf("%w: %s", errPackageNotFound, importPath)
}

func TestCallableHeaderGeneration(t *testing.T) {
	t.Parallel()

	t.Run("without qualifier", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockPkgLoader := NewMockPackageLoader()
		mockPkgLoader.AddPackageFromSource(".", localPackageSource)
		mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", `package run
func LocalFunc() int { return 0 }`)

		args := []string{"generator", "run.LocalFunc", "--name", "LocalFuncImp"}

		err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		code := string(mockFS.files["generated_LocalFuncImp.go"])
		if !strings.Contains(code, "package mypkg") {
			t.Error("Expected package declaration in header")
		}

		if !strings.Contains(code, `"github.com/toejough/imptest/imptest"`) {
			t.Error("Expected imptest import in header")
		}
	})

	t.Run("with qualifier", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockPkgLoader := NewMockPackageLoader()
		mockPkgLoader.AddPackageFromSource(".", `package current
import "github.com/foo/external"
var _ = external.Process`)
		mockPkgLoader.AddPackageFromSource("github.com/foo/external", `package external
type Data struct { Value string }
func Process(input Data) Data { return input }`)

		args := []string{"generator", "external.Process", "--name", "ProcessImp"}

		err := run.Run(args, func(_ string) string { return "current" }, mockFS, mockPkgLoader, io.Discard)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		code := string(mockFS.files["generated_ProcessImp.go"])
		if !strings.Contains(code, "package current") {
			t.Error("Expected package declaration in header")
		}

		if !strings.Contains(code, `external "github.com/foo/external"`) {
			t.Logf("Generated code:\n%s", code)
			t.Error("Expected qualified import in header")
		}
	})
}

func TestRunCallable_GenericMultipleTypeParameters(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function that uses a generic type with multiple type parameters
	sourceCode := `package run

type KeyValue[K comparable, V any] struct {
	Key   K
	Value V
}

func ProcessKeyValue(kv *KeyValue[string, int]) string {
	return kv.Key
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessKeyValue", "--name", "ProcessKeyValueImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessKeyValueImp.go"]
	if !ok {
		t.Fatal("Expected ProcessKeyValueImp.go to be created")
	}

	contentStr := string(content)

	// Verify generic type with multiple parameters is handled correctly
	expected := []string{
		"type ProcessKeyValueImp struct",
		"func NewProcessKeyValueImp",
		"func (s *ProcessKeyValueImp) Start",
		"*run.KeyValue[string, int]", // Generic type with multiple params and qualifier
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRunCallable_GenericTypeParameter(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function that uses a generic type instantiation (single type parameter)
	sourceCode := `package run

type Container[T any] struct {
	Value T
}

func ProcessContainer(c *Container[int]) int {
	return c.Value
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessContainer", "--name", "ProcessContainerImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessContainerImp.go"]
	if !ok {
		t.Fatal("Expected ProcessContainerImp.go to be created")
	}

	contentStr := string(content)

	// Verify generic type instantiation is handled correctly
	expected := []string{
		"type ProcessContainerImp struct",
		"func NewProcessContainerImp",
		"func (s *ProcessContainerImp) Start",
		"*run.Container[int]", // Generic type with qualifier
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRunCallable_ImportPathNotFound(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Package with a function that uses exported types
	sourceCode := `package run

type MyType struct {
	Value int
}

func ProcessData(data *MyType) int {
	return data.Value
}
`
	// Local package WITHOUT the needed import
	localSource := `package mypkg

import "fmt"
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localSource) // Has imports, but not "run"
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	// Try to generate callable for run.ProcessData, but "run" package not imported in local package
	args := []string{"impgen", "run.ProcessData", "--name", "ProcessDataImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error when import path not found, got nil")

		return
	}

	if !strings.Contains(err.Error(), "symbol (interface or function) not found") {
		t.Errorf("Expected 'symbol (interface or function) not found' error, got: %v", err)
	}
}

func TestRunCallable_InlineStructType(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function that uses an inline struct type
	sourceCode := `package run

func ProcessStruct(data struct{ Name string; Age int }) string {
	return data.Name
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessStruct", "--name", "ProcessStructImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessStructImp.go"]
	if !ok {
		t.Fatal("Expected ProcessStructImp.go to be created")
	}

	contentStr := string(content)

	// Verify inline struct type is handled
	expected := []string{
		"type ProcessStructImp struct",
		"func NewProcessStructImp",
		"func (s *ProcessStructImp) Start",
		"struct",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRunCallable_LocalFunction(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Source with a simple local function (no qualified name)
	sourceCode := `package mypkg

func SimpleAdd(a, b int) int {
	return a + b
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// Generate callable for local function without package qualifier
	args := []string{"impgen", "SimpleAdd", "--name", "SimpleAddImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_SimpleAddImp.go"]
	if !ok {
		t.Fatal("Expected SimpleAddImp.go to be created")
	}

	contentStr := string(content)

	// Verify structure for local function
	expected := []string{
		"type SimpleAddImp struct",
		"type SimpleAddImpReturn struct",
		"func NewSimpleAddImp",
		"func (s *SimpleAddImp) Start(a, b int)",
		"func (s *SimpleAddImp) ExpectReturnedValuesAre(v1 int)",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}

	// Verify no package import is added for local function
	if strings.Contains(contentStr, `import`) {
		// Should only have imports for testing and reflect, not for any custom package
		if !strings.Contains(contentStr, `"reflect"`) && !strings.Contains(contentStr, `"testing"`) {
			t.Error("Unexpected import found for local function")
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

	args := []string{"impgen", "run.Divide", "--name", "DivideImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_DivideImp.go"]
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
		"func (s *DivideImp) ExpectReturnedValuesAre(v1 int, v2 int)",
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

func TestRunCallable_PackageLoadErrorForExportedTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Package with a function that uses exported types
	sourceCode := `package run

type MyType struct {
	Value int
}

func ProcessData(data *MyType) int {
	return data.Value
}
`
	mockPkgLoader := NewMockPackageLoader()
	// Register the target package but NOT the local package
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)
	// Don't register "." - this will cause Load(".") to fail

	args := []string{"impgen", "run.ProcessData", "--name", "ProcessDataImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error when loading local package for callable with exported types, got nil")

		return
	}

	if !strings.Contains(err.Error(), "failed to load local package") {
		t.Errorf("Expected 'failed to load local package' error, got: %v", err)
	}
}

func TestRunCallable_SelectorExprType(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function that uses a selector expression type (pkg.Type)
	sourceCode := `package run

import "time"

func ProcessTime(t time.Time) string {
	return t.String()
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessTime", "--name", "ProcessTimeImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessTimeImp.go"]
	if !ok {
		t.Fatal("Expected ProcessTimeImp.go to be created")
	}

	contentStr := string(content)

	// Verify selector expression (time.Time) is handled
	expected := []string{
		"type ProcessTimeImp struct",
		"func NewProcessTimeImp",
		"func (s *ProcessTimeImp) Start",
		"time.Time", // Selector expression type
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRunCallable_SliceReturnType(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function with slice return type (non-comparable)
	sourceCode := `package run

func GetNames() []string {
	return []string{"alice", "bob"}
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.GetNames", "--name", "GetNamesImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_GetNamesImp.go"]
	if !ok {
		t.Fatal("Expected GetNamesImp.go to be created")
	}

	contentStr := string(content)

	// Callable wrapper imports imptest for matcher support
	if !strings.Contains(contentStr, `"github.com/toejough/imptest/imptest"`) {
		t.Error("Expected imptest import")
	}

	// Should use imptest.MatchValue in ExpectReturnedValuesShould
	if !strings.Contains(contentStr, "imptest.MatchValue") {
		t.Error("Expected imptest.MatchValue for matcher-based comparison")
	}

	// Should generate both ExpectReturnedValuesAre and ExpectReturnedValuesShould
	if !strings.Contains(contentStr, "ExpectReturnedValuesAre") {
		t.Error("Expected ExpectReturnedValuesAre method")
	}

	if !strings.Contains(contentStr, "ExpectReturnedValuesShould") {
		t.Error("Expected ExpectReturnedValuesShould method")
	}
}

func TestRun_AutoDetect_Function(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	sourceCode := `
package mypkg
func MyFunc(a int) string { return "" }
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// No --call flag, but it's a function
	args := []string{"generator", "MyFunc", "--name", "MyFuncImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_MyFuncImp.go"]
	if !ok {
		t.Fatal("Expected MyFuncImp.go to be created")
	}

	if !strings.Contains(string(content), "type MyFuncImpReturn struct") {
		t.Error("Expected callable wrapper to be generated")
	}
}

func TestRun_AutoDetect_Interface(t *testing.T) {
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

	// No --call flag
	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_MyImp.go"]
	if !ok {
		t.Fatal("Expected MyImp.go to be created")
	}

	if !strings.Contains(string(content), "type MyImpMock struct") {
		t.Error("Expected interface mock to be generated")
	}
}

func TestRun_AutoDetect_Method(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	sourceCode := `
package mypkg
type MyType struct{}
func (m *MyType) MyMethod(a int) {}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// No --call flag, but it's a method
	args := []string{"generator", "MyType.MyMethod", "--name", "MyMethodImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_MyMethodImp.go"]
	if !ok {
		t.Fatal("Expected MyMethodImp.go to be created")
	}

	if !strings.Contains(string(content), "func (s *MyMethodImp) Start(a int)") {
		t.Error("Expected callable wrapper to be generated for method")
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
	args := []string{"impgen", "run.ProcessData", "--name", "ProcessDataImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessDataImp.go"]
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
	args := []string{"impgen", "run.ProcessMap", "--name", "ProcessMapImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessMapImp.go"]
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
	args := []string{"impgen", "run.PrintSum", "--name", "PrintSumImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_PrintSumImp.go"]
	if !ok {
		t.Fatal("Expected PrintSumImp.go to be created")
	}

	contentStr := string(content)

	expected := []string{
		"type PrintSumImp struct",
		"type PrintSumImpReturn struct",
		"func NewPrintSumImp",
		"func (s *PrintSumImp) Start(a, b int)",
		"func (s *PrintSumImp) ExpectReturnedValuesAre(v1 int)",
		"ReturnChan",
		"PanicChan",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ComplexImp.go"]
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

func TestRun_ComplexTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	sourceCode := `package mypkg
import "os"
type ComplexInterface interface {
	ProcessChannels(in <-chan int, out chan<- string)
	ProcessFile(f *os.File) error
	Nested(data [5]*int)
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "ComplexInterface", "--name", "ComplexImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ComplexImp.go"]
	if !ok {
		t.Fatal("Expected ComplexImp.go to be created")
	}

	contentStr := string(content)

	expected := []string{
		"<-chan int",
		"chan<- string",
		"*os.File",
		"[5]*int",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
		}
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_EmbeddedImp.go"]
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

func TestRun_ExternalCallable(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	externalSource := `package external
func DoSomething(a int) string { return "" }
`
	mockPkgLoader.AddPackageFromSource("github.com/foo/external", externalSource)

	mockPkgLoader.AddPackageFromSource(".", externalImportSource)

	args := []string{"impgen", "external.DoSomething", "--name", "DoSomethingImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_DoSomethingImp.go"]; !ok {
		t.Error("Expected DoSomethingImp.go to be created")
	}
}

func TestRun_ExternalCallable_UnexportedName(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	externalSource := `package external
type MyData struct{}
func doSomething(d MyData) {}
`
	mockPkgLoader.AddPackageFromSource("github.com/foo/external", externalSource)

	mockPkgLoader.AddPackageFromSource(".", externalImportSource)

	// doSomething is unexported, but uses exported MyData.
	args := []string{"impgen", "external.doSomething", "--name", "DoSomethingImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_DoSomethingImp.go"]; !ok {
		t.Error("Expected DoSomethingImp.go to be created")
	}
}

func TestRun_ExternalInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	// External package source
	externalSource := `package external
type Secret int
type Service interface {
	Do(s Secret)
}
`
	mockPkgLoader.AddPackageFromSource("github.com/foo/external", externalSource)

	// Local package importing external
	mockPkgLoader.AddPackageFromSource(".", externalImportSource)

	args := []string{"impgen", "external.Service", "--name", "ExternalImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_ExternalImp.go"]; !ok {
		t.Error("Expected ExternalImp.go to be created")
	}
}

func TestRun_ExternalInterface_NoExportedTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	externalSource := `package external
type service interface {
	Do(a int)
}
`
	mockPkgLoader.AddPackageFromSource("github.com/foo/external", externalSource)
	mockPkgLoader.AddPackageFromSource(".", externalImportSource)

	args := []string{"impgen", "external.service", "--name", "ServiceImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_ServiceImp.go"]; !ok {
		t.Error("Expected ServiceImp.go to be created")
	}
}

func TestRun_ExternalInterface_NoExportedTypes_WithTypeParam(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	externalSource := `package external
type service[T any] interface {
	Do(a T)
}
`
	mockPkgLoader.AddPackageFromSource("github.com/foo/external", externalSource)
	mockPkgLoader.AddPackageFromSource(".", externalImportSource)

	args := []string{"impgen", "external.service", "--name", "ServiceImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_ServiceImp.go"]; !ok {
		t.Error("Expected ServiceImp.go to be created")
	}
}

func TestRun_ExternalInterface_UnexportedName(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	externalSource := `package external
type MyData struct{}
type service interface {
	Do(d MyData)
}
`
	mockPkgLoader.AddPackageFromSource("github.com/foo/external", externalSource)

	mockPkgLoader.AddPackageFromSource(".", externalImportSource)

	// service is unexported, but uses exported MyData.
	// We need to use its full name for findSymbol to find it.
	args := []string{"impgen", "external.service", "--name", "ServiceImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_ServiceImp.go"]; !ok {
		t.Error("Expected ServiceImp.go to be created")
	}
}

func TestRun_ExternalInterface_UnexportedType(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	// External package with unexported type in interface
	externalSource := `package external
type secret int
type Service interface {
	Do(s secret)
}
`
	mockPkgLoader.AddPackageFromSource("github.com/foo/external", externalSource)

	// Local package
	mockPkgLoader.AddPackageFromSource(".", externalImportSource)

	args := []string{"impgen", "external.Service", "--name", "ExternalImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Fatal("Expected error due to unexported type in external interface")
	}

	if !strings.Contains(err.Error(), "type 'secret': unexported type is not accessible") {
		t.Errorf("Unexpected error message: %v", err)
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_StringerImp.go"]
	if !ok {
		t.Error("Expected StringerImp.go to be created")
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "func (m *StringerImpMock) String() string") {
		t.Error("Expected Stringer method generated from fmt.Stringer")
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error loading nonsense package path, got nil")
	}
}

func TestRun_GenericInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Generic interface with type parameters
	sourceCode := `package mypkg

type GenericInterface[T any, U comparable] interface {
	Process(item T) U
	Compare(a, b U) bool
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "GenericInterface", "--name", "GenericImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_GenericImp.go"]
	if !ok {
		t.Error("Expected GenericImp.go to be created")
	}

	contentStr := string(content)

	// Verify generic type parameters are rendered correctly
	expected := []string{
		"type GenericImp[T any, U comparable] struct",
		"type GenericImpMock[T any, U comparable] struct",
		"func NewGenericImp[T any, U comparable](t *testing.T) *GenericImp[T, U]",
		"func (m *GenericImpMock[T, U]) Process(item T) U",
		"func (m *GenericImpMock[T, U]) Compare(a U, b U) bool",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRun_InterfaceWithExportedTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	sourceCode := `package mypkg
type MyData struct { Name string }
type MyInterface interface {
	Process(d MyData)
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	// local source imports the package
	localPackageSource := `package run_test
import "github.com/toejough/imptest/UAT/run"
var _ = run.MyData{}
`
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)

	args := []string{"impgen", "run.MyInterface", "--name", "MyImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_MyImp.go"]
	if !ok {
		t.Fatal("Expected MyImp.go to be created")
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "run.MyData") {
		t.Errorf("Expected generated code to contain %q", "run.MyData")
	}
}

func TestRun_InterfaceWithGenericTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	sourceCode := `package mypkg
type Container[T any] struct { Value T }
type Pair[T, U any] struct { First T; Second U }
type GenericInterface interface {
	ProcessContainer(c Container[int])
	ProcessPair(p Pair[string, bool])
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "GenericInterface", "--name", "GenericImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_GenericImp.go"]
	if !ok {
		t.Fatal("Expected GenericImp.go to be created")
	}

	contentStr := string(content)

	expected := []string{
		"Container[int]",
		"Pair[string, bool]",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
		}
	}
}

func TestRun_InterfaceWithMissingTypeInfo(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Interface with undefined type (to trigger missing type info)
	sourceCode := `package mypkg

type Processor interface {
	ProcessData(data UndefinedType) int
}
`
	// Create a mock loader that will have incomplete type info
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"impgen", "Processor", "--name", "ProcessorImp"}

	// This should still generate code even with type errors (we're conservative)
	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should successfully generate code
	if _, ok := mockFS.files["generated_ProcessorImp.go"]; !ok {
		t.Fatal("Expected ProcessorImp.go to be created")
	}
}

func TestRun_InterfaceWithMixedExportedAndGeneric(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	sourceCode := `package mypkg
type MyData struct { Name string }
type MyInterface[T any] interface {
	Process(d MyData, item T)
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	localPackageSource := `package run_test
import "github.com/toejough/imptest/UAT/run"
var _ = run.MyData{}
`
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)

	args := []string{"impgen", "run.MyInterface", "--name", "MyImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_MyImp.go"]
	if !ok {
		t.Fatal("Expected MyImp.go to be created")
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "MyImp[T any]") {
		t.Errorf("Expected generic implementation, got:\n%s", contentStr)
	}
}

func TestRun_InterfaceWithNonComparableTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Interface with slice and map parameters (non-comparable types)
	sourceCode := `package mypkg

type DataProcessor interface {
	ProcessSlice(data []string) int
	ProcessMap(m map[string]int) bool
	ProcessInt(n int) int
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"impgen", "DataProcessor", "--name", "DataProcessorImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_DataProcessorImp.go"]
	if !ok {
		t.Fatal("Expected DataProcessorImp.go to be created")
	}

	contentStr := string(content)

	// Should include reflect import for non-comparable types
	if !strings.Contains(contentStr, `import "reflect"`) {
		t.Error("Expected reflect import for non-comparable types")
	}

	// Should use reflect.DeepEqual for slice parameter
	if !strings.Contains(contentStr, "reflect.DeepEqual(methodCall.data, data)") {
		t.Error("Expected reflect.DeepEqual for slice parameter")
	}

	// Should use reflect.DeepEqual for map parameter
	if !strings.Contains(contentStr, "reflect.DeepEqual(methodCall.m, m)") {
		t.Error("Expected reflect.DeepEqual for map parameter")
	}

	// Should use != for comparable int parameter
	if !strings.Contains(contentStr, "methodCall.n != n") {
		t.Error("Expected != comparison for int parameter")
	}
}

func TestRun_InterfaceWithOnlyComparableTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Interface with only comparable types
	sourceCode := `package mypkg

type Calculator interface {
	Add(a int, b int) int
	Concat(s1 string, s2 string) string
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"impgen", "Calculator", "--name", "CalculatorImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_CalculatorImp.go"]
	if !ok {
		t.Fatal("Expected CalculatorImp.go to be created")
	}

	contentStr := string(content)

	// Should NOT include reflect import when all types are comparable
	if strings.Contains(contentStr, `import "reflect"`) {
		t.Error("Should not have reflect import when all types are comparable")
	}

	// Should use != for all comparisons
	if !strings.Contains(contentStr, "methodCall.a != a") {
		t.Error("Expected != comparison for int parameter")
	}

	if !strings.Contains(contentStr, "methodCall.s1 != s1") {
		t.Error("Expected != comparison for string parameter")
	}
}

func TestRun_LocalPackageLoadErrorForForeignInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()
	// Don't register any packages - Load will fail when trying to load "." to resolve imports

	// Use a qualified interface name (pkg.Interface) which requires loading local package
	args := []string{"generator", "pkg.SomeInterface", "--name", "TestImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error loading local package for foreign interface, got nil")
	}
}

func TestRun_MalformedAliasedImportPath(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()

	// Create a file with a malformed aliased import
	fset := token.NewFileSet()
	file := &ast.File{
		Name: &ast.Ident{Name: "mypkg"},
		Imports: []*ast.ImportSpec{
			{
				Name: &ast.Ident{Name: "pkg"},
				Path: &ast.BasicLit{
					Value: `malformed-alias`, // Invalid - not quoted properly
				},
			},
		},
	}
	mockPkgLoader.packages["."] = mockPackage{
		files: []*ast.File{file},
		fset:  fset,
	}

	// Try to reference the aliased package - should fail with error about malformed import
	args := []string{"generator", "pkg.SomeInterface", "--name", "TestImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error for malformed aliased import path, got nil")
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error for malformed import path, got nil")
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error when interface is missing")
	}
}

func TestRun_PackageLoaderError(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()
	// Don't register any packages - Load will fail

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error from package loader, got nil")
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error when package not in imports, got nil")
	}
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_MyImp.go"]; !ok {
		t.Error("Expected MyImp.go to be created")
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

	err := run.Run(args, env, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should create generated_MyImp_test.go when in a _test package
	if _, ok := mockFS.files["generated_MyImp_test.go"]; !ok {
		t.Error("Expected generated_MyImp_test.go to be created in _test package")
		// Show what files were actually created for debugging
		for f := range mockFS.files {
			t.Logf("Found file: %s", f)
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ValueImp.go"]
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

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err == nil {
		t.Error("Expected error on write failure")
	}
}

// Tests for WithCache

func TestWithCache_CacheHit(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	sourceCode := simpleInterfaceSource
	// Add source file to mock filesystem for signature calculation
	mockFS.files["source.go"] = []byte(sourceCode)

	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// Use the same cache filesystem across both runs
	mockCacheFS := NewMockCacheFileSystem()
	// Set up go.mod so FindProjectRoot succeeds
	mockCacheFS.SetStat("/test/project/go.mod", nil)

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	// First run - populate cache
	err := run.WithCache(args, envWithPkgName, mockFS, mockPkgLoader, mockCacheFS, io.Discard)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// Keep source file but clear generated file to verify we're reading from cache
	generatedFile := mockFS.files["generated_MyImp.go"]
	mockFS.files = make(map[string][]byte)
	mockFS.files["source.go"] = []byte(sourceCode)

	// Second run - should hit cache and restore file
	err = run.WithCache(args, envWithPkgName, mockFS, mockPkgLoader, mockCacheFS, io.Discard)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_MyImp.go"]; !ok {
		t.Error("Expected file to be restored from cache")
	}

	// Verify the content matches what was generated originally
	if string(mockFS.files["generated_MyImp.go"]) != string(generatedFile) {
		t.Error("Cached content doesn't match original")
	}
}

func TestWithCache_DifferentArgs(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	sourceCode := `package mypkg
type MyInterface interface {
	DoSomething()
}
type OtherInterface interface {
	DoOther()
}`
	// Add source file to mock filesystem for signature calculation
	mockFS.files["source.go"] = []byte(sourceCode)

	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	// Use the same cache filesystem across both runs
	mockCacheFS := NewMockCacheFileSystem()
	// Set up go.mod so FindProjectRoot succeeds
	mockCacheFS.SetStat("/test/project/go.mod", nil)

	// First run with MyInterface
	args1 := []string{"generator", "MyInterface", "--name", "MyImp"}

	err := run.WithCache(args1, envWithPkgName, mockFS, mockPkgLoader, mockCacheFS, io.Discard)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// Second run with OtherInterface - should be cache miss (different args)
	args2 := []string{"generator", "OtherInterface", "--name", "OtherImp"}

	err = run.WithCache(args2, envWithPkgName, mockFS, mockPkgLoader, mockCacheFS, io.Discard)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	// Both files should exist
	if _, ok := mockFS.files["generated_MyImp.go"]; !ok {
		t.Error("Expected MyImp.go to exist")
	}

	if _, ok := mockFS.files["generated_OtherImp.go"]; !ok {
		t.Error("Expected OtherImp.go to exist")
	}
}

func TestWithCache_RunError(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockPkgLoader := NewMockPackageLoader()
	// Don't register any packages - will cause Run() to fail

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	err := run.WithCache(args, envWithPkgName, mockFS, mockPkgLoader, NewMockCacheFileSystem(), io.Discard)
	if err == nil {
		t.Error("Expected error when Run() fails")
	}
}

func TestWithCache_SignatureCalculationFails(t *testing.T) {
	t.Parallel()

	// Test fallback when filesystem Glob fails
	mockFS := NewMockFileSystem()
	// Override Glob to return an error
	mockFS.globHook = func(_ string) ([]string, error) {
		return nil, errors.New("glob failed")
	}

	sourceCode := simpleInterfaceSource
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	// Should still work even if signature calculation fails
	err := run.WithCache(args, envWithPkgName, mockFS, mockPkgLoader, NewMockCacheFileSystem(), io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_MyImp.go"]; !ok {
		t.Error("Expected file to be created even if caching fails")
	}
}

func TestWithCache_WriteErrorFromCache(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	sourceCode := simpleInterfaceSource
	// Add source file to mock filesystem for signature calculation
	mockFS.files["source.go"] = []byte(sourceCode)

	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", sourceCode)

	args := []string{"generator", "MyInterface", "--name", "MyImp"}

	// First run - populate cache
	err := run.WithCache(args, envWithPkgName, mockFS, mockPkgLoader, NewMockCacheFileSystem(), io.Discard)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// Configure write to fail
	mockFS.writeHook = func(_ string, _ []byte) error {
		return errWriteFailed
	}

	// Second run - write fails (will take cache miss path since FindProjectRoot fails with mockFS)
	err = run.WithCache(args, envWithPkgName, mockFS, mockPkgLoader, NewMockCacheFileSystem(), io.Discard)
	if err == nil {
		t.Fatal("Expected error when writing fails")
	}

	if !strings.Contains(err.Error(), "error writing") && !strings.Contains(err.Error(), "write failed") {
		t.Errorf("Expected write error, got: %v", err)
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

	args := []string{"impgen", "run.Player.Play", "--name", "PlayerPlayImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_PlayerPlayImp.go"]
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

	args := []string{"impgen", "run.Calculator.Add", "--name", "CalculatorAddImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_CalculatorAddImp.go"]
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

// unexported constants.
const (
	externalImportSource = `package mypkg
import "github.com/foo/external"
var _ external.Service
`
	localPackageSource = `package mypkg
import "github.com/toejough/imptest/UAT/run"
`
	pkgName               = "mypkg"
	simpleInterfaceSource = `package mypkg
type MyInterface interface {
	DoSomething()
}`
)

// unexported variables.
var (
	errPackageNotFound = errors.New("package not found")
	errWriteFailed     = errors.New("write failed")
)

type mockPackage struct {
	files     []*ast.File
	fset      *token.FileSet
	typesInfo *types.Info
}

type mockWriteCloser struct {
	fs   *MockCacheFileSystem
	path string
	buf  bytes.Buffer
}

func (m *mockWriteCloser) Close() error {
	m.fs.files[m.path] = m.buf.Bytes()

	return nil
}

//nolint:wrapcheck // test helper
func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

// envWithPkgName returns the test package name, ignoring the provided cwd parameter.
func envWithPkgName(_ string) string { return pkgName }
