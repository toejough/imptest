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
	t        _imptest.TestReporter
	callable func()
}

// WrapPanicCallHandle represents a single call to the wrapped function.
type WrapPanicCallHandle struct {
	*_imptest.CallableController[WrapPanicReturnsReturn]
}

// Start executes the wrapped function in a goroutine.
func (w *WrapPanicWrapperHandle) Start() *WrapPanicCallHandle {
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

// ExpectPanic verifies the function panics with the expected value.
func (h *WrapPanicCallHandle) ExpectPanic(expected any) {
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
func (h *WrapPanicCallHandle) ExpectPanicMatch(matcher any) {
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
		t:        t,
		callable: fn,
	}
}
