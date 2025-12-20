package embedded_test

import (
	"io"
	"testing"

	embedded "github.com/toejough/imptest/UAT/08-embedded-interfaces"
	"github.com/toejough/imptest/imptest"
)

//go:generate go run ../../impgen/main.go embedded.ReadCloser --name ReadCloserImp

func TestEmbeddedInterfaces(t *testing.T) {
	t.Parallel()

	mock := NewReadCloserImp(t)

	go func() {
		_, _ = embedded.ProcessStream(mock.Mock)
	}()

	// Read is embedded from io.Reader
	// Note: []byte is not comparable, so it uses reflect.DeepEqual automatically.
	mock.ExpectCallIs.Read().ExpectArgsShould(imptest.Any()).InjectResults(5, nil)

	// Close is embedded from Closer
	mock.ExpectCallIs.Close().ExpectArgsAre().InjectResult(nil)
}

// TODO: not sure this test adds much value beyond the above?

func TestEmbeddedInterfaceError(t *testing.T) {
	t.Parallel()

	mock := NewReadCloserImp(t)

	go func() {
		_, _ = embedded.ProcessStream(mock.Mock)
	}()

	// Simulate a read error
	mock.ExpectCallIs.Read().ExpectArgsShould(imptest.Any()).InjectResults(0, io.EOF)

	// Verify Close is still called
	mock.ExpectCallIs.Close().InjectResult(nil)
}
