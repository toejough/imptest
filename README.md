# imptest

An IMPure function TEST tool.

There are plenty of test tools written to facilitate testing pure functions: Inputs -> Outputs. 

Impure functions, on the other hand, are characterized by calls to _other_ functions. The whole point of some (_most_?) functions is that they coordinate calls to other functions. 

We often don't want to validate what those other functions _do_, as we already have tests for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we can use integration tests or end-to-end testing. 

This library is here to help where we really do just want to test that the function-under-test makes the calls it's supposed to, in the right order, shuffling inputs and outputs between them correctly.

Let's look at the tests to see how this really works. 

```go
import (
	"strings"
	"testing"

	"github.com/toejough/imptest/imptest"
)

// ==General Intent for Imptest & its API levels==

// Things Imptest does:
// track activity from function under test
// match activity to expectations
// respond to activity
// support concurrency

// Level 1:
// track activity from function under test: wrap dependency funcs to track their calls, manually track return/panic
// match activity to expectations: receive activity from chan, manually check type, args
// respond to activity: manually send response type & args to chan
// support concurrency: manually send activity back to chan if not the one we wanted

// Level 2:
// track activity from function under test: wrap dep structs of funcs to track their calls, auto track return/panic
// match activity to expectations: receive activity & check type, args via simple sugar funcs
// respond to activity: send response type & args via simple sugar funcs
// support concurrency: auto track and compare expectations to activity

// Level 3 (planned, but not implemented yet):
// track activity from function under test: generate dep structs of funcs to track their calls
// match activity to expectations: complex matchers?
// respond to activity: ???
// support concurrency: ???

// Level 4 (under consideration):
// provide deps that can be called to set expectations, like calling code...
// * calls like `call.foo(a, b, c).sendReturn()`
// * returns/panics from an example func: `imp.Run(func(){panic('yikes')})` to indicate the expected panic

// ==Test consts & structs==

const (
	anyString = "literally anything"
	anyInt    = 42 // literally anything
)

type depStruct1 struct {
	// the data fields are intentionally unused - they're there to help validate that they'll be properly skipped by the
	// imp when it tries to mimic and replace the functions
	privateDataField string //nolint:unused
	PublicDataField  string
	Dep1             func() string
	Dep2             func(x int, y string) bool
}

type depStruct2 struct {
	D1 func()
	D2 func()
	D3 func()
	D4 func()
	D5 func()
}

// ==Tests==

// TestL2ReceiveCallSendReturn tests matching a dependency call and pushing a return more simply, with a
// dependency struct.
func TestL2ReceiveCallSendReturn(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct1) string {
		return deps.Dep1()
	}
	// and a struct of dependencies to mimic
	depsToMimic := depStruct1{} //nolint:exhaustruct
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	// and a string to return from the dependency call
	returnString := anyString

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)
	// Then the next thing the function under test does is make a call matching our expectations
	call := imp.ExpectCall("Dep1")
	// When we push a return string
	call.SendReturn(returnString)
	// Then the next thing the function under test does is return values matching our expectations
	imp.ExpectReturn(returnString)
}

// TestL2ReceiveCallSendPanic tests matching a dependency call and pushing a panic more simply, with a
// dependency struct.
func TestL2ReceiveCallSendPanic(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct1) string {
		return deps.Dep1()
	}
	// and a struct of dependencies to mimic
	depsToMimic := depStruct1{}
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	// and a string to panic from the dependency call
	panicString := anyString

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)

	// Then the next thing the function under test does is make a call matching our expectations
	// (and then When we push a panic value...)
	imp.ExpectCall("Dep1").SendPanic(panicString)

	// Then the next thing the function under test does is panic with a value matching our expectations
	imp.ExpectPanic(panicString)
}

// TestL2ArgPassing tests passing args around inside a function.
func TestL2ArgPassing(t *testing.T) {
	t.Parallel()

	// Given
	funcToTest := func(y string, x int, deps depStruct1) (bool, int) {
		return !deps.Dep2(x, y), x * len(y)
	}
	depsToMimic := depStruct1{} //nolint:exhaustruct
	imp := imptest.NewImp(t, &depsToMimic)
	dep2Result := true
	returnBool, returnInt := false, anyInt*len(anyString)

	// When
	imp.Start(funcToTest, anyString, anyInt, depsToMimic)

	// Then
	imp.ExpectCall("Dep2", anyInt, anyString).SendReturn(dep2Result)
	imp.ExpectReturn(returnBool, returnInt)
}

// TestL2Pure demonstrates testing a pure function.
func TestL2Pure(t *testing.T) {
	t.Parallel()

	// Given
	funcToTest := func(x int, y string) int { return x * len(y) }
	imp := imptest.NewImp(t)
	returnInt := anyInt * len(anyString)

	// When
	imp.Start(funcToTest, anyInt, anyString)

	// Then
	imp.ExpectReturn(returnInt)
}

// TestL2Concurrency tests that we can even use the L2 API to test concurrent calls to dependency functions.
func TestL2Concurrency(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(d depStruct2) {
		go d.D1()
		go d.D2()
		go d.D3()
		go d.D4()
		go d.D5()
	}
	// and a dependency struct with functions to replace with mimics
	deps := depStruct2{}
	// and a helpful imp...
	imp := imptest.NewImp(t, &deps)

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, deps)

	// When we set expectations for the function calls
	imp.Concurrently(
		func() { imp.ExpectCall("D5").SendReturn() },
		func() { imp.ExpectCall("D2").SendReturn() },
		func() { imp.ExpectCall("D4").SendReturn() },
		func() { imp.ExpectReturn() },
		func() { imp.ExpectCall("D1").SendReturn() },
		func() { imp.ExpectCall("D3").SendReturn() },
	)
}

// TestL2L1Mix demonstrates mixing L2 for convenience and L1 for finer control.
func TestL2L1Mix(t *testing.T) {
	t.Parallel()

	// ==L2 stuff, which is easier to read but gives you less control==
	// Given a function to test
	funcToTest := func(deps depStruct1) string {
		return deps.Dep1()
	}
	// and a struct of dependencies to mimic
	depsToMimic := depStruct1{}
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	// and a string to return from the dependency call
	returnString := anyString

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)
	// Then the next thing the function under test does is make a call matching our expectations
	call := imp.ExpectCall("Dep1")
	// When we push a return string
	call.SendReturn(returnString)

	// ==L1 stuff, for when you need more control over matching and failure messaging than L2 allows==
	// Then the next thing the function under test does is return values matching our expectations
	functionReturned := <-imp.ActivityChan
	if functionReturned.Type != imptest.ActivityTypeReturn {
		t.Fatal("expected a return action but got something else")
	}

	returns := functionReturned.ReturnVals
	if len(returns) != 1 {
		t.Fatalf("Expected only one return")
	}

	retString := returns[0].(string) //nolint:forcetypeassert
	if !strings.HasPrefix(retString, returnString) {
		t.Fatalf(
			"expected the return string to have a prefix of the sent return from the dependency call (%s),"+
				"but it didn't. Instead it was just '%s'.",
			returnString, retString,
		)
	}
}
```
## What's going on here?

