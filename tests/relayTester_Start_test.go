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
	tester := imptest.NewRelayTester(rapidT)

	// Given FUT
	argFunc := generateRandomValueNonFunc(rapidT)

	// When the func is run with something that isn't a function
	defer expectPanicWith(rapidT, "must pass a function")
	tester.Start(argFunc)
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

	rapid.Check(t, testStartPanicsWithWrongNumArgs)
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

func TestStartPanicsWithWrongArgTypes(t *testing.T) {
	t.Parallel()

	// Given testing needs
	tester := imptest.NewRelayTester(t)

	// Given FUT
	// could use rapid and reflect to make random types, but that feels
	// like overkill.
	argFunc := func(_ int) {}

	// When the func is run with the wrong number of args
	defer expectPanicWith(t, "Wrong arg type")
	tester.Start(argFunc, "1")
}

func TestStartIsOkWithVariadicFuncs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	tester := imptest.NewRelayTester(t)

	// Given FUT with variadic args
	testFunc := func(_ ...any) {}

	// When func is run with start
	// Then no failure, because the variadic args were accepted
	tester.Start(testFunc, "first", "second", "third")
}

func TestStartCallsFuncWithArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	tester := imptest.NewRelayTester(t)
	doneChan := make(chan struct{})

	// Given FUT
	var argsPassed []any

	testFunc := func(args ...any) {
		argsPassed = args

		close(doneChan)
	}

	// When func is run with start
	intArg := 1
	strArg := "hi there"
	structArg := struct{}{}
	tester.Start(testFunc, intArg, strArg, &structArg)

	// When we wait for the testFunc to finish
	select {
	case <-doneChan:
	case <-time.After(time.Second):
		panic("doneChan never closed, indicating function never completed.")
	}

	// Then the args are equivalent
	expectedArgs := []any{intArg, strArg, &structArg}
	if !reflect.DeepEqual(argsPassed, expectedArgs) {
		t.Fatalf(
			"Expected %#v but the func was called with %#v",
			argsPassed,
			expectedArgs,
		)
	}
}

func TestStartRunsFUTInGoroutine(t *testing.T) {
	t.Parallel()

	// Given test needs
	tester := imptest.NewRelayTester(t)
	// Given inputs
	lockchan := make(chan struct{})
	waitchan := make(chan struct{})
	donechan := make(chan struct{})
	// Given setup
	runs := 0
	// this function won't return synchronously, so if we
	// return immediately, we know it was either run in a goroutine
	// or not at all.
	wait := func() {
		// we will know this ran n times
		runs++
		// wait till lockchan is closed
		<-lockchan
		close(donechan)
	}

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

	// When closing the lockchan
	close(lockchan)
	select {
	case <-donechan:
	case <-time.After(time.Second):
		panic("donechan never closed, indicating function never completed.")
	}

	// Then we should be done with our single run
	if runs != 1 {
		t.Fatalf("Expected a single run, but got %d instead", runs)
	}
}

func TestStartFailsOnPanic(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	doneChan := make(chan struct{})
	// Given FUT that panics
	panickyFunc := func() {
		defer close(doneChan)
		panic("intentionally panicky")
	}

	mockedt.Wrap(func() {
		// When Start is called with the func
		tester.Start(panickyFunc)
		// When we wait for the func to finish
		select {
		case <-doneChan:
		case <-time.After(time.Second):
			panic("doneChan never closed, indicating function never completed.")
		}
	})

	// Then the test is marked as failed with the panic message
	if !mockedt.Failed() {
		t.Fatal("Test should've failed due to the panic, but it didn't.")
	}
}
