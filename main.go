package main

import (
	"fmt"
	"reflect"
)

func sumThree(a, b, c int) int {
	return a + b + c
}

func makeLog[T any](f T) T {
	// fptr is a pointer to a function.
	// Obtain the function value itself (likely nil) as a reflect.Value
	// so that we can query its type and then set the value.
	t := reflect.TypeOf(f)

	log := func(in []reflect.Value) []reflect.Value {
		fmt.Println(in)
		numOut := t.NumOut()
		out := []reflect.Value{}
		for i := range numOut {
			out = append(out, reflect.New(t.Out(i)).Elem())
		}
		return out
	}

	// Make a function of the right type.
	v := reflect.MakeFunc(t, log)

	// Assign it to the value fn represents.
	return v.Interface().(T)
}

func main() {
	logSumThree := makeLog(sumThree)
	logSumThree(1, 2, 3)
}
