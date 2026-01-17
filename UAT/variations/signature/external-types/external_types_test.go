package externalimports_test

//go:generate impgen externalimports.FileHandler --dependency

import (
	"testing"

	// Import for impgen to resolve the package.
	_ "github.com/toejough/imptest/UAT/variations/signature/external-types"
)

func TestExternalTypesCompile(t *testing.T) {
	t.Parallel()
	// If this compiles, the imports are correct - v2 API test
	mock, _ := MockFileHandler(t)
	_ = mock
}
