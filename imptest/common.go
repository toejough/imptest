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

	panicIfNotFunc(function)
	panicIfWrongNumArgs(function, expectedArgs)
	panicIfWrongArgTypes(function, expectedArgs)

	name := GetFuncName(function)
	assertCalledNameIs(tester, called, name)

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
	panicIfNotFunc(f)
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	// this suffix gets appended sometimes. It's unimportant, as far as I can tell.
	name = strings.TrimSuffix(name, "-fm")

	return name
}

// panicIfNotFunc panics if the given object is not a function. It also takes the caller
// so that it can report where the panic came from.
func panicIfNotFunc(evaluate Function) {
	kind := reflect.ValueOf(evaluate).Kind()
	if kind != reflect.Func {
		panic(fmt.Sprintf("must pass a function. received a %s instead.",
			kind.String(),
		))
	}
}

// panicIfWrongNumArgs panics if the number of args given don't match the number of args
// the given function takes.
func panicIfWrongNumArgs(function Function, args []any) {
	numArgs := len(args)

	reflectedFunc := reflect.TypeOf(function)
	numFunctionArgs := reflectedFunc.NumIn()

	if numArgs < numFunctionArgs {
		panic(fmt.Sprintf("Too few args passed. The func (%s) takes %d args,"+
			" but only %d were passed",
			GetFuncName(function),
			numFunctionArgs,
			numArgs,
		))
	} else if numFunctionArgs < numArgs {
		panic(fmt.Sprintf("Too many args passed. The func (%s) only takes %d values,"+
			" but %d were passed",
			GetFuncName(function),
			numFunctionArgs,
			numArgs,
		))
	}
}

// panicIfWrongNumReturns panics if the number of returns given don't match the number of values
// the given function returns.
func panicIfWrongNumReturns(function reflect.Value, returns []any) {
	numReturns := len(returns)

	reflectedFunc := function.Type()
	numFunctionReturns := reflectedFunc.NumOut()

	if numReturns < numFunctionReturns {
		panic(fmt.Sprintf("Too few returns passed. The func (%s) returns %d values,"+
			" but only %d were passed",
			GetFuncName(function.Interface()),
			numFunctionReturns,
			numReturns,
		))
	} else if numFunctionReturns < numReturns {
		panic(fmt.Sprintf("Too many returns passed. The func (%s) only returns %d values,"+
			" but %d were passed",
			GetFuncName(function.Interface()),
			numFunctionReturns,
			numReturns,
		))
	}
}

// panicIfWrongArgTypes panics if the types of args given don't match the types of args
// the given function takes.
func panicIfWrongArgTypes(function Function, args []any) {
	reflectedFunc := reflect.TypeOf(function)

	for index := range args {
		arg := args[index]
		// if the func type is a pointer and the passed Arg is nil, that's ok, too.
		if arg == nil && isNillableKind(reflectedFunc.In(index).Kind()) {
			continue
		}

		functionArgType := reflectedFunc.In(index)
		argType := reflect.TypeOf(arg)

		if functionArgType == argType {
			continue
		}

		if functionArgType.Kind() == reflect.Interface && argType.Implements(functionArgType) {
			continue
		}

		panic(fmt.Sprintf("Wrong arg type. The arg type at index %d for func (%s) is %s,"+
			" but a value of type %s was passed",
			index,
			GetFuncName(function),
			getTypeName(functionArgType),
			getTypeName(argType),
		))
	}
}

// panicIfWrongReturnTypes panics if the types of returns given don't match the types of values
// the given function returns.
func panicIfWrongReturnTypes(function reflect.Value, returns []any) {
	reflectedFunc := function.Type()

	for index := range returns {
		ret := returns[index]
		// if the func type is a pointer and the passed Arg is nil, that's ok, too.
		if ret == nil && isNillableKind(reflectedFunc.Out(index).Kind()) {
			continue
		}

		functionReturnType := reflectedFunc.Out(index)
		retType := reflect.TypeOf(ret)

		if functionReturnType == retType {
			continue
		}

		if functionReturnType.Kind() == reflect.Interface && retType.Implements(functionReturnType) {
			continue
		}

		panic(fmt.Sprintf("Wrong return type. The return type at index %d for func (%s) is %s,"+
			" but a value of type %s was passed",
			index,
			GetFuncName(function.Interface()),
			getTypeName(functionReturnType),
			getTypeName(retType),
		))
	}
}

// panicIfInvalidReturns panics if either the number or type of the given values are mismatched
// with the return signature of the function.
func panicIfInvalidReturns(function reflect.Value, assertedReturns []any) {
	panicIfWrongNumReturns(function, assertedReturns)
	panicIfWrongReturnTypes(function, assertedReturns)
}
