// Package imptest provides impure function testing functionality.
package imptest

import (
	"encoding/json"
	"fmt"
	"iter"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/muesli/reflow/indent"
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
func WrapFunc[T any](tester Tester, function T, calls chan YieldedValue, options ...WrapOption) (T, string) {
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
			&Call{
				opts.name,
				unreflectValues(args),
				injectedValueChan,
				funcType,
				tester,
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
	Call       *Call
}

func (out *YieldedValue) String() string {
	switch out.Type {
	case YieldedCall:
		return strings.Join([]string{
			"call to",
			out.Call.ID,
			"with args",
			fmt.Sprintf("%#v", out.Call.Args),
		}, " ")
	case YieldedReturn:
		return strings.Join([]string{
			"completed & returned with",
			fmt.Sprintf("%#v", out.ReturnVals),
		}, " ")
	case YieldedPanic:
		return strings.Join([]string{
			"panicked",
			"with value",
			fmt.Sprintf("%#v", out.PanicVal),
		}, " ")
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
	Type              reflect.Type
	t                 Tester
}

// Return returns the given values in the func call.
func (c Call) Return(returnVals ...any) {
	c.t.Helper()
	// make sure these are at least the right number of returns
	expectedNumReturns := c.Type.NumOut()
	if len(returnVals) != expectedNumReturns {
		c.t.Fatalf(
			"%d returns were pushed, but %s only returns %d values",
			len(returnVals), c.ID, expectedNumReturns,
		)
	}
	// make sure these are at least assignable
	for rvi := range returnVals {
		actual := reflect.TypeOf(returnVals[rvi])
		expected := c.Type.Out(rvi)

		if actual != nil && !actual.AssignableTo(expected) {
			c.t.Fatalf(
				"unable to push return value %d for the call to %s: a value of type %v was pushed, "+
					"but that is unassignable to the expected type (%v)",
				rvi, c.ID, actual, expected,
			)
		}
	}
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
	funcTester.Differ = func(actual, expected any) (string, error) {
		if !reflect.DeepEqual(actual, expected) {
			return fmt.Sprintf("actual: %#v\nexpected: %#v", actual, expected), nil
		}

		return "", nil
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
					nil,
				}
			} else {
				t.OutputChan <- YieldedValue{
					YieldedReturn,
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
func (t *FuncTester) Called() Call {
	for next := range t.iterOut() {
		if next.Type == YieldedCall {
			return *next.Call
		}
	}

	t.T.Fatalf("Expected a call, but none was found. Yielded outputs from the function: %s", formatOutput(t.outputBuffer))

	panic("should never get here - the code within the iterator will panic if we can't get a good value")
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) Call {
	t.T.Helper()

	diffs := []string{}

	// for as many things as have been output by the function under test...
	for next := range t.iterOut() {
		// was this thing a call?
		if next.Type != YieldedCall {
			continue
		}
		// did it match our expected call id?
		if next.Call.ID != expectedCallID {
			continue
		}
		// were the args matching?
		diff, err := t.Differ(next.Call.Args, expectedArgs)
		// was there an error matching the args?
		if err != nil {
			t.T.Fatalf("unable to assert arg equality: %s", err.Error())
		}
		// if there was a difference from expectations, record it & try again
		if diff != "" {
			diffs = append(diffs, diff)
			continue
		}

		// if we got here: this was a call, of the expected name, and the args matched.
		// return the call.
		return *next.Call
	}

	// if we popped out here, it means there's no other output to iterate, and nothing totally matched.
	// if we have diffs, it means we _did_ find the matching call, but the args were different.
	// the most common case is a sequential test, in which we found the one matching call, but the args were different...
	// ... handle that messaging.
	const spacesToIndent = 4
	if len(diffs) == 1 {
		t.T.Fatalf(
			"Found expected call to %s, but the args differed:\n%s",
			expectedCallID, indent.String(diffs[0], spacesToIndent),
		)
	}
	// if we have no diffs, it means we _did not_ find the matching call.
	if len(diffs) == 0 {
		t.T.Fatalf(
			"Failed to find expected call to %s. Instead found the following output from the function under test:\n%s",
			expectedCallID, indent.String(formatOutput(t.outputBuffer), spacesToIndent),
		)
	}
	// the final case is if we have multiple diffs, we found multiple matching calls, but they all had differing args.
	t.T.Fatalf(
		"Found multiple call to %s, but the args differed in each case:\n%s",
		expectedCallID, indent.String(strings.Join(diffs, "\n"), spacesToIndent),
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

	diff, err := t.Differ(expectedReturnValues, returnVals)
	if err != nil {
		t.T.Fatalf("unable to assert return value equality: %s", err.Error())
	}

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

	diff, err := t.Differ(expectedPanic, panicVal)
	if err != nil {
		t.T.Fatalf("unable to assert panic value equality: %s", err.Error())
	}

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

			if !yield(*next) {
				// if we don't want to keep going, we've found the match we want. remove it from the buffer!
				t.outputBuffer = slices.Delete(t.outputBuffer, nextIndex, nextIndex+1)

				return
			}

			nextIndex++
		}
	}
}

// nextOutput gets the next output from the queue or the func outputs.
func (t *FuncTester) nextOutput(nextIndex int) (*YieldedValue, bool) {
	t.T.Helper()

	// if we have more items in the buffer, return the next one.
	if nextIndex < len(t.outputBuffer) {
		return &t.outputBuffer[nextIndex], true
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

			return &actualOutput, true
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

	// if we're not allowed to pull more, return not ok
	return nil, false
}

type Differ func(any, any) (string, error)

func formatOutput(outputs []YieldedValue) string {
	formatted := []string{}

	for _, funcOut := range outputs {
		formatted = append(formatted, funcOut.String())
	}

	return "\n\t" + strings.Join(formatted, "\n\t")
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

func NewImp(tester Tester, funcStructs ...any) *Tester2 {
	ftester := NewFuncTester(tester)
	tester2 := &Tester2{ft: ftester}

	for _, fs := range funcStructs {
		// get all methods of the funcStructs
		fsType := reflect.ValueOf(fs).Elem().Type()
		fsValue := reflect.ValueOf(fs).Elem()
		numFields := fsType.NumField()
		fields := make([]fieldPair, numFields)

		for i := range numFields {
			fields[i].Type = fsType.Field(i)
			fields[i].Value = fsValue.Field(i)
		}

		// reduce to fields that are functions
		functionFields := []fieldPair{}

		for i := range numFields {
			if fields[i].Type.Type.Kind() != reflect.Func {
				continue
			}

			functionFields = append(functionFields, fields[i])
		}

		// intercept them all
		for i := range functionFields {
			wrapFuncField(tester, functionFields[i], ftester.OutputChan)
		}
	}

	return tester2
}

type fieldPair struct {
	Type  reflect.StructField
	Value reflect.Value
}

type Tester2 struct {
	ft *FuncTester
}

func (t *Tester2) ExpectCall(f string) *Call2 {
	// TODO: collapse tester2 stuff, we shouldn't need to care about "subcall"
	c := &Call2{t: t, f: f}
	return c
}

func (t *Tester2) Start(f any, args ...any) *Tester2 {
	// start the test
	t.ft.Start(f, args...)

	return t
}

func (t *Tester2) ExpectReturns(returned ...any) {
	t.ft.AssertReturned(returned...)
}

type Call2 struct {
	t       *Tester2
	f       string
	subcall Call `exhaustruct:"optional"`
}

func (c *Call2) ExpectArgs(args ...any) *Call2 {
	c.subcall = c.t.ft.AssertCalled(c.f, args...)
	c.subcall.t.Helper()

	return c
}

func (c *Call2) ExpectArgsJSON(args ...any) *Call2 {
	c.t.ft.T.Helper()

	originalDiffer := c.t.ft.Differ
	defer func() { c.t.ft.Differ = originalDiffer }()

	c.t.ft.Differ = jsonDiffer
	c.subcall = c.t.ft.AssertCalled(c.f, args...)

	return c
}

func (c *Call2) ExpectArgsFmt(args ...any) *Call2 {
	c.t.ft.T.Helper()

	originalDiffer := c.t.ft.Differ
	defer func() { c.t.ft.Differ = originalDiffer }()

	c.t.ft.Differ = fmtDiffer
	c.subcall = c.t.ft.AssertCalled(c.f, args...)

	return c
}

func fmtDiffer(actual, expected any) (string, error) {
	var actualString, expectedString string

	actualArr, isArray := actual.([]any)
	if isArray {
		actualString = fmtArray(actualArr)
	} else {
		actualString = fmt.Sprintf("%v", actual)
	}

	expectedArr, isArray := expected.([]any)

	if isArray {
		expectedString = fmtArray(expectedArr)
	} else {
		expectedString = fmt.Sprintf("%v", expected)
	}

	const spacesToIndent = 4
	if actualString != expectedString {
		return fmt.Sprintf(
			"actual: \n%s\nexpected: \n%s",
			indent.String(actualString, spacesToIndent), indent.String(expectedString, spacesToIndent),
		), nil
	}

	return "", nil
}

func fmtArray(aa []any) string {
	formattedArray := []string{}
	for i := range aa {
		formattedArray = append(formattedArray, fmt.Sprintf("%v", aa[i]))
	}

	return strings.Join(formattedArray, "\n")
}

func jsonDiffer(actual, expected any) (string, error) {
	actualJSON, err := json.MarshalIndent(actual, "", "    ")
	if err != nil {
		return "", fmt.Errorf("unable to diff %#v and %#v: error converting the first to json: %w", actual, expected, err)
	}

	expectedJSON, err := json.MarshalIndent(expected, "", "    ")
	if err != nil {
		return "", fmt.Errorf("unable to diff %#v and %#v: error converting the second to json: %w", actual, expected, err)
	}

	actualString := string(actualJSON)
	expectedString := string(expectedJSON)

	if actualString != expectedString {
		return fmt.Sprintf("actual: %s\nexpected: %s", actualString, expectedString), nil
	}

	return "", nil
}

func (c *Call2) PushReturns(returns ...any) {
	c.subcall.t.Helper()
	c.subcall.Return(returns...)
}

func wrapFuncField(tester Tester, funcField fieldPair, calls chan YieldedValue) {
	name := funcField.Type.Name

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := funcField.Type.Type

	relayer := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		injectedValueChan := make(chan injectedValue)

		// Submit this call to the calls channel
		calls <- YieldedValue{
			YieldedCall,
			nil,
			nil,
			&Call{
				name,
				unreflectValues(args),
				injectedValueChan,
				funcType,
				tester,
			},
		}

		outputV := <-injectedValueChan

		switch outputV.Type {
		case InjectedReturn:
			returnValues := make([]reflect.Value, len(outputV.ReturnValues))

			// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
			for rvi, a := range outputV.ReturnValues {
				// special casing to avoid a completely nil return hosing things up
				v := reflect.ValueOf(a)
				if !v.IsValid() {
					v = reflect.Zero(funcType.Out(rvi))
				}

				returnValues[rvi] = v
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
	wrappedValue := reflect.MakeFunc(funcType, relayer)

	funcField.Value.Set(wrappedValue)
}
