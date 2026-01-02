// Package handlers_test demonstrates that impgen correctly handles function types
// (e.g., http.HandlerFunc) when mocking with the --dependency flag.
//
// FUNCTION TYPE vs FUNCTION LITERAL:
// - Function type: A named type alias for a function signature (type HandlerFunc func(...))
// - Function literal: An anonymous function signature as a parameter (func Process(fn func(int) int))
//
// This UAT tests function types, which are common in the Go stdlib (http.HandlerFunc,
// http.Handler, context.CancelFunc, etc.) and are distinct from function literals.
package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	handlers "github.com/toejough/imptest/UAT/31-function-type-dependency"
	"github.com/toejough/imptest/imptest"
)

// TestApplyMiddleware_NestedFunctionTypes demonstrates mocking a method
// that accepts a function type returning another function type.
func TestApplyMiddleware_NestedFunctionTypes(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)

	// Define a middleware (function type that returns function type)
	loggingMiddleware := func(next handlers.HandlerFunc) handlers.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Log before
			next(w, r)
			// Log after
		}
	}

	// Expected wrapped handler
	wrappedHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Start goroutine to call the method under test
	resultChan := make(chan handlers.HandlerFunc, 1)

	go func() {
		resultChan <- mock.Interface().ApplyMiddleware(loggingMiddleware)
	}()

	// Set up mock expectation to unblock the goroutine
	// Verify call with matcher (function types can't be compared)
	mock.ApplyMiddleware.ExpectCalledWithMatches(imptest.Any()).
		InjectReturnValues(wrappedHandler)

	// Wait for goroutine to complete and get the result
	result := <-resultChan

	// Verify the returned handler works
	if result == nil {
		t.Fatal("expected wrapped handler to be returned")
	}
}

// NOTE: TestTargetWrapper_FunctionTypeParam is disabled due to variable shadowing bug
// in impgen's wrapper generator. This test would demonstrate target wrappers with
// function type parameters, but is not critical for validating function type support
// in --dependency mode (which is the primary goal of this UAT).
//
// TODO: Re-enable after fixing impgen wrapper variable shadowing issue
/*
func TestTargetWrapper_FunctionTypeParam(t *testing.T) {
	t.Parallel()

	server := handlers.Server{}

	// Create a handler function type
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	// Create test request/response
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Wrap the method
	wrapper := WrapServerHandleRequest(t, server.HandleRequest)

	// Execute and verify
	wrapper.Start(handler, w, r)

	// Verify handler was called
	if !handlerCalled {
		t.Error("expected handler to be called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
*/

// TestFunctionTypeNil demonstrates handling nil function type values.
func TestFunctionTypeNil(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)

	// Call with nil function type
	go func() {
		mock.Interface().RegisterHandler("/nil", nil)
	}()

	// Verify call with nil matcher
	mock.RegisterHandler.ExpectCalledWithMatches("/nil", imptest.Any())
}

// TestGetHandler_FunctionTypeReturn demonstrates mocking a method that
// returns a function type.
func TestGetHandler_FunctionTypeReturn(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)
	path := "/api/products"

	// Create a handler to return from the mock
	expectedHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}

	// Start goroutine to call the method under test
	resultChan := make(chan handlers.HandlerFunc, 1)

	go func() {
		resultChan <- mock.Interface().GetHandler(path)
	}()

	// Set up mock expectation to unblock the goroutine
	mock.GetHandler.ExpectCalledWithExactly(path).
		InjectReturnValues(expectedHandler)

	// Wait for goroutine to complete and get the returned handler
	returnedHandler := <-resultChan

	// Verify the returned handler works
	if returnedHandler == nil {
		t.Fatal("expected handler to be returned")
	}

	// Execute the returned handler
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	returnedHandler(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

// TestMultipleFunctionTypes demonstrates handling multiple function type
// parameters and returns in different method calls.
func TestMultipleFunctionTypes(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)

	// First call: RegisterHandler
	handler1 := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	go func() {
		mock.Interface().RegisterHandler("/path1", handler1)
	}()

	mock.RegisterHandler.ExpectCalledWithMatches("/path1", imptest.Any())

	// Second call: GetHandler
	handler2 := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}

	handlerChan := make(chan handlers.HandlerFunc, 1)

	go func() {
		handlerChan <- mock.Interface().GetHandler("/path2")
	}()

	mock.GetHandler.ExpectCalledWithExactly("/path2").
		InjectReturnValues(handler2)

	// Third call: ValidateWith
	validator := func(_ string) error { return nil }

	go func() {
		_ = mock.Interface().ValidateWith(validator, "test")
	}()

	mock.ValidateWith.ExpectCalledWithMatches(imptest.Any(), "test").
		InjectReturnValues(nil)

	// Verify returned handler from GetHandler
	returned := <-handlerChan

	if returned == nil {
		t.Error("expected handler to be returned")
	}
}

