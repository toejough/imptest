package v2

// TargetFunction wraps a function under test.
// F is the function type (e.g., func(int, int) int)
type TargetFunction[F any] struct {
	imp      *Imp
	callable F
}

// NewTargetFunction creates a new wrapper for testing a function.
func NewTargetFunction[F any](imp *Imp, callable F) *TargetFunction[F] {
	return &TargetFunction[F]{
		imp:      imp,
		callable: callable,
	}
}

// CallWith is a placeholder - actual implementation will be code-generated
// for type-safe argument passing
func (tf *TargetFunction[F]) CallWith(args ...any) *TargetCall {
	// TODO: Implement actual call logic
	return &TargetCall{
		imp: tf.imp,
	}
}

// TargetCall represents a call to a target function.
type TargetCall struct {
	imp      *Imp
	ordered  bool // true = ordered (default), false = unordered
	returned bool
	panicked bool
	// TODO: Add actual return values and panic value
}

// ExpectReturnsEqual verifies the function returned exact values.
func (tc *TargetCall) ExpectReturnsEqual(expected ...any) {
	tc.imp.t.Helper()
	// TODO: Implement verification logic
	tc.imp.t.Fatal("ExpectReturnsEqual not yet implemented")
}

// ExpectReturnsMatch verifies return values match the given matchers.
func (tc *TargetCall) ExpectReturnsMatch(matchers ...any) {
	tc.imp.t.Helper()
	// TODO: Implement matcher verification logic
	tc.imp.t.Fatal("ExpectReturnsMatch not yet implemented")
}

// ExpectPanicEquals verifies the function panicked with an exact value.
func (tc *TargetCall) ExpectPanicEquals(expected any) {
	tc.imp.t.Helper()
	// TODO: Implement panic verification logic
	tc.imp.t.Fatal("ExpectPanicEquals not yet implemented")
}

// ExpectPanicMatches verifies the panic value matches the given matcher.
func (tc *TargetCall) ExpectPanicMatches(matcher Matcher) {
	tc.imp.t.Helper()
	// TODO: Implement panic matcher verification logic
	tc.imp.t.Fatal("ExpectPanicMatches not yet implemented")
}

// Eventually switches to unordered mode where the call will wait for
// matching interactions, queueing mismatches.
func (tc *TargetCall) Eventually() *TargetCall {
	tc.ordered = false
	return tc
}

// GetReturns returns the actual return values from the call.
// This is a placeholder - actual implementation will be code-generated.
type TargetReturns struct {
	R1 any
	R2 any
	// More fields will be generated as needed
}

func (tc *TargetCall) GetReturns() *TargetReturns {
	tc.imp.t.Helper()
	// TODO: Implement get returns logic
	tc.imp.t.Fatal("GetReturns not yet implemented")
	return nil
}

// GetPanic returns the panic value if the function panicked.
func (tc *TargetCall) GetPanic() any {
	tc.imp.t.Helper()
	// TODO: Implement get panic logic
	tc.imp.t.Fatal("GetPanic not yet implemented")
	return nil
}

// TargetInterface wraps an interface implementation under test.
// I is the interface type
type TargetInterface[I any] struct {
	imp      *Imp
	instance I
	// TODO: Methods will be added based on the interface
}

// NewTargetInterface creates a new wrapper for testing an interface implementation.
func NewTargetInterface[I any](imp *Imp, instance I) *TargetInterface[I] {
	return &TargetInterface[I]{
		imp:      imp,
		instance: instance,
	}
}

// Note: Interface methods like Add, Subtract, etc. will be code-generated
// for each specific interface type