The basic idea is to find a way to treat impure function call activity as pure input/output data, thereby allowing us to write fast, repeatable tests for it, just like we do for pure functions.

This library treats everything after the invocation of the function under test as "activity", which it encapsulates in a `FunctionActivity` object and pushes onto a channel. Calls to dependency functions are intercepted this way, as is the final return/panic from the function under test. The test code, then, can pull activity from the channel, inspect it, and react to it. In the case of dependency calls, there is a response channel provided in the `FunctionActivity`, which allows the test code to send a response (either an instruction to return or to panic with some values).

It's this call & response via channels that is one of the key distinguishing features of this library vs other mock libraries - each test instruction is interacting synchronously with the function under test. If you want to make sure calls happen in a certain order, you check for them in that order. If you want to know what happens if a dependency returns true 30 times and then false the 31st time, you can control that in the test code itself, rather than having to write yet another mock implementation.

Oops, buried the lede there - that's one of the best things about this library: at _most_ you have to write a single mock for the dependencies, and you don't even have to implement any logic - all Imptest needs is the function signature! 

Thanks to the magic of `reflect.MakeFunc`, all we need is a function stub, and then any test can use that stub as the baseline to mimic, and from there each test can push arbitrary responses from that function on the fly! Instead of implementing `returnsTrueMock`, `returnsFalseMock`, `returnsTrue5TimesThenFalseMock`, etc, you pass in `struct{BoolFunc func() bool}` and then in your test call `imp.ExpectCall("BoolFunc").SendReturn(true)`, etc!

## API's 

There are two loose groupings of API's: L1 & L2. 

L1 is about the very minimum level of implementation abstraction. They
also give you the most direct control and access to the underlying data from the function under test and its dependency
calls. 

L2 is about more syntactic sugar and readability for the most common use cases. You can see examples in the test
snippet above, as well as plans for future usability improvements.

Both sets of API's are reasonably complete - you can do everything in the following list with either:

* mimic dependency functions 
* start your function under test with arbitrary args 
* wait for a dependency call 
* validate the call & args 
* send responses for the depenency call to perform (either returns or panics) 
* wait for the function under test to return or panic 
* set expectations concurrently 

The `Concurrently` function is just a convenience wrapper around waitgroups - it runs each passed in function in a
goroutine and waits to return till all of them have.

Generally speaking, the L2 API's do what you'd expect - they match calls/returns/panics, and error with decent messages
and diffs when they find differences. Comparisons are largely `reflect.DeepEqual` based, but there are some special
cases carved out for things like comparing typed and untyped `nil`. Error message diffs are based on string comparisons
of `spew` output. If you want more control over either the comparisons or the error messaging, the recommandation is to
fall back to the L1 API's, as in the final example in the tests above.

## alternatives/inspirations
Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?

https://github.com/stretchr/testify/issues/741, highlights some challenges, and is answered by the author with some additional syntax and functionality. I still found myself wondering if something with what I considered simpler syntax was possible, mostly out of curiosity, and to learn. 

A couple (gulp) _years_ later, I'm pretty happy with what I've come up with. I hope you will be, too!
