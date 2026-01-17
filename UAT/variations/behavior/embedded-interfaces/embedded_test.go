package embedded_test

import (
	"io"
	"testing"

	"github.com/toejough/imptest/match"
	embedded "github.com/toejough/imptest/UAT/variations/behavior/embedded-interfaces"
)

// TestEmbeddedInterfaceError demonstrates error handling with embedded interfaces.
func TestEmbeddedInterfaceError(t *testing.T) {
	t.Parallel()

	mock, imp := MockReadCloser(t)

	go func() {
		_, _ = embedded.ProcessStream(mock)
	}()

	// Simulate a read error
	imp.Read.ArgsShould(match.BeAny).Return(0, io.EOF)

	// Verify Close is still called (standard Go cleanup pattern)
	imp.Close.Called().Return(nil)
}

//go:generate impgen embedded.ReadCloser --dependency

// TestEmbeddedInterfaces demonstrates how imptest automatically expands
// embedded interfaces.
//
// Key Requirements Met:
//  1. Automatic Expansion: Methods from embedded interfaces (like io.Reader
//     and io.Closer) are automatically discovered and included in the mock.
//  2. Transitive Mocking: Verify interactions with deep interface hierarchies
//     without manual boilerplate.
func TestEmbeddedInterfaces(t *testing.T) {
	t.Parallel()

	mock, imp := MockReadCloser(t)

	go func() {
		_, _ = embedded.ProcessStream(mock)
	}()

	// Read is embedded from io.Reader
	// Note: []byte is not comparable, so it uses reflect.DeepEqual automatically.
	imp.Read.ArgsShould(match.BeAny).Return(5, nil)

	// Close is embedded from Closer (no args, so Called is called)
	imp.Close.Called().Return(nil)
}
