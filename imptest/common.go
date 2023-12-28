// Package imptest provides impure function testing functionality.
package imptest

// This file provides commonly used types, values, and functions that are not large enough
// to justify spliting out into their  own files.

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// Function is here to help us distinguish functions internally, because there is no single
// function _type_ in go.
type Function any

// AssertCallIs checks whether the given call matches the function and args given, and fails
// the test if not.
func AssertCallIs(tester Tester, called *Call, function Function, expectedArgs ...any) *Call {
	tester.Helper()

	name := GetFuncName(function)
	assertCalledNameIs(tester, called, name)

	reflectedFunc := reflect.TypeOf(function)
	supportedNumArgs := reflectedFunc.NumIn()
	expectedNumArgs := len(expectedArgs)

	if expectedNumArgs < supportedNumArgs {
		panic(fmt.Sprintf(
			"too few args in the expected argument list (%d)"+
				// I want to keep these error messages independent
				" compared to the number of arguments (%s) supports (%d)",
			expectedNumArgs,
			GetFuncName(function),
			supportedNumArgs,
		))
	} else if expectedNumArgs > supportedNumArgs {
		panic(fmt.Sprintf(
			"too many args in the expected argument list (%d)"+
				" compared to the number of arguments (%s) supports (%d)",
			expectedNumArgs,
			GetFuncName(function),
			supportedNumArgs,
		))
	}

	for index := range expectedArgs {
		argAsserted := expectedArgs[index]
		assertedType := reflect.TypeOf(argAsserted).Name()
		actualType := reflectedFunc.In(index).Name()

		if assertedType != actualType {
			panic(fmt.Sprintf(
				"Wrong type."+
					"The type asserted for the arg at index %d is %s,"+
					"but the actual type for that arg for function %s is %s",
				index,
				assertedType,
				GetFuncName(function),
				actualType,
			))
		}
	}

	assertArgsAre(tester, called, expectedArgs...)

	return called
}

// assertCalledNameIs checks whether the given name matches the name of the function in the Call,
// and fails the test if not.
func assertCalledNameIs(t Tester, c *Call, expectedName string) {
	t.Helper()

	if c.Name() != expectedName {
		t.Fatalf("wrong func called: the called function was expected to be %s, but was %s instead", expectedName, c.Name())
	}
}

// assertArgsAre checks whether the given args match the args in the Call, and fails the
// test if not.
func assertArgsAre(tester Tester, theCall *Call, expectedArgs ...any) {
	tester.Helper()

	if !reflect.DeepEqual(theCall.Args(), expectedArgs) {
		tester.Fatalf("wrong values: the function %s was expected to be called with %#v but was called with %#v",
			theCall.Name(), expectedArgs, theCall.Args(),
		)
	}
}

// GetFuncName gets the function's name.
func GetFuncName(f Function) string {
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	// this suffix gets appended sometimes. It's unimportant, as far as I can tell.
	name = strings.TrimSuffix(name, "-fm")

	return name
}

// panicIfNotFunc panics if the given object is not a function. It also takes the caller
// so that it can report where the panic came from.
// TODO: don't pass the caller.
func panicIfNotFunc(evaluate Function) {
	kind := reflect.ValueOf(evaluate).Kind()
	if kind != reflect.Func {
		panic(fmt.Sprintf("must pass a function. received a %s instead.",
			kind.String(),
		))
	}
}
