// Code generated manually for UAT-42. DO NOT EDIT.

package callable_test

import (
	_reflect "reflect"

	_imptest "github.com/toejough/imptest/imptest"
)

// WrapMultiplyReturnsReturn holds the return values from the wrapped function.
type WrapMultiplyReturnsReturn struct {
	Result0 int
}

// WrapMultiplyWrapper wraps a function for testing.
type WrapMultiplyWrapperHandle struct {
	Method *WrapMultiplyWrapperMethod
}

type WrapMultiplyWrapperMethod struct {
	t        _imptest.TestReporter
	callable func(a, b int) int
}

// WrapMultiplyCallHandle represents a single call to the wrapped function.
type WrapMultiplyCallHandle struct {
	*_imptest.CallableController[WrapMultiplyReturnsReturn]
}

// Start executes the wrapped function in a goroutine.
func (w *WrapMultiplyWrapperMethod) Start(a, b int) *WrapMultiplyCallHandle {
	handle := &WrapMultiplyCallHandle{
		CallableController: _imptest.NewCallableController[WrapMultiplyReturnsReturn](w.t),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		ret0 := w.callable(a, b)
		handle.ReturnChan <- WrapMultiplyReturnsReturn{Result0: ret0}
	}()
	return handle
}

// ExpectReturnsEqual verifies the function returned the expected values.
func (h *WrapMultiplyCallHandle) ExpectReturnsEqual(v0 int) {
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

// ExpectReturnsMatch verifies the return values match the given matchers.
func (h *WrapMultiplyCallHandle) ExpectReturnsMatch(v0 any) {
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

// ExpectPanicEquals verifies the function panics with the expected value.
func (h *WrapMultiplyCallHandle) ExpectPanicEquals(expected any) {
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
func (h *WrapMultiplyCallHandle) ExpectPanicMatches(matcher any) {
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

// WrapMultiply wraps a function for testing.
func WrapMultiply(t _imptest.TestReporter, fn func(a, b int) int) *WrapMultiplyWrapperHandle {
	return &WrapMultiplyWrapperHandle{
		Method: &WrapMultiplyWrapperMethod{
			t:        t,
			callable: fn,
		},
	}
}
