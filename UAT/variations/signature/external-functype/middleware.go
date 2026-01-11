// Package middleware demonstrates mocking interfaces with external function types.
package middleware

import "net/http"

// HTTPMiddleware wraps HTTP handlers with middleware functionality.
// This interface demonstrates using external function types (http.HandlerFunc) as parameters.
type HTTPMiddleware interface {
	// Wrap takes an http.HandlerFunc and returns a wrapped version.
	Wrap(handler http.HandlerFunc) http.HandlerFunc
}
