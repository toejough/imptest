// Package named_test demonstrates that impgen correctly handles named parameters
// and named return values, which are common Go patterns for improving readability.
package named_test

import (
	"context"
	"testing"

	named "github.com/toejough/imptest/UAT/variations/signature/named-params"
)

// Generate dependency mock for interface with named params/returns
//go:generate impgen named.UserRepository --dependency

// Generate target wrapper for method with named returns
//go:generate impgen named.Calculator.Divide --target

// Generate target wrapper for function with named params/returns
//go:generate impgen named.ProcessUser --target

// TestDependencyWithNamedParams demonstrates that dependency mocks work correctly
// with interfaces that have named parameters and named return values.
func TestDependencyWithNamedParams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mock := MockUserRepository(t)

	// Run code under test
	go func() {
		user, err := mock.Mock.GetUser(ctx, 123)
		_ = user
		_ = err
	}()

	// Verify mock handles named parameters and returns correctly
	mock.Method.GetUser.ExpectCalledWithExactly(ctx, 123).
		InjectReturnValues(named.User{ID: 123, Name: "Alice"}, nil)
}

// TestFunctionWithNamedParams demonstrates that function wrappers handle
// named parameters and returns correctly.
func TestFunctionWithNamedParams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := MockUserRepository(t)

	// Wrap the ProcessUser function for testing
	wrapper := WrapProcessUser(t, named.ProcessUser)

	// Start the wrapped function
	call := wrapper.Method.Start(ctx, 456, mockRepo.Mock)

	// Handle the repository call
	mockRepo.Method.GetUser.ExpectCalledWithExactly(ctx, 456).
		InjectReturnValues(named.User{ID: 456, Name: "Bob"}, nil)

	// Verify the wrapper received correct return values
	call.ExpectReturnsEqual(named.User{ID: 456, Name: "Bob"}, nil)
}

// TestMultipleMethods demonstrates that mocks handle multiple methods
// with different named parameter/return combinations.
func TestMultipleMethods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mock := MockUserRepository(t)

	go func() {
		// Test SaveUser (named params + named returns)
		savedUser, err := mock.Mock.SaveUser(ctx, named.User{ID: 789, Name: "Charlie"})
		_ = savedUser
		_ = err

		// Test DeleteUser (named params + single named return)
		err = mock.Mock.DeleteUser(ctx, 789)
		_ = err

		// Test CountUsers (named param + named returns)
		count, err := mock.Mock.CountUsers(ctx)
		_ = count
		_ = err
	}()

	// Handle SaveUser
	mock.Method.SaveUser.ExpectCalledWithExactly(ctx, named.User{ID: 789, Name: "Charlie"}).
		InjectReturnValues(named.User{ID: 789, Name: "Charlie"}, nil)

	// Handle DeleteUser
	mock.Method.DeleteUser.ExpectCalledWithExactly(ctx, 789).
		InjectReturnValues(nil)

	// Handle CountUsers
	mock.Method.CountUsers.ExpectCalledWithExactly(ctx).
		InjectReturnValues(3, nil)
}

// TestTargetWithNamedReturns demonstrates that target wrappers work correctly
// with methods that have named return values.
func TestTargetWithNamedReturns(t *testing.T) {
	t.Parallel()

	// Create calculator instance
	calc := named.Calculator{}

	// Wrap the Divide method for testing
	wrapper := WrapCalculatorDivide(t, calc.Divide)

	// Execute and verify named returns (quotient, remainder, err)
	wrapper.Method.Start(10, 3).ExpectReturnsEqual(3, 1, nil)
}
