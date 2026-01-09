// Code generated manually for UAT-42. DO NOT EDIT.

package callable_test

import (
	_imptest "github.com/toejough/imptest/imptest"
)

// WrapPanicWithMessageReturnsReturn holds the return values from the wrapped function.
type WrapPanicWithMessageReturnsReturn struct {
}

// WrapPanicWithMessageWrapper wraps a function for testing.
type WrapPanicWithMessageWrapperHandle struct {
	Method     *WrapPanicWithMessageWrapperMethod
	Controller *_imptest.TargetController
}

type WrapPanicWithMessageWrapperMethod struct {
	t          _imptest.TestReporter
	controller *_imptest.TargetController
	callable   func(msg string)
}

// WrapPanicWithMessageCallHandle represents a single call to the wrapped function.
type WrapPanicWithMessageCallHandle struct {
	*_imptest.CallableController[WrapPanicWithMessageReturnsReturn]
	controller        *_imptest.TargetController
	pendingCompletion *_imptest.PendingCompletion
}

// Eventually returns a pending completion for async expectation registration.
func (h *WrapPanicWithMessageCallHandle) Eventually() *_imptest.PendingCompletion {
	if h.pendingCompletion == nil {
		h.pendingCompletion = h.controller.RegisterPendingCompletion()
		// Start a goroutine to wait for completion and notify the pending completion
		go func() {
			h.WaitForResponse()
			h.pendingCompletion.SetCompleted(h.Returned, h.Panicked)
		}()
	}

	return h.pendingCompletion
}

// Start executes the wrapped function in a goroutine.
func (w *WrapPanicWithMessageWrapperMethod) Start(msg string) *WrapPanicWithMessageCallHandle {
	handle := &WrapPanicWithMessageCallHandle{
		CallableController: _imptest.NewCallableController[WrapPanicWithMessageReturnsReturn](w.t),
		controller:         w.controller,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		w.callable(msg)
		handle.ReturnChan <- WrapPanicWithMessageReturnsReturn{}
	}()

	return handle
}

// ExpectPanicEquals verifies the function panics with the expected value.
func (h *WrapPanicWithMessageCallHandle) ExpectPanicEquals(expected any) {
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
func (h *WrapPanicWithMessageCallHandle) ExpectPanicMatches(matcher any) {
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

// WrapPanicWithMessage wraps a function for testing.
func WrapPanicWithMessage(t _imptest.TestReporter, fn func(msg string)) *WrapPanicWithMessageWrapperHandle {
	ctrl := _imptest.NewTargetController(t)

	return &WrapPanicWithMessageWrapperHandle{
		Method: &WrapPanicWithMessageWrapperMethod{
			t:          t,
			controller: ctrl,
			callable:   fn,
		},
		Controller: ctrl,
	}
}
