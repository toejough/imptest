package visitor_test

//go:generate ../../bin/impgen TreeWalker
//go:generate ../../bin/impgen CountFiles

import (
	"io/fs"
	"testing"
	"time"

	imptest "github.com/toejough/imptest/imptest"
	visitor "github.com/toejough/imptest/UAT/15-callback-visitor"
)

// mockDirEntry is a simple fs.DirEntry implementation for testing.
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode          { return 0 }
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

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
	callable.ExpectReturnedValues(2, nil)
}
