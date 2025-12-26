package methods

// Calculator provides arithmetic operations.
// In real applications, methods might have side effects (logging, metrics, database calls)
// that make them difficult to test directly.
type Calculator struct {
	multiplier int
}

// NewCalculator creates a calculator with a multiplier factor.
func NewCalculator(multiplier int) *Calculator {
	return &Calculator{multiplier: multiplier}
}

// Add returns the sum of two integers.
func (c *Calculator) Add(a, b int) int {
	return a + b
}

// Divide returns the quotient and whether the division was successful.
// Returns false if attempting to divide by zero.
func (c *Calculator) Divide(numerator, denominator int) (int, bool) {
	if denominator == 0 {
		return 0, false
	}

	return numerator / denominator, true
}

// Multiply applies the calculator's multiplier to the input.
func (c *Calculator) Multiply(value int) int {
	return value * c.multiplier
}

// ProcessValue demonstrates a method that might panic in production.
// This method applies a series of operations that could fail.
func (c *Calculator) ProcessValue(value int) int {
	if value < 0 {
		panic("negative values not supported")
	}

	result := c.Multiply(value)

	return result + processingOffset
}

// unexported constants.
const (
	processingOffset = 10
)
