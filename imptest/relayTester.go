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
		t:            t,
		relay:        NewCallRelay(),
		function:     nil,
		args:         nil,
		returnValues: nil,
	}
}

// RelayTester is a convenience wrapper over interacting with the CallRelay and
// a testing library that generally follows the interface of the standard test.T.
type RelayTester struct {
	t            Tester
	relay        *CallRelay
	function     Function
	args         []any
	returnValues []any
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
	panicIfInvalidCall(function, args)

	rt.function = function
	rt.args = args

	go func() {
		defer func() { rt.relay.Shutdown() }()

		rt.returnValues = call(function, args)
	}()
}

// call calls the given function with the given args, and returns the return values from that call.
func call(f Function, args []any) []any {
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

	reflectedFunc := reflect.TypeOf(rt.function)

	for index := range assertedReturns {
		returned := rt.returnValues[index]
		returnAsserted := assertedReturns[index]
		// TODO: need a better equality check, like for functions
		// if the func type is a pointer and the passed Arg is nil, that's ok, too.
		if returnAsserted == nil && isNillableKind(reflectedFunc.Out(index).Kind()) {
			continue
		}

		if !reflect.DeepEqual(returned, returnAsserted) {
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

// getTypeName gets the type's name, if it has one. If it does not have one, getTypeName
// will return the type's string.
func getTypeName(t reflect.Type) string {
	if t.Name() != "" {
		return t.Name()
	}

	return t.String()
}

// isNillableKind returns true if the kind passed is nillable.
// According to https://pkg.go.dev/reflect#Value.IsNil, this is the case for
// chan, func, interface, map, pointer, or slice kinds.
func isNillableKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	case reflect.Invalid, reflect.Bool, reflect.Int,
		reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Array,
		reflect.String, reflect.Struct, reflect.UnsafePointer:
		return false
	default:
		// This code is only coverable if go itself introduces a new kind into the reflect package
		// ...or if we just force a new kind int during whitebox testing?
		// I'm not worried about writing a test to cover this.
		panic(fmt.Sprintf("unable to check for nillability for unknown kind %s", kind.String()))
	}
}

// PutCallputs the function and args onto the underlying CallRelay as a Call.
func (rt *RelayTester) PutCall(f Function, a ...any) *Call {
	panicIfInvalidCall(f, a)

	return rt.relay.putCall(f, a...)
}

// GetNextCall gets the next Call from the underlying CallRelay.
func (rt *RelayTester) GetNextCall() *Call {
	call, err := rt.relay.Get()
	if err != nil {
		rt.t.Fatalf(err.Error())
		return nil
	}

	return call
}

// AssertNextCallIs gets the next Call from the underlying CallRelay and checks that the
// given function and args match that Call. Otherwise, it fails the test.
func (rt *RelayTester) AssertNextCallIs(function Function, args ...any) *Call {
	panicIfInvalidCall(function, args)

	call := rt.GetNextCall()

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
