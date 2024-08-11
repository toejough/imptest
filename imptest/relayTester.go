// Package imptest provides impure function testing functionality.
package imptest

// This file provides RelayTester.

import (
	"reflect"
	"time"
)

// Testing philosophy:
//
// Constructors ("New...") are implicitly tested by their use in other tests.
// Other public functions and methods are explicitly tested.
// Private functions are implicitly tested via the public functions' tests.

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

// Behavioral properties philosophy:
//
// Every public function will satisfy a number of behavioral properties it guarantees.
// Those properties will be tested.

// Dependency philosophy:
//
// Function dependencies shall be passed in via a final argument to the function.
// Method dependencies shall be passed in via arguments to the type's constructor.
// Dependency argument types shall be interfaces, not structs.

// **Constructors**

// NewRelayTester creates and returns a pointer to a new RelayTester with a
// new CallRelay set up, with one-second default timeouts.
//
// It is provided so that the normal user won't have to fill in a bunch of
// sane dependencies themselves. If you want to provide the dependencies yourself,
// please use NewRelayTesterCustom.
func NewRelayTester(t Tester) *RelayTester {
	return NewRelayTesterCustom(t, NewCallRelay(&defaultCallRelayDeps{}), time.Second)
}

// defaultCallRelayDeps is the default implementation of CallRelayDeps, which uses the
// standard lib time.After to supply the After method.
type defaultCallRelayDeps struct{}

// After takes a duration and returns a channel which returns the time elapsed once the duration
// has been met or exceeded.
func (deps *defaultCallRelayDeps) After(d time.Duration) <-chan time.Time { return time.After(d) }

// NewCall returns a pointer to a new Call object.
func (deps *defaultCallRelayDeps) NewCall(function Function, args ...any) *Call {
	return NewCall(&defaultCallDeps{}, time.Second, function, args...)
}

// defaultCallDeps is the default implementation of CallDeps, which uses the
// standard lib time.After to supply the After method.
type defaultCallDeps struct{}

// After takes a duration and returns a channel which returns the time elapsed once the duration
// has been met or exceeded.
func (deps *defaultCallDeps) After(d time.Duration) <-chan time.Time { return time.After(d) }

// NewRelayTesterCustom provides a pointer to a new RelayTester with the given test object and CallRelay.
// If you want a new RelayTester with sane default call-relay and timeouts already set, please
// use NewRelayTester.
func NewRelayTesterCustom(t Tester, r *CallRelay, d time.Duration) *RelayTester {
	return &RelayTester{
		t:              t,
		relay:          r,
		defaultTimeout: d,
		function:       nil,
		returnValues:   nil,
	}
}

// RelayTester is a convenience wrapper over interacting with the CallRelay and
// a testing library that generally follows the interface of the standard test.T.
type (
	RelayTester struct {
		t              Tester
		relay          *CallRelay
		defaultTimeout time.Duration
		function       Function
		returnValues   []any
	}
)

// **Public Functions & Methods**

// Start calls the given function with the given args in a goroutine and returns immediately.
// The intent is to start the function running and then return control flow to the calling
// test to act as a coroutine, passing data back and forth to the function under test to verify
// correct behavior.
//
// Tested properties you can depend on:
//
//	-[x] Start will panic if 'function' is anything other than a function.
//	-[x] Start will panic if 'function' takes a different number of args than are passed as 'args'
//	-[x] Start will panic if 'function' takes args of different types than are passed as 'args'
//	-[x] Start will make 'function' available for inspection by AssertReturned
//	-[x] Start will call the function in a goroutine and return control to the caller immediately.
//	-[x] Start will call the function with the given arguments.
//	-[x] Start will call the function exactly once.
//	-[x] Start will make the function's return values available via GetReturns and AssertReturned.
//	-[x] Start will recover any panic from the function and call Tester.Fatalf with it.
//	-[x] Start will shut down the CallRelay when the function exits.
func (rt *RelayTester) Start(function Function, args ...any) {
	panicIfInvalidCall(function, args)

	rt.function = function

	go func() {
		defer func() {
			rt.relay.Shutdown()

			if r := recover(); r != nil {
				rt.t.Fatalf("caught panic from started function: %v", r)
			}
		}()

		rt.returnValues = callFunc(function, args)
	}()
}

