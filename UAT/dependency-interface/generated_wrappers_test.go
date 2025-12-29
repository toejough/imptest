// Code generated manually for v2 API demonstration. DO NOT EDIT.

package dependencyinterface_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapLoadAndProcess creates a wrapper for the LoadAndProcess method.
func WrapLoadAndProcess(imp imptest.TestReporter, fn func(int) (string, error)) *LoadAndProcessWrapper {
	return &LoadAndProcessWrapper{
		imp: imp.(*imptest.Imp),
		fn:  fn,
	}
}

// LoadAndProcessWrapper wraps a LoadAndProcess function for testing.
type LoadAndProcessWrapper struct {
	imp        *imptest.Imp
	fn         func(int) (string, error)
	returnChan chan LoadAndProcessReturns
	panicChan  chan any
	returned   *LoadAndProcessReturns
	panicked   any
}

// LoadAndProcessReturns holds the return values from LoadAndProcess.
type LoadAndProcessReturns struct {
	R1 string
	R2 error
}

// Start executes the function in a goroutine and returns the wrapper for chaining.
func (w *LoadAndProcessWrapper) Start(id int) *LoadAndProcessWrapper {
	w.returnChan = make(chan LoadAndProcessReturns, 1)
	w.panicChan = make(chan any, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()
		r1, r2 := w.fn(id)
		w.returnChan <- LoadAndProcessReturns{R1: r1, R2: r2}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *LoadAndProcessWrapper) WaitForResponse() {
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
func (w *LoadAndProcessWrapper) ExpectReturnsEqual(expectedR1 string, expectedR2 error) {
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

// WrapSaveProcessed creates a wrapper for the SaveProcessed method.
func WrapSaveProcessed(imp imptest.TestReporter, fn func(int, string) error) *SaveProcessedWrapper {
	return &SaveProcessedWrapper{
		imp: imp.(*imptest.Imp),
		fn:  fn,
	}
}

// SaveProcessedWrapper wraps a SaveProcessed function for testing.
type SaveProcessedWrapper struct {
	imp        *imptest.Imp
	fn         func(int, string) error
	returnChan chan SaveProcessedReturns
	panicChan  chan any
	returned   *SaveProcessedReturns
	panicked   any
}

// SaveProcessedReturns holds the return values from SaveProcessed.
type SaveProcessedReturns struct {
	R1 error
}

// Start executes the function in a goroutine and returns the wrapper for chaining.
func (w *SaveProcessedWrapper) Start(id int, input string) *SaveProcessedWrapper {
	w.returnChan = make(chan SaveProcessedReturns, 1)
	w.panicChan = make(chan any, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()
		r1 := w.fn(id, input)
		w.returnChan <- SaveProcessedReturns{R1: r1}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *SaveProcessedWrapper) WaitForResponse() {
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
func (w *SaveProcessedWrapper) ExpectReturnsEqual(expectedR1 error) {
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

// ExpectPanicEquals verifies the function panicked with an exact value.
func (w *LoadAndProcessWrapper) ExpectPanicEquals(expected any) {
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
