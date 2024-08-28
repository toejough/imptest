// Package imptest provides impure function testing functionality.
package imptest

// This file provides RelayTester.

import (
	"reflect"
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
