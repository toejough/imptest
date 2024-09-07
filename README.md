# imptest

An IMPure function TEST tool.

There are plenty of test tools written to facilitate testing pure functions: Inputs -> Outputs. 

Impure functions, on the other hand, are characterized by calls to _other_ functions. The whole point of some functions is that they coordinate calls to other functions. 

We often don't want to validate what those other functions _do_, as we already have tests for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we can use integration tests or end-to-end testing. 

This library is here to help where we really do just want to test that the function-under-test makes the calls it's supposed to, in the right order, shuffling inputs and outputs between them correctly.

Let's start with an example that has no arguments or returns. It's... purely impure.

```go
func DoThings() {
    thing1()
    thing2()
    thing3() 
}
```

We want one test:

* DoThingsRunsExpectedFuncsInOrder
    * thing1() is called
    * thing2() is called
    * thing3() is called
    * function returns

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
	thing4 func() bool
	thing5 func(int) string
	thing6 func(string) int
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

	// Given convenience test wrapper
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

========>MADE IT HERE WITH README UPDATES<========
Functions don't only call other functions - they can also panic and kick off other goroutines to run things in parallel. Let's examine each scenario.

Panics are actually fairly easy to capture.

```go
func DoThings() {
    pkgDeps.thing1()
    if pkgDeps.thing2() {
        panic("on purpose?!") 
    }
}

// The test replaces those functions in order to test they are called
func TestDoThingsIsFineIfThing2ReturnsFalse(t *testing.T) {
    // Given deps replaced
    tester := imptest.NewTester(t)
    // WrapFunc returns a function of the same signature, but which:
    // * puts the given function on the relay for test validation
    // * waits for the test to tell it to return before returning
    // It also returns an ID, to compare against, because go does not allow    
    // us to compare functions.
    pkgDeps.thing1, id1 = tester.WrapFunc(thing1)
    pkgDeps.thing2, id2 = tester.WrapFunc(thing2)
    pkgDeps.thing3, id3 = tester.WrapFunc(thing3)

    // When DoThings is started
    tester.Start(func(){DoThings()})

    // Then the functions are called in the following order
    tester.AssertCalled(id1).Return()
    tester.AssertCalled(id2).Return(false)

    // Then the function is done
    tester.AssertReturned()
}

func TestDoThingsPanicsIfThings2ReturnsTrue(t *testing.T) {
    // Given deps replaced
    tester := imptest.NewTester(t)
    // WrapFunc returns a function of the same signature, but which:
    // * puts the given function on the relay for test validation
    // * waits for the test to tell it to return before returning
    // It also returns an ID, to compare against, because go does not allow    
    // us to compare functions.
    pkgDeps.thing1, id1 = tester.WrapFunc(thing1)
    pkgDeps.thing2, id2 = tester.WrapFunc(thing2)
    pkgDeps.thing3, id3 = tester.WrapFunc(thing3)

    // When DoThings is started
    tester.Start(func(){DoThings()})

    // Then the functions are called in the following order
    tester.AssertCalled(id1).Return()
    tester.AssertCalled(id2).Return(true)
    tester.AssertPanicked("on purpose?!")

    // Then the function is done
    tester.AssertReturned()
}
```

The `AssertPanicked` function has a partner `Panicked` which returns the content of the panic.

Concurrent calls fired off by goroutines are more difficult to capture, but not by much.

```go
func DoThings(x, y int) {
    go pkgDeps.thing1(x)
    go func() {
        z := pkgDeps.thing2(y)
        pkgDeps.thing3(z)
    }()
}

func TestDoThingsConcurrently(t *testing.T) {
    // Given deps replaced
    tester := imptest.NewTester(t)
    pkgDeps.thing1, id1 = tester.WrapFunc(thing1)
    pkgDeps.thing2, id2 = tester.WrapFunc(thing2)
    pkgDeps.thing3, id3 = tester.WrapFunc(thing3)

    // When DoThings is started
    tester.Start(func(){DoThings(1, 2)})

    // Then the functions are called in any order
    // SetGoroutines tells the tester that it should pull up to the number given calls off of the 
    // relay. Each call to AssertCalled will pull up to that many calls off of the relay before asserting
    // that one of them matches. 
    tester.SetGoroutines(2) 
    tester.AssertCalled(id1, 1).Return() // this could be second or third, and would be fine
    tester.AssertCalled(id2, 2).Return(3) // this could be first, and would be fine
    tester.AssertCalled(id3, 3).Return() // this must be called after the wait for id2

    // Then the function is done
    tester.AssertReturned()
}
```

Some other niceties:

* `SetGoroutines(0)` will allow an arbitrary number of goroutines, if you don't care to track their count.
* `Timeout(duration)` will set the timeout for all future assertions/next calls. The default is 1s.

# alternatives/inspirations
Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?

In the straightforward use cases, you only get to specify simple call/return behavior, with no guarantees about ordering, and you need to unset handlers for repeated calls for the same function.

On the other hand, there's https://github.com/stretchr/testify/issues/741, which calls some of this out, and which is answered by the author with some additional syntax and functionality. I still found myself wondering if something with what I considered simpler syntax was possible, mostly out of curiosity, and to learn.
