package externalimports_test

//go:generate impgen externalimports.FileHandler --dependency

import (
	"testing"
)

func TestExternalTypesCompile(t *testing.T) {
	t.Parallel()
	// If this compiles, the imports are correct - v2 API test
	mock := MockFileHandler(t)
	_ = mock.Mock
}
