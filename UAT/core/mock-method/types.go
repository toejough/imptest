// Package mockmethod demonstrates mocking individual struct methods.
package mockmethod

// Counter is a struct type with methods that we want to mock individually.
// Unlike mocking the whole struct, we mock single method signatures.
type Counter struct {
	val int
}

// Add adds a value to the counter and returns the new value.
func (c *Counter) Add(n int) int {
	c.val += n
	return c.val
}

// Dec decrements the counter and returns the new value.
func (c *Counter) Dec() int {
	c.val--
	return c.val
}

// Inc increments the counter and returns the new value.
func (c *Counter) Inc() int {
	c.val++
	return c.val
}

// Value returns the current counter value.
func (c *Counter) Value() int {
	return c.val
}

// UseCounterAdd is a function that depends on a function matching Counter.Add's signature.
// This demonstrates injecting a mocked method.
func UseCounterAdd(addFunc func(n int) int, value int) int {
	return addFunc(value)
}

// UseCounterInc is a function that depends on a function matching Counter.Inc's signature.
func UseCounterInc(incFunc func() int) int {
	return incFunc()
}
