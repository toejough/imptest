// Code generated manually for UAT-42. DO NOT EDIT.

package callable_test

import (
	_imptest "github.com/toejough/imptest"
)

// WrapSideEffectReturnsReturn holds the return values from the wrapped function.
type WrapSideEffectReturnsReturn struct {
}

// WrapSideEffectWrapper wraps a function for testing.
type WrapSideEffectWrapperHandle struct {
	t        _imptest.TestReporter
	callable func(x int)
}

// WrapSideEffectCallHandle represents a single call to the wrapped function.
type WrapSideEffectCallHandle struct {
	*_imptest.CallableController[WrapSideEffectReturnsReturn]
}

// Start executes the wrapped function in a goroutine.
func (w *WrapSideEffectWrapperHandle) Start(x int) *WrapSideEffectCallHandle {
	handle := &WrapSideEffectCallHandle{
		CallableController: _imptest.NewCallableController[WrapSideEffectReturnsReturn](w.t),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		w.callable(x)
		handle.ReturnChan <- WrapSideEffectReturnsReturn{}
	}()
	return handle
}

// ExpectPanic verifies the function panics with the expected value.
func (h *WrapSideEffectCallHandle) ExpectPanic(expected any) {
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
func (h *WrapSideEffectCallHandle) ExpectPanicMatch(matcher any) {
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

// WrapSideEffect wraps a function for testing.
func WrapSideEffect(t _imptest.TestReporter, fn func(x int)) *WrapSideEffectWrapperHandle {
	return &WrapSideEffectWrapperHandle{
		t:        t,
		callable: fn,
	}
}
