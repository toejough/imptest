// Code generated manually for UAT-42. DO NOT EDIT.

package callable_test

import (
	"time"

	_reflect "reflect"

	_imptest "github.com/toejough/imptest/imptest"
)

// WrapSlowAddReturnsReturn holds the return values from the wrapped function.
type WrapSlowAddReturnsReturn struct {
	Result0 int
}

// WrapSlowAddWrapper wraps a function for testing.
type WrapSlowAddWrapperHandle struct {
	Method     *WrapSlowAddWrapperMethod
	Controller *_imptest.TargetController
}

type WrapSlowAddWrapperMethod struct {
	t          _imptest.TestReporter
	controller *_imptest.TargetController
	callable   func(a, b int, delay time.Duration) int
}

// WrapSlowAddCallHandle represents a single call to the wrapped function.
type WrapSlowAddCallHandle struct {
	*_imptest.CallableController[WrapSlowAddReturnsReturn]
	controller        *_imptest.TargetController
	pendingCompletion *_imptest.PendingCompletion
	// Eventually is the async version of this call handle for registering non-blocking expectations.
	Eventually *WrapSlowAddCallHandleEventually
}

// WrapSlowAddCallHandleEventually wraps a call handle for async expectation registration.
type WrapSlowAddCallHandleEventually struct {
	h *WrapSlowAddCallHandle
}

func (e *WrapSlowAddCallHandleEventually) ensureStarted() *_imptest.PendingCompletion {
	if e.h.pendingCompletion == nil {
		e.h.pendingCompletion = e.h.controller.RegisterPendingCompletion()
		go func() {
			e.h.WaitForResponse()
			e.h.pendingCompletion.SetCompleted(e.h.Returned, e.h.Panicked)
		}()
	}
	return e.h.pendingCompletion
}

// ExpectReturnsEqual registers an async expectation for return values.
func (e *WrapSlowAddCallHandleEventually) ExpectReturnsEqual(values ...any) {
	e.ensureStarted().ExpectReturnsEqual(values...)
}

// ExpectPanicEquals registers an async expectation for a panic value.
func (e *WrapSlowAddCallHandleEventually) ExpectPanicEquals(value any) {
	e.ensureStarted().ExpectPanicEquals(value)
}

// Start executes the wrapped function in a goroutine.
func (w *WrapSlowAddWrapperMethod) Start(a, b int, delay time.Duration) *WrapSlowAddCallHandle {
	handle := &WrapSlowAddCallHandle{
		CallableController: _imptest.NewCallableController[WrapSlowAddReturnsReturn](w.t),
		controller:         w.controller,
	}
	handle.Eventually = &WrapSlowAddCallHandleEventually{h: handle}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handle.PanicChan <- r
			}
		}()
		ret0 := w.callable(a, b, delay)
		handle.ReturnChan <- WrapSlowAddReturnsReturn{Result0: ret0}
	}()

	return handle
}

// ExpectReturnsEqual verifies the function returned the expected values.
func (h *WrapSlowAddCallHandle) ExpectReturnsEqual(v0 int) {
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
func (h *WrapSlowAddCallHandle) ExpectReturnsMatch(v0 any) {
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
func (h *WrapSlowAddCallHandle) ExpectPanicEquals(expected any) {
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
func (h *WrapSlowAddCallHandle) ExpectPanicMatches(matcher any) {
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

// WrapSlowAdd wraps a function for testing.
func WrapSlowAdd(t _imptest.TestReporter, fn func(a, b int, delay time.Duration) int) *WrapSlowAddWrapperHandle {
	ctrl := _imptest.NewTargetController(t)

	return &WrapSlowAddWrapperHandle{
		Method: &WrapSlowAddWrapperMethod{
			t:          t,
			controller: ctrl,
			callable:   fn,
		},
		Controller: ctrl,
	}
}
