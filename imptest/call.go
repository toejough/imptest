// Package imptest provides impure function testing functionality.
package imptest

// This file provides Call.

import (
	"fmt"
	"reflect"
	"time"
)

// Call is the basic type that represents a function call.
type (
	Call struct {
		function Function
		args     []any
		returns  chan []any
	}
)

// newCall will create a new Call, set up with the given function and args, as well as a return
// channel for injecting returns into and filling them from.
func newCall(f Function, args ...any) *Call {
	return &Call{function: f, args: args, returns: make(chan []any)}
}

// Name returns the name of the function.
func (c Call) Name() string {
	return GetFuncName(c.function)
}

// Args returns the args passed to the function.
func (c Call) Args() []any {
	return c.args
}

// InjectReturnsWithin injects return values into the underlying return channel. These are
// intended to be filled into the functions own internal return values via FillReturns.
// InjectReturnsWithin panics if the injected values are not consumed by FillReturns
// within the given duration.
func (c Call) InjectReturnsWithin(duration time.Duration, returnValues ...any) {
	panicIfInvalidReturns(c.function, returnValues)

	select {
	case c.returns <- returnValues:
		return
	case <-time.After(duration):
		panic("fill was not called: timed out waiting for " + c.Name() + " to read the injected return values")
	}
}

// FillReturns fills the given return pointers with values from the internal return channel.
// These are intended to come from the values passed to InjectReturns.
func (c Call) FillReturns(returnPointers ...any) {
	returnValues := <-c.returns

	panicIfWrongNumReturns(c.function, returnPointers)

	// necessarily in-line to prove to nilaway that the loops are fine
	if len(returnPointers) != len(returnValues) {
		panic(fmt.Sprintf(
			"num values to fill (%d) != num values injected (%d)",
			len(returnPointers),
			len(returnValues),
		))
	}

	panicIfNotAllPointers(returnPointers)
	panicIfNotEqualTypes(c.function, returnPointers, returnValues)

	// Fill the pointers
	for index := range returnValues {
		returnPointerValue := reflect.ValueOf(returnPointers[index]).Elem()

		// handle untyped nils
		if isUntypedNil(returnValues[index]) {
			returnPointerValue.SetZero()
			continue
		}

		// USEFUL SNIPPETS FROM JSON.UNMARSHAL
		// if returnPointerValue.Kind() != reflect.Pointer || returnPointerValue.IsNil() {
		// 	return &InvalidUnmarshalError{reflect.TypeOf(v)}
		// }
		// v.Set(reflect.ValueOf(oi))
		returnPointerValue.Set(reflect.ValueOf(returnValues[index]))
	}
}

// panicIfNotEqualTypes panics of the types in the first argument set aren't the same as those
// in the second.
func panicIfNotEqualTypes(function Function, returnPointers []any, returnValues []any) {
	for index := range returnValues {
		returnPointerValue := reflect.ValueOf(returnPointers[index]).Elem()
		returnPointerType := getTypeName(returnPointerValue.Type())
		expectedArgType := getTypeName(reflect.TypeOf(function).Out(index))

		if returnPointerType != expectedArgType {
			panic(fmt.Sprintf(
				"wrong return type: return value %d to be filled was type %s, but func (%s) returns type %s",
				index,
				returnPointerType,
				GetFuncName(function),
				expectedArgType,
			))
		}
	}
}

// panicIfNotAllPointers panics if all the args aren't pointers.
func panicIfNotAllPointers(returnPointers []any) {
	// USEFUL SNIPPETS FROM JSON.UNMARSHAL
	// if returnPointerValue.Kind() != reflect.Pointer || returnPointerValue.IsNil() {
	// 	return &InvalidUnmarshalError{reflect.TypeOf(v)}
	// }
	// v.Set(reflect.ValueOf(oi))
	for index := range returnPointers {
		returnPointerValue := reflect.ValueOf(returnPointers[index])
		if returnPointerValue.Kind() != reflect.Pointer || returnPointerValue.IsNil() {
			panic(fmt.Sprintf("cannot fill value into non-pointer at index %d", index))
		}
	}
}
