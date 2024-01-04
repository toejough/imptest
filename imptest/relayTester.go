// Package imptest provides impure function testing functionality.
package imptest

// This file provides RelayTester.

import (
	"reflect"
	"time"
)

// NewRelayTesterCustom provides a pointer to a new RelayTester with the given test object and CallRelay.
func NewRelayTesterCustom(t Tester, r *CallRelay, d time.Duration) *RelayTester {
	return &RelayTester{
		t:              t,
		relay:          r,
		defaultTimeout: d,
		function:       nil,
		args:           nil,
		returnValues:   nil,
	}
}

// NewRelayTester creates and returns a pointer to a new RelayTester with a
// new default CallRelay set up.
func NewRelayTester(t Tester) *RelayTester {
	return NewRelayTesterCustom(t, NewCallRelay(&defaultCallRelayDeps{}, time.Second), time.Second)
}

// defaultCallRelayDeps is the default implementation of CallRelayDeps, which uses the
// standard lib time.After to supply the After method.
type defaultCallRelayDeps struct{}

// After takes a duration and returns a channel which returns the time elapsed once the duration
// has been met or exceeded.
func (deps *defaultCallRelayDeps) After(d time.Duration) <-chan time.Time { return time.After(d) }

// NewCall returns a pointer to a new Call object.
func (deps *defaultCallRelayDeps) NewCall(duration time.Duration, function Function, args ...any) *Call {
	return NewCall(&defaultCallDeps{}, duration, function, args...)
}

// defaultCallDeps is the default implementation of CallDeps, which uses the
// standard lib time.After to supply the After method.
type defaultCallDeps struct{}

// After takes a duration and returns a channel which returns the time elapsed once the duration
// has been met or exceeded.
func (deps *defaultCallDeps) After(d time.Duration) <-chan time.Time { return time.After(d) }

// RelayTester is a convenience wrapper over interacting with the CallRelay and
// a testing library that generally follows the interface of the standard test.T.
type (
	RelayTester struct {
		t              Tester
		relay          *CallRelay
		defaultTimeout time.Duration
		function       Function
		args           []any
		returnValues   []any
	}
)

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
	panicIfInvalidCall(function, args)

	rt.function = function
	rt.args = args

	go func() {
		defer func() { rt.relay.Shutdown() }()

		rt.returnValues = callFunc(function, args)
	}()
}

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

// AssertFinishes checks that the underlying relay was shut down within the default time,
// implying that the function under test was done within the default time.
// Otherwise it fails the test.
func (rt *RelayTester) AssertFinishes() {
	rt.AssertDoneWithin(rt.defaultTimeout)
}

// AssertDoneWithin checks that the underlying relay was shut down within the given time,
// implying that the function under test was done within the given time.
// Otherwise it fails the test.
func (rt *RelayTester) AssertDoneWithin(d time.Duration) {
	rt.t.Helper()

	if err := rt.relay.WaitForShutdown(d); err != nil {
		rt.t.Fatalf("the relay has not shut down yet: %s", err)
	}
}

// AssertReturned checks that the function returned the given values. Otherwise it fails the test.
func (rt *RelayTester) AssertReturned(assertedReturns ...any) {
	panicIfInvalidReturns(rt.function, assertedReturns)

	for index := range assertedReturns {
		returned := rt.returnValues[index]
		returnAsserted := assertedReturns[index]

		if !deepEqual(returned, returnAsserted) {
			rt.t.Fatalf(
				"The func returned the wrong value for a return: "+
					"the return value at index %d was expected to be %#v, "+
					"but it was %#v",
				index, returnAsserted, returned,
			)

			return
		}
	}
}

// PutCallputs the function and args onto the underlying CallRelay as a Call.
func (rt *RelayTester) PutCall(f Function, a ...any) *Call {
	panicIfInvalidCall(f, a)

	return rt.relay.putCall(f, a...)
}

// GetNextCall gets the next Call from the underlying CallRelay.
// GetNextCall fails the test if the call was not available within the default timeout.
func (rt *RelayTester) GetNextCall() *Call {
	return rt.GetNextCallWithin(rt.defaultTimeout)
}

// GetNextCallWithin gets the next Call from the underlying CallRelay.
// GetNextCallWithin fails the test if the call was not available within the given timeout.
func (rt *RelayTester) GetNextCallWithin(d time.Duration) *Call {
	call, err := rt.relay.GetCallWithin(d)
	if err != nil {
		rt.t.Fatalf(err.Error())
		return nil
	}

	return call
}

// AssertNextCallIs gets the next Call from the underlying CallRelay and checks that the
// given function and args match that Call. Otherwise, it fails the test.
// AssertNextCallIs fails the test if the call is not available within the default timeout.
func (rt *RelayTester) AssertNextCallIs(function Function, args ...any) *Call {
	return rt.AssertNextCallWithin(rt.defaultTimeout, function, args...)
}

// AssertNextCallWithin gets the next Call from the underlying CallRelay and checks that the
// given function and args match that Call. Otherwise, it fails the test.
// AssertNextCallWithin fails the test if the call is not available within the given timeout.
func (rt *RelayTester) AssertNextCallWithin(d time.Duration, function Function, args ...any) *Call {
	panicIfInvalidCall(function, args)

	call := rt.GetNextCallWithin(d)

	return AssertCallIs(rt.t, call, function, args...)
}

// panicIfInvalidCall panics if the passed function is in fact not a function.
// panicIfInvalidCall panics if the arg num or type is mismatched with the function's signature.
func panicIfInvalidCall(function Function, args []any) {
	panicIfNotFunc(function)
	panicIfWrongNumArgs(function, args)
	panicIfWrongArgTypes(function, args)
}

// GetReturns gets the values returned by the function.
func (rt *RelayTester) GetReturns() []any { return rt.returnValues }

// Tester is the necessary testing interface for use with RelayTester.
type Tester interface {
	Helper()
	Fatalf(message string, args ...any)
	Failed() bool
}
