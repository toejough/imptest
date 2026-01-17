// Code generated manually for UAT-42. DO NOT EDIT.

package callable_test

import (
	_reflect "reflect"

	_imptest "github.com/toejough/imptest"
)

// WrapConditionalReturnsReturn holds the return values from the wrapped function.
type WrapConditionalReturnsReturn struct {
	Result0 int
}

// WrapConditionalWrapper wraps a function for testing.
type WrapConditionalWrapperHandle struct {
	t        _imptest.TestReporter
	callable func(x int) int
}

// WrapConditionalCallHandle represents a single call to the wrapped function.
type WrapConditionalCallHandle struct {
	*_imptest.CallableController[WrapConditionalReturnsReturn]
}

// Start executes the wrapped function in a goroutine.
func (w *WrapConditionalWrapperHandle) Start(x int) *WrapConditionalCallHandle {
	handle := &WrapConditionalCallHandle{
		CallableController: _imptest.NewCallableController[WrapConditionalReturnsReturn](w.t),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		ret0 := w.callable(x)
		handle.ReturnChan <- WrapConditionalReturnsReturn{Result0: ret0}
	}()
	return handle
}

// ExpectReturn verifies the function returned the expected values.
func (h *WrapConditionalCallHandle) ExpectReturn(v0 int) {
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
func (h *WrapConditionalCallHandle) ExpectReturnMatch(v0 any) {
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
func (h *WrapConditionalCallHandle) ExpectPanic(expected any) {
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
func (h *WrapConditionalCallHandle) ExpectPanicMatch(matcher any) {
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

// WrapConditional wraps a function for testing.
func WrapConditional(t _imptest.TestReporter, fn func(x int) int) *WrapConditionalWrapperHandle {
	return &WrapConditionalWrapperHandle{
		t:        t,
		callable: fn,
	}
}
