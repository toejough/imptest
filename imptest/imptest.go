// Package imptest provides impure function testing functionality.
package imptest

import (
	"fmt"
	"iter"
	"reflect"
	"runtime"
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
	funcID := getFuncName(function)
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
	funcTester.Timeout = 500 * time.Millisecond //nolint:mnd,gomnd
	funcTester.Differ = func(a, b any) string {
		if !reflect.DeepEqual(a, b) {
			return fmt.Sprintf("%#v != %#v", a, b)
		}

		return ""
	}

	for _, o := range options {
		funcTester = o(funcTester)
	}

	return funcTester
}

type FuncTesterOption func(*FuncTester) *FuncTester

func WithTimeout(timeout time.Duration) FuncTesterOption {
	return func(ft *FuncTester) *FuncTester {
		ft.Timeout = timeout
		return ft
	}
}

func WithDiffer(differ func(any, any) string) FuncTesterOption {
	return func(ft *FuncTester) *FuncTester {
		ft.Differ = differ
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
	T            Tester
	OutputChan   chan YieldedValue
	outputBuffer []YieldedValue
	bufferMaxLen int
	Timeout      time.Duration
	hasReturned  bool
	returnedVals []any
	hasPanicked  bool
	panickedVal  any
	Differ       Differ
}

func (t *FuncTester) SwapDiffer(d Differ) Differ {
	pd := t.Differ
	t.Differ = d

	return pd
}

type Differ func(any, any) string

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

	t.T.Fatalf("Expected a call, but none was found. Yielded outputs from the function: %s", formatOutput(t.outputBuffer))

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) YieldedValue {
	t.T.Helper()

	diffs := []string{}

	for next := range t.iterOut() {
		if next.Type == YieldedCall {
			if next.ID == expectedCallID {
				diff := t.Differ(next.Args, expectedArgs)
				if diff == "" {
					return next
				}

				diffs = append(diffs, diff)
			}
		}
	}

	t.T.Fatalf(
		"Expected a call matching %v with %v args, but none was found. \n"+
			"Yielded outputs from the function: %s\n"+
			"diffs for matching func ID's: %s",
		expectedCallID,
		expectedArgs,
		formatOutput(t.outputBuffer),
		strings.Join(diffs, "\n"),
	)

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

	// if we already registered a return, then just return that
	if t.hasReturned {
		return t.returnedVals
	}

	// iterate through the outputs from the function under test, until we find a return
	for next := range t.iterOut() {
		if next.Type == YieldedReturn {
			// register the return
			t.returnedVals = next.returnVals
			t.hasReturned = true

			// return the return!
			return t.returnedVals
		}
	}

	// error if there was no return
	t.T.Fatalf(
		"Expected a return, but none was found. Yielded outputs from the function: %s",
		formatOutput(t.outputBuffer),
	)

	panic("should never get here - linters just can't know that the test functions will panic")
}

