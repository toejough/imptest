package crossfile_test

//go:generate impgen crossfile.FileSystem --dependency

import (
	"testing"

	// Import for impgen to resolve the package.
	_ "github.com/toejough/imptest/UAT/variations/signature/cross-file-external"
)

func TestCrossFileImportsCompile(t *testing.T) {
	t.Parallel()
	// If this compiles, the imports are correctly resolved from types.go
	mock, _ := MockFileSystem(t)
	_ = mock
}
