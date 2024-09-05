// Package imptest provides impure function testing functionality.
package imptest

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func WrapFunc[T any](function T, calls chan FuncCall) (T, string) {
	// creates a unique ID for the function
	// TODO: allow users to override the ID
	funcID := GetFuncName(function)

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := reflect.TypeOf(function)

	relayer := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive return values on
		returnValuesChan := make(chan []any)

		// Create a channel to receive a panic value on
		panicValueChan := make(chan any)

		// Submit this call to the calls channel
		calls <- FuncCall{
			funcID,
			unreflectValues(args),
			returnValuesChan,
			panicValueChan,
		}

		select {
		case returnValuesReflected := <-returnValuesChan:
			returnValues := make([]reflect.Value, len(returnValuesReflected))

			// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
			for i, a := range returnValuesReflected {
				returnValues[i] = reflect.ValueOf(a)
			}

			return returnValues
		// if we're supposed to panic, do.
		case panicValue := <-panicValueChan:
			panic(panicValue)
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

type FuncCall struct {
	ID               string
	Args             []any
	ReturnValuesChan chan []any
	PanicValueChan   chan any
}

// NewFuncTester returns a newly initialized FuncTester.
func NewFuncTester(tester Tester) *FuncTester {
	tester.Helper()

	calls := make(chan FuncCall)
	returnFunc, returnID := ReturnFunc(calls)
	panicFunc, panicID := PanicFunc(calls)

	funcTester := new(FuncTester)
	funcTester.T = tester
	funcTester.CallChan = calls
	funcTester.returnFunc = returnFunc
	funcTester.panicFunc = panicFunc
	funcTester.returnID = returnID
	funcTester.panicID = panicID
	funcTester.bufferMaxLen = 1

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
	T                Tester
	CallChan         chan FuncCall
	returnFunc       func(...any)
	panicFunc        func(any)
	returnID         string
	panicID          string
	callBuffer       []FuncCall
	bufferStartIndex int
	bufferMaxLen     int
	bufferNextIndex  int
}

// Start starts the function.
func (t *FuncTester) Start(function any, args ...any) {
	// record when the func is done so we can test that, too
	go func() {
		var rVals []any

		defer func() {
			p := recover()
			if p != nil {
				t.panicFunc(p)
			} else {
				t.returnFunc(rVals...)
			}
		}()

		rVals = callFunc(function, args)
	}()
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) FuncCall {
	t.T.Helper()

	return t.assertMatch(expectedCallID, expectedArgs)
}

func (t *FuncTester) assertMatch(expectedCallID string, expectedArgs []any) FuncCall {
	t.bufferNextIndex = 0

	for {
		// get the next thing
		next := t.nextCall()

		var expectation string

		switch expectedCallID {
		case t.returnID:
			expectation = "return from function under test"
		case t.panicID:
			expectation = "panic from function under test"
		default:
			expectation = "call ID of " + expectedCallID
		}

		// if match, remove from the buffer & return
		if next.ID == expectedCallID && reflect.DeepEqual(next.Args, expectedArgs) {
			t.callBuffer = append(t.callBuffer[:t.bufferNextIndex], t.callBuffer[t.bufferNextIndex+1:]...)
			return next
		}

		// t.T.Logf(
		// 	"No match between expected (%s)\nand next (%s)",
		// 	fmt.Sprintf("ID: %s, Args: %#v", expectedCallID, expectedArgs),
		// 	fmt.Sprintf("ID: %s, Args: %#v", next.ID, actualArgs),
		// )
		t.bufferNextIndex++
		logMessage := fmt.Sprintf(
			"\n"+
				"Looking for %s\n"+
				"  with args %v,\n"+
				"but the only calls found were %s.\n"+
				"bufferMaxLen: %d.\n"+
				"bufferStartIndex: %d\n"+
				"bufferNextIndex: %d",
			expectation,
			expectedArgs,
			formatCalls(t.callBuffer),
			t.bufferMaxLen,
			t.bufferStartIndex,
			t.bufferNextIndex,
		)

		// t.T.Logf(logMessage)

		// if we have tried and failed to match calls, such that the total
		// buffered calls are now equal to or greater than the
		// numConcurrentFuncs, then the function under test has called things
		// in an unexpected way. One of the calls in unmatchedCalls should've
		// matched.
		if t.bufferNextIndex >= t.bufferMaxLen {
			t.T.Fatal(logMessage)
		}
	}
}

func formatCalls(calls []FuncCall) string {
	formatted := []string{}

	for _, funcCall := range calls {
		formattedCall := fmt.Sprintf("\nCall %s\n"+
			"  with args %v",
			funcCall.ID,
			funcCall.Args,
		)
		formatted = append(formatted, formattedCall)
	}

	return strings.Join(formatted, "")
}

// nextCall gets the next call from the queue or the calls.
func (t *FuncTester) nextCall() FuncCall {
	if t.bufferNextIndex < len(t.callBuffer) {
		next := t.callBuffer[t.bufferNextIndex]
		// t.T.Logf("returning next from call queue: %#v", next)

		return next
	}

	// t.T.Logf(
	// 	"waiting for the next call from the channel\n"+
	// 		"len(callQueue): %d\n"+
	// 		"maxQueueLen: %d\n"+
	// 		"queueStartIndex: %d\n"+
	// 		"callQueue: %#v",
	// 	len(t.callQueue),
	// 	t.maxQueueLen,
	// 	t.queueStartIndex,
	// 	t.callQueue,
	// )

	// TODO: allow a timeout to be set
	select {
	case actualCall, open := <-t.CallChan:
		if !open {
			t.T.Fatal("expected a call to be available, but the calls channel was already closed")
			panic("only necessary because nilchecker doesn't know what to do with my mocked tester")
		}

		t.callBuffer = append(t.callBuffer, actualCall)

		// t.T.Logf("returning next from call channel: %#v", actualCall)

		return actualCall
	case <-time.After(1 * time.Second):
		t.T.Fatal("expected a call to be available, but the test timed out waiting after 1s")
		panic("only necessary because linters don't know what to do with my mocked tester")
	}
}

func (t *FuncTester) Close() {
	close(t.CallChan)
}

// AssertReturned asserts that the function under test returned the given values.
func (t *FuncTester) AssertReturned(expectedReturnValues ...any) {
	t.T.Helper()

	t.assertMatch(t.returnID, expectedReturnValues)
}

// Return returns the given values in the func call.
func (c FuncCall) Return(returnVals ...any) {
	c.ReturnValuesChan <- returnVals
	close(c.ReturnValuesChan)
}

// Panic makes the func call result in a panic with the given value.
func (c FuncCall) Panic(panicVal any) {
	c.PanicValueChan <- panicVal
	close(c.PanicValueChan)
}

// AssertPanicked asserts that the function under test paniced with the given value.
func (t *FuncTester) AssertPanicked(expectedPanic any) {
	t.T.Helper()

	t.assertMatch(t.panicID, []any{expectedPanic})
}

// TODO: make ID it's own type

func ReturnFunc(calls chan FuncCall) (func(...any), string) {
	// creates a unique ID for the function
	// TODO: allow users to override the ID
	// TODO: add a random unique element to the end
	funcID := "returnFunc"

	// create the function, that when called:
	// * puts its ID onto the call channel

	returnFunc := func(rVals ...any) {
		// Submit this call to the calls channel
		calls <- FuncCall{
			funcID,
			rVals,
			nil,
			nil,
		}
	}

	// returns both the wrapped func and the ID
	return returnFunc, funcID
}

func PanicFunc(calls chan FuncCall) (func(any), string) {
	// creates a unique ID for the function
	funcID := "panicFunc"

	// create the function, that when called:
	// * puts its ID onto the call channel
	panicFunc := func(pVal any) {
		// Submit this call to the calls channel
		calls <- FuncCall{
			funcID,
			[]any{pVal},
			nil,
			nil,
		}
	}

	// returns both the wrapped func and the ID
	return panicFunc, funcID
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
