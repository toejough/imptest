// Package imptest provides impure function testing functionality.
package imptest

// This file provides commonly used types, values, and functions that are not large enough
// to justify spliting out into their  own files.

import (
	"reflect"
	"runtime"
	"strings"
)

// Function is here to help us distinguish functions internally, because there is no single
// function _type_ in go.
type Function any

// GetFuncName gets the function's name.
func GetFuncName(f Function) string {
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	// this suffix gets appended sometimes. It's unimportant, as far as I can tell.
	name = strings.TrimSuffix(name, "-fm")

	return name
}
