// Package named demonstrates mocking interfaces and functions with named
// parameters and return values.
package named

import (
	"context"
	"errors"
	"fmt"
)

// Exported variables.
var (
	ErrDivisionByZero = errors.New("division by zero")
	ErrNotFound       = errors.New("user not found")
)

// Calculator demonstrates a target function with named returns.
type Calculator struct{}

// Divide demonstrates named return values in a function.
// Named returns (quotient, remainder, err) make the API clearer.
func (c Calculator) Divide(dividend, divisor int) (quotient, remainder int, err error) {
	if divisor == 0 {
		return 0, 0, ErrDivisionByZero
	}

	return dividend / divisor, dividend % divisor, nil
}

// User represents a user entity.
type User struct {
	ID   int
	Name string
}

// UserRepository demonstrates named parameters and named return values.
// This is a common Go pattern for improving code readability.
type UserRepository interface {
	// GetUser demonstrates named parameters (ctx, userID) and named returns (user, err).
	// Named returns make the function signature more self-documenting.
	GetUser(ctx context.Context, userID int) (user User, err error)

	// SaveUser demonstrates mixed named parameters.
	SaveUser(ctx context.Context, user User) (savedUser User, err error)

	// DeleteUser demonstrates named parameters with single named return.
	DeleteUser(ctx context.Context, userID int) (err error)

	// CountUsers demonstrates named parameter with named return value.
	CountUsers(ctx context.Context) (count int, err error)
}

// ProcessUser demonstrates a standalone function with named parameters and returns.
func ProcessUser(ctx context.Context, userID int, repo UserRepository) (user User, err error) {
	user, err = repo.GetUser(ctx, userID)
	if err != nil {
		return User{}, fmt.Errorf("failed to get user %d: %w", userID, err)
	}

	return user, nil
}
