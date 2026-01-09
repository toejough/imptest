// Code generated manually for UAT-42. DO NOT EDIT.

package callable_test

import (
	_imptest "github.com/toejough/imptest"
)

// WrapPanicReturnsReturn holds the return values from the wrapped function.
type WrapPanicReturnsReturn struct {
}

// WrapPanicWrapper wraps a function for testing.
type WrapPanicWrapperHandle struct {
	Method *WrapPanicWrapperMethod
}

type WrapPanicWrapperMethod struct {
	t        _imptest.TestReporter
	callable func()
}

// WrapPanicCallHandle represents a single call to the wrapped function.
type WrapPanicCallHandle struct {
	*_imptest.CallableController[WrapPanicReturnsReturn]
}

// Start executes the wrapped function in a goroutine.
func (w *WrapPanicWrapperMethod) Start() *WrapPanicCallHandle {
	handle := &WrapPanicCallHandle{
		CallableController: _imptest.NewCallableController[WrapPanicReturnsReturn](w.t),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		w.callable()
		handle.ReturnChan <- WrapPanicReturnsReturn{}
	}()
	return handle
}

// ExpectPanicEquals verifies the function panics with the expected value.
func (h *WrapPanicCallHandle) ExpectPanicEquals(expected any) {
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
func (h *WrapPanicCallHandle) ExpectPanicMatches(matcher any) {
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

// WrapPanic wraps a function for testing.
func WrapPanic(t _imptest.TestReporter, fn func()) *WrapPanicWrapperHandle {
	return &WrapPanicWrapperHandle{
		Method: &WrapPanicWrapperMethod{
			t:        t,
			callable: fn,
		},
	}
}
