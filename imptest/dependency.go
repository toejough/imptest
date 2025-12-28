package imptest

// DependencyCall represents an expected call to a dependency.
// This type is used by generated mock code.
type DependencyCall struct {
	imp         *Imp
	expectation *Expectation
}

// InjectReturnValues specifies the values the mock should return when called.
func (dc *DependencyCall) InjectReturnValues(values ...any) {
	dc.expectation.returnValues = values
	dc.expectation.shouldPanic = false
}

// InjectPanicValue specifies that the mock should panic with the given value.
func (dc *DependencyCall) InjectPanicValue(value any) {
	dc.expectation.panicValue = value
	dc.expectation.shouldPanic = true
}

// Eventually switches to unordered mode for this expectation.
func (dc *DependencyCall) Eventually() *DependencyCall {
	dc.expectation.ordered = false
	return dc
}

// GetArgs returns the actual arguments that were passed to the dependency.
// Code generation will create properly typed versions of this.
type DependencyArgs struct {
	A1 any
	A2 any
	// More fields will be generated as needed
}

func (dc *DependencyCall) GetArgs() *DependencyArgs {
	dc.imp.Helper()
	if !dc.expectation.called {
		dc.imp.Fatalf("GetArgs called but expectation was never matched")
		return nil
	}
	// For now, return a struct with up to 2 args
	// Code generation will create properly typed versions
	result := &DependencyArgs{}
	if len(dc.expectation.actualArgs) > 0 {
		result.A1 = dc.expectation.actualArgs[0]
	}
	if len(dc.expectation.actualArgs) > 1 {
		result.A2 = dc.expectation.actualArgs[1]
	}
	return result
}
