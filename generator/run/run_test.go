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

var errWriteFailed = errors.New("write failed")

var errNotImplemented = errors.New("not implemented")

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	cwd       string
	files     map[string][]byte
	dirs      map[string][]os.DirEntry
	writeHook func(name string, data []byte) error
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		cwd:   appDir,
		files: make(map[string][]byte),
		dirs:  make(map[string][]os.DirEntry),
	}
}

func (m *MockFileSystem) Getwd() (string, error) {
	return m.cwd, nil
}

func (m *MockFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	if entries, ok := m.dirs[name]; ok {
		return entries, nil
	}

	return nil, os.ErrNotExist
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
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
