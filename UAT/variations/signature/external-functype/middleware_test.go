package middleware_test

//go:generate impgen middleware.HTTPMiddleware --dependency

import (
	"net/http"
	"testing"

	"github.com/toejough/imptest"
)

// TestHTTPMiddleware tests using an interface with external function type parameters.
//
// This UAT verifies that impgen can generate mocks for interfaces that have
// methods with parameters using external function types like http.HandlerFunc.
//
// This provides coverage for the findExternalTypeAlias function in codegen_common.go.
func TestHTTPMiddleware(t *testing.T) {
	t.Parallel()

	// Create the mock
	mock := MockHTTPMiddleware(t)

	// Create a simple handler
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("original"))
	}

	// Create a wrapped handler to return (explicitly typed as http.HandlerFunc)
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("wrapped"))
	})

	// Call the mock in a goroutine
	done := make(chan http.HandlerFunc)

	go func() {
		result := mock.Mock.Wrap(handler)
		done <- result
	}()

	// Expect the Wrap call and inject the wrapped handler
	call := mock.Method.Wrap.Eventually.ExpectCalledWithMatches(imptest.Any())
	call.InjectReturnValues(wrappedHandler)

	// Wait for the result
	result := <-done

	// Verify we got a non-nil handler back
	if result == nil {
		t.Fatal("Expected non-nil handler")
	}
}
