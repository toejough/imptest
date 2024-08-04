// Package imptest provides impure function testing functionality.
package imptest

import (
	"reflect"

	"github.com/google/uuid"
)

func WrapFunc[T any](function T, calls chan FuncCall) (T, string) {
	// creates a unique ID for the function
	id := GetFuncName(function) + "_" + uuid.New().String()
	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	t := reflect.TypeOf(function)

	relayer := func(in []reflect.Value) []reflect.Value {
		out := make(chan []any)
		calls <- FuncCall{id, in, out}
		outVals := []reflect.Value{}
		outAny := <-out
		for _, a := range outAny {
			outVals = append(outVals, reflect.ValueOf(a))
		}
		return outVals
	}

	// Make a function of the right type.
	v := reflect.MakeFunc(t, relayer)

	// Assign it to the value fn represents.

	// returns both
	return v.Interface().(T), id
}

type FuncCall struct {
	ID  string
	in  []reflect.Value
	Out chan []any
}
