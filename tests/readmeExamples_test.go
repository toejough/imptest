package imptest_test

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
