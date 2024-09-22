// Package imptest provides impure function testing functionality.
package imptest

import (
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
	"time"
)

func WithName(name string) func(string) string {
	return func(_ string) string {
		return name
	}
}

type WrapOption func(string) string

func WrapFunc[T any](function T, calls chan YieldedValue, options ...WrapOption) (T, string) {
	// creates a unique ID for the function
	funcID := GetFuncName(function)
	for _, o := range options {
		funcID = o(funcID)
	}

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := reflect.TypeOf(function)

	relayer := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive output values on
		outputValuesChan := make(chan injectedValue)

		// Submit this call to the calls channel
		calls <- YieldedValue{
			YieldedCall,
			funcID,
			unreflectValues(args),
			nil,
			nil,
			outputValuesChan,
		}

		outputV := <-outputValuesChan

		switch outputV.Type {
		case InjectedReturn:
			returnValues := make([]reflect.Value, len(outputV.ReturnValues))

			// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
			for i, a := range outputV.ReturnValues {
				returnValues[i] = reflect.ValueOf(a)
			}

			return returnValues
		// if we're supposed to panic, do.
		case InjectedPanic:
			panic(outputV.PanicValue)
		default:
			panic("imptest failure - unrecognized outputValue type was passed")
		}
	}

	// Make a function of the right type.
	// Ignore the type assertion lint check - we are depending on MakeFunc to
	// return the correct type, as documented. If it fails to, the only thing
	// we'd do is panic anyway.
	wrapped := reflect.MakeFunc(funcType, relayer).Interface().(T) //nolint:forcetypeassert

	// returns both the wrapped func and the ID
	return wrapped, funcID
}

type yieldType int

const (
	YieldedReturn yieldType = iota
	YieldedPanic  yieldType = iota
	YieldedCall   yieldType = iota
)

type injectionType int

const (
	InjectedReturn injectionType = iota
	InjectedPanic  injectionType = iota
)

type injectedValue struct {
	Type         injectionType
	ReturnValues []any
	PanicValue   any
}

type YieldedValue struct {
	Type            yieldType
	ID              string
	Args            []any
	panicVal        any
	returnVals      []any
	outputValueChan chan injectedValue
}

func (out *YieldedValue) String() string {
	switch out.Type {
	case YieldedCall:
		return strings.Join([]string{
			"call",
			"with name",
			out.ID,
			"with args",
			fmt.Sprintf("%#v", out.Args),
		}, "\n")
	case YieldedReturn:
		return strings.Join([]string{
			"return",
			"with values",
			fmt.Sprintf("%#v", out.returnVals),
		}, "\n")
	case YieldedPanic:
		return strings.Join([]string{
			"panic",
			"with value",
			fmt.Sprintf("%#v", out.panicVal),
		}, "\n")
	default:
		panic("got an unexpected output type")
	}
}

// NewFuncTester returns a newly initialized FuncTester.
func NewFuncTester(tester Tester, options ...FuncTesterOption) *FuncTester {
	tester.Helper()

	calls := make(chan YieldedValue)

	funcTester := new(FuncTester)
	funcTester.T = tester
	funcTester.OutputChan = calls
	funcTester.bufferMaxLen = 1
	// I want this to be a magic number, it's half a second
	funcTester.timeout = 500 * time.Millisecond //nolint:mnd,gomnd
	funcTester.comparator = reflect.DeepEqual

	for _, o := range options {
		funcTester = o(funcTester)
	}

	return funcTester
}

type FuncTesterOption func(*FuncTester) *FuncTester

func WithTimeout(timeout time.Duration) FuncTesterOption {
	return func(ft *FuncTester) *FuncTester {
		ft.timeout = timeout
		return ft
	}
}

func WithComparator(comp func(any, any) bool) FuncTesterOption {
	return func(ft *FuncTester) *FuncTester {
		ft.comparator = comp
		return ft
	}
}

type Tester interface {
	Helper()
	Fatal(args ...any)
	Fatalf(message string, args ...any)
	Logf(message string, args ...any)
}

// Tester contains the *testing.T and the chan FuncCall.
type FuncTester struct {
	T               Tester
	OutputChan      chan YieldedValue
	outputBuffer    []YieldedValue
	bufferMaxLen    int
	bufferNextIndex int
	timeout         time.Duration
	hasReturned     bool
	returnedVals    []any
	hasPanicked     bool
	panickedVal     any
	comparator      func(any, any) bool
}

func (t *FuncTester) Timeout() time.Duration {
	return t.timeout
}

// Start starts the function.
func (t *FuncTester) Start(function any, args ...any) {
	// record when the func is done so we can test that, too
	go func() {
		var rVals []any

		defer func() {
			panicVal := recover()
			if panicVal != nil {
				t.OutputChan <- YieldedValue{
					YieldedPanic,
					"",
					nil,
					panicVal,
					nil,
					nil,
				}
			} else {
				t.OutputChan <- YieldedValue{
					YieldedReturn,
					"",
					nil,
					nil,
					rVals,
					nil,
				}
			}
		}()

		rVals = callFunc(function, args)
	}()
}

// Called returns the FuncCall for inspection by the test.
func (t *FuncTester) Called() YieldedValue {
	for next := range t.iterOut() {
		if next.Type == YieldedCall {
			return next
		}
	}

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) YieldedValue {
	t.T.Helper()

	for next := range t.iterOut() {
		if next.Type == YieldedCall {
			if next.ID == expectedCallID && t.comparator(next.Args, expectedArgs) {
				return next
			}
		}
	}

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
}

func formatOutput(outputs []YieldedValue) string {
	formatted := []string{}

	for _, funcOut := range outputs {
		formatted = append(formatted, funcOut.String())
	}

	return strings.Join(formatted, "\n")
}

