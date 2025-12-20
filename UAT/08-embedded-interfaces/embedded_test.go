package embedded_test

import (
	"io"
	"testing"

	embedded "github.com/toejough/imptest/UAT/08-embedded-interfaces"
	"github.com/toejough/imptest/imptest"
)

// TODO: there's a _lot_ of generated code. Is there any way to reduce how much is generated, vs calling out to an imp
// library for common things?

//go:generate go run ../../impgen/main.go embedded.ReadCloser --name ReadCloserImp

func TestEmbeddedInterfaces(t *testing.T) {
	t.Parallel()

	imp := NewReadCloserImp(t)

	go func() {
		_, _ = embedded.ProcessStream(imp.Mock)
	}()

	// Read is embedded from io.Reader
	// Note: []byte is not comparable, so it uses reflect.DeepEqual automatically.
	imp.ExpectCallIs.Read().ExpectArgsShould(imptest.Any()).InjectResults(5, nil)

	// Close is embedded from Closer
	imp.ExpectCallIs.Close().ExpectArgsAre().InjectResult(nil)
}

// TODO: not sure this test adds much value beyond the above?

func TestEmbeddedInterfaceError(t *testing.T) {
	t.Parallel()

	imp := NewReadCloserImp(t)

	go func() {
		_, _ = embedded.ProcessStream(imp.Mock)
	}()

	// Simulate a read error
	imp.ExpectCallIs.Read().ExpectArgsShould(imptest.Any()).InjectResults(0, io.EOF)

	// Verify Close is still called
	imp.ExpectCallIs.Close().InjectResult(nil)
}
