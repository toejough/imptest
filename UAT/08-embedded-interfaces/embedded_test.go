package embedded_test

import (
	"io"
	"testing"

	embedded "github.com/toejough/imptest/UAT/08-embedded-interfaces"
	"github.com/toejough/imptest/imptest"
)

// TestEmbeddedInterfaceError demonstrates error handling with embedded interfaces.
func TestEmbeddedInterfaceError(t *testing.T) {
	t.Parallel()

	mock := MockReadCloser(t)

	go func() {
		_, _ = embedded.ProcessStream(mock.Interface())
	}()

	// Simulate a read error
	mock.Read.ExpectCalledWithMatches(imptest.Any()).InjectReturnValues(0, io.EOF)

	// Verify Close is still called (standard Go cleanup pattern)
	mock.Close.ExpectCalledWithExactly().InjectReturnValues(nil)
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

	mock := MockReadCloser(t)

	go func() {
		_, _ = embedded.ProcessStream(mock.Interface())
	}()

	// Read is embedded from io.Reader
	// Note: []byte is not comparable, so it uses reflect.DeepEqual automatically.
	mock.Read.ExpectCalledWithMatches(imptest.Any()).InjectReturnValues(5, nil)

	// Close is embedded from Closer (no args, so ExpectCalledWithExactly is called with no arguments)
	mock.Close.ExpectCalledWithExactly().InjectReturnValues(nil)
}