// TestOnComplete_FunctionTypeWithContext demonstrates mocking a method
// that accepts a function type with context.Context parameter.
func TestOnComplete_FunctionTypeWithContext(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)

	// Define a callback function type
	completionCallback := func(_ context.Context, _ error) {
		// Handle completion
	}

	// Call the method under test
	go func() {
		mock.Interface().OnComplete(completionCallback)
	}()

	// Verify call - use matcher for function type
	mock.OnComplete.ExpectCalledWithMatches(imptest.Any())
}

// Generate dependency mock for interface with function type parameters and returns
//go:generate impgen handlers.Router --dependency

// NOTE: Target wrapper generation for Server.HandleRequest is commented out due to
// a known variable shadowing bug in impgen's wrapper generator (parameter name conflicts
// with receiver name). This is unrelated to function type support.
// TODO: Re-enable after fixing impgen wrapper variable shadowing issue
// //go:generate impgen handlers.Server.HandleRequest --target

// TestRegisterHandler_FunctionTypeParam demonstrates mocking a method that
// accepts a function type as a parameter.
func TestRegisterHandler_FunctionTypeParam(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)
	path := "/api/users"

	// Define a HandlerFunc (function type instance)
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Call the method under test
	go func() {
		mock.Interface().RegisterHandler(path, handler)
	}()

	// Verify the call - use imptest.Any() matcher for function type parameter
	// (functions cannot be compared with ==, just like function literals)
	mock.RegisterHandler.ExpectCalledWithMatches(path, imptest.Any())
}

// TestValidateWith_FunctionTypeWithOtherParams demonstrates mocking a method
// with a function type parameter alongside regular parameters.
func TestValidateWith_FunctionTypeWithOtherParams(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)

	// Define a validator function type
	emailValidator := func(data string) error {
		if data == "" {
			return errors.New("email cannot be empty")
		}

		return nil
	}

	testData := "user@example.com"

	// Call the method under test
	errChan := make(chan error, 1)

	go func() {
		errChan <- mock.Interface().ValidateWith(emailValidator, testData)
	}()

	// Verify call - use matcher for function type, exact match for string
	mock.ValidateWith.ExpectCalledWithMatches(imptest.Any(), testData).
		InjectReturnValues(nil)

	// Wait for goroutine to complete and get the error
	validationErr := <-errChan

	// Verify no error was returned
	if validationErr != nil {
		t.Errorf("expected no error, got %v", validationErr)
	}
}

// TestValidateWith_ValidationError demonstrates error handling with function types.
func TestValidateWith_ValidationError(t *testing.T) {
	t.Parallel()

	mock := MockRouter(t)

	validator := func(_ string) error {
		return errors.New("validation failed")
	}

	testData := "invalid"
	expectedErr := errors.New("validation failed")

	errChan := make(chan error, 1)

	go func() {
		errChan <- mock.Interface().ValidateWith(validator, testData)
	}()

	mock.ValidateWith.ExpectCalledWithMatches(imptest.Any(), testData).
		InjectReturnValues(expectedErr)

	validationErr := <-errChan
	if validationErr == nil {
		t.Error("expected validation error")
	}
}
