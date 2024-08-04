// Package imptest provides impure function testing functionality.
package imptest

import (
	"reflect"

	"github.com/google/uuid"
)

func WrapFunc[T any](function T, calls chan FuncCall) (T, string) {
	// creates a unique ID for the function
	funcID := GetFuncName(function) + "_" + uuid.New().String()

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := reflect.TypeOf(function)

	relayer := func(args []reflect.Value) (returnValues []reflect.Value) {
		// Create a channel to receive return values on
		returnValuesChan := make(chan []any)

		// Submit this call to the calls channel
		calls <- FuncCall{funcID, args, returnValuesChan}

		// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
		for _, a := range <-returnValuesChan {
			returnValues = append(returnValues, reflect.ValueOf(a))
		}

		return returnValues
	}

	// Make a function of the right type.
	// Ignore the type assertion lint check - we are depending on MakeFunc to
	// return the correct type, as documented. If it fails to, the only thing
	// we'd do is panic anyway.
	wrapped := reflect.MakeFunc(funcType, relayer).Interface().(T) //nolint: forcetypeassert

	// returns both the wrapped func and the ID
	return wrapped, funcID
}

type FuncCall struct {
	ID               string
	args             []reflect.Value
	ReturnValuesChan chan []any
}
