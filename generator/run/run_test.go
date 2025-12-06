package run_test

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/toejough/imptest/generator/run"
)

const (
	appDir  = "/app"
	pkgName = "mypkg"
)

var (
	errCannotReadDir  = errors.New("cannot read dir")
	errCannotReadFile = errors.New("cannot read file")
)

const skipInterfaceSource = `package mypkg
type SkipInterface interface {
	Method()
}`

var errWriteFailed = errors.New("write failed")

var errNotImplemented = errors.New("not implemented")

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	cwd         string
	files       map[string][]byte
	dirs        map[string][]os.DirEntry
	readDirErr  map[string]error
	readFileErr map[string]error
	writeHook   func(name string, data []byte) error
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		cwd:         appDir,
		files:       make(map[string][]byte),
		dirs:        make(map[string][]os.DirEntry),
		readDirErr:  make(map[string]error),
		readFileErr: make(map[string]error),
	}
}

func (m *MockFileSystem) Getwd() (string, error) {
	return m.cwd, nil
}

func (m *MockFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	if err, ok := m.readDirErr[name]; ok {
		return nil, err
	}

	if entries, ok := m.dirs[name]; ok {
		return entries, nil
	}

	return nil, os.ErrNotExist
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if err, ok := m.readFileErr[name]; ok {
		return nil, err
	}

	if content, ok := m.files[name]; ok {
		return content, nil
	}

	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFile(name string, data []byte, _ os.FileMode) error {
	if m.writeHook != nil {
		return m.writeHook(name, data)
	}

	m.files[name] = data

	return nil
}

// MockDirEntry implements os.DirEntry for testing.
type MockDirEntry struct {
	name  string
	isDir bool
}

func (m MockDirEntry) Name() string               { return m.name }
func (m MockDirEntry) IsDir() bool                { return m.isDir }
func (m MockDirEntry) Type() fs.FileMode          { return 0 }
func (m MockDirEntry) Info() (fs.FileInfo, error) { return nil, errNotImplemented }

func TestRun_Success(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockFS.cwd = appDir

	// Setup mock files
	sourceCode := `
package mypkg

type MyInterface interface {
	DoSomething()
}
`
	mockFS.files[appDir+"/source.go"] = []byte(sourceCode)
	mockFS.dirs[appDir] = []os.DirEntry{
		MockDirEntry{name: "source.go", isDir: false},
	}

	args := []string{"generator", "MyInterface", "--name", "MyImp"}
	env := func(key string) string {
		if key == "GOPACKAGE" {
			return pkgName
		}

		return ""
	}

	err := run.Run(args, env, mockFS)
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
	mockFS.cwd = appDir

	// Setup mock files with NO interface
	sourceCode := `
package mypkg

type MyStruct struct {}
`
	mockFS.files[appDir+"/source.go"] = []byte(sourceCode)
	mockFS.dirs[appDir] = []os.DirEntry{
		MockDirEntry{name: "source.go", isDir: false},
	}

	args := []string{"generator", "MyInterface"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS)
	if err == nil {
		t.Error("Expected error when interface is missing")
	}
}

func TestRun_WriteError(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockFS.cwd = appDir

	sourceCode := `
package mypkg

type MyInterface interface {
	DoSomething()
}
`
	mockFS.files[appDir+"/source.go"] = []byte(sourceCode)
	mockFS.dirs[appDir] = []os.DirEntry{
		MockDirEntry{name: "source.go", isDir: false},
	}

	// Fail on write
	mockFS.writeHook = func(_ string, _ []byte) error {
		return errWriteFailed
	}

	args := []string{"generator", "MyInterface"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS)
	if err == nil {
		t.Error("Expected error on write failure")
	}
}

func TestRun_ComplexInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockFS.cwd = appDir

	sourceCode := `
package mypkg

type ComplexInterface interface {
	Method1(a int, b string) (bool, error)
	Method2(fn func(int) int)
	Method3(a, b int)
	Method4() (x, y int)
}
`
	mockFS.files[appDir+"/source.go"] = []byte(sourceCode)
	mockFS.dirs[appDir] = []os.DirEntry{
		MockDirEntry{name: "source.go", isDir: false},
	}

	args := []string{"generator", "ComplexInterface", "--name", "ComplexImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS)
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
	mockFS.cwd = appDir

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
	mockFS.files[appDir+"/values.go"] = []byte(sourceCode)
	mockFS.dirs[appDir] = []os.DirEntry{
		MockDirEntry{name: "values.go", isDir: false},
	}

	args := []string{"generator", "ValueInterface", "--name", "ValueImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS)
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

func TestRun_ForeignInterface(t *testing.T) {
	t.Parallel()

	// This test requires actual disk access because packages.Load hits the disk/module system.
	// We can't mock the filesystem for packages.Load easily without internal changes.
	// However, we CAN test the 'getPackageAndMatchName' logic if we mock ReadDir/ReadFile enough
	// for a local file to resolve imports, OR we just trust packages.Load works and test the flow.

	// Let's rely on standard packages like "fmt" which are always available.

	mockFS := NewMockFileSystem()
	cwd, _ := os.Getwd() // We need a real CWD for packages.Load to work relative to something?
	// packages.Load behaves relative to current dir usually.

	// Issue: mockFS is only used for OUR file operations (ReadDir/ReadFile called by run.go),
	// but packages.Load uses the REAL OS filesystem.
	// If we provide "fmt.Stringer", getPackageAndMatchName parses the import from OUR local files first
	// to find the full import path if "fmt" was a local alias, but here "fmt" is the import path.

	mockFS.cwd = cwd

	// We create a dummy file that imports "fmt" so we can "resolve" it if needed,
	// but the code logic for "fmt.Stringer" in getPackageAndMatchName:
	// 1. Checks if "fmt.Stringer" has dot -> Yes.
	// 2. targetPkgImport = "fmt", matchName = "Stringer".
	// 3. parsePackageFiles(pkgDir, fs) -> scans local files to see imports.

	// So we need a local file that imports "fmt".
	sourceCode := `package mypkg
import "fmt"
var _ fmt.Stringer
`
	mockFS.files[cwd+"/dummy.go"] = []byte(sourceCode)

	// MockDirEntry needs to be robust
	mockFS.dirs[cwd] = []os.DirEntry{
		MockDirEntry{name: "dummy.go", isDir: false},
	}

	args := []string{"generator", "fmt.Stringer", "--name", "StringerImp"}
	env := func(_ string) string { return pkgName }

	// Run needs to:
	// 1. Parse local dummy.go (via mockFS)
	// 2. Find "fmt" import.
	// 3. Call parsePackageAST("fmt", ...) -> calls packages.Load("fmt") -> REAL FS access.
	// 4. Find Stringer interface in fmt package (Real AST).
	// 5. Generate code (MockFS write).

	err := run.Run(args, env, mockFS)
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

func TestRun_ParseFiles_ReadErrors(t *testing.T) {
	t.Parallel()

	// 1. ReadDir Error
	t.Run("ReadDir Error", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.readDirErr[appDir] = errCannotReadDir

		args := []string{"generator", "MyInterface", "--name", "MyImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err == nil {
			t.Error("Expected error from ReadDir, got nil")
		}
	})

	// 2. ReadFile Error - parsePackageFiles continues on error, so interface won't be found
	t.Run("ReadFile Error", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.dirs[appDir] = []os.DirEntry{
			MockDirEntry{name: "file.go", isDir: false},
		}
		mockFS.readFileErr[appDir+"/file.go"] = errCannotReadFile

		args := []string{"generator", "MyInterface", "--name", "MyImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err == nil {
			t.Error("Expected error finding interface (due to read fail), got nil")
		}
	})

	// 3. Parse Error (Invalid Go syntax)
	t.Run("Parse Error", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.dirs[appDir] = []os.DirEntry{
			MockDirEntry{name: "bad.go", isDir: false},
		}
		mockFS.files[appDir+"/bad.go"] = []byte("package bad\n func { invalid syntax }")

		args := []string{"generator", "MyInterface", "--name", "MyImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err == nil {
			t.Error("Expected error (interface not found), got nil")
		}
	})
}

func TestRun_ParseFiles_Filtering(t *testing.T) {
	t.Parallel()

	// 4. Directory entry (should be skipped)
	t.Run("Skip Directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		sourceCode := skipInterfaceSource
		mockFS.dirs[appDir] = []os.DirEntry{
			MockDirEntry{name: "subdir", isDir: true},
			MockDirEntry{name: "valid.go", isDir: false},
		}
		mockFS.files[appDir+"/valid.go"] = []byte(sourceCode)

		args := []string{"generator", "SkipInterface", "--name", "SkipImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err != nil {
			t.Errorf("Should skip directory and find interface, got error: %v", err)
		}
	})

	// 5. Non-.go file (should be skipped)
	t.Run("Skip Non-Go File", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		sourceCode := skipInterfaceSource
		mockFS.dirs[appDir] = []os.DirEntry{
			MockDirEntry{name: "readme.txt", isDir: false},
			MockDirEntry{name: "valid.go", isDir: false},
		}
		mockFS.files[appDir+"/valid.go"] = []byte(sourceCode)

		args := []string{"generator", "SkipInterface", "--name", "SkipImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err != nil {
			t.Errorf("Should skip non-.go file and find interface, got error: %v", err)
		}
	})
}

