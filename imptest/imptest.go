// Package imptest provides impure function testing functionality.
package imptest

import (
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
	args             []any
	ReturnValuesChan chan []any
	PanicValueChan   chan any
}

// NewFuncTester returns a newly initialized FuncTester.
func NewFuncTester(t *testing.T) *FuncTester {
	t.Helper()

	return &FuncTester{
		T:            t,
		Calls:        make(chan FuncCall),
		Panic:        nil,
		ReturnValues: []any{},
	}
}

// Tester contains the *testing.T and the chan FuncCall.
type FuncTester struct {
	T            *testing.T
	Calls        chan FuncCall
	Panic        any
	ReturnValues []any
}

// Start starts the function.
func (t *FuncTester) Start(function any, args ...any) {
	// record when the func is done so we can test that, too
	go func() {
		defer func() {
			close(t.Calls)

			t.Panic = recover()
		}()

		t.ReturnValues = callFunc(function, args)
	}()
}

// AssertCalled asserts that the passed in fuction and args match.
func (t *FuncTester) AssertCalled(expectedCallID string, expectedArgs ...any) FuncCall {
	t.T.Helper()

	actualCall, open := <-t.Calls
	if !open {
		t.T.Fatalf("expected a call to %s, but the function under test returned early", expectedCallID)
	}

	if actualCall.ID != expectedCallID {
		t.T.Fatalf(
			"wrong callID: expected the function %s to be called, but %s was called instead",
			expectedCallID,
			actualCall.ID,
		)
	}

	if !reflect.DeepEqual(actualCall.args, expectedArgs) {
		t.T.Fatalf("wrong values: the function %s was expected to be called with %#v, but was called with %#v",
			expectedCallID, expectedArgs, actualCall.args,
		)
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
	if !reflect.DeepEqual(t.ReturnValues, expectedReturnValues) {
		t.T.Fatalf("wrong values: the function under test was expected to return %#v, but returned %#v",
			expectedReturnValues, t.ReturnValues,
		)
	}
}

// Return returns the given values in the func call.
func (c FuncCall) Return(returnVals ...any) {
	c.ReturnValuesChan <- returnVals
	close(c.ReturnValuesChan)
}

// Return returns the given values in the func call.
func (c FuncCall) Panic(panicVal any) {
	c.PanicValueChan <- panicVal
	close(c.PanicValueChan)
}

// AssertPanicked asserts that the function under test paniced with the given value.
func (t *FuncTester) AssertPanicked(expectedPanic any) {
	t.T.Helper()

	// Then there are no more calls
	_, open := <-t.Calls
	if open {
		t.T.Fatal("the function under test was not done, but a panic was expected")
	}

	if !reflect.DeepEqual(t.Panic, expectedPanic) {
		t.T.Fatalf("wrong panic: the function under test was expected to panic with %#v  but %#v was found instead",
			expectedPanic, t.Panic,
		)
	}
}
