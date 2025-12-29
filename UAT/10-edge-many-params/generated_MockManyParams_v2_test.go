// Code generated manually for v2 API migration (Phase 2). DO NOT EDIT.
// TODO: Replace with template-generated code in future phase.

package many_params_test

import (
	mp "github.com/toejough/imptest/UAT/10-edge-many-params"
	"github.com/toejough/imptest/imptest"
)

// ManyParamsMock is the mock implementation returned by MockManyParams.
type ManyParamsMock struct {
	imp     *imptest.Imp
	Process *imptest.DependencyMethod
}

// MockManyParams creates a new mock for the ManyParams interface.
func MockManyParams(testReporter imptest.TestReporter) *ManyParamsMock {
	imp, ok := testReporter.(*imptest.Imp)
	if !ok {
		imp = imptest.NewImp(testReporter)
	}

	return &ManyParamsMock{
		imp:     imp,
		Process: imptest.NewDependencyMethod(imp, "Process"),
	}
}

// Interface returns the mock as a ManyParams interface implementation.
func (m *ManyParamsMock) Interface() mp.ManyParams {
	return &manyParamsImpl{mock: m}
}

// manyParamsImpl implements the ManyParams interface by forwarding to the mock.
type manyParamsImpl struct {
	mock *ManyParamsMock
}

// Process implements ManyParams.Process by sending a call to the Controller and blocking on response.
func (impl *manyParamsImpl) Process(a, b, c, d, e, f, g, h, i, j int) string {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Process",
		Args:         []any{a, b, c, d, e, f, g, h, i, j},
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
	var result string
	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(string); ok {
			result = value
		}
	}

	return result
}
