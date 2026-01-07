package externalfuncs_test

//go:generate impgen http.HandlerFunc --target

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHttpHandlerFunc tests wrapping an external function type from the standard library.
//
// This UAT verifies that impgen can generate wrappers for external function types
// like http.HandlerFunc, which is defined in the standard library as:
//
//	type HandlerFunc func(ResponseWriter, *Request)
//
// This provides coverage for the findExternalTypeAlias function in codegen_common.go.
func TestHttpHandlerFunc(t *testing.T) {
	t.Parallel()

	// Create a test handler function
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	}

	// Wrap the handler for testing
	wrapper := WrapHandlerFunc(t, handler)

	// Create test request and recorder
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	recorder := httptest.NewRecorder()

	// Start the handler in a goroutine and verify completion
	wrapper.Start(recorder, req).ExpectCompletes()
}