// AssertReturned asserts that the function under test returned the given values.
func (t *FuncTester) AssertReturned(expectedReturnValues ...any) {
	t.T.Helper()

	returnVals := t.Returned()

	diff := t.Differ(expectedReturnValues, returnVals)
	if diff != "" {
		t.T.Fatalf("\n"+
			"Looking for the function to return\n"+
			"  with %#v,\n"+
			"but it returned with %#v instead.\n"+
			"diff: %s",
			expectedReturnValues,
			returnVals,
			diff,
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

		nextIndex := 0

		for {
			next, ok := t.nextOutput(nextIndex)
			if !ok {
				return
			}

			if !yield(next) {
				// if we don't want to keep going, we've found the match we want. remove it from the buffer!
				t.outputBuffer = slices.Delete(t.outputBuffer, nextIndex, nextIndex+1)

				return
			}

			nextIndex++
		}
	}
}

// AssertPanicked asserts that the function under test paniced with the given value.
func (t *FuncTester) AssertPanicked(expectedPanic any) {
	t.T.Helper()

	panicVal := t.Panicked()

	diff := t.Differ(expectedPanic, panicVal)
	if diff != "" {
		t.T.Fatalf("\n"+
			"Looking for the function to panic\n"+
			"  with %#v,\n"+
			"but it panicked with %#v instead.\n"+
			"diff: %s",
			expectedPanic,
			panicVal,
			diff,
		)
	}
}

// nextOutput gets the next output from the queue or the func outputs.
func (t *FuncTester) nextOutput(nextIndex int) (YieldedValue, bool) {
	t.T.Helper()

	// if we have more items in the buffer, return the next one.
	if nextIndex < len(t.outputBuffer) {
		return t.outputBuffer[nextIndex], true
	}

	// if we're allowed to pull more, pull, add to the buffer, and return what was pulled.
	for len(t.outputBuffer) < t.bufferMaxLen {
		select {
		case actualOutput, open := <-t.OutputChan:
			if !open {
				t.T.Fatal("expected an output to be available, but the outputs channel was already closed")
				panic("only necessary because nilchecker doesn't know what to do with my mocked tester")
			}

			t.outputBuffer = append(t.outputBuffer, actualOutput)

			return actualOutput, true
		case <-time.After(t.Timeout):
			logMessage := fmt.Sprintf(
				"\n"+
					"Looking for output\n"+
					"but the test timed out with a queue with %s.\n"+
					"bufferMaxLen: %d.\n"+
					"bufferNextIndex: %d\n"+
					"timeout: %v",
				formatOutput(t.outputBuffer),
				t.bufferMaxLen,
				nextIndex,
				t.Timeout,
			)

			t.T.Fatalf(logMessage)
			// t.T.Fatalf("expected a call to be available, but the test timed out waiting after %v", t.timeout)
			panic("only necessary because linters don't know what to do with my mocked tester")
		}
	}

	// if we're not allowed to pull any more, return _not ok_
	return YieldedValue{}, false //nolint:exhaustruct // we're intentionally returning a nil value
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

// Function is here to help us distinguish functions internally, because there is no single
// function _type_ in go.
type Function any

// getFuncName gets the function's name.
func getFuncName(f Function) string {
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	// this suffix gets appended sometimes. It's unimportant, as far as I can tell.
	name = strings.TrimSuffix(name, "-fm")

	return name
}

// Error philosophy:
//
// Failures: conditions which signal expected failures the user is testing for (this is a test library), should
// trigger a test failure.
//
// Panics: conditions which signal an error which it is not generally reasonable to expect a caller to recover from,
// which instead imply programmer intervention is necessary to resolve, should trigger an explanatory panic for the
// programmer to track down.
//
// Errors: all other error conditions should trigger an error with sufficient detail to enable the caller to take
// corrective action

// Behavioral properties philosophy:
//
// Every public function will satisfy a number of behavioral properties it guarantees.
// Those properties will be tested.

// **Private Functions & Methods**

// callFunc calls the given function with the given args, and returns the return values from that callFunc.
func callFunc(f Function, args []any) []any {
	rf := reflect.ValueOf(f)
	rArgs := reflectValuesOf(args)
	rReturns := rf.Call(rArgs)

	return unreflectValues(rReturns)
}

// unreflectValues returns the actual values of the reflected values.
func unreflectValues(rArgs []reflect.Value) []any {
	// tricking nilaway with repeated appends till this issue is closed
	// https://github.com/uber-go/nilaway/pull/60
	// args := make([]any, len(rArgs))
	if len(rArgs) == 0 {
		return nil
	}

	args := []any{}

	for i := range rArgs {
		// args[i] = rArgs[i].Interface()
		args = append(args, rArgs[i].Interface())
	}

	return args
}

// reflectValuesOf returns reflected values for all of the values.
func reflectValuesOf(args []any) []reflect.Value {
	rArgs := make([]reflect.Value, len(args))
	for i := range args {
		rArgs[i] = reflect.ValueOf(args[i])
	}

	return rArgs
}