func TestRun_ParseFiles_GeneratedGo(t *testing.T) {
	t.Parallel()

	// 6. Skip generated.go file
	t.Run("Skip generated.go", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		sourceCode := skipInterfaceSource
		mockFS.dirs[appDir] = []os.DirEntry{
			MockDirEntry{name: "generated.go", isDir: false},
			MockDirEntry{name: "valid.go", isDir: false},
		}
		mockFS.files[appDir+"/generated.go"] = []byte("package mypkg\n// should be skipped")
		mockFS.files[appDir+"/valid.go"] = []byte(sourceCode)

		args := []string{"generator", "SkipInterface", "--name", "SkipImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err != nil {
			t.Errorf("Should skip generated.go and find interface, got error: %v", err)
		}
	})
}

func TestRun_EmbeddedInterface(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockFS.cwd = appDir

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
	mockFS.files[appDir+"/embedded.go"] = []byte(sourceCode)
	mockFS.dirs[appDir] = []os.DirEntry{
		MockDirEntry{name: "embedded.go", isDir: false},
	}

	args := []string{"generator", "EmbeddedInterface", "--name", "EmbeddedImp"}
	env := func(_ string) string { return pkgName }

	err := run.Run(args, env, mockFS)
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

func TestRun_ParseAST_Error(t *testing.T) {
	t.Parallel()

	// Test 1: Empty package path (dotted name with no matching import)
	t.Run("Empty Package Path", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		cwd, _ := os.Getwd()
		mockFS.cwd = cwd

		// Create a file with NO imports
		sourceCode := `package mypkg
`
		mockFS.files[cwd+"/test.go"] = []byte(sourceCode)
		mockFS.dirs[cwd] = []os.DirEntry{
			MockDirEntry{name: "test.go", isDir: false},
		}

		// Use "pkg.Interface" but there's no import for "pkg"
		// getPackageAndMatchName will return "" for package path
		// parsePackageAST will hit the pkgImportPath == "" branch
		args := []string{"generator", "pkg.Interface", "--name", "TestImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err == nil {
			t.Error("Expected error (interface not found), got nil")
		}
	})

	// Test 2: Invalid package that packages.Load will fail on
	t.Run("Packages Load Error", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		cwd, _ := os.Getwd()
		mockFS.cwd = cwd

		// Create a file with an import
		sourceCode := `package mypkg
import "invalid/package/path/that/does/not/exist/anywhere"
`
		mockFS.files[cwd+"/test.go"] = []byte(sourceCode)
		mockFS.dirs[cwd] = []os.DirEntry{
			MockDirEntry{name: "test.go", isDir: false},
		}

		// Use the imported package name
		args := []string{"generator", "anywhere.Interface", "--name", "TestImp"}
		env := func(_ string) string { return pkgName }

		err := run.Run(args, env, mockFS)
		if err == nil {
			t.Error("Expected error loading invalid package, got nil")
		}
	})
}
