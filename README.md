# imptest

An IMPure function TEST tool.

There are plenty of test tools written to facilitate testing pure functions: Inputs -> Outputs. 

Impure functions, on the other hand, are characterized by calls to _other_ functions. The whole point of some functions is that they coordinate calls to other functions. 

We often don't want to validate what those other functions _do_, as we already have tests for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we can use integration tests or end-to-end testing. 

This library is here to help where we really do just want to test that the function-under-test makes the calls it's supposed to, in the right order, shuffling inputs and outputs between them correctly.

Let's start with an example that has no arguments or returns. It's... purely impure. Imagine each of these functions performs some expensive, non-idempotent, difficult-to-verify 3rd party action. Mostly you trust that when called, these do what they're supposed to, and you want an easy way to test that _your_ code makes the correct calls.

```go
func DoThings() {
    thing1()
    thing2()
    thing3() 
}
```

We want one test:

* `DoThingsRunsExpectedFuncsInOrder`
    * `thing1()` is called
    * `thing2()` is called
    * `thing3()` is called
    * your function returns

We literally don't care about the underlying functionality, we just want to ensure proper ordering and conditions at the DoThings level of abstraction. How do we test that? 

Somehow we need to get notified, at runtime, whether `thing1` is called. The imptest way to do this is to mock `thing1`, replacing it with a call that instead puts `thing1` on a channel for the test to inspect before moving on.

You can choose any dependency injection method you like. For simplicity, the following examples will assume a struct that stores the functions which we want to test for.

```go
package imptest_test

import (
	"testing"

	"github.com/toejough/protest/imptest"
)

// DoThings now calls functions from a dependencies struct.
func DoThings(deps doThingsDeps) {
	deps.thing1()
	deps.thing2()
	deps.thing3()
}

type doThingsDeps struct {
	thing1 func()
	thing2 func()
	thing3 func()
}

// The test replaces those functions in order to test they are called.
func TestDoThingsRunsExpectedFuncsInOrder(t *testing.T) {
	t.Parallel()
	// Given a call channel to track the calls
	calls := make(chan imptest.FuncCall)

	// Given the dependencies are replaced by functions which place their calls on the channel

	// WrapFunc returns a function of the same signature, but which:
	// * puts the given function on the calls channel for test validation
	// * waits for the test to tell it to return before returning
	// It also returns an ID, to compare against, because go does not allow us
	// to compare functions.
	var (
		id1, id2, id3 string
		deps          doThingsDeps
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, calls)
	deps.thing2, id2 = imptest.WrapFunc(thing2, calls)
	deps.thing3, id3 = imptest.WrapFunc(thing3, calls)

	// When DoThings is started
	go func() {
		// record when the func is done so we can test that, too
		defer close(calls)
		DoThings(deps)
	}()

	// Then thing1 is called
	funcCall1 := <-calls
	if funcCall1.ID != id1 {
		t.Fail()
	}

	// When thing1 returns
	funcCall1.ReturnValuesChan <- []any{} // no returns

	// Then thing2 is called
	funcCall2 := <-calls
	if funcCall2.ID != id2 {
		t.Fail()
	}

	// When thing2 returns
	funcCall2.ReturnValuesChan <- nil // for no returns, can also inject nil

	// Then thing3 is called
	funcCall3 := <-calls
	if funcCall3.ID != id3 {
		t.Fail()
	}

	// When thing3 returns
	funcCall3.ReturnValuesChan <- nil

	// Then there are no more calls
	_, open := <-calls
	if open {
		t.Fail()
	}
}
```

A slightly less verbose version of the test is generally helpful for parsing and understanding, so Imptest provides some syntactic sugar.

```go
// The test replaces those functions in order to test they are called.
func TestDoThingsRunsExpectedFuncsInOrderSimply(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper that includes the call channel
	tester := imptest.NewFuncTester(t)

	// Given deps replaced
	var (
		id1, id2, id3 string
		deps          doThingsDeps
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.CallChan)
	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThings, deps)

	// Then the functions are called in the following order
	tester.AssertCalled(id1).Return()
	tester.AssertCalled(id2).Return()
	tester.AssertCalled(id3).Return()

	// Then the function returned
	tester.AssertReturned()
}
```

Let's explore a more complex example, where we have a more interesting function, which acts on returns from the subfunctions.

```go
func DoThingsWithBranch(deps doThingsDeps) {
	deps.thing1()

	if deps.thing4() {
		deps.thing2()
	}
}
```

Now we would like two tests:

* when `thing2` returns `false`, we should only expect to call `thing1` and `thing2` before returning.
* when `thing2` returns `true`, we should expect to call all three functions before returning.

