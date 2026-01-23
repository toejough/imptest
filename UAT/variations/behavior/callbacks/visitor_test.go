package visitor_test

//go:generate impgen visitor.TreeWalker --dependency
//go:generate impgen visitor.CountFiles --target
//go:generate impgen visitor.WalkFunc --target

import (
	"io/fs"
	"testing"

	visitor "github.com/toejough/imptest/UAT/variations/behavior/callbacks"
	. "github.com/toejough/imptest/match" //nolint:revive // Dot import for matcher DSL
)

func TestCallbackMatcherSupport(t *testing.T) {
	t.Parallel()

	// V2 Pattern: Create mock and start the function under test
	mock, imp := MockTreeWalker(t)

	// Start the function under test in a goroutine
	wrapperCall := StartCountFiles(t, visitor.CountFiles, mock, "/test")

	// V2 Pattern: Wait for the Walk call without Eventually to see if that's the issue
	call := imp.Walk.ArgsShould("/test", BeAny)

	// V2 Pattern: Extract the callback from args using the typed wrapper
	args := call.GetArgs()
	callback := args.Fn

	// V2 Pattern: Invoke the callback directly with test data
	// This demonstrates that we've successfully extracted and can invoke the callback
	err := callback("/test/file.txt", mockDirEntry{name: "file.txt", isDir: false}, nil)
	if err != nil {
		t.Errorf("Expected callback to return nil, got %v", err)
	}

	// Inject return value to let Walk complete
	call.Return(nil)

	// Verify the function completed successfully
	wrapperCall.ReturnsEqual(1, nil)
}

func TestCallbackPanicSupport(t *testing.T) {
	t.Parallel()

	// V2 Pattern: Create mock
	mock, imp := MockTreeWalker(t)

	// Start goroutine that will pass a panicking callback to the mock
	go func() {
		_ = mock.Walk("/test", func(_ string, _ fs.DirEntry, _ error) error {
			panic("test panic") // Callback panics on its own
		})
	}()

	// V2 Pattern: Wait for the Walk call
	call := imp.Eventually.Walk.ArgsShould("/test", BeAny)

	// V2 Pattern: Extract callback and invoke it, catching the panic
	rawArgs := call.RawArgs()

	callback, ok := rawArgs[1].(func(string, fs.DirEntry, error) error)
	if !ok {
		t.Fatal("Expected callback to be a function")
	}

	// Invoke callback and verify it panics with expected value
	var panicValue any

	func() {
		defer func() {
			panicValue = recover()
		}()

		_ = callback("/test/file.txt", mockDirEntry{name: "file.txt", isDir: false}, nil)
	}()

	if panicValue != "test panic" {
		t.Errorf("Expected callback to panic with 'test panic', got %v", panicValue)
	}

	// Let Walk return
	call.Return(nil)
}

func TestCountFiles(t *testing.T) {
	t.Parallel()

	// V2 Pattern: Create mock and start function under test
	mock, imp := MockTreeWalker(t)

	// Start the code under test
	wrapperCall := StartCountFiles(t, visitor.CountFiles, mock, "/test")

	// V2 Pattern: Wait for the Walk call
	call := imp.Eventually.Walk.ArgsShould("/test", BeAny)

	// V2 Pattern: Extract the callback from args
	// When using Eventually(), we get the base DependencyCall, so we use RawArgs()
	rawArgs := call.RawArgs()

	callback, ok := rawArgs[1].(func(string, fs.DirEntry, error) error)
	if !ok {
		t.Fatal("Expected callback to be a function")
	}

	// V2 Pattern: Invoke the callback multiple times with test data
	// Verify each invocation returns the expected value
	err := callback("/test/a.txt", mockDirEntry{name: "a.txt", isDir: false}, nil)
	if err != nil {
		t.Errorf("Expected callback to return nil for a.txt, got %v", err)
	}

	err = callback("/test/b.txt", mockDirEntry{name: "b.txt", isDir: false}, nil)
	if err != nil {
		t.Errorf("Expected callback to return nil for b.txt, got %v", err)
	}

	err = callback("/test/dir", mockDirEntry{name: "dir", isDir: true}, nil)
	if err != nil {
		t.Errorf("Expected callback to return nil for dir, got %v", err)
	}

	// Finally, let Walk return
	call.Return(nil)

	// Verify the results - should count only the 2 non-directory entries
	wrapperCall.ReturnsEqual(2, nil)
}

func TestWalkWithNamedType(t *testing.T) {
	t.Parallel()

	// V2 Pattern: Create mock
	mock, imp := MockTreeWalker(t)

	// Call WalkWithNamedType in a goroutine to trigger the mock
	go func() {
		_ = mock.WalkWithNamedType("/data", func(_ string, _ fs.DirEntry, _ error) error {
			return nil
		})
	}()

	// V2 Pattern: Wait for and verify the WalkWithNamedType call
	call := imp.Eventually.WalkWithNamedType.ArgsShould("/data", BeAny)

	// V2 Pattern: Extract callback from args
	// When using Eventually(), we get the base DependencyCall, so we use RawArgs()
	rawArgs := call.RawArgs()

	callback, ok := rawArgs[1].(visitor.WalkFunc)
	if !ok {
		t.Fatal("Expected callback to be a visitor.WalkFunc")
	}

	// Invoke the callback with the named type - should work just like inline function types
	err := callback("/data/file.txt", mockDirEntry{name: "file.txt", isDir: false}, nil)
	if err != nil {
		t.Errorf("Expected callback to return nil, got %v", err)
	}

	// Let the method return
	call.Return(nil)
}

type mockDirEntry struct {
	name  string
	isDir bool
}

//nolint:nilnil // Test mock implementation - error scenario not tested
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func (m mockDirEntry) IsDir() bool { return m.isDir }

func (m mockDirEntry) Name() string { return m.name }

func (m mockDirEntry) Type() fs.FileMode { return 0 }
