package imptest_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
	"pgregory.net/rapid"
)

func TestStartPanicsWithNonFunction(t *testing.T) {
	t.Parallel()
	rapid.Check(t, testStartPanicsWithNonFunction)
}

func testStartPanicsWithNonFunction(rapidT *rapid.T) {
	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := generateRandomValueNonFunc(rapidT)

	mockedt.Wrap(func() {
		// When the func is run with something that isn't a function
		defer expectPanicWith(mockedt, "must pass a function")
		tester.Start(argFunc)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		rapidT.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func generateRandomValueNonFunc(rapidT *rapid.T) any {
	kindFuncs := mapOfKindFuncs()
	delete(kindFuncs, "Func")
	vals := valuesOf(kindFuncs)

	index := rapid.IntRange(0, len(vals)-1).Draw(rapidT, "index")

	return vals[index]()
}

func mapOfKindFuncs() map[string]func() any {
	return map[string]func() any{
		"Bool":       func() any { var v bool; return v },
		"Int":        func() any { var v int; return v },
		"Int8":       func() any { var v int8; return v },
		"Int16":      func() any { var v int16; return v },
		"Int32":      func() any { var v int32; return v },
		"Int64":      func() any { var v int64; return v },
		"Uint":       func() any { var v uint; return v },
		"Uint8":      func() any { var v uint8; return v },
		"Uint16":     func() any { var v uint16; return v },
		"Uint32":     func() any { var v uint32; return v },
		"Uint64":     func() any { var v uint64; return v },
		"Uintptr":    func() any { var v uintptr; return v },
		"Float32":    func() any { var v float32; return v },
		"Float64":    func() any { var v float64; return v },
		"Complex64":  func() any { var v complex64; return v },
		"Complex128": func() any { var v complex128; return v },
		"Array":      func() any { var v [0]int; return v },
		"Chan":       func() any { var v chan int; return v },
		"Func":       func() any { var v func(); return v },
		"Interface":  func() any { var v any; return v },
		"Map":        func() any { var v map[int]int; return v },
		"Pointer":    func() any { var v *int; return v },
		"Slice":      func() any { var v []int; return v },
		"String":     func() any { var v string; return v },
		"Struct":     func() any { var v struct{}; return v },
		// "UnsafePointer": func() any { var v nope; return v },
	}
}

func valuesOf[K comparable, V any](m map[K]V) []V {
	values := []V{}
	for _, v := range m {
		values = append(values, v)
	}

	return values
}

func TestStartPanicsWithWrongNumArgs(t *testing.T) {
	t.Parallel()

	mockedt := newMockedTestingT()

	// When the test is run
	mockedt.Wrap(func() {
		rapid.Check(mockedt, testStartPanicsWithWrongNumArgs)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func testStartPanicsWithWrongNumArgs(rapidT *rapid.T) {
	// Given testing needs
	tester := imptest.NewRelayTester(rapidT)

	// Given FUT
	numArgs := rapid.IntRange(0, 5).Draw(rapidT, "numArgs")
	argFunc := funcOfNArgs(numArgs)

	// Given the wrong number of args
	numArgsToPass := rapid.IntRange(0, 10).Filter(func(i int) bool { return i != numArgs }).Draw(rapidT, "numArgsToPass")
	argsToPass := make([]any, numArgsToPass)

	// Then we expect a panic
	expectedMessage := "Too few args"
	if numArgs < numArgsToPass {
		expectedMessage = "Too many args"
	}
	defer expectPanicWith(rapidT, expectedMessage)

	// (when the test is actually run)
	tester.Start(argFunc, argsToPass...)
}

func funcOfNArgs(numArgs int) any {
	args := make([]reflect.Type, numArgs)

	for i := range args {
		args[i] = reflect.TypeOf(0)
	}

	argFunc := reflect.New(reflect.FuncOf(args, nil, false)).Elem().Interface()

	return argFunc
}

func TestStartRunsFUTInGoroutine(t *testing.T) {
	t.Parallel()

	// Given test needs
	tester := imptest.NewRelayTester(t)
	// Given inputs
	lockchan := make(chan struct{})
	waitchan := make(chan struct{})
	wait := func() {
		<-lockchan
	}

	// release the lock at the end of the test
	defer close(lockchan)

	// When the func is run
	go func() {
		tester.Start(wait)
		close(waitchan)
	}()

	// Then the return from waitchan should be immediate
	select {
	case <-waitchan:
	case <-time.After(time.Second):
		t.Error("waitchan never closed, indicating function was run synchronously instead of in a goroutine.")
	}
}

func TestStartPanicsWithWrongArgTypes(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(mockedt, "Wrong arg type")
		tester.Start(argFunc, "1")
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}
