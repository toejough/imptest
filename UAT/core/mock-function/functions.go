// Package mockfunction contains example functions to demonstrate mocking
// package-level functions with --dependency flag.
package mockfunction

import (
	"context"
	"errors"
	"fmt"
)

// Exported variables.
var (
	ErrInputEmpty    = errors.New("input cannot be empty")
	ErrOrderNotFound = errors.New("order not found")
)

// Order represents a business entity.
type Order struct {
	ID     int
	Status string
	Total  float64
}

// FormatPrice is a pure function with no error return.
// This demonstrates mocking a function that doesn't return error.
func FormatPrice(amount float64, currency string) string {
	// In real code, this would format the price
	return fmt.Sprintf("%s%.2f", currency, amount)
}

// Notify is a function with no return values.
// This demonstrates mocking a void function.
func Notify(userID int, message string) {
	// In real code, this would send a notification
	_, _ = userID, message
}

// ProcessOrder is a package-level function that processes an order.
// This demonstrates mocking a function that has parameters and returns.
func ProcessOrder(ctx context.Context, orderID int) (*Order, error) {
	// In real code, this would fetch and process the order
	_ = ctx

	return &Order{ID: orderID, Status: "processed", Total: 0}, nil
}

// TransformData processes data with complex type signatures.
// This tests collection of imports from complex types (maps, slices, funcs).
func TransformData(items []*Order, lookup map[string]*Order, processor func(*Order) error) (*Order, error) {
	// In real code, this would transform the data
	_, _, _ = items, lookup, processor

	return nil, nil
}

// ValidateInput is a simple validation function.
// This demonstrates mocking a function with simpler signature.
func ValidateInput(input string) error {
	if input == "" {
		return ErrInputEmpty
	}

	return nil
}

// init prevents deadcode from removing the package-level functions.
// In real code, these functions would be called from production code.
//
//nolint:gochecknoinits // Required to prevent deadcode elimination in UAT
func init() {
	_ = []any{ProcessOrder, ValidateInput, FormatPrice, Notify, TransformData}
}
