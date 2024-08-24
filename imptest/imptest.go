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
	}
}

// Tester contains the *testing.T and the chan FuncCall.
type FuncTester struct {
	T             *testing.T
	Calls         chan FuncCall
	Panic         any
	ReturnValues  []any
	maxGoroutines int
	returnFunc    func()
	panicFunc     func()
	returnID      string
	panicID       string
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

	// check if the call is on the queue
	// if it is, remove it & return it.
	// if it isn't, and the queue isn't full, pull another call & check it.
	// if it matches, return it.
	// if not add it to the queue and go back to checking the queue size.
	// add the mismatch message to a failure queue
	// if the queue is full, fail with the failure queue
	// if the channel is closed, fail with the failure queue and a closed message
	actualCall, open := <-t.Calls
	if !open {
		t.T.Fatalf("expected a call to %s, but the calls channel was already closed", expectedCallID)
	}

	if actualCall.ID != expectedCallID {
		t.T.Fatalf(
			"wrong callID: expected the function %s to be called, but %s was called instead",
			expectedCallID,
			actualCall.ID,
		)
	}

	if !reflect.DeepEqual(actualCall.Args, expectedArgs) {
		t.T.Fatalf("wrong values: the function %s was expected to be called with %#v, but was called with %#v",
			expectedCallID, expectedArgs, actualCall.Args,
		)
	}

	return actualCall
}

