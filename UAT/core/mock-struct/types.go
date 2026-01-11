// Package mockstruct demonstrates mocking a struct type by wrapping its methods
// into an interface that can be mocked.
package mockstruct

import "errors"

// Exported variables.
var (
	ErrNotFound = errors.New("not found")
)

// Calculator is a struct type with methods that we want to mock.
// Unlike an interface, this is a concrete implementation.
type Calculator struct {
	memory int
}

// Add returns the sum of two integers.
func (c *Calculator) Add(a, b int) int {
	return a + b
}

// Get retrieves the current memory value.
func (c *Calculator) Get() (int, error) {
	if c.memory == 0 {
		return 0, ErrNotFound
	}

	return c.memory, nil
}

// Reset clears the memory (void method).
func (c *Calculator) Reset() {
	c.memory = 0
}

// Store saves a value to memory and returns the previous value.
func (c *Calculator) Store(value int) int {
	prev := c.memory
	c.memory = value

	return prev
}

// UseCalculator is a helper that exercises the Calculator methods.
func UseCalculator(calc interface {
	Add(a, b int) int
	Store(value int) int
	Get() (int, error)
	Reset()
},
) {
	const (
		a       = 1
		b       = 2
		storeMe = 42
	)

	_ = calc.Add(a, b)
	_ = calc.Store(storeMe)
	_, _ = calc.Get()
	calc.Reset()
}