// AssertFinishes checks that the underlying relay was shut down within the default time,
// implying that the function under test was done within the default time.
// Otherwise it fails the test.
//
// Tested properties you can depend on:
//   - [x] AssertFinishes returns once the underlying relay is shut down
//   - [x] AssertFinishes will not return successfully before the underlying relay is shut down
//   - [x] AssertFinishes will fail the test when the defaultTimeout time elapses before the underlying
//     relay is shut down. The timeout countdown starts when AssertFinishes is called.
func (rt *RelayTester) AssertFinishes() {
	rt.AssertDoneWithin(rt.defaultTimeout)
}

// AssertDoneWithin checks that the underlying relay was shut down within the given time,
// implying that the function under test was done within the given time.
// Otherwise it fails the test.
//
// Tested properties you can depend on:
//   - [x] AssertDoneWithin returns once the underlying relay is shut down
//   - [x] AssertDoneWithin will not return successfully before the underlying relay is shut down
//   - [x] AssertDoneWithin will fail the test when the given time elapses before the underlying
//     relay is shut down. The timeout countdown starts when AssertDoneWithin is called.
func (rt *RelayTester) AssertDoneWithin(d time.Duration) {
	rt.t.Helper()

	if err := rt.relay.WaitForShutdown(d); err != nil {
		rt.t.Fatalf("the relay has not shut down yet: %s", err)
	}
}

// AssertReturned checks that the function returned the given values. Otherwise it fails the test.
//
// Tested properties you can depend on:
//   - [x] AssertReturned will panic if there are a different number of returns passed in than the
//     function passed to Start returned
//   - [x] AssertReturned will panic if the types passed are different than the types the function
//     passed to Start returns
//   - [x] AssertReturned will fail the test if the values passed in do not match those returned by
//     the function passed to Start.
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

// PutCall puts the function and args onto the underlying CallRelay as a Call.
//
// Tested properties you can depend on:
//   - [x] PutCall will panic if the function is not a function
//   - [x] PutCall will panic if the args passed do not match the number of args the function takes
//   - [x] PutCall will panic if the args passed do not match the types of the args the function takes
//   - [x] PutCall will put the call onto the underlying CallRelay for inspection by the test via AssertNextCallIs
//     and GetNextCall, and their "within" variations.
func (rt *RelayTester) PutCall(f Function, a ...any) *Call {
	panicIfInvalidCall(f, a)

	return rt.relay.putCall(f, a...)
}

// GetNextCall gets the next Call from the underlying CallRelay.
// GetNextCall fails the test if the call was not available within the default timeout.
//
// Tested properties you can depend on:.
//   - [x] GetNextCall will return the next call which has not already been popped by previous calls
//     to GetNextCall or AssertNextCallIs, from the underlying CallRelay, in the order
//     inserted by PutCall.
//   - [x] GetNextCall will fail the test if the defaultTimeout passes before the next call is available.
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

// GetReturns gets the values returned by the function.
func (rt *RelayTester) GetReturns() []any { return rt.returnValues }

// Tester is the necessary testing interface for use with RelayTester.
type Tester interface {
	Helper()
	Fatalf(message string, args ...any)
	Failed() bool
}

// **Private Functions & Methods**

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

// reflectValuesOf returns reflected values for all of the values.
func reflectValuesOf(args []any) []reflect.Value {
	rArgs := make([]reflect.Value, len(args))
	for i := range args {
		rArgs[i] = reflect.ValueOf(args[i])
	}

	return rArgs
}

// panicIfInvalidCall panics if the passed function is in fact not a function.
// panicIfInvalidCall panics if the arg num or type is mismatched with the function's signature.
func panicIfInvalidCall(function Function, args []any) {
	panicIfNotFunc(function)
	panicIfWrongNumArgs(function, args)
	panicIfWrongArgTypes(function, args)
}
