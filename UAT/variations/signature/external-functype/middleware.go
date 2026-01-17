// Package middleware demonstrates mocking interfaces with external function types.
package middleware

import "net/http"

type HTTPMiddleware interface {
	// Wrap takes an http.HandlerFunc and returns a wrapped version.
	Wrap(handler http.HandlerFunc) http.HandlerFunc
}