```go
type doThingsDeps struct {
	thing1 func()
	thing2 func()
	thing3 func()
	thing4 func() bool
}

// The test replaces those functions in order to test they are called
func TestDoThingsAvoidsThings3IfThings2ReturnsFalse(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	tester := imptest.NewFuncTester(t)

	// Given deps replaced
	var (
		id1, id4 string
		deps     doThingsDeps
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThingsWithBranch, deps)

	// Then the functions are called in the following order
	tester.AssertCalled(id1).Return()
	tester.AssertCalled(id4).Return(false)

	// Then the function is done
	tester.AssertReturned()
}

func TestDoThingsCallsThings3IfThings2ReturnsTrue(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	tester := imptest.NewFuncTester(t)

	// Given deps replaced
	var (
		id1, id2, id4 string
		deps          doThingsDeps
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.CallChan)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThingsWithBranch, deps)

	// Then the functions are called in the following order
	tester.AssertCalled(id1).Return()
	tester.AssertCalled(id4).Return(true)
	tester.AssertCalled(id2).Return()

	// Then the function is done
	tester.AssertReturned()
}
```

Adding arguments and more returns is fairly trivial.

```go
type doThingsDeps struct {
	thing1 func()
	thing2 func()
	thing3 func()
	thing4 func() bool
	thing5 func(int) string
	thing6 func(string) int
}

func DoThingsWithArgs(x int, deps doThingsDeps) int {
	y := deps.thing5(x)
	return deps.thing6(y)
}

func TestDoThingsRunsExpectedFuncsWithArgs(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	tester := imptest.NewFuncTester(t)

	// Given deps replaced
	var (
		id5, id6 string
		deps     doThingsDeps
	)

	deps.thing5, id5 = imptest.WrapFunc(thing5, tester.CallChan)
	deps.thing6, id6 = imptest.WrapFunc(thing6, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThingsWithArgs, 1, deps)

	// Then the functions are called in the following order
	tester.AssertCalled(id5, 1).Return("hi")
	tester.AssertCalled(id6, "hi").Return(2)

	// Then the function returned as expected
	tester.AssertReturned(2)
}
```

Functions don't only call other functions - they can also panic and kick off other goroutines to run things in parallel. Let's examine each scenario.

Panics are actually fairly easy to capture.

```go
func DoThingsThatPanic() {
	panic("on purpose?!")
}

func TestDoThingsThatPanic(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	tester := imptest.NewFuncTester(t)

	// When DoThings is started
	tester.Start(DoThingsThatPanic)
	tester.AssertPanicked("on purpose?!")
}
```

Imptest can also handle _injecting_ panics.

``` go
func DoThingsWithPanic(deps doThingsDeps) (panicVal string) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool

			panicVal, ok = r.(string)
			if !ok {
				panic(r)
			}
		}
	}()

	deps.thing1()

	return
}

// The test replaces those functions in order to test they are called.
func TestDoThingsWithPanic(t *testing.T) {
	t.Parallel()

	// convenience test wrapper
	tester := imptest.NewFuncTester(t)

	var (
		deps doThingsDeps
		id1  string
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThingsWithPanic, deps)

	// Then id7 is called. When it panics...
	tester.AssertCalled(id1).Panic("omg what?")

	// Then the function returns the panic message
	tester.AssertReturned("omg what?")
}
```

Concurrent calls fired off by goroutines are more difficult to capture, but imptest can handle this, too.

```go
type doThingsDeps struct {
	thing1 func()
	thing2 func()
	thing3 func()
	thing4 func() bool
	thing5 func(int) string
	thing6 func(string) int
	thing7 func(bool) 
}

func DoThingsConcurrently(deps doThingsDeps) {
	go deps.thing3()
	go func() {
		z := deps.thing4()
		deps.thing7(z)
	}()
}

func TestDoThingsConcurrently(t *testing.T) {
	t.Parallel()

	// Given deps replaced
	tester := imptest.NewFuncTester(t)

	var (
		deps          doThingsDeps
		id3, id4, id7 string
	)

	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.CallChan)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.CallChan)
	deps.thing7, id7 = imptest.WrapFunc(thing7, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThingsConcurrently, deps)

	// Then the functions are called in any order
	tester.Concurrently(func() {
		tester.AssertCalled(id3).Return()
	}, func() {
		tester.AssertCalled(id4).Return(true)
		tester.AssertCalled(id7, true).Return()
	}, func() {
		tester.AssertReturned()
	})
	tester.Close()
}
```

## Customization

The test helper makes some assumptions, each of which you can override:
* timeouts for waiting for a call are defaulted to 500ms. You can override with `NewFuncTester(t, imptest.WithTimeout(duration))`.
* names for mocked functions are defaulted to `runtime.FuncForPC(...).Name()`. You can override with `WrapFunc(f, callsChan, WithName(name))`.
* by default, the test helper performs comparisons with `runtime.DeepEqual`. You can override with `NewFuncTester(t, imptest.WithComparator(comparisonFunc))`.

For even more custom inspection, comparison, and diffing of calls, returns, and panics, you can use `Called`, `Returned`, or `Panicked` in place of their `Assert[Called|Returned|Panicked]` functions. Doing so will get you the underlying FuncCall, return value list, or panic value, respectively.

## alternatives/inspirations
Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?

https://github.com/stretchr/testify/issues/741, highlights some challenges, and is answered by the author with some additional syntax and functionality. I still found myself wondering if something with what I considered simpler syntax was possible, mostly out of curiosity, and to learn.
