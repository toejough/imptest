// Package imptest provides impure function testing functionality.
package imptest

import (
	"reflect"
	"testing"
)

func WrapFunc[T any](function T, calls chan FuncCall) (T, string) {
	// creates a unique ID for the function
	// TODO: allow users to override the ID
	// TODO: drop the uuid
	funcID := GetFuncName(function)

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := reflect.TypeOf(function)

	relayer := func(args []reflect.Value) (returnValues []reflect.Value) {
		// Create a channel to receive return values on
		returnValuesChan := make(chan []any)

		// Submit this call to the calls channel
		calls <- FuncCall{funcID, unreflectValues(args), returnValuesChan}

		// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
		for _, a := range <-returnValuesChan {
			returnValues = append(returnValues, reflect.ValueOf(a))
		}

		return returnValues
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
	args             []any
	ReturnValuesChan chan []any
}

// NewFuncTester returns a newly initialized FuncTester.
func NewFuncTester(t *testing.T, c chan FuncCall) *FuncTester {
	t.Helper()

	return &FuncTester{
		T:            t,
		Calls:        c,
		ReturnValues: []any{},
	}
}

// Tester contains the *testing.T and the chan FuncCall.
type FuncTester struct {
	T            *testing.T
	Calls        chan FuncCall
	ReturnValues []any
}

// Start starts the function.
func (t *FuncTester) Start(function any, args ...any) {
	// record when the func is done so we can test that, too
	go func() {
		defer func() {
			t.T.Helper()

			close(t.Calls)

			if r := recover(); r != nil {
				t.T.Fatalf("caught panic from started function: %v", r)
			}
		}()

		t.ReturnValues = callFunc(function, args)
	}()
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) FuncCall {
	t.T.Helper()

	actualCall := <-t.Calls
	if actualCall.ID != expectedCallID {
		t.T.Fatalf(
			"wrong callID: expected the function %s to be called, but %s was called instead",
			expectedCallID,
			actualCall.ID,
		)
	}

	// TODO: just simplify the comparison. Deep-equal or bust as the default.
	actualArgs := actualCall.args
	for i := range expectedArgs {
		if !deepEqual(actualArgs[i], expectedArgs[i]) {
			t.T.Fatalf("wrong values: the function %s was expected to be called with %#v at index %d but was called with %#v",
				expectedCallID, expectedArgs[i], i, actualArgs[i],
			)
		}
	}

	return actualCall
}

// AssertReturned asserts that the function under test returned the given values.
func (t *FuncTester) AssertReturned(expectedReturnValues ...any) {
	t.T.Helper()

	// Then there are no more calls
	_, open := <-t.Calls
	if open {
		t.T.Fail()
	}

	// TODO: create a basic diff function based on json marshalling
	// TODO: allow users to override the diff function
	for i := range expectedReturnValues {
		if !deepEqual(t.ReturnValues[i], expectedReturnValues[i]) {
			t.T.Fatalf("wrong values: the function under test was expected to return %#v at index %d but returned %#v",
				expectedReturnValues[i], i, t.ReturnValues[i],
			)
		}
	}
}

// Return returns the given values in the func call.
func (c FuncCall) Return(returnVals ...any) {
	c.ReturnValuesChan <- returnVals
}
