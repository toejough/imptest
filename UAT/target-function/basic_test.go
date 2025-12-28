// Package targetfunction_test demonstrates wrapping target functions with imptest v2.
//
// Test Taxonomy Coverage:
//
//	What:     Target ✓ | Dependency x
//	Type:     Function ✓ | Interface x
//	Mode:     Ordered ✓ | Unordered ✓
//	Matching: Exact ✓ | Matcher ✓
//	Outcome:  Return ✓ | Panic ✓
//	Source:   Type ✓ | Definition ✓
//
// Wrapper Sources (function signatures used for code generation):
//
//	WrapBinaryOp   ← type BinaryOp func(a, b int) int
//	WrapAdd        ← Add(a, b int) int
//	WrapDivide     ← Divide(a, b int) int
//	WrapConcurrent ← Concurrent(i int) int
package targetfunction_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/imptest"
)

// BinaryOp is a function type that takes two integers and returns one
type BinaryOp func(a, b int) int

// multiply is an instance of BinaryOp
var multiply BinaryOp = func(a, b int) int {
	return a * b
}

// Add adds two integers and returns the result
func Add(a, b int) int {
	return a + b
}

// TestTargetFunction_Type_Ordered_Exact_Returns demonstrates wrapping a function
// from a named type (vs a function definition)
func TestTargetFunction_Type_Ordered_Exact_Returns(t *testing.T) {
	WrapBinaryOp(t, multiply).CallWith(3, 4).ExpectReturnsEqual(12)
}

// TestTargetFunction_Ordered_Exact_Returns demonstrates wrapping a function
// with ordered expectations and exact value matching for returns
func TestTargetFunction_Ordered_Exact_Returns(t *testing.T) {
	WrapAdd(t, Add).CallWith(2, 3).ExpectReturnsEqual(5)
}

// TestTargetFunction_Ordered_Matcher_Returns demonstrates using matchers
// for return value verification
func TestTargetFunction_Ordered_Matcher_Returns(t *testing.T) {
	// Use gomega matcher for flexible validation
	WrapAdd(t, Add).CallWith(2, 3).ExpectReturnsMatch(BeNumerically(">", 0))
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
	// Verify the function panicked with exact value
	WrapDivide(t, Divide).CallWith(10, 0).ExpectPanicEquals("division by zero")
}

// TestTargetFunction_Ordered_Matcher_Panic demonstrates panic matching
func TestTargetFunction_Ordered_Matcher_Panic(t *testing.T) {
	// Match any panic
	WrapDivide(t, Divide).CallWith(10, 0).ExpectPanicMatches(imptest.Any())
}

// TestTargetFunction_Ordered_GetReturns demonstrates getting actual values
func TestTargetFunction_Ordered_GetReturns(t *testing.T) {
	// Get the actual return value for custom assertions
	returns := WrapAdd(t, Add).CallWith(2, 3).GetReturns()
	if returns.R1 != 5 {
		t.Errorf("expected 5, got %d", returns.R1)
	}
}

// Concurrent is a sample function that returns its input.
// It's not actually concurrent, but used to demonstrate concurrent testing and ordering.
func Concurrent(i int) int {
	return i
}

// TestTargetFunction_Ordered_Coordinated demonstrates coordinating multiple
// interactions with a shared Imp coordinator.
func TestTargetFunction_Ordered_Coordinated(t *testing.T) {
	// Use a shared Imp to coordinate multiple calls
	imp := imptest.NewImp(t)

	// Launch two calls without waiting for their responses
	c1 := WrapConcurrent(imp, Concurrent).CallWith(1)
	c2 := WrapConcurrent(imp, Concurrent).CallWith(2)

	// Now verify their returns in order
	c1.ExpectReturnsEqual(1)
	c2.ExpectReturnsEqual(2)
}

// TestTargetFunction_Unordered_Coordinated demonstrates coordinating unordered
// expectations across interactions via Eventually.
func TestTargetFunction_Unordered_Coordinated(t *testing.T) {
	// Use a shared Imp to coordinate multiple calls
	imp := imptest.NewImp(t)

	// Launch two calls without waiting for their responses
	c1 := WrapConcurrent(imp, Concurrent).CallWith(1)
	c2 := WrapConcurrent(imp, Concurrent).CallWith(2)

	// Verify returns in reverse order using Eventually
	c2.Eventually().ExpectReturnsEqual(2)
	c1.Eventually().ExpectReturnsEqual(1)
}
