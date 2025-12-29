// Code generated manually for v2 API demonstration. DO NOT EDIT.

package dependencyinterface_test

import (
	"github.com/toejough/imptest/imptest"
)

// DataStoreMock is the mock implementation returned by MockDataStore.
type DataStoreMock struct {
	imp *imptest.Imp
	Get    *imptest.DependencyMethod
	Save   *imptest.DependencyMethod
	Delete *imptest.DependencyMethod
}

// MockDataStore creates a new mock for the DataStore interface.
func MockDataStore(t imptest.TestReporter) *DataStoreMock {
	imp := t.(*imptest.Imp) // If given Imp, use it directly
	if imp == nil {
		// If given testing.T, create new Imp
		imp = imptest.NewImp(t)
	}

	return &DataStoreMock{
		imp:    imp,
		Get:    imptest.NewDependencyMethod(imp, "Get"),
		Save:   imptest.NewDependencyMethod(imp, "Save"),
		Delete: imptest.NewDependencyMethod(imp, "Delete"),
	}
}

// Interface returns the mock as a DataStore interface implementation.
func (m *DataStoreMock) Interface() DataStore {
	return &dataStoreImpl{mock: m}
}

// dataStoreImpl implements the DataStore interface by forwarding to the mock.
type dataStoreImpl struct {
	mock *DataStoreMock
}

// Get implements DataStore.Get by sending a call to the Controller and blocking on response.
func (impl *dataStoreImpl) Get(id int) (string, error) {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Get",
		Args:         []any{id},
		ResponseChan: responseChan,
	}

	// Send call to Controller
	impl.mock.imp.CallChan <- call

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

// Save implements DataStore.Save by sending a call to the Controller and blocking on response.
func (impl *dataStoreImpl) Save(id int, data string) error {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Save",
		Args:         []any{id, data},
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

	var err error
	if len(resp.ReturnValues) > 0 {
		if e, ok := resp.ReturnValues[0].(error); ok {
			err = e
		}
	}

	return err
}

// Delete implements DataStore.Delete by sending a call to the Controller and blocking on response.
func (impl *dataStoreImpl) Delete(id int) error {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Delete",
		Args:         []any{id},
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

	var err error
	if len(resp.ReturnValues) > 0 {
		if e, ok := resp.ReturnValues[0].(error); ok {
			err = e
		}
	}

	return err
}
