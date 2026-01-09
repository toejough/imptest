package crossfile_test

//go:generate impgen crossfile.FileSystem --dependency

import (
	"testing"
)

func TestCrossFileImportsCompile(t *testing.T) {
	t.Parallel()
	// If this compiles, the imports are correctly resolved from types.go
	mock := MockFileSystem(t)
	_ = mock.Mock
}
