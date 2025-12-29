package imptest

// Eventually switches to unordered mode for this expectation.
// In unordered mode, GetCallWithTimeout will wait for a matching call,
// queueing non-matching calls that arrive first.
// TODO: Implement when adding Eventually support to generated code
// func (dc *DependencyCall) Eventually() *DependencyCall {
// 	return dc
// }

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
type DependencyCall struct {
	imp  *Imp
	call *GenericCall
}

func (dc *DependencyCall) GetArgs() *DependencyArgs {
	dc.imp.Helper()

	// Build the args struct from the call's args
	result := &DependencyArgs{}

	if len(dc.call.Args) > 0 {
		result.A1 = dc.call.Args[0]
	}

	if len(dc.call.Args) > minArgsForA2 {
		result.A2 = dc.call.Args[1]
	}

	if len(dc.call.Args) > minArgsForA3 {
		result.A3 = dc.call.Args[2]
	}

	if len(dc.call.Args) > minArgsForA4 {
		result.A4 = dc.call.Args[3]
	}

	if len(dc.call.Args) > minArgsForA5 {
		result.A5 = dc.call.Args[4]
	}

	return result
}

// InjectPanicValue specifies that the mock should panic with the given value.
// This sends a panic response to the mock's response channel, unblocking it.
func (dc *DependencyCall) InjectPanicValue(value any) {
	dc.call.MarkDone()

	dc.call.ResponseChan <- GenericResponse{
		Type:       "panic",
		PanicValue: value,
	}
}

// InjectReturnValues specifies the values the mock should return when called.
// This sends the response to the mock's response channel, unblocking it.
func (dc *DependencyCall) InjectReturnValues(values ...any) {
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
}

// NewDependencyMethod creates a new DependencyMethod.
// This is used by generated mock code.
func NewDependencyMethod(imp *Imp, methodName string) *DependencyMethod {
	return &DependencyMethod{
		imp:        imp,
		methodName: methodName,
	}
}

// ExpectCalledWithExactly waits for a call to this method with exactly the specified arguments.
// Uses reflection-based DeepEqual for argument matching.
func (dm *DependencyMethod) ExpectCalledWithExactly(args ...any) *DependencyCall {
	validator := func(actualArgs []any) bool {
		if len(actualArgs) != len(args) {
			return false
		}

		for i, expected := range args {
			if !valuesEqual(actualArgs[i], expected) {
				return false
			}
		}

		return true
	}

	call := dm.imp.GetCallWithTimeout(0, dm.methodName, validator)

	return newDependencyCall(dm.imp, call)
}

// ExpectCalledWithMatches waits for a call to this method with arguments matching the given matchers.
// Each matcher should implement the Matcher interface (compatible with gomega matchers).
func (dm *DependencyMethod) ExpectCalledWithMatches(matchers ...any) *DependencyCall {
	validator := func(actualArgs []any) bool {
		if len(actualArgs) != len(matchers) {
			return false
		}

		for index, m := range matchers {
			matcher, ok := m.(Matcher)
			if !ok {
				return false
			}

			success, _ := matcher.Match(actualArgs[index])
			if !success {
				return false
			}
		}

		return true
	}

	call := dm.imp.GetCallWithTimeout(0, dm.methodName, validator)

	return newDependencyCall(dm.imp, call)
}

// Eventually switches to unordered mode where the expectation will wait
// for a matching call, queueing non-matching calls that arrive first.
// TODO: Implement when adding Eventually support
// func (dm *DependencyMethod) Eventually() *DependencyMethodEventually {
// 	return &DependencyMethodEventually{
// 		imp:        dm.imp,
// 		methodName: dm.methodName,
// 		timeout:    30 * time.Second, // Default timeout for Eventually
// 	}
// }

// DependencyMethodEventually provides Eventually (unordered) mode expectations.
// TODO: Implement when adding Eventually support
// type DependencyMethodEventually struct {
// 	imp        *Imp
// 	methodName string
// 	timeout    time.Duration
// }
//
// // ExpectCalledWithExactly waits (with timeout) for a call with exact arguments.
// func (dme *DependencyMethodEventually) ExpectCalledWithExactly(args ...any) *DependencyCall {
// 	validator := func(actualArgs []any) bool {
// 		if len(actualArgs) != len(args) {
// 			return false
// 		}
// 		for i, expected := range args {
// 			if !valuesEqual(actualArgs[i], expected) {
// 				return false
// 			}
// 		}
// 		return true
// 	}
//
// 	call := dme.imp.GetCallWithTimeout(dme.timeout, dme.methodName, validator)
// 	return newDependencyCall(dme.imp, call)
// }
//
// // ExpectCalledWithMatches waits (with timeout) for a call with matching arguments.
// func (dme *DependencyMethodEventually) ExpectCalledWithMatches(matchers ...any) *DependencyCall {
// 	validator := func(actualArgs []any) bool {
// 		if len(actualArgs) != len(matchers) {
// 			return false
// 		}
// 		for i, m := range matchers {
// 			matcher, ok := m.(Matcher)
// 			if !ok {
// 				return false
// 			}
// 			success, _ := matcher.Match(actualArgs[i])
// 			if !success {
// 				return false
// 			}
// 		}
// 		return true
// 	}
//
// 	call := dme.imp.GetCallWithTimeout(dme.timeout, dme.methodName, validator)
// 	return newDependencyCall(dme.imp, call)
// }

// unexported constants.
const (
	// Argument position constants for DependencyArgs fields.
	minArgsForA2 = 1
	minArgsForA3 = 2
	minArgsForA4 = 3
	minArgsForA5 = 4
)

// newDependencyCall creates a DependencyCall from a GenericCall.
// This is called by generated mock code after receiving a call from the Controller.
func newDependencyCall(imp *Imp, call *GenericCall) *DependencyCall {
	return &DependencyCall{
		imp:  imp,
		call: call,
	}
}
