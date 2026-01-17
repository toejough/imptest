// Code generated manually for UAT-42. DO NOT EDIT.

package callable_test

import (
	"time"

	_reflect "reflect"

	_imptest "github.com/toejough/imptest"
)

// WrapSlowMultiplyReturnsReturn holds the return values from the wrapped function.
type WrapSlowMultiplyReturnsReturn struct {
	Result0 int
}

// WrapSlowMultiplyWrapper wraps a function for testing.
type WrapSlowMultiplyWrapperHandle struct {
	t        _imptest.TestReporter
	callable func(a int, delay time.Duration) int
}

// WrapSlowMultiplyCallHandle represents a single call to the wrapped function.
type WrapSlowMultiplyCallHandle struct {
	*_imptest.CallableController[WrapSlowMultiplyReturnsReturn]
}

// Start executes the wrapped function in a goroutine.
func (w *WrapSlowMultiplyWrapperHandle) Start(a int, delay time.Duration) *WrapSlowMultiplyCallHandle {
	handle := &WrapSlowMultiplyCallHandle{
		CallableController: _imptest.NewCallableController[WrapSlowMultiplyReturnsReturn](w.t),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		ret0 := w.callable(a, delay)
		handle.ReturnChan <- WrapSlowMultiplyReturnsReturn{Result0: ret0}
	}()
	return handle
}

// ExpectReturn verifies the function returned the expected values.
func (h *WrapSlowMultiplyCallHandle) ExpectReturn(v0 int) {
	h.T.Helper()
	h.WaitForResponse()

	if h.Returned != nil {
		if !_reflect.DeepEqual(h.Returned.Result0, v0) {
			h.T.Fatalf("expected return value 0 to be %v, got %v", v0, h.Returned.Result0)
		}
		return
	}

	h.T.Fatalf("expected function to return, but it panicked with: %v", h.Panicked)
}

// ExpectReturnMatch verifies the return values match the given matchers.
func (h *WrapSlowMultiplyCallHandle) ExpectReturnMatch(v0 any) {
	h.T.Helper()
	h.WaitForResponse()

	if h.Returned != nil {
		var ok bool
		var msg string
		ok, msg = _imptest.MatchValue(h.Returned.Result0, v0)
		if !ok {
			h.T.Fatalf("return value 0: %s", msg)
		}
		return
	}

	h.T.Fatalf("expected function to return, but it panicked with: %v", h.Panicked)
}

// ExpectPanic verifies the function panics with the expected value.
func (h *WrapSlowMultiplyCallHandle) ExpectPanic(expected any) {
	h.T.Helper()
	h.WaitForResponse()

	if h.Panicked != nil {
		ok, msg := _imptest.MatchValue(h.Panicked, expected)
		if !ok {
			h.T.Fatalf("panic value: %s", msg)
		}
		return
	}

	h.T.Fatalf("expected function to panic, but it returned")
}

// ExpectPanicMatch verifies the function panics with a value matching the given matcher.
func (h *WrapSlowMultiplyCallHandle) ExpectPanicMatch(matcher any) {
	h.T.Helper()
	h.WaitForResponse()

	if h.Panicked != nil {
		ok, msg := _imptest.MatchValue(h.Panicked, matcher)
		if !ok {
			h.T.Fatalf("panic value: %s", msg)
		}
		return
	}

	h.T.Fatalf("expected function to panic, but it returned")
}

// WrapSlowMultiply wraps a function for testing.
func WrapSlowMultiply(t _imptest.TestReporter, fn func(a int, delay time.Duration) int) *WrapSlowMultiplyWrapperHandle {
	return &WrapSlowMultiplyWrapperHandle{
		t:        t,
		callable: fn,
	}
}
