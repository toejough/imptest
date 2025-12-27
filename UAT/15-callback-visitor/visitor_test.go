package visitor_test

//go:generate impgen visitor.TreeWalker
//go:generate impgen visitor.CountFiles

import (
	"io/fs"
	"testing"
	"time"

	visitor "github.com/toejough/imptest/UAT/15-callback-visitor"
	imptest "github.com/toejough/imptest/imptest"
)

//nolint:varnamelen // Standard Go testing convention
func TestCountFiles(t *testing.T) {
	t.Parallel()

	walker := NewTreeWalkerImp(t)

	// Start the code under test using callable wrapper
	callable := NewCountFilesImp(t, visitor.CountFiles).Start(walker.Mock, "/test")

	// Wait for the Walk call
	call := walker.Within(time.Second).ExpectCallIs.Walk().ExpectArgsShould("/test", imptest.Any())

	// Invoke the callback multiple times with test data
	// Expected API: call.InvokeFn(args...).ExpectReturned(result)
	call.InvokeFn("/test/a.txt", mockDirEntry{name: "a.txt", isDir: false}, nil).ExpectReturned(nil)
	call.InvokeFn("/test/b.txt", mockDirEntry{name: "b.txt", isDir: false}, nil).ExpectReturned(nil)
	call.InvokeFn("/test/dir", mockDirEntry{name: "dir", isDir: true}, nil).ExpectReturned(nil)

	// Finally, let Walk return
	call.InjectResult(nil)

	// Verify the results - should count only the 2 non-directory entries
	callable.ExpectReturnedValuesAre(2, nil)
}

func TestWalkWithNamedType(t *testing.T) {
	t.Parallel()

	walker := NewTreeWalkerImp(t)

	// Call WalkWithNamedType in a goroutine to trigger the mock
	go func() {
		_ = walker.Mock.WalkWithNamedType("/data", func(_ string, _ fs.DirEntry, _ error) error {
			return nil
		})
	}()

	// Wait for and verify the WalkWithNamedType call
	call := walker.Within(time.Second).ExpectCallIs.WalkWithNamedType().ExpectArgsShould("/data", imptest.Any())

	// Invoke the callback with the named type - should work just like inline function types
	call.InvokeFn("/data/file.txt", mockDirEntry{name: "file.txt", isDir: false}, nil).ExpectReturned(nil)

	// Let the method return
	call.InjectResult(nil)
}

// mockDirEntry is a simple fs.DirEntry implementation for testing.
type mockDirEntry struct {
	name  string
	isDir bool
}

//nolint:nilnil // Test mock implementation - error scenario not tested
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func (m mockDirEntry) IsDir() bool { return m.isDir }

func (m mockDirEntry) Name() string { return m.name }

func (m mockDirEntry) Type() fs.FileMode { return 0 }
