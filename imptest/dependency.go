package imptest

// DependencyFunction creates a mock for a function dependency.
// F is the function type (e.g., func(int) (string, error))
type DependencyFunction[F any] struct {
	imp *Imp
	// TODO: Store the mock function implementation
}

// NewDependencyFunction creates a new mock for a function dependency.
func NewDependencyFunction[F any](imp *Imp) *DependencyFunction[F] {
	return &DependencyFunction[F]{
		imp: imp,
	}
}

// ExpectCalledWithExactly sets up an expectation that the function will be
// called with exactly the specified arguments.
func (df *DependencyFunction[F]) ExpectCalledWithExactly(args ...any) *DependencyCall {
	// Add expectation with exact args matching, ordered mode by default
	exp := df.imp.AddExpectation("DependencyFunction", args, nil, true)
	return &DependencyCall{
		imp:         df.imp,
		expectation: exp,
	}
}

// ExpectCalledWithMatches sets up an expectation using matchers for arguments.
func (df *DependencyFunction[F]) ExpectCalledWithMatches(matchers ...any) *DependencyCall {
	// Convert matchers to Matcher type
	matcherList := make([]Matcher, len(matchers))
	for i, m := range matchers {
		if matcher, ok := m.(Matcher); ok {
			matcherList[i] = matcher
		} else {
			df.imp.Helper()
			df.imp.Fatalf("argument %d is not a Matcher", i)
		}
	}

	exp := df.imp.AddExpectation("DependencyFunction", nil, matcherList, true)
	return &DependencyCall{
		imp:         df.imp,
		expectation: exp,
	}
}

// Func returns the actual function that should be passed to code under test.
// When this function is called, it will verify expectations and inject responses.
func (df *DependencyFunction[F]) Func() F {
	// TODO: Return a function that intercepts calls
	var zero F
	return zero
}

// DependencyCall represents an expected call to a dependency.
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
// This is a placeholder - actual implementation will be code-generated.
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

// DependencyInterface creates a mock for an interface dependency.
// I is the interface type
type DependencyInterface[I any] struct {
	imp *Imp
	// TODO: Methods will be added based on the interface
}

// NewDependencyInterface creates a new mock for an interface dependency.
func NewDependencyInterface[I any](imp *Imp) *DependencyInterface[I] {
	return &DependencyInterface[I]{
		imp: imp,
	}
}

// Interface returns the actual interface instance that should be passed to code under test.
func (di *DependencyInterface[I]) Interface() I {
	// TODO: Return a mock implementation
	var zero I
	return zero
}

// Note: Interface methods like Get, Save, etc. will be code-generated
// for each specific interface type
