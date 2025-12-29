// Code generated manually for v2 API migration (Phase 2). DO NOT EDIT.
// TODO: Replace with template-generated code in future phase.

package basic_test

import (
	basic "github.com/toejough/imptest/UAT/01-basic-interface-mocking"
	"github.com/toejough/imptest/imptest"
)

// OpsMock is the mock implementation returned by MockOps.
type OpsMock struct {
	imp    *imptest.Imp
	Add    *imptest.DependencyMethod
	Store  *imptest.DependencyMethod
	Log    *imptest.DependencyMethod
	Notify *imptest.DependencyMethod
	Finish *imptest.DependencyMethod
}

// MockOps creates a new mock for the Ops interface.
func MockOps(testReporter imptest.TestReporter) *OpsMock {
	imp, ok := testReporter.(*imptest.Imp)
	if !ok {
		// If given testing.T, create new Imp
		imp = imptest.NewImp(testReporter)
	}

	return &OpsMock{
		imp:    imp,
		Add:    imptest.NewDependencyMethod(imp, "Add"),
		Store:  imptest.NewDependencyMethod(imp, "Store"),
		Log:    imptest.NewDependencyMethod(imp, "Log"),
		Notify: imptest.NewDependencyMethod(imp, "Notify"),
		Finish: imptest.NewDependencyMethod(imp, "Finish"),
	}
}

// Interface returns the mock as an Ops interface implementation.
func (m *OpsMock) Interface() basic.Ops {
	return &opsImpl{mock: m}
}

// opsImpl implements the Ops interface by forwarding to the mock.
type opsImpl struct {
	mock *OpsMock
}

// Add implements Ops.Add by sending a call to the Controller and blocking on response.
func (impl *opsImpl) Add(arg1, arg2 int) int {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Add",
		Args:         []any{arg1, arg2},
		ResponseChan: responseChan,
	}

	// Send call to Controller
	impl.mock.imp.CallChan <- call

	// Block waiting for test to inject response
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

	// Extract return value
	var result int
	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(int); ok {
			result = value
		}
	}

	return result
}

// Store implements Ops.Store by sending a call to the Controller and blocking on response.
func (impl *opsImpl) Store(key string, value any) (int, error) {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Store",
		Args:         []any{key, value},
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

	var result int
	var err error

	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(int); ok {
			result = value
		}
	}

	if len(resp.ReturnValues) > 1 {
		if e, ok := resp.ReturnValues[1].(error); ok {
			err = e
		}
	}

	return result, err
}

// Log implements Ops.Log by sending a call to the Controller and blocking on response.
func (impl *opsImpl) Log(message string) {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Log",
		Args:         []any{message},
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}
}

// Notify implements Ops.Notify by sending a call to the Controller and blocking on response.
func (impl *opsImpl) Notify(message string, ids ...int) bool {
	// Convert variadic args to []any
	args := make([]any, 0, 1+len(ids))
	args = append(args, message)

	for _, id := range ids {
		args = append(args, id)
	}

	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Notify",
		Args:         args,
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

	var result bool
	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(bool); ok {
			result = value
		}
	}

	return result
}

// Finish implements Ops.Finish by sending a call to the Controller and blocking on response.
func (impl *opsImpl) Finish() bool {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Finish",
		Args:         []any{},
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

	var result bool
	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(bool); ok {
			result = value
		}
	}

	return result
}