func (t *FuncTester) Close() {
	close(t.OutputChan)
}

// Return returns the given values in the func call.
func (out YieldedValue) Return(returnVals ...any) {
	out.outputValueChan <- injectedValue{
		InjectedReturn,
		returnVals,
		nil,
	}
	close(out.outputValueChan)
}

func (t *FuncTester) Returned() []any {
	t.T.Helper()

	if t.hasReturned {
		return t.returnedVals
	}

	for next := range t.iterOut() {
		if next.Type == YieldedReturn {
			t.returnedVals = next.returnVals
			t.hasReturned = true

			return t.returnedVals
		}
	}

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
}

// AssertReturned asserts that the function under test returned the given values.
func (t *FuncTester) AssertReturned(expectedReturnValues ...any) {
	t.T.Helper()

	returnVals := t.Returned()

	if !t.comparator(expectedReturnValues, returnVals) {
		t.T.Fatalf("\n"+
			"Looking for the function to return\n"+
			"  with %#v,\n"+
			"but it returned with %#v instead.\n",
			expectedReturnValues,
			returnVals,
		)
	}
}

// Panic makes the func call result in a panic with the given value.
func (out YieldedValue) Panic(panicVal any) {
	out.outputValueChan <- injectedValue{
		InjectedPanic,
		nil,
		panicVal,
	}
	close(out.outputValueChan)
}

// Panicked returns the panicked value.
func (t *FuncTester) Panicked() any {
	t.T.Helper()

	if t.hasPanicked {
		return t.panickedVal
	}

	for next := range t.iterOut() {
		if next.Type == YieldedPanic {
			t.panickedVal = next.panicVal
			t.hasPanicked = true

			return t.panickedVal
		}
	}

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
}

func (t *FuncTester) iterOut() iter.Seq[YieldedValue] {
	return func(yield func(YieldedValue) bool) {
		t.T.Helper()
		t.bufferNextIndex = 0

		for {
			next := t.nextOutput()
			if !yield(next) {
				t.outputBuffer = slices.Delete(t.outputBuffer, t.bufferNextIndex, t.bufferNextIndex+1)
				return
			}

			t.bufferNextIndex++
		}
	}
}

// AssertPanicked asserts that the function under test paniced with the given value.
func (t *FuncTester) AssertPanicked(expectedPanic any) {
	t.T.Helper()

	panicVal := t.Panicked()

	if !t.comparator(expectedPanic, panicVal) {
		t.T.Fatalf("\n"+
			"Looking for the function to panic\n"+
			"  with %#v,\n"+
			"but it panicked with %#v instead.\n",
			expectedPanic,
			panicVal,
		)
	}
}

// nextOutput gets the next output from the queue or the func outputs.
func (t *FuncTester) nextOutput() YieldedValue {
	t.T.Helper()

	if t.bufferNextIndex < len(t.outputBuffer) {
		return t.outputBuffer[t.bufferNextIndex]
	}

	for len(t.outputBuffer) < t.bufferMaxLen {
		select {
		case actualOutput, open := <-t.OutputChan:
			if !open {
				t.T.Fatal("expected an output to be available, but the outputs channel was already closed")
				panic("only necessary because nilchecker doesn't know what to do with my mocked tester")
			}

			t.outputBuffer = append(t.outputBuffer, actualOutput)

			return actualOutput
		case <-time.After(t.timeout):
			logMessage := fmt.Sprintf(
				"\n"+
					"Looking for output\n"+
					"but the test timed out with a queue with %s.\n"+
					"bufferMaxLen: %d.\n"+
					"bufferNextIndex: %d\n"+
					"timeout: %v",
				formatOutput(t.outputBuffer),
				t.bufferMaxLen,
				t.bufferNextIndex,
				t.timeout,
			)

			t.T.Fatalf(logMessage)
			// t.T.Fatalf("expected a call to be available, but the test timed out waiting after %v", t.timeout)
			panic("only necessary because linters don't know what to do with my mocked tester")
		}
	}

	// TODO: for assertion functions, make it clear what output was being looked for
	// TODO: for normal getter functions, make it clear what kind of output was being looked for
	t.T.Fatalf(
		"\n"+
			"Looking for an output\n"+
			"but it was not found with a queue with %s.\n"+
			"bufferMaxLen: %d.\n"+
			"bufferNextIndex: %d\n"+
			"timeout: %v",
		formatOutput(t.outputBuffer),
		t.bufferMaxLen,
		t.bufferNextIndex,
		t.timeout,
	)
	panic("this is only necessary because nothing knows what to do with the mocked test type")
}

// Concurrently marks the current size of the call queue, such that assertion
// calls made within the passed functions only start from the marked location
// in the queue. It also limits the maximum size of the queue by the number of
// concurrent functions that have yet to complete. It is nestable - nested
// calls to Concurrently will push a new mark onto a queue of marks, and pop it
// off when complete.
func (t *FuncTester) Concurrently(funcs ...func()) {
	// add len(funcs) for each func we just added
	t.bufferMaxLen += len(funcs)

	// run each function.
	for _, concurrentCheck := range funcs {
		// reduce the t.bufferMaxLen. The expectation for concurrently is that
		// you have spun off some goroutines and are managing the expected
		// concurrent calls now from within the concurrently's functions.
		// Imagine each expected goroutine has a concurrent-call token. For the
		// first iteration, then, we're removing the calling goroutine's token.
		// Subsequent loops remove the prior function's token. That leaves a
		// single token at the end of the cycle, which is effectively returned
		// to the calling goroutine.
		t.bufferMaxLen--
		// run the func!
		concurrentCheck()
	}
}
