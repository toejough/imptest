package externalimports_test

//go:generate ../../bin/impgen FileHandler

import (
	"testing"
)

func TestExternalTypesCompile(t *testing.T) {
	// If this compiles, the imports are correct
	_ = NewFileHandlerImp(t)
}
