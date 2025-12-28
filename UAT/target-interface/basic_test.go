package targetinterface_test

import (
	"testing"

	imptest "github.com/toejough/imptest/imptest/v2"
)

// Calculator defines basic arithmetic operations
type Calculator interface {
	Add(a, b int) int
	Subtract(a, b int) int
	Divide(a, b int) (int, error)
}

// BasicCalculator implements Calculator
type BasicCalculator struct{}

func (c *BasicCalculator) Add(a, b int) int {
	return a + b
}

func (c *BasicCalculator) Subtract(a, b int) int {
	return a - b
}

func (c *BasicCalculator) Divide(a, b int) (int, error) {
	if b == 0 {
		panic("division by zero")
	}
	return a / b, nil
}

// TestTargetInterface_Ordered_Exact_Returns demonstrates wrapping an interface
// with ordered expectations and exact value matching
func TestTargetInterface_Ordered_Exact_Returns(t *testing.T) {
	imp := imptest.NewImp(t)

	// Create the actual implementation
	calc := &BasicCalculator{}

	// Wrap it with target interface wrapper
	target := imptest.NewTargetInterface(imp, calc)

	// Call a method through the wrapper
	call := target.Add.CallWith(2, 3)

	// Verify exact return value
	call.ExpectReturnsEqual(5)
}

// TestTargetInterface_Ordered_Matcher_Returns demonstrates matcher validation
func TestTargetInterface_Ordered_Matcher_Returns(t *testing.T) {
	imp := imptest.NewImp(t)
	calc := &BasicCalculator{}
	target := imptest.NewTargetInterface(imp, calc)

	call := target.Subtract.CallWith(10, 3)

	// Use matcher for validation
	call.ExpectReturnsMatch(imptest.Satisfies(func(v any) bool {
		result, ok := v.(int)
		return ok && result > 0 && result < 10
	}))
}

// TestTargetInterface_Ordered_Exact_Panic demonstrates panic verification
func TestTargetInterface_Ordered_Exact_Panic(t *testing.T) {
	imp := imptest.NewImp(t)
	calc := &BasicCalculator{}
	target := imptest.NewTargetInterface(imp, calc)

	call := target.Divide.CallWith(10, 0)

	// Verify panic with exact value
	call.ExpectPanicEquals("division by zero")
}

// TestTargetInterface_Ordered_GetReturns demonstrates getting actual values
func TestTargetInterface_Ordered_GetReturns(t *testing.T) {
	imp := imptest.NewImp(t)
	calc := &BasicCalculator{}
	target := imptest.NewTargetInterface(imp, calc)

	call := target.Divide.CallWith(10, 2)

	// Get actual return values
	result := call.GetReturns()
	if result.R1 != 5 {
		t.Errorf("expected result 5, got %d", result.R1)
	}
	if result.R2 != nil {
		t.Errorf("expected no error, got %v", result.R2)
	}
}

// TestTargetInterface_Ordered_MultipleReturns demonstrates multiple return values
func TestTargetInterface_Ordered_MultipleReturns(t *testing.T) {
	imp := imptest.NewImp(t)
	calc := &BasicCalculator{}
	target := imptest.NewTargetInterface(imp, calc)

	call := target.Divide.CallWith(10, 2)

	// Verify multiple return values
	call.ExpectReturnsEqual(5, nil)
}
