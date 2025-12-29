package targetfunction_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapBinaryOp wraps a BinaryOp function for testing.
func WrapBinaryOp(t imptest.TestReporter, fn BinaryOp) *WrapBinaryOpWrapper {
	return &WrapBinaryOpWrapper{
		imp:        imptest.NewImp(t),
		fn:         fn,
		returnChan: make(chan WrapBinaryOpReturns, 1),
		panicChan:  make(chan any, 1),
	}
}

// WrapBinaryOpWrapper provides a fluent API for calling and verifying BinaryOp.
type WrapBinaryOpWrapper struct {
	imp        *imptest.Imp
	fn         BinaryOp
	returnChan chan WrapBinaryOpReturns
	panicChan  chan any
	returned   *WrapBinaryOpReturns
	panicked   any
}

// Start begins execution of the function in a goroutine with the provided arguments.
func (w *WrapBinaryOpWrapper) Start(arg1, arg2 int) *WrapBinaryOpWrapper {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()

		result := w.fn(arg1, arg2)
		w.returnChan <- WrapBinaryOpReturns{R1: result}
	}()

	return w
}

// WaitForResponse blocks until the function completes (return or panic).
func (w *WrapBinaryOpWrapper) WaitForResponse() {
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
func (w *WrapBinaryOpWrapper) ExpectReturnsEqual(expected int) {
	w.imp.Helper()
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to return, but it panicked with: %v", w.panicked)
		return
	}

	if w.returned.R1 != expected {
		w.imp.Fatalf("expected %v, got %v", expected, w.returned.R1)
	}
}

// WrapBinaryOpReturns provides type-safe access to return values.
type WrapBinaryOpReturns struct {
	R1 int
}

// WrapAdd wraps the Add function for testing.
func WrapAdd(t imptest.TestReporter, fn func(int, int) int) *WrapAddWrapper {
	return &WrapAddWrapper{
		imp:        imptest.NewImp(t),
		fn:         fn,
		returnChan: make(chan WrapAddReturns, 1),
		panicChan:  make(chan any, 1),
	}
}

// WrapAddWrapper provides a fluent API for calling and verifying Add.
type WrapAddWrapper struct {
	imp        *imptest.Imp
	fn         func(int, int) int
	returnChan chan WrapAddReturns
	panicChan  chan any
	returned   *WrapAddReturns
	panicked   any
}

// Start begins execution of the function in a goroutine.
func (w *WrapAddWrapper) Start(arg1, arg2 int) *WrapAddWrapper {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()

		result := w.fn(arg1, arg2)
		w.returnChan <- WrapAddReturns{R1: result}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *WrapAddWrapper) WaitForResponse() {
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

// ExpectReturnsEqual verifies exact return values.
func (w *WrapAddWrapper) ExpectReturnsEqual(expected int) {
	w.imp.Helper()
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to return, but it panicked with: %v", w.panicked)
		return
	}

	if w.returned.R1 != expected {
		w.imp.Fatalf("expected %v, got %v", expected, w.returned.R1)
	}
}

// ExpectReturnsMatch verifies return values match matchers.
func (w *WrapAddWrapper) ExpectReturnsMatch(matchers ...any) {
	w.imp.Helper()
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to return, but it panicked with: %v", w.panicked)
		return
	}

	if len(matchers) != 1 {
		w.imp.Fatalf("expected 1 matcher, got %d", len(matchers))
		return
	}

	matcher, ok := matchers[0].(imptest.Matcher)
	if !ok {
		w.imp.Fatalf("argument 0 is not a Matcher")
		return
	}

	success, err := matcher.Match(w.returned.R1)
	if err != nil {
		w.imp.Fatalf("matcher error: %v", err)
		return
	}

	if !success {
		w.imp.Fatalf("return value: %s", matcher.FailureMessage(w.returned.R1))
	}
}

// WrapAddReturns holds return values.
type WrapAddReturns struct {
	R1 int
}

// GetReturns returns type-safe return values.
func (w *WrapAddWrapper) GetReturns() *WrapAddReturns {
	w.imp.Helper()
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("cannot get returns: function panicked with %v", w.panicked)
		return nil
	}

	return w.returned
}

