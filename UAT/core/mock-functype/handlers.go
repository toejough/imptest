package handlers

import (
	"context"
	"net/http"
)

// Callback is a function type for async completion handlers.
// Function types with context are common in concurrent Go code.
type Callback func(ctx context.Context, err error)

// HandlerFunc is a function type mimicking http.HandlerFunc.
// This is a common pattern where a function signature is given a name
// and can be used as a type in interfaces and function parameters.
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// Middleware is a function type that wraps a HandlerFunc to add behavior.
// This demonstrates function types that return other function types.
type Middleware func(next HandlerFunc) HandlerFunc

// Router demonstrates an interface that accepts and returns function types.
// This tests that impgen can handle function types as both parameters and return values.
type Router interface {
	// RegisterHandler accepts a function type as a parameter.
	// This is similar to http.HandleFunc(path string, handler func(...)).
	RegisterHandler(path string, handler HandlerFunc)

	// GetHandler returns a function type.
	// Tests that function types can be returned from mocked methods.
	GetHandler(path string) HandlerFunc

	// ApplyMiddleware accepts a function type that returns a function type.
	// This tests nested function type handling.
	ApplyMiddleware(middleware Middleware) HandlerFunc

	// ValidateWith accepts a validator function type and data.
	// Tests multiple parameters where one is a function type.
	ValidateWith(validator Validator, data string) error

	// OnComplete registers a callback function type.
	// Tests function types with multiple parameters including context.
	OnComplete(callback Callback)
}

// Server demonstrates a struct with methods that use function types.
type Server struct{}

// HandleRequest accepts a HandlerFunc function type.
// This can be wrapped with --target flag.

// Validator is a function type for validating request data.
// Simple function types like this are common for callback patterns.
type Validator func(data string) error
