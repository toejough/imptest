// Code generated manually for v2 API demonstration. DO NOT EDIT.

package dependencyfunction_test

import (
	"github.com/toejough/imptest/imptest"
)

// FetcherMock is the mock for the Fetcher function type.
type FetcherMock struct {
	imp    *imptest.Imp
	method *imptest.DependencyMethod
}

// MockFetcher creates a new mock for the Fetcher function type.
func MockFetcher(t imptest.TestReporter) *FetcherMock {
	imp := t.(*imptest.Imp)
	if imp == nil {
		imp = imptest.NewImp(t)
	}

	return &FetcherMock{
		imp:    imp,
		method: imptest.NewDependencyMethod(imp, "Fetcher"),
	}
}

// Func returns the mock as a Fetcher function.
func (m *FetcherMock) Func() Fetcher {
	return func(id int) (string, error) {
		responseChan := make(chan imptest.GenericResponse, 1)

		call := &imptest.GenericCall{
			MethodName:   "Fetcher",
			Args:         []any{id},
			ResponseChan: responseChan,
		}

		// Send call to Controller
		m.imp.CallChan <- call

		// Block waiting for test to inject response
		resp := <-responseChan

		if resp.Type == "panic" {
			panic(resp.PanicValue)
		}

		// Extract return values
		var result string
		var err error
		if len(resp.ReturnValues) > 0 {
			if v, ok := resp.ReturnValues[0].(string); ok {
				result = v
			}
		}
		if len(resp.ReturnValues) > 1 {
			if e, ok := resp.ReturnValues[1].(error); ok {
				err = e
			}
		}

		return result, err
	}
}

// ExpectCalledWithExactly waits for a call with exact arguments.
func (m *FetcherMock) ExpectCalledWithExactly(args ...any) *imptest.DependencyCall {
	return m.method.ExpectCalledWithExactly(args...)
}

// ExpectCalledWithMatches waits for a call with matching arguments.
func (m *FetcherMock) ExpectCalledWithMatches(matchers ...any) *imptest.DependencyCall {
	return m.method.ExpectCalledWithMatches(matchers...)
}

// ValidatorMock is the mock for the Validator function type.
type ValidatorMock struct {
	imp    *imptest.Imp
	method *imptest.DependencyMethod
}

// MockValidator creates a new mock for the Validator function type.
func MockValidator(t imptest.TestReporter) *ValidatorMock {
	imp := t.(*imptest.Imp)
	if imp == nil {
		imp = imptest.NewImp(t)
	}

	return &ValidatorMock{
		imp:    imp,
		method: imptest.NewDependencyMethod(imp, "Validator"),
	}
}

// Func returns the mock as a Validator function.
func (m *ValidatorMock) Func() Validator {
	return func(value int) bool {
		responseChan := make(chan imptest.GenericResponse, 1)

		call := &imptest.GenericCall{
			MethodName:   "Validator",
			Args:         []any{value},
			ResponseChan: responseChan,
		}

		m.imp.CallChan <- call
		resp := <-responseChan

		if resp.Type == "panic" {
			panic(resp.PanicValue)
		}

		var result bool
		if len(resp.ReturnValues) > 0 {
			if v, ok := resp.ReturnValues[0].(bool); ok {
				result = v
			}
		}

		return result
	}
}

// ExpectCalledWithExactly waits for a call with exact arguments.
func (m *ValidatorMock) ExpectCalledWithExactly(args ...any) *imptest.DependencyCall {
	return m.method.ExpectCalledWithExactly(args...)
}

// ExpectCalledWithMatches waits for a call with matching arguments.
func (m *ValidatorMock) ExpectCalledWithMatches(matchers ...any) *imptest.DependencyCall {
	return m.method.ExpectCalledWithMatches(matchers...)
}
