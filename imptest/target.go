package imptest

import "reflect"

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
		Imp: tf.imp,
	}
}

// TargetCall represents a call to a target function.
// Fields are exported so generated code can populate them.
type TargetCall struct {
	Imp          *Imp
	Ordered      bool  // true = ordered (default), false = unordered
	Returned     bool  // true if function returned normally
	Panicked     bool  // true if function panicked
	ReturnValues []any // actual return values if returned
	PanicValue   any   // actual panic value if panicked
}

// ExpectReturnsEqual verifies the function returned exact values.
func (tc *TargetCall) ExpectReturnsEqual(expected ...any) {
	tc.Imp.Helper()

	if tc.Panicked {
		tc.Imp.Fatalf("expected function to return, but it panicked with: %v", tc.PanicValue)
		return
	}

	if !tc.Returned {
		tc.Imp.Fatalf("expected function to return, but it didn't complete")
		return
	}

	if len(tc.ReturnValues) != len(expected) {
		tc.Imp.Fatalf("expected %d return values, got %d", len(expected), len(tc.ReturnValues))
		return
	}

	for i, exp := range expected {
		if !valuesEqual(tc.ReturnValues[i], exp) {
			tc.Imp.Fatalf("return value %d: expected %v, got %v", i, FormatValue(exp), FormatValue(tc.ReturnValues[i]))
			return
		}
	}
}

// ExpectReturnsMatch verifies return values match the given matchers.
func (tc *TargetCall) ExpectReturnsMatch(matchers ...any) {
	tc.Imp.Helper()

	if tc.Panicked {
		tc.Imp.Fatalf("expected function to return, but it panicked with: %v", tc.PanicValue)
		return
	}

	if !tc.Returned {
		tc.Imp.Fatalf("expected function to return, but it didn't complete")
		return
	}

	if len(tc.ReturnValues) != len(matchers) {
		tc.Imp.Fatalf("expected %d matchers, got %d return values", len(matchers), len(tc.ReturnValues))
		return
	}

	for i, m := range matchers {
		matcher, ok := m.(Matcher)
		if !ok {
			tc.Imp.Fatalf("argument %d is not a Matcher", i)
			return
		}

		success, err := matcher.Match(tc.ReturnValues[i])
		if err != nil {
			tc.Imp.Fatalf("return value %d: matcher error: %v", i, err)
			return
		}
		if !success {
			tc.Imp.Fatalf("return value %d: %s", i, matcher.FailureMessage(tc.ReturnValues[i]))
			return
		}
	}
}

// ExpectPanicEquals verifies the function panicked with an exact value.
func (tc *TargetCall) ExpectPanicEquals(expected any) {
	tc.Imp.Helper()

	if !tc.Panicked {
		tc.Imp.Fatalf("expected function to panic, but it returned normally")
		return
	}

	if !valuesEqual(tc.PanicValue, expected) {
		tc.Imp.Fatalf("expected panic with %v, got %v", FormatValue(expected), FormatValue(tc.PanicValue))
		return
	}
}

// ExpectPanicMatches verifies the panic value matches the given matcher.
func (tc *TargetCall) ExpectPanicMatches(matcher Matcher) {
	tc.Imp.Helper()

	if !tc.Panicked {
		tc.Imp.Fatalf("expected function to panic, but it returned normally")
		return
	}

	success, err := matcher.Match(tc.PanicValue)
	if err != nil {
		tc.Imp.Fatalf("panic matcher error: %v", err)
		return
	}
	if !success {
		tc.Imp.Fatalf("panic value: %s", matcher.FailureMessage(tc.PanicValue))
		return
	}
}

// Eventually switches to unordered mode where the call will wait for
// matching interactions, queueing mismatches.
func (tc *TargetCall) Eventually() *TargetCall {
	tc.Ordered = false
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
	tc.Imp.Helper()

	if tc.Panicked {
		tc.Imp.Fatalf("cannot get returns: function panicked with %v", tc.PanicValue)
		return nil
	}

	if !tc.Returned {
		tc.Imp.Fatalf("cannot get returns: function didn't complete")
		return nil
	}

	// For now, return a struct with up to 2 return values
	// Code generation will create properly typed versions
	result := &TargetReturns{}
	if len(tc.ReturnValues) > 0 {
		result.R1 = tc.ReturnValues[0]
	}
	if len(tc.ReturnValues) > 1 {
		result.R2 = tc.ReturnValues[1]
	}
	return result
}

// GetPanic returns the panic value if the function panicked.
func (tc *TargetCall) GetPanic() any {
	tc.Imp.Helper()

	if !tc.Panicked {
		tc.Imp.Fatalf("cannot get panic: function returned normally")
		return nil
	}

	return tc.PanicValue
}

// valuesEqual checks if two values are equal using reflect.DeepEqual.
func valuesEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
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
