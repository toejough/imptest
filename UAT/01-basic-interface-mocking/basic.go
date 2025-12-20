package basic

// BasicOps demonstrates the core mocking features of imptest.
// It covers single and multiple return values, void methods, and variadic arguments.
type BasicOps interface {
	// Add demonstrates a simple method with parameters and a single return value.
	Add(a, b int) int

	// Store demonstrates a method with multiple return values (common for error handling).
	Store(key string, value any) (int, error)

	// Log demonstrates a void method (no return values).
	Log(message string)

	// Notify demonstrates variadic arguments.
	Notify(message string, ids ...int) bool
}

// PerformOps is a helper that uses the BasicOps interface.
func PerformOps(ops BasicOps) {
	const (
		val1 = 1
		val2 = 2
		val3 = 3
	)

	_ = ops.Add(val1, val2)
	_, _ = ops.Store("foo", "bar")
	ops.Log("action performed")
	_ = ops.Notify("alert", val1, val2, val3)
}
