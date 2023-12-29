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
		// TODO: consistently make internal reps the reflect.Values?
		rFunc   reflect.Value
		args    []any
		returns chan []any
	}
)

// newCall will create a new Call, set up with the given function and args, as well as a return
// channel for injecting returns into and filling them from.
func newCall(rf reflect.Value, args ...any) *Call {
	return &Call{rFunc: rf, args: args, returns: make(chan []any)}
}

// Name returns the name of the function.
func (c Call) Name() string {
	return GetFuncName(c.rFunc.Interface())
}

// Args returns the args passed to the function.
func (c Call) Args() []any {
	return c.args
}

// InjectReturns injects return values into the underlying return channel. These are
// intended to be filled into the functions own internal return values via FillReturns.
func (c Call) InjectReturns(returnValues ...any) {
	panicIfInvalidReturns(c.rFunc, returnValues)

	select {
	case c.returns <- returnValues:
		return
	case <-time.After(1 * time.Second):
		panic("fill was not called: timed out waiting for " + c.Name() + " to read the injected return values")
	}
}

// FillReturns fills the given return pointers with values from the internal return channel.
// These are intended to come from the values passed to InjectReturns.
func (c Call) FillReturns(returnPointers ...any) {
	returnValues := <-c.returns

	if len(returnPointers) != len(returnValues) {
		panic(fmt.Sprintf(
			"wrong number of returns: "+
				"the length of the pointer array to fill with return values (%d) does not match the "+
				" length of the return value array injected by the test (%d)",
			len(returnPointers),
			len(returnValues),
		))
	}

	for index := range returnValues {
		// USEFUL SNIPPETS FROM JSON.UNMARSHAL
		// if returnPointerValue.Kind() != reflect.Pointer || returnPointerValue.IsNil() {
		// 	return &InvalidUnmarshalError{reflect.TypeOf(v)}
		// }
		// v.Set(reflect.ValueOf(oi))
		returnPointerValue := reflect.ValueOf(returnPointers[index])
		if returnPointerValue.Kind() != reflect.Pointer || returnPointerValue.IsNil() {
			panic("cannot fill value into non-pointer")
		}

		returnPointerType := reflect.TypeOf(returnPointerValue.Elem().Interface()).Name()
		expectedArgType := c.rFunc.Type().Out(index).Name()

		if returnPointerType != expectedArgType {
			panic(fmt.Sprintf(
				"wrong return type: return value %d to be filled was type %s, but func (%s) returns type %s",
				index,
				returnPointerType,
				GetFuncName(c.rFunc.Interface()),
				expectedArgType,
			))
		}

		returnedValue := reflect.ValueOf(returnValues[index])

		// handle nils
		if !returnedValue.IsValid() {
			returnPointerValue.Elem().SetZero()
			continue
		}
		// Use Elem instead of directly using Set for setting pointers
		returnPointerValue.Elem().Set(returnedValue)
	}
}
