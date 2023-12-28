// Package imptest provides impure function testing functionality.
package imptest

// This file provides RelayTester.

import (
	"fmt"
	"reflect"
	"time"
)

// NewTester provides a pointer to a new RelayTester with the given test object set and
// a new CallRelay properly set up.
func NewTester(t Tester) *RelayTester {
	return &RelayTester{
		T:        t,
		Relay:    NewCallRelay(),
		function: nil,
		returns:  nil,
	}
}

// RelayTester is a convenience wrapper over interacting with the CallRelay and
// a testing library that generally follows the interface of the standard test.T.
type RelayTester struct {
	// TODO: is T unnecessarily public?
	T Tester
	// TODO: is Relay unnecessarily public?
	Relay    *CallRelay
	function Function
	returns  []reflect.Value
}

// Start calls the give function with the given args in a goroutine and returns immediately.
// The intent is to start the function running and then return control flow to the calling
// test in order to have it assert various calls are happening in the right order, inject
// the necessary return values into them, and finally assert that the function is complete
// and returned the right values.
//
// Start will catch panics, reporting them as fatal test failures.
//
// Start will shut down the CallRelay for the tester when the function returns.
//
// Start will store the return values from the function.
func (rt *RelayTester) Start(function Function, args ...any) {
	rt.function = function

	go func() {
		// get args as reflect.Values
		rArgs := make([]reflect.Value, len(args))
		for i := range args {
			rArgs[i] = reflect.ValueOf(args[i])
		}

		// define some cleanup
		defer func() {
			// catch and handle bad args
			if r := recover(); r != nil {
				rt.T.Fatalf("failed to call %s with args (%v): %v", GetFuncName(function), rArgs, r)
			}

			// always shutdown afterwards
			rt.Relay.Shutdown()
		}()

		// actually call the function
		rt.returns = reflect.ValueOf(function).Call(rArgs)
	}()
}

// AssertDoneWithin checks that the underlying relay was shut down within the given time,
// implying that the function under test was done within the given time.
// Otherwise it fails the test.
func (rt *RelayTester) AssertDoneWithin(d time.Duration) {
	rt.T.Helper()

	if err := rt.Relay.WaitForShutdown(d); err != nil {
		rt.T.Fatalf("the relay has not shut down yet: %s", err)
	}
}

// AssertReturned checks that the function returned the given values. Otherwise it fails the test.
func (rt *RelayTester) AssertReturned(assertedReturns ...any) {
	lenReturnsAsserted := len(assertedReturns)

	reflectedFunc := reflect.TypeOf(rt.function)
	numFunctionReturns := reflectedFunc.NumOut()

	if numFunctionReturns > lenReturnsAsserted {
		panic(fmt.Sprintf("Too few return values asserted. The func (%s) returns %d values,"+
			" but only %d were asserted",
			GetFuncName(rt.function),
			numFunctionReturns,
			lenReturnsAsserted,
		))
	} else if numFunctionReturns < lenReturnsAsserted {
		panic(fmt.Sprintf("Too many return values asserted. The func (%s) only returns %d values,"+
			" but %d were asserted",
			GetFuncName(rt.function),
			numFunctionReturns,
			lenReturnsAsserted,
		))
	}

	for index := range assertedReturns {
		returned := rt.returns[index].Interface()
		returnAsserted := assertedReturns[index]
		// if the func type is a pointer and the passed Arg is nil, that's ok, too.
		if returnAsserted == nil && reflectedFunc.Out(index).Kind() == reflect.Pointer {
			continue
		}
		// TODO: investigate undefined types. pointers don't seem to have a name.
		returnType := reflectedFunc.Out(index).Name()
		assertedType := reflect.TypeOf(returnAsserted).Name()

		if returnType != assertedType {
			panic(fmt.Sprintf("Wrong return type asserted. The return at index %d from func (%s) is %s,"+
				" but a value of type %s was asserted",
				index,
				GetFuncName(rt.function),
				returnType,
				assertedType,
			))
		}

		if !reflect.DeepEqual(returned, returnAsserted) {
			rt.T.Fatalf(
				"The func returned the wrong value for a return: "+
					"the return value at index %d was expected to be %#v, "+
					"but it was %#v",
				index, returnAsserted, returned,
			)

			return
		}
	}
}

// PutCall puts the function and args onto the underlying CallRelay as a Call.
func (rt *RelayTester) PutCall(f Function, a ...any) *Call { return rt.Relay.PutCall(f, a...) }

// GetNextCall gets the next Call from the underlying CallRelay.
func (rt *RelayTester) GetNextCall() *Call {
	call, err := rt.Relay.Get()
	if err != nil {
		rt.T.Fatalf(err.Error())
		return nil
	}

	return call
}

// AssertNextCallIs gets the next Call from the underlying CallRelay and checks that the
// given function and args match that Call. Otherwise, it fails the test.
func (rt *RelayTester) AssertNextCallIs(function Function, args ...any) *Call {
	rt.T.Helper()
	panicIfNotFunc(function)

	call := rt.GetNextCall()

	if rt.T.Failed() {
		return nil
	}

	return AssertCallIs(rt.T, call, function, args...)
}

// GetReturns gets the values returned by the function.
func (rt *RelayTester) GetReturns() []reflect.Value { return rt.returns }

// Tester is the necessary testing interface for use with RelayTester.
type Tester interface {
	Helper()
	// FIXME: this really needs to panic, but the testing mock doesn't, and that causes all kinds of other
	// awkward logic in this library...
	Fatalf(message string, args ...any)
	Failed() bool
}
