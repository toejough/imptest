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
		t,
		make(chan FuncCall),
		nil,
		[]any{},
		1,
	}
}

// Tester contains the *testing.T and the chan FuncCall.
type FuncTester struct {
	T             *testing.T
	Calls         chan FuncCall
	Panic         any
	ReturnValues  []any
	maxGoroutines int
}

// Start starts the function.
func (t *FuncTester) Start(function any, args ...any) {
	// record when the func is done so we can test that, too
	go func() {
		defer func() {
			// FIXME: This can't go this way when the function under test starts goroutines and just returns immediately. If we close the channel, then there's no way for the goroutines to put their calls on the channel.
			// is there any way for us to know how many goroutines a function spawned, and wait for those?
			// num goroutines may be able to check this, but probably not in a parallel mode.
			// there's a possible solution over here: https://github.com/golang/go/blob/master/src/net/http/main_test.go#L26-L51
			// like we could read the stack & see what goroutines were fired off by us?
			// likely, we really just need to not close if goroutines is anything other than 1.
			// in that case, and actually every case, we need our own indicator that we think this func has returned,
			// besides closing the call channel.
			close(t.Calls)

			t.Panic = recover()
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

// SetGoroutines sets the number of goroutines to read till finding the expected call.
func (t *FuncTester) SetGoroutines(num int) {
	t.maxGoroutines = num
}

// TODO: make ID it's own type
// ExpectCall creates a new ExpectedCall, which will perform the various Assert commands when Enforced.
func (t *FuncTester) ExpectCall(id string, args ...any) *ExpectedCall {
	return &ExpectedCall{t, id, args, nil}
}

// ExpectedCall contains the tester, expected call ID, expected args, and any
// forced returns/panic, which can all be asserted and executed with the
// Enforce call.
type ExpectedCall struct {
	t                  *FuncTester
	id                 string
	args               []any
	forcedReturnValues []any
}

// ForceReturn sets the value(s) you want to force to be returned if/when the expected call is found.
func (t *ExpectedCall) ForceReturn(returnValues ...any) *ExpectedCall {
	t.forcedReturnValues = returnValues
	return t
}

// Enforce enforces the expectations with the underlying tester.
func (t *ExpectedCall) Enforce() {
	t.t.AssertCalled(t.id, t.args...).Return(t.forcedReturnValues...)
}

// ExpectReturn creates a new ExpectedReturn, which will perform the check for a returned value when Enforced.
func (t *FuncTester) ExpectReturn(args ...any) *ExpectedReturn {
	return &ExpectedReturn{t, args}
}

// ExpectedReturn contains the tester and expected return values, which can be asserted with the Enforce call.
type ExpectedReturn struct {
	t            *FuncTester
	returnValues []any
}

// Enforce enforces the expectations with the underlying tester.
func (t *ExpectedReturn) Enforce() {
	t.t.AssertReturned(t.returnValues...)
}

// ExpectOrdered returns an ExpectedOrdered, which will walk through each
// enforceable one by one and enforce it when the ExpectOrdered is Enforced.
func (t *FuncTester) ExpectOrdered(enforceables ...enforceable) *ExpectedOrdered {
	return &ExpectedOrdered{t, enforceables}
}

// ExpectedOrdered contains a tester and the enforceables that will be enforced in order when Enforce is called.
type ExpectedOrdered struct {
	t            *FuncTester
	enforceables []enforceable
}

// Enforce enforces the expectations with the underlying tester.
func (eo *ExpectedOrdered) Enforce() {
	for _, e := range eo.enforceables {
		e.Enforce()
	}
}

// an enforceable is something that has the Enforce method.
type enforceable interface {
	Enforce()
	EnforceIfMatch() (enforced, matchable bool)
}

// ExpectUnordered returns an ExpectedUnordered, which will walk through each
// enforceable one by one and enforce it when the ExpectUnordered is Enforced.
func (t *FuncTester) ExpectUnordered(enforceables ...enforceable) *ExpectedUnordered {
	return &ExpectedUnordered{t, enforceables}
}

// ExpectedOrdered contains a tester and the enforceables that will be enforced in order when Enforce is called.
type ExpectedUnordered struct {
	t            *FuncTester
	enforceables []enforceable
}

// Enforce enforces the expectations with the underlying tester.
func (eo *ExpectedUnordered) Enforce() {
	// FIXME: fix the order of operations here - the whole point is that this is unordered.
	// any of the underlying enforceables might be enforceable, or might not, and it's ok, so long as one of them is, right now. For iterable enforceables, we only want to check the first/next item, not the entire list.
	// get a copy of our enforceables
	enforceables := []enforceable{}
	copy(enforceables, eo.enforceables)
	// loop through them till they've all been enforced
	for len(enforceables) > 0 {
		// none have been enforced yet
		enforced := false
		// need a temporary place to store the still-matchable enforceables after we've tried to enforce them.
		newEnforceables := []enforceable{}
		// loop through the current set, and try to enforce.
		for _, e := range enforceables {
			// only try to enforce if we haven't been successful yet.
			if !enforced {
				var matchable bool
				enforced, matchable, checkedAgainst = e.EnforceIfMatch()
				// if this enforceable isn't matchable anymore (it was a single item that was matched, or a collection and all of its items have been matched), then skip adding it to the new list
				if !matchable {
					continue
				}
			}
			// add the enforceable to the new list
			newEnforceables = append(newEnforceables, e)
		}
		// error out if we were unable to enforce anything
		if !enforced {
			eo.t.T.Fatalf("unable to match. Expected any of %v, but the current state is %v", expected, eo.t.State())
		}
		// flip the new list into the main list to keep going
		enforceables = newEnforceables
	}
}
