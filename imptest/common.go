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

	actualArgs := theCall.Args()

	for i := range expectedArgs {
		if !deepEqual(actualArgs[i], expectedArgs[i]) {
			tester.Fatalf("wrong values: the function %s was expected to be called with %#v at index %d but was called with %#v",
				theCall.Name(), expectedArgs[i], i, actualArgs[i],
			)
		}
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
func panicIfWrongNumReturns(function Function, returns []any) {
	numReturns := len(returns)

	reflectedFunc := reflect.TypeOf(function)
	numFunctionReturns := reflectedFunc.NumOut()

	if numReturns < numFunctionReturns {
		panic(fmt.Sprintf("Too few returns passed. The func (%s) returns %d values,"+
			" but only %d were passed",
			GetFuncName(function),
			numFunctionReturns,
			numReturns,
		))
	} else if numFunctionReturns < numReturns {
		panic(fmt.Sprintf("Too many returns passed. The func (%s) only returns %d values,"+
			" but %d were passed",
			GetFuncName(function),
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
func panicIfWrongReturnTypes(function Function, returns []any) {
	reflectedFunc := reflect.TypeOf(function)

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
			GetFuncName(function),
			getTypeName(functionReturnType),
			getTypeName(retType),
		))
	}
}

// panicIfInvalidReturns panics if either the number or type of the given values are mismatched
// with the return signature of the function.
func panicIfInvalidReturns(function Function, assertedReturns []any) {
	panicIfWrongNumReturns(function, assertedReturns)
	panicIfWrongReturnTypes(function, assertedReturns)
}

// deepEqual checks whether two values are deeply equal.
// deepEqual calls functions equal if their names are equal.
// For everything else it depends on reflect.DeepEqual.
func deepEqual(actual, expected any) bool {
	// handle, for instance, nil == (*int)nil
	// check out https://groups.google.com/g/golang-nuts/c/rVO7ld8KIXI?pli=1 or
	// for more. Rob Pike agrees this is weird & gross. "unsatisfactory."
	if isNil(actual) && isNil(expected) {
		return true
	}

	// Special handling for functions. For our purposes, call funcs with the same names equal.
	if reflect.TypeOf(actual).Kind() == reflect.Func &&
		reflect.TypeOf(expected).Kind() == reflect.Func &&
		GetFuncName(actual) == GetFuncName(expected) {
		return true
	}

	return reflect.DeepEqual(actual, expected)
}

// isNil returns whether the value is nil.
func isNil(value any) bool { return isUntypedNil(value) || isTypedNil(value) }

// isTypedNil returns whether the value is a typed nil.
func isTypedNil(value any) bool {
	reflectedValue := reflect.ValueOf(value)
	return isNillableKind(reflectedValue.Kind()) && reflectedValue.IsNil()
}

// isUntypedNil returns whether the value is an untyped nil.
func isUntypedNil(value any) bool { return !reflect.ValueOf(value).IsValid() }

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
		panic("unable to check for nillability for unknown kind " + kind.String())
	}
}
