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

// =The Core types and functions that let any of this happen=

// WrapFunc mocks the given function, creating a named coroutine from it, to be used in testing.
// The coroutine will yield a Call onto the given YieldedValue channel whenever it is called.
// This Call can be checked for its name (ID) and args. The calling function will wait for the
// Call.Return or Call.Panic methods to be called. Calling either of those methods will cause the
// coroutine to return or panic with the values as passed.
func WrapFunc[T any](function T, calls chan YieldedValue, options ...WrapOption) (T, string) {
	opts := WrapOptions{name: getFuncName(function)}
	for _, o := range options {
		opts = o(opts)
	}

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := reflect.TypeOf(function)

	relayer := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		injectedValueChan := make(chan injectedValue)

		// Submit this call to the calls channel
		calls <- YieldedValue{
			YieldedCall,
			nil,
			nil,
			Call{
				opts.name,
				unreflectValues(args),
				injectedValueChan,
			},
		}

		outputV := <-injectedValueChan

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
	return wrapped, opts.name
}

type YieldedValue struct {
	Type       YieldType
	PanicVal   any
	ReturnVals []any
	Call       Call
}

func (out *YieldedValue) String() string {
	switch out.Type {
	case YieldedCall:
		return strings.Join([]string{
			"call",
			"with name",
			out.Call.ID,
			"with args",
			fmt.Sprintf("%#v", out.Call.Args),
		}, "\n")
	case YieldedReturn:
		return strings.Join([]string{
			"return",
			"with values",
			fmt.Sprintf("%#v", out.ReturnVals),
		}, "\n")
	case YieldedPanic:
		return strings.Join([]string{
			"panic",
			"with value",
			fmt.Sprintf("%#v", out.PanicVal),
		}, "\n")
	default:
		panic("got an unexpected output type")
	}
}

type YieldType int

const (
	YieldedReturn YieldType = iota
	YieldedPanic  YieldType = iota
	YieldedCall   YieldType = iota
)

type Call struct {
	ID                string
	Args              []any
	injectedValueChan chan injectedValue
}

// Return returns the given values in the func call.
func (c Call) Return(returnVals ...any) {
	c.injectedValueChan <- injectedValue{
		InjectedReturn,
		returnVals,
		nil,
	}
	close(c.injectedValueChan)
}

// Panic makes the func call result in a panic with the given value.
func (c Call) Panic(panicVal any) {
	c.injectedValueChan <- injectedValue{
		InjectedPanic,
		nil,
		panicVal,
	}
	close(c.injectedValueChan)
}

type injectedValue struct {
	Type         injectionType
	ReturnValues []any
	PanicValue   any
}

type injectionType int

const (
	InjectedReturn injectionType = iota
	InjectedPanic  injectionType = iota
)

type WrapOption func(WrapOptions) WrapOptions

type WrapOptions struct {
	name string
}

func WithName(name string) WrapOption {
	return func(wo WrapOptions) WrapOptions {
		wo.name = name
		return wo
	}
}

// getFuncName gets the function's name.
func getFuncName(f function) string {
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	// this suffix gets appended sometimes. It's unimportant, as far as I can tell.
	name = strings.TrimSuffix(name, "-fm")

	return name
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

// function is here to help us distinguish functions internally, because there is no single
// function _type_ in go.
type function any

// =Test Simplification and readability abstractions=

// NewFuncTester returns a newly initialized FuncTester.
func NewFuncTester(tester Tester) *FuncTester {
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

	return funcTester
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
					panicVal,
					nil,
					Call{}, //nolint:exhaustruct // passing a zero value on purpose
				}
			} else {
				t.OutputChan <- YieldedValue{
					YieldedReturn,
					nil,
					rVals,
					Call{}, //nolint:exhaustruct // passing a zero value on purpose
				}
			}
		}()

		rVals = callFunc(function, args)
	}()
}

// Called returns the FuncCall for inspection by the test.
func (t *FuncTester) Called() Call {
	for next := range t.iterOut() {
		if next.Type == YieldedCall {
			return next.Call
		}
	}

	t.T.Fatalf("Expected a call, but none was found. Yielded outputs from the function: %s", formatOutput(t.outputBuffer))

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) Call {
	t.T.Helper()

	diffs := []string{}

	for next := range t.iterOut() {
		if next.Type == YieldedCall {
			if next.Call.ID == expectedCallID {
				diff := t.Differ(next.Call.Args, expectedArgs)
				if diff == "" {
					return next.Call
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

func (t *FuncTester) Close() {
	close(t.OutputChan)
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
			t.returnedVals = next.ReturnVals
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

// Panicked returns the panicked value.
func (t *FuncTester) Panicked() any {
	t.T.Helper()

	if t.hasPanicked {
		return t.panickedVal
	}

	for next := range t.iterOut() {
		if next.Type == YieldedPanic {
			t.panickedVal = next.PanicVal
			t.hasPanicked = true

			return t.panickedVal
		}
	}

	// error if there was no panic
	t.T.Fatalf(
		"Expected a panic, but none was found. Yielded outputs from the function: %s",
		formatOutput(t.outputBuffer),
	)

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
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

	// if we're not allowed to pu_not ok_
	return YieldedValue{}, false //nolint:exhaustruct
}

type Differ func(any, any) string

func formatOutput(outputs []YieldedValue) string {
	formatted := []string{}

	for _, funcOut := range outputs {
		formatted = append(formatted, funcOut.String())
	}

	return strings.Join(formatted, "\n")
}

// callFunc calls the given function with the given args, and returns the return values from that callFunc.
func callFunc(f function, args []any) []any {
	rf := reflect.ValueOf(f)
	rArgs := reflectValuesOf(args)
	rReturns := rf.Call(rArgs)

	return unreflectValues(rReturns)
}

// reflectValuesOf returns reflected values for all of the values.
func reflectValuesOf(args []any) []reflect.Value {
	rArgs := make([]reflect.Value, len(args))
	for i := range args {
		rArgs[i] = reflect.ValueOf(args[i])
	}

	return rArgs
}
