package core

import "fmt"

// DependencyArgs provides access to the actual arguments that were passed to the dependency.
// Code generation will create properly typed versions of this.
type DependencyArgs struct {
	A1 any
	A2 any
	A3 any
	A4 any
	A5 any
	// More fields will be generated as needed
}

// DependencyCall represents an expected call to a dependency.
// This type wraps a GenericCall from the Controller and provides the v2 API.
// In async mode (Eventually()), it wraps a PendingExpectation instead.
type DependencyCall struct {
	call    *GenericCall        // set in synchronous mode
	pending *PendingExpectation // set in async mode (Eventually)
}

// GetArgs returns the arguments passed to the mock method in this call.

// Build the args struct from the call's args

// Panic specifies that the mock should panic with the given value.
// This sends a panic response to the mock's response channel, unblocking it.
// In async mode, this can be called before or after the call is matched.
func (dc *DependencyCall) Panic(value any) {
	if dc.pending != nil {
		// Async mode - delegate to PendingExpectation
		dc.pending.Panic(value)

		return
	}

	// Synchronous mode - send directly
	dc.call.MarkDone()

	dc.call.ResponseChan <- GenericResponse{
		Type:       "panic",
		PanicValue: value,
	}
}

// RawArgs returns the raw argument slice for use by generated code.
// This allows generated code to create type-safe args accessors.
// In async mode, blocks until a call is matched.
func (dc *DependencyCall) RawArgs() []any {
	if dc.pending != nil {
		// Async mode - wait for match and return matched args
		return dc.pending.GetMatchedArgs()
	}

	// Synchronous mode - return call args directly
	return dc.call.Args
}

// Return specifies the values the mock should return when called.
// This sends the response to the mock's response channel, unblocking it.
// In async mode, this can be called before or after the call is matched.
func (dc *DependencyCall) Return(values ...any) {
	if dc.pending != nil {
		// Async mode - delegate to PendingExpectation
		dc.pending.Return(values...)

		return
	}

	// Synchronous mode - send directly
	dc.call.MarkDone()

	dc.call.ResponseChan <- GenericResponse{
		Type:         "return",
		ReturnValues: values,
	}
}

// DependencyMethod represents a method on a mocked interface.
// It provides methods to set up expectations for that specific method.
// Code generation creates instances of this for each method in an interface.
type DependencyMethod struct {
	imp        *Imp
	methodName string
	eventually bool

	// Eventually is the async version of this method.
	// Use this for concurrent code where calls may arrive out of order.
	// Example: imp.Add.Eventually.Expect(1, 2)
	Eventually *DependencyMethod
}

// NewDependencyMethod creates a new DependencyMethod.
// This is used by generated mock code.
func NewDependencyMethod(imp *Imp, methodName string) *DependencyMethod {
	depMethod := &DependencyMethod{
		imp:        imp,
		methodName: methodName,
		eventually: false,
	}
	// Initialize the Eventually field as a copy with eventually mode enabled
	depMethod.Eventually = &DependencyMethod{
		imp:        imp,
		methodName: methodName,
		eventually: true,
	}

	return depMethod
}

// Expect waits for a call to this method with exactly the specified arguments.
// Uses reflection-based DeepEqual for argument matching. Returns detailed error messages
// when arguments don't match.
// In eventually mode, this returns immediately (non-blocking) and registers a pending expectation.
func (dm *DependencyMethod) Expect(args ...any) *DependencyCall {
	validator := func(actualArgs []any) error {
		if len(actualArgs) != len(args) {
			//nolint:err113 // validation error with dynamic context
			return fmt.Errorf("expected %d args, got %d", len(args), len(actualArgs))
		}

		for i, expected := range args {
			if !valuesEqual(actualArgs[i], expected) {
				//nolint:err113 // validation error with dynamic context
				return fmt.Errorf("arg %d: expected %#v, got %#v", i, expected, actualArgs[i])
			}
		}

		return nil
	}

	if dm.eventually {
		// Async mode - register pending expectation and return immediately
		pending := dm.imp.RegisterPendingExpectation(dm.methodName, validator)

		return &DependencyCall{
			pending: pending,
		}
	}

	// Synchronous mode - block until call arrives
	call := dm.imp.GetCallOrdered(0, dm.methodName, validator)

	return newDependencyCall(call)
}

// MatchAny waits for a call to this method with any arguments.
// Use when you don't care about the argument values.
// In eventually mode, this returns immediately (non-blocking) and registers a pending expectation.

// Always matches

// Async mode - register pending expectation and return immediately

// Synchronous mode - block until call arrives

// Match waits for a call to this method with arguments matching the given matchers.
// Each matcher should implement the Matcher interface (compatible with gomega matchers).
// Returns detailed error messages when matchers don't match.
// In eventually mode, this returns immediately (non-blocking) and registers a pending expectation.
func (dm *DependencyMethod) Match(matchers ...any) *DependencyCall {
	validator := func(actualArgs []any) error {
		if len(actualArgs) != len(matchers) {
			//nolint:err113 // validation error with dynamic context
			return fmt.Errorf("expected %d args, got %d", len(matchers), len(actualArgs))
		}

		for index, m := range matchers {
			ok, failureMsg := MatchValue(actualArgs[index], m)
			if !ok {
				if failureMsg != "" {
					//nolint:err113 // validation error with dynamic context
					return fmt.Errorf("arg %d: %s", index, failureMsg)
				}
				//nolint:err113 // validation error with dynamic context
				return fmt.Errorf("arg %d: matcher failed for value %#v", index, actualArgs[index])
			}
		}

		return nil
	}

	if dm.eventually {
		// Async mode - register pending expectation and return immediately
		pending := dm.imp.RegisterPendingExpectation(dm.methodName, validator)

		return &DependencyCall{
			pending: pending,
		}
	}

	// Synchronous mode - block until call arrives
	call := dm.imp.GetCallOrdered(0, dm.methodName, validator)

	return newDependencyCall(call)
}

// newDependencyCall creates a DependencyCall from a GenericCall.
// This is called by generated mock code after receiving a call from the Controller.
func newDependencyCall(call *GenericCall) *DependencyCall {
	return &DependencyCall{
		call: call,
	}
}
