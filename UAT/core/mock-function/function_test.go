// Package mockfunction_test demonstrates mocking package-level functions
// using the --dependency flag.
//
// This UAT tests Issue #43: Function as dependency (mock a function by signature)
//
// The key insight is that a package-level function like:
//
//	func ProcessOrder(ctx context.Context, orderID int) (*Order, error)
//
// can be mocked by extracting its signature and generating a mock similar to
// how we mock interfaces. The mock provides a Func() method that returns a
// function with the same signature.
package mockfunction_test

import (
	"context"
	"errors"
	"testing"

	mockfunction "github.com/toejough/imptest/UAT/core/mock-function"
	"github.com/toejough/imptest/imptest"
)

// TestMockFunction_Eventually demonstrates using Eventually for concurrent calls.
func TestMockFunction_Eventually(t *testing.T) {
	t.Parallel()

	mock := MockValidateInput(t)

	// Start multiple goroutines calling the mock
	go func() { _ = mock.Func()("input1") }()
	go func() { _ = mock.Func()("input2") }()

	// Match calls in any order using Eventually
	mock.Eventually().ExpectCalledWithExactly("input2").InjectReturnValues(nil)
	mock.Eventually().ExpectCalledWithExactly("input1").InjectReturnValues(nil)
}

// TestMockFunction_GetArgs demonstrates accessing typed arguments.
func TestMockFunction_GetArgs(t *testing.T) {
	t.Parallel()

	mock := MockFormatPrice(t)

	go func() {
		_ = mock.Func()(123.45, "EUR")
	}()

	call := mock.ExpectCalledWithMatches(imptest.Any(), imptest.Any())

	// Access typed arguments
	args := call.GetArgs()
	if args.Amount != 123.45 {
		t.Errorf("expected amount 123.45, got %v", args.Amount)
	}

	if args.Currency != "EUR" {
		t.Errorf("expected currency EUR, got %v", args.Currency)
	}

	call.InjectReturnValues("€123.45")
}

// TestMockFunction_NoError demonstrates mocking a function without error return.
func TestMockFunction_NoError(t *testing.T) {
	t.Parallel()

	mock := MockFormatPrice(t)

	amount := 99.99
	currency := "USD"
	expected := "$99.99"

	resultChan := make(chan string, 1)

	go func() {
		resultChan <- mock.Func()(amount, currency)
	}()

	mock.ExpectCalledWithExactly(amount, currency).InjectReturnValues(expected)

	result := <-resultChan
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestMockFunction_NoReturns demonstrates mocking a void function.
func TestMockFunction_NoReturns(t *testing.T) {
	t.Parallel()

	mock := MockNotify(t)

	userID := 42
	message := "Hello!"

	done := make(chan struct{}, 1)

	go func() {
		mock.Func()(userID, message)

		done <- struct{}{}
	}()

	// For void functions, still need to call InjectReturnValues to unblock the mock
	mock.ExpectCalledWithExactly(userID, message).InjectReturnValues()

	<-done
}

// TestMockFunction_Simple demonstrates mocking a simpler function.
func TestMockFunction_Simple(t *testing.T) {
	t.Parallel()

	mock := MockValidateInput(t)

	testData := "valid input"

	resultChan := make(chan error, 1)

	go func() {
		resultChan <- mock.Func()(testData)
	}()

	mock.ExpectCalledWithExactly(testData).InjectReturnValues(nil)

	err := <-resultChan
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestMockFunction_WithError demonstrates mocking a function that returns an error.
func TestMockFunction_WithError(t *testing.T) {
	t.Parallel()

	mock := MockProcessOrder(t)

	ctx := context.Background()
	orderID := 999
	expectedErr := mockfunction.ErrOrderNotFound

	resultChan := make(chan error, 1)

	go func() {
		_, err := mock.Func()(ctx, orderID)
		resultChan <- err
	}()

	mock.ExpectCalledWithExactly(ctx, orderID).InjectReturnValues(nil, expectedErr)

	err := <-resultChan
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestMockFunction_WithMatchers demonstrates using matchers for flexible matching.
func TestMockFunction_WithMatchers(t *testing.T) {
	t.Parallel()

	mock := MockProcessOrder(t)

	ctx := context.Background()
	orderID := 456
	expectedOrder := &mockfunction.Order{ID: orderID, Status: "pending", Total: 50.00}

	resultChan := make(chan *mockfunction.Order, 1)

	go func() {
		order, _ := mock.Func()(ctx, orderID)
		resultChan <- order
	}()

	// Use matchers instead of exact values
	mock.ExpectCalledWithMatches(imptest.Any(), imptest.Any()).InjectReturnValues(expectedOrder, nil)

	order := <-resultChan
	if order.ID != expectedOrder.ID {
		t.Errorf("expected order ID %d, got %d", expectedOrder.ID, order.ID)
	}
}

// Generate mocks for package-level functions
//go:generate impgen mockfunction.ProcessOrder --dependency
//go:generate impgen mockfunction.ValidateInput --dependency
//go:generate impgen mockfunction.FormatPrice --dependency
//go:generate impgen mockfunction.Notify --dependency

// TestMockFunction_WithReturns demonstrates mocking a function that returns values.
func TestMockFunction_WithReturns(t *testing.T) {
	t.Parallel()

	mock := MockProcessOrder(t)

	ctx := context.Background()
	orderID := 123
	expectedOrder := &mockfunction.Order{ID: orderID, Status: "completed", Total: 99.99}

	// Start goroutine that calls the mock function
	resultChan := make(chan struct {
		order *mockfunction.Order
		err   error
	}, 1)

	go func() {
		// mock.Func() returns a function with the same signature as ProcessOrder
		order, err := mock.Func()(ctx, orderID)
		resultChan <- struct {
			order *mockfunction.Order
			err   error
		}{order, err}
	}()

	// Set up expectation and inject return values
	mock.ExpectCalledWithExactly(ctx, orderID).InjectReturnValues(expectedOrder, nil)

	// Verify results
	result := <-resultChan
	if result.err != nil {
		t.Errorf("expected no error, got %v", result.err)
	}

	if result.order.ID != expectedOrder.ID {
		t.Errorf("expected order ID %d, got %d", expectedOrder.ID, result.order.ID)
	}
}
