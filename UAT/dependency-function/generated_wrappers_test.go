// Code generated manually for v2 API demonstration. DO NOT EDIT.

package dependencyfunction_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapProcessData creates a wrapper for the ProcessData function.
func WrapProcessData(imp imptest.TestReporter, fn func(int, Fetcher) (string, error)) *ProcessDataWrapper {
	return &ProcessDataWrapper{
		imp: imp.(*imptest.Imp),
		fn:  fn,
	}
}

// ProcessDataWrapper wraps a ProcessData function for testing.
type ProcessDataWrapper struct {
	imp        *imptest.Imp
	fn         func(int, Fetcher) (string, error)
	returnChan chan ProcessDataReturns
	panicChan  chan any
	returned   *ProcessDataReturns
	panicked   any
}

// ProcessDataReturns holds the return values from ProcessData.
type ProcessDataReturns struct {
	R1 string
	R2 error
}

// Start executes the function in a goroutine and returns the wrapper for chaining.
func (w *ProcessDataWrapper) Start(id int, fetcher Fetcher) *ProcessDataWrapper {
	w.returnChan = make(chan ProcessDataReturns, 1)
	w.panicChan = make(chan any, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()
		r1, r2 := w.fn(id, fetcher)
		w.returnChan <- ProcessDataReturns{R1: r1, R2: r2}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *ProcessDataWrapper) WaitForResponse() {
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

// ExpectReturnsEqual verifies the function returned exact values.
func (w *ProcessDataWrapper) ExpectReturnsEqual(expectedR1 string, expectedR2 error) {
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to return, but it panicked with: %v", w.panicked)
		return
	}

	if w.returned.R1 != expectedR1 {
		w.imp.Fatalf("return value 1: expected %q, got %q", expectedR1, w.returned.R1)
		return
	}

	if w.returned.R2 != expectedR2 {
		w.imp.Fatalf("return value 2: expected %v, got %v", expectedR2, w.returned.R2)
		return
	}
}

// ExpectPanicEquals verifies the function panicked with an exact value.
func (w *ProcessDataWrapper) ExpectPanicEquals(expected any) {
	w.WaitForResponse()

	if w.panicked == nil {
		w.imp.Fatalf("expected function to panic, but it returned normally")
		return
	}

	if w.panicked != expected {
		w.imp.Fatalf("expected panic with %v, got %v", expected, w.panicked)
		return
	}
}

// WrapValidateAndProcess creates a wrapper for the ValidateAndProcess function.
func WrapValidateAndProcess(imp imptest.TestReporter, fn func(int, Validator) bool) *ValidateAndProcessWrapper {
	return &ValidateAndProcessWrapper{
		imp: imp.(*imptest.Imp),
		fn:  fn,
	}
}

// ValidateAndProcessWrapper wraps a ValidateAndProcess function for testing.
type ValidateAndProcessWrapper struct {
	imp        *imptest.Imp
	fn         func(int, Validator) bool
	returnChan chan ValidateAndProcessReturns
	panicChan  chan any
	returned   *ValidateAndProcessReturns
	panicked   any
}

// ValidateAndProcessReturns holds the return values from ValidateAndProcess.
type ValidateAndProcessReturns struct {
	R1 bool
}

// Start executes the function in a goroutine and returns the wrapper for chaining.
func (w *ValidateAndProcessWrapper) Start(value int, validator Validator) *ValidateAndProcessWrapper {
	w.returnChan = make(chan ValidateAndProcessReturns, 1)
	w.panicChan = make(chan any, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()
		r1 := w.fn(value, validator)
		w.returnChan <- ValidateAndProcessReturns{R1: r1}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *ValidateAndProcessWrapper) WaitForResponse() {
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

// ExpectReturnsEqual verifies the function returned exact values.
func (w *ValidateAndProcessWrapper) ExpectReturnsEqual(expectedR1 bool) {
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to return, but it panicked with: %v", w.panicked)
		return
	}

	if w.returned.R1 != expectedR1 {
		w.imp.Fatalf("return value 1: expected %v, got %v", expectedR1, w.returned.R1)
		return
	}
}