// WrapDivide wraps the Divide function for testing.
func WrapDivide(t imptest.TestReporter, fn func(int, int) int) *WrapDivideWrapper {
	return &WrapDivideWrapper{
		imp:        imptest.NewImp(t),
		fn:         fn,
		returnChan: make(chan WrapDivideReturns, 1),
		panicChan:  make(chan any, 1),
	}
}

// WrapDivideWrapper provides a fluent API for calling and verifying Divide.
type WrapDivideWrapper struct {
	imp        *imptest.Imp
	fn         func(int, int) int
	returnChan chan WrapDivideReturns
	panicChan  chan any
	returned   *WrapDivideReturns
	panicked   any
}

// Start begins execution of the function in a goroutine.
func (w *WrapDivideWrapper) Start(arg1, arg2 int) *WrapDivideWrapper {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()

		result := w.fn(arg1, arg2)
		w.returnChan <- WrapDivideReturns{R1: result}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *WrapDivideWrapper) WaitForResponse() {
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

// ExpectPanicEquals verifies the function panicked with an exact value.
func (w *WrapDivideWrapper) ExpectPanicEquals(expected any) {
	w.imp.Helper()
	w.WaitForResponse()

	if w.returned != nil {
		w.imp.Fatalf("expected function to panic, but it returned normally")
		return
	}

	if w.panicked != expected {
		w.imp.Fatalf("expected panic with %v, got %v", expected, w.panicked)
	}
}

// ExpectPanicMatches verifies the panic value matches the given matcher.
func (w *WrapDivideWrapper) ExpectPanicMatches(matcher imptest.Matcher) {
	w.imp.Helper()
	w.WaitForResponse()

	if w.returned != nil {
		w.imp.Fatalf("expected function to panic, but it returned normally")
		return
	}

	success, err := matcher.Match(w.panicked)
	if err != nil {
		w.imp.Fatalf("panic matcher error: %v", err)
		return
	}

	if !success {
		w.imp.Fatalf("panic value: %s", matcher.FailureMessage(w.panicked))
	}
}

// WrapDivideReturns holds return values.
type WrapDivideReturns struct {
	R1 int
}

// WrapConcurrent wraps the Concurrent function for testing.
func WrapConcurrent(t imptest.TestReporter, fn func(int) int) *WrapConcurrentWrapper {
	return &WrapConcurrentWrapper{
		imp:        imptest.NewImp(t),
		fn:         fn,
		returnChan: make(chan WrapConcurrentReturns, 1),
		panicChan:  make(chan any, 1),
	}
}

// WrapConcurrentWrapper provides a fluent API for calling and verifying Concurrent.
type WrapConcurrentWrapper struct {
	imp        *imptest.Imp
	fn         func(int) int
	returnChan chan WrapConcurrentReturns
	panicChan  chan any
	returned   *WrapConcurrentReturns
	panicked   any
}

// Start begins execution of the function in a goroutine.
func (w *WrapConcurrentWrapper) Start(value int) *WrapConcurrentWrapper {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.panicChan <- r
			}
		}()

		result := w.fn(value)
		w.returnChan <- WrapConcurrentReturns{R1: result}
	}()

	return w
}

// WaitForResponse blocks until the function completes.
func (w *WrapConcurrentWrapper) WaitForResponse() {
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

// ExpectReturnsEqual verifies exact return values.
func (w *WrapConcurrentWrapper) ExpectReturnsEqual(expected int) {
	w.imp.Helper()
	w.WaitForResponse()

	if w.panicked != nil {
		w.imp.Fatalf("expected function to return, but it panicked with: %v", w.panicked)
		return
	}

	if w.returned.R1 != expected {
		w.imp.Fatalf("expected %v, got %v", expected, w.returned.R1)
	}
}

// Eventually switches to unordered mode (placeholder for now).
func (w *WrapConcurrentWrapper) Eventually() *WrapConcurrentWrapper {
	// TODO: Implement unordered mode
	return w
}

// WrapConcurrentReturns holds return values.
type WrapConcurrentReturns struct {
	R1 int
}