// AssertReturned asserts that the function under test returned the given values.
func (t *FuncTester) AssertReturned(expectedReturnValues ...any) {
	t.T.Helper()

	expectedCallID := t.returnID

	actualCall, open := <-t.Calls
	if !open {
		t.T.Fatalf("expected a call to %s, but the calls channel was already closed", expectedCallID)
	}

	if actualCall.ID != expectedCallID {
		t.T.Fatalf(
			"wrong callID: expected the function %s to be called, but %s was called instead",
			expectedCallID,
			actualCall.ID,
		)
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

// Panic makes the func call result in a panic with the given value.
func (c FuncCall) Panic(panicVal any) {
	c.PanicValueChan <- panicVal
	close(c.PanicValueChan)
}

// AssertPanicked asserts that the function under test paniced with the given value.
func (t *FuncTester) AssertPanicked(expectedPanic any) {
	t.T.Helper()

	expectedCallID := t.panicID

	actualCall, open := <-t.Calls
	if !open {
		t.T.Fatalf("expected a call to %s, but the calls channel was already closed", expectedCallID)
	}

	if actualCall.ID != expectedCallID {
		t.T.Fatalf(
			"wrong callID: expected the function %s to be called, but %s was called instead",
			expectedCallID,
			actualCall.ID,
		)
	}

	if !reflect.DeepEqual(t.Panic, expectedPanic) {
		t.T.Fatalf("wrong panic: the function under test was expected to panic with %#v  but %#v was found instead",
			expectedPanic, t.Panic,
		)
	}
}

// SetGoroutines sets the number of goroutines to read till finding the expected call.
func (t *FuncTester) SetGoroutines(num int) {
	t.maxGoroutines = num
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

func (t *FuncTester) ExpectUnordered(expected ...Enforceable) *ExpectedUnordered {
	return &ExpectedUnordered{t, expected}
}

type Enforceable interface {
	Enforce()
}

type ExpectedUnordered struct {
	t        *FuncTester
	expected []Enforceable
}

func (eu *ExpectedUnordered) Enforce() {
	eu.t.T.Helper()
	for {
		// Get the next possible expected calls to match against. If none, return.
		expectedCalls := eu.peekNextExpectedCalls()
		if len(expectedCalls) == 0 {
			return
		}
		concreteCalls := []string{}
		for _, ec := range expectedCalls {
			concreteCalls = append(concreteCalls, fmt.Sprintf("ID: %s, Args: %v\n", ec.id, ec.args))
		}
		// Get the next call
		call, ok := <-eu.t.Calls
		// TODO: consolidate the closed check in a "next" call
		if !ok {
			eu.t.T.Fatalf("expected any of %v, but the calls channel was closed", concreteCalls)
		}
		callString := fmt.Sprintf("ID: %s, Args: %v", call.ID, call.Args)
		// Match one of them
		matched := eu.match(call, expectedCalls)
		// If there's no match, fail
		if matched == nil {
			eu.t.T.Fatalf("expected any of %v, but found %s instead", concreteCalls, callString)
		}
		// pop the match off of the list.
		eu.pop(matched)
		// enforce it. We've already checked that this is the right one, now we just need to make the followup action happen
		if matched.doPanic {
			call.Panic(matched.panicValue)
		} else if call.ReturnValuesChan != nil {
			call.Return(matched.returns...)
		}
		// repeat
	}
}

func (eu *ExpectedUnordered) peekNextExpectedCalls() []*ExpectedCall {
	expectedCalls := []*ExpectedCall{}
	for _, expected := range eu.expected {
		switch v := expected.(type) {
		case *ExpectedCall:
			expectedCalls = append(expectedCalls, v)
		case *ExpectedUnordered:
			expectedCalls = append(expectedCalls, v.peekNextExpectedCalls()...)
		case *ExpectedOrdered:
			expectedCalls = append(expectedCalls, v.peekNextExpectedCalls()...)
		}
	}
	return expectedCalls
}

func (eo *ExpectedOrdered) peekNextExpectedCalls() []*ExpectedCall {
	expectedCalls := []*ExpectedCall{}
	for _, expected := range eo.expected {
		switch v := expected.(type) {
		case *ExpectedCall:
			expectedCalls = append(expectedCalls, v)
		case *ExpectedUnordered:
			expectedCalls = append(expectedCalls, v.peekNextExpectedCalls()...)
		case *ExpectedOrdered:
			expectedCalls = append(expectedCalls, v.peekNextExpectedCalls()...)
		}
	}
	return expectedCalls
}

func (eu *ExpectedUnordered) match(call FuncCall, expectedCalls []*ExpectedCall) *ExpectedCall {
	for _, expected := range expectedCalls {
		if expected.id != call.ID {
			continue
		}
		if !reflect.DeepEqual(call.Args, expected.args) {
			continue
		}

		return expected
	}

	return nil
}

func (eu *ExpectedUnordered) pop(matched *ExpectedCall) bool {
	popped := false
	newEnforceables := []Enforceable{}
	for _, expected := range eu.expected {
		if popped {
			newEnforceables = append(newEnforceables, expected)
			continue
		}
		switch v := expected.(type) {
		case *ExpectedCall:
			if v.id == matched.id && reflect.DeepEqual(v.args, matched.args) {
				popped = true
				continue
			}
		case *ExpectedUnordered:
			popped = v.pop(matched)
			if len(v.expected) == 0 {
				continue
			}
		case *ExpectedOrdered:
			popped = v.pop(matched)
			if len(v.expected) == 0 {
				continue
			}
		}
		newEnforceables = append(newEnforceables, expected)
	}
	eu.expected = newEnforceables
	return popped
}

func (eo *ExpectedOrdered) pop(matched *ExpectedCall) bool {
	popped := false
	newEnforceables := []Enforceable{}
	for _, expected := range eo.expected {
		if popped {
			newEnforceables = append(newEnforceables, expected)
			continue
		}
		switch v := expected.(type) {
		case *ExpectedCall:
			if v.id == matched.id && reflect.DeepEqual(v.args, matched.args) {
				popped = true
				continue
			}
		case *ExpectedUnordered:
			popped = v.pop(matched)
			if len(v.expected) == 0 {
				continue
			}
		case *ExpectedOrdered:
			popped = v.pop(matched)
			if len(v.expected) == 0 {
				continue
			}
		}
		newEnforceables = append(newEnforceables, expected)
	}
	eo.expected = newEnforceables
	return popped
}

type ExpectedCall struct {
	t          *FuncTester
	id         string
	args       []any
	returns    []any
	panicValue any
	doPanic    bool
}

func (ec *ExpectedCall) Enforce() {
	ec.t.T.Helper()
	call := ec.t.AssertCalled(ec.id, ec.args...)

	if ec.doPanic {
		call.Panic(ec.panicValue)
	} else if call.ReturnValuesChan != nil {
		call.Return(ec.returns...)
	}
}

func (t *FuncTester) ExpectCall(id string, args ...any) *ExpectedCall {
	return &ExpectedCall{
		t,
		id,
		args,
		nil,
		nil,
		false,
	}
}

func (ec *ExpectedCall) ForceReturn(values ...any) *ExpectedCall {
	ec.returns = values
	return ec
}

func (t *FuncTester) ExpectReturn(args ...any) *ExpectedCall {
	return &ExpectedCall{
		t,
		t.returnID,
		args,
		nil,
		nil,
		false,
	}
}

type ExpectedOrdered struct {
	t        *FuncTester
	expected []Enforceable
}

func (eo *ExpectedOrdered) Enforce() {
	eo.t.T.Helper()
	for _, e := range eo.expected {
		e.Enforce()
	}
}

func (t *FuncTester) ExpectOrdered(expected ...Enforceable) *ExpectedOrdered {
	return &ExpectedOrdered{t, expected}
}
