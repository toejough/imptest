// Code generated manually for UAT-42. DO NOT EDIT.

package callhandle_test

import (
	_reflect "reflect"

	_imptest "github.com/toejough/imptest/imptest"
)

// WrapDivideReturnsReturn holds the return values from the wrapped function.
type WrapDivideReturnsReturn struct {
	Result0 int
	Result1 bool
}

// WrapDivideWrapper wraps a function for testing.
type WrapDivideWrapper struct {
	t        _imptest.TestReporter
	callable func(a, b int) (int, bool)
}

// WrapDivideCallHandle represents a single call to the wrapped function.
type WrapDivideCallHandle struct {
	*_imptest.CallableController[WrapDivideReturnsReturn]
}

// Start executes the wrapped function in a goroutine.
func (w *WrapDivideWrapper) Start(a, b int) *WrapDivideCallHandle {
	handle := &WrapDivideCallHandle{
		CallableController: _imptest.NewCallableController[WrapDivideReturnsReturn](w.t),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		ret0, ret1 := w.callable(a, b)
		handle.ReturnChan <- WrapDivideReturnsReturn{Result0: ret0, Result1: ret1}
	}()
	return handle
}

// ExpectReturnsEqual verifies the function returned the expected values.
func (h *WrapDivideCallHandle) ExpectReturnsEqual(v0 int, v1 bool) {
	h.T.Helper()
	h.WaitForResponse()

	if h.Returned != nil {
		if !_reflect.DeepEqual(h.Returned.Result0, v0) {
			h.T.Fatalf("expected return value 0 to be %v, got %v", v0, h.Returned.Result0)
		}
		if !_reflect.DeepEqual(h.Returned.Result1, v1) {
			h.T.Fatalf("expected return value 1 to be %v, got %v", v1, h.Returned.Result1)
		}
		return
	}

	h.T.Fatalf("expected function to return, but it panicked with: %v", h.Panicked)
}

// ExpectReturnsMatch verifies the return values match the given matchers.
func (h *WrapDivideCallHandle) ExpectReturnsMatch(v0 any, v1 any) {
	h.T.Helper()
	h.WaitForResponse()

	if h.Returned != nil {
		var ok bool
		var msg string
		ok, msg = _imptest.MatchValue(h.Returned.Result0, v0)
		if !ok {
			h.T.Fatalf("return value 0: %s", msg)
		}
		ok, msg = _imptest.MatchValue(h.Returned.Result1, v1)
		if !ok {
			h.T.Fatalf("return value 1: %s", msg)
		}
		return
	}

	h.T.Fatalf("expected function to return, but it panicked with: %v", h.Panicked)
}

// ExpectPanicEquals verifies the function panics with the expected value.
func (h *WrapDivideCallHandle) ExpectPanicEquals(expected any) {
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

// ExpectPanicMatches verifies the function panics with a value matching the given matcher.
func (h *WrapDivideCallHandle) ExpectPanicMatches(matcher any) {
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

// WrapDivide wraps a function for testing.
func WrapDivide(t _imptest.TestReporter, fn func(a, b int) (int, bool)) *WrapDivideWrapper {
	return &WrapDivideWrapper{
		t:        t,
		callable: fn,
	}
}
