// Package imptest provides impure function testing functionality.
package imptest

import (
	"fmt"
	"reflect"
	"testing"
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
			if len(returnValuesReflected) == 0 {
				return nil
			}

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
	wrapped := reflect.MakeFunc(funcType, relayer).Interface().(T) //nolint: forcetypeassert

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
func NewFuncTester(t *testing.T) *FuncTester {
	t.Helper()

	calls := make(chan FuncCall)
	returnFunc, returnID := ReturnFunc(calls)
	panicFunc, panicID := PanicFunc(calls)

	return &FuncTester{
		t,
		calls,
		nil,
		[]any{},
		1,
		returnFunc,
		panicFunc,
		returnID,
		panicID,
		[]int{},
		[]FuncCall{},
		0,
		0,
	}
}

// Tester contains the *testing.T and the chan FuncCall.
type FuncTester struct {
	T               *testing.T
	Calls           chan FuncCall
	Panic           any
	ReturnValues    []any
	maxGoroutines   int
	returnFunc      func()
	panicFunc       func()
	returnID        string
	panicID         string
	marks           []int
	callQueue       []FuncCall
	queueStartIndex int
	maxQueueLen     int
}

// Start starts the function.
func (t *FuncTester) Start(function any, args ...any) {
	// record when the func is done so we can test that, too
	go func() {
		defer func() {
			t.Panic = recover()
			if t.Panic != nil {
				t.panicFunc()
			} else {
				t.returnFunc()
			}
		}()

		t.ReturnValues = callFunc(function, args)
	}()
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) FuncCall {
	t.T.Helper()

	return t.assertMatch(expectedCallID, expectedArgs)
}

func (t *FuncTester) assertMatch(expectedCallID string, expectedArgs []any) FuncCall {
	unmatchedCalls := []FuncCall{}

	for {
		// get the next thing
		next := t.nextCall()

		// TODO: clean this up in terms of better switching & abstraction
		actualArgs := next.Args
		expectationID := "call ID of " + expectedCallID

		if expectedCallID == t.returnID {
			actualArgs = t.ReturnValues
			expectationID = "return from function under test"
		} else if expectedCallID == t.panicID {
			actualArgs = []any{t.Panic}
			expectationID = "panic from function under test"
		}

		// TODO: is this mapping unnecessary?
		actualID := next.ID

		// if match, shove other checked calls back onto the queue & return
		if actualID == expectedCallID && reflect.DeepEqual(actualArgs, expectedArgs) {
			t.callQueue = append(t.callQueue, unmatchedCalls...)
			return next
		}

		t.T.Logf(
			"No match between expected (%s)\nand next (%s)",
			fmt.Sprintf("ID: %s, Args: %#v", expectedCallID, expectedArgs),
			fmt.Sprintf("ID: %s, Args: %#v", actualID, actualArgs),
		)

		// if no match, put call on the stack of checked calls
		unmatchedCalls = append(unmatchedCalls, next)

		// if !more, fail with message about what we expected to find vs what we got
		if len(unmatchedCalls)+len(t.callQueue) > t.maxQueueLen+1 {
			t.T.Fatalf(
				"Expected %s,"+
					"with args %#v,\nbut the only calls found were %#v.\n"+
					"len(unmatchedCalls): %d.\nlen(callQueue): %d,\nmaxQueueLen: %d.\n"+
					" queueStartIndex: %d.\ncallQueue: %#v",
				expectationID,
				expectedArgs,
				unmatchedCalls,
				len(unmatchedCalls),
				len(t.callQueue),
				t.maxQueueLen,
				t.queueStartIndex,
				t.callQueue,
			)
		}
	}
}

// nextCall gets the next call from the queue or the calls.
func (t *FuncTester) nextCall() FuncCall {
	if len(t.callQueue[t.queueStartIndex:]) > 0 {
		next := t.callQueue[t.queueStartIndex]

		if t.queueStartIndex > 0 {
			t.callQueue = append(t.callQueue[0:t.queueStartIndex-1], t.callQueue[t.queueStartIndex+1:]...)
		} else {
			t.callQueue = t.callQueue[t.queueStartIndex+1:]
		}

		t.T.Logf("returning next from call queue: %#v", next)

		return next
	}

	actualCall, open := <-t.Calls
	if !open {
		t.T.Fatal("expected a call to be available, but the calls channel was already closed")
	}

	t.T.Logf("returning next from call channel: %#v", actualCall)

	return actualCall
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

func ReturnFunc(calls chan FuncCall) (func(), string) {
	// creates a unique ID for the function
	// TODO: allow users to override the ID
	// TODO: add a random unique element to the end
	funcID := "returnFunc"

	// create the function, that when called:
	// * puts its ID onto the call channel

	returnFunc := func() {
		// Submit this call to the calls channel
		calls <- FuncCall{
			funcID,
			nil,
			nil,
			nil,
		}
	}

	// returns both the wrapped func and the ID
	return returnFunc, funcID
}

func PanicFunc(calls chan FuncCall) (func(), string) {
	// creates a unique ID for the function
	// TODO: allow users to override the ID
	// TODO: add a random unique element to the end
	funcID := "panicFunc"

	// create the function, that when called:
	// * puts its ID onto the call channel
	panicFunc := func() {
		// Submit this call to the calls channel
		calls <- FuncCall{
			funcID,
			nil,
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
	// read the current queue length
	mark := len(t.callQueue)
	// add a mark for that length
	t.marks = append(t.marks, mark)
	// reset queue start index to the latest mark
	t.queueStartIndex = t.marks[len(t.marks)-1]
	// add len(funcs) -1 as a max for queue length
	t.maxQueueLen += len(funcs) - 1
	// run each function.
	for _, f := range funcs {
		f()
		// reset queue start index to the latest mark
		t.queueStartIndex = t.marks[len(t.marks)-1]
		// reduce the max queue length
		t.maxQueueLen--
	}
	// at the end, pop the mark we added
	t.marks = t.marks[0 : len(t.marks)-1]
}
