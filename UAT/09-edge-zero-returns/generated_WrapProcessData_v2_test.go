// Code generated manually for v2 API migration (Phase 2). DO NOT EDIT.
// TODO: Replace with template-generated code in future phase.

package zero_returns_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapProcessData wraps the ProcessData function for testing.
func WrapProcessData(testReporter imptest.TestReporter, fn func(string, int)) *WrapProcessDataWrapper {
	imp, ok := testReporter.(*imptest.Imp)
	if !ok {
		imp = imptest.NewImp(testReporter)
	}

	return &WrapProcessDataWrapper{
		imp:        imp,
		fn:         fn,
		returnChan: make(chan WrapProcessDataReturns, 1),
		panicChan:  make(chan any, 1),
	}
}

// WrapProcessDataWrapper provides a fluent API for calling and verifying ProcessData.
type WrapProcessDataWrapper struct {
	imp        *imptest.Imp
	fn         func(string, int)
	returnChan chan WrapProcessDataReturns
	panicChan  chan any
	returned   *WrapProcessDataReturns
	panicked   any
}

// Start begins execution of the function in a goroutine with the provided arguments.
func (w *WrapProcessDataWrapper) Start(data string, count int) *WrapProcessDataWrapper {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()

		w.fn(data, count)
		w.returnChan <- WrapProcessDataReturns{}
	}()

	return w
}

// WaitForCompletion blocks until the function completes (return or panic).
func (w *WrapProcessDataWrapper) WaitForCompletion() {
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

// ExpectCompletes verifies the function completed without panicking.
func (w *WrapProcessDataWrapper) ExpectCompletes() {
	w.imp.Helper()
	w.WaitForCompletion()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to complete, but it panicked with: %v", w.panicked)
	}
}

// ExpectPanicEquals verifies the function panicked with the exact value.
func (w *WrapProcessDataWrapper) ExpectPanicEquals(expected any) {
	w.imp.Helper()
	w.WaitForCompletion()

	if w.panicked == nil {
		w.imp.Fatalf("expected panic with %v, but function completed normally", expected)
		return
	}

	if w.panicked != expected {
		w.imp.Fatalf("expected panic with %v, got %v", expected, w.panicked)
	}
}

// WrapProcessDataReturns provides type-safe access to return values (empty for void functions).
type WrapProcessDataReturns struct{}
