package targetfunction_test

import (
	"testing"

	imptest "github.com/toejough/imptest/imptest/v2"
)

// Add adds two integers and returns the result
func Add(a, b int) int {
	return a + b
}

// TestTargetFunction_Ordered_Exact_Returns demonstrates wrapping a function
// with ordered expectations and exact value matching for returns
func TestTargetFunction_Ordered_Exact_Returns(t *testing.T) {
	// Setup: Create central coordinator
	imp := imptest.NewImp(t)

	// Create target wrapper for the function under test
	target := imptest.NewTargetFunction(imp, Add)

	// Execute: Call the function with specific arguments
	call := target.CallWith(2, 3)

	// Verify: Expect exact return value (ordered - expects next interaction)
	call.ExpectReturnsEqual(5)
}

// TestTargetFunction_Ordered_Matcher_Returns demonstrates using matchers
// for return value verification
func TestTargetFunction_Ordered_Matcher_Returns(t *testing.T) {
	imp := imptest.NewImp(t)
	target := imptest.NewTargetFunction(imp, Add)

	call := target.CallWith(2, 3)

	// Use matcher instead of exact value
	call.ExpectReturnsMatch(imptest.Satisfies(func(v any) bool {
		result, ok := v.(int)
		return ok && result > 0
	}))
}

// Divide divides two integers, panicking on division by zero
func Divide(a, b int) int {
	if b == 0 {
		panic("division by zero")
	}
	return a / b
}

// TestTargetFunction_Ordered_Exact_Panic demonstrates verifying panics
func TestTargetFunction_Ordered_Exact_Panic(t *testing.T) {
	imp := imptest.NewImp(t)
	target := imptest.NewTargetFunction(imp, Divide)

	call := target.CallWith(10, 0)

	// Verify the function panicked with exact value
	call.ExpectPanicEquals("division by zero")
}

// TestTargetFunction_Ordered_Matcher_Panic demonstrates panic matching
func TestTargetFunction_Ordered_Matcher_Panic(t *testing.T) {
	imp := imptest.NewImp(t)
	target := imptest.NewTargetFunction(imp, Divide)

	call := target.CallWith(10, 0)

	// Match any panic
	call.ExpectPanicMatches(imptest.Any())
}

// TestTargetFunction_Ordered_GetReturns demonstrates getting actual values
func TestTargetFunction_Ordered_GetReturns(t *testing.T) {
	imp := imptest.NewImp(t)
	target := imptest.NewTargetFunction(imp, Add)

	call := target.CallWith(2, 3)

	// Get the actual return value for custom assertions
	result := call.GetReturns().R1
	if result != 5 {
		t.Errorf("expected 5, got %d", result)
	}
}
