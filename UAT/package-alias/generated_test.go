// Code generated manually for v2 API demonstration. DO NOT EDIT.

package packagealias_test

import (
	"time"

	"github.com/toejough/imptest/imptest"
)

// TimeServiceMock is the mock implementation returned by MockTimeService.
type TimeServiceMock struct {
	imp   *imptest.Imp
	Now   *imptest.DependencyMethod
	Sleep *imptest.DependencyMethod
}

// MockTimeService creates a new mock for the TimeService interface.
func MockTimeService(t imptest.TestReporter) *TimeServiceMock {
	imp := t.(*imptest.Imp)
	if imp == nil {
		imp = imptest.NewImp(t)
	}

	return &TimeServiceMock{
		imp:   imp,
		Now:   imptest.NewDependencyMethod(imp, "Now"),
		Sleep: imptest.NewDependencyMethod(imp, "Sleep"),
	}
}

// Interface returns the mock as a TimeService interface implementation.
func (m *TimeServiceMock) Interface() TimeService {
	return &timeServiceImpl{mock: m}
}

// timeServiceImpl implements the TimeService interface.
type timeServiceImpl struct {
	mock *TimeServiceMock
}

// Now implements TimeService.Now.
func (impl *timeServiceImpl) Now() time.Time {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Now",
		Args:         []any{},
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

	var result time.Time
	if len(resp.ReturnValues) > 0 {
		if v, ok := resp.ReturnValues[0].(time.Time); ok {
			result = v
		}
	}

	return result
}

// Sleep implements TimeService.Sleep.
func (impl *timeServiceImpl) Sleep(d time.Duration) {
	responseChan := make(chan imptest.GenericResponse, 1)

	call := &imptest.GenericCall{
		MethodName:   "Sleep",
		Args:         []any{d},
		ResponseChan: responseChan,
	}

	impl.mock.imp.CallChan <- call
	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}
}

// WrapGetCurrentHour creates a wrapper for the GetCurrentHour function.
func WrapGetCurrentHour(imp imptest.TestReporter, fn func(TimeService) int) *GetCurrentHourWrapper {
	return &GetCurrentHourWrapper{
		imp: imp.(*imptest.Imp),
		fn:  fn,
	}
}

// GetCurrentHourWrapper wraps a GetCurrentHour function for testing.
type GetCurrentHourWrapper struct {
	imp        *imptest.Imp
	fn         func(TimeService) int
	returnChan chan GetCurrentHourReturns
	panicChan  chan any
	returned   *GetCurrentHourReturns
	panicked   any
}

// GetCurrentHourReturns holds the return values from GetCurrentHour.
type GetCurrentHourReturns struct {
	R1 int
}

// Start executes the function in a goroutine and returns the wrapper for chaining.
func (w *GetCurrentHourWrapper) Start(ts TimeService) *GetCurrentHourWrapper {
	w.returnChan = make(chan GetCurrentHourReturns, 1)
	w.panicChan = make(chan any, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()
		r1 := w.fn(ts)
		w.returnChan <- GetCurrentHourReturns{R1: r1}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *GetCurrentHourWrapper) WaitForResponse() {
	if w.returned != nil || w.panicked != nil {
		return
	}
	select {
	case ret := <-w.returnChan:
		w.returned = &ret
	case p := <-w.panicChan:
		w.panicked = p
	}
}

// ExpectReturnsEqual verifies the function returned exact values.
func (w *GetCurrentHourWrapper) ExpectReturnsEqual(expectedR1 int) {
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to return, but it panicked with: %v", w.panicked)
		return
	}

	if w.returned.R1 != expectedR1 {
		w.imp.Fatalf("return value 1: expected %v, got %v", expectedR1, w.returned.R1)
		return
	}
}
