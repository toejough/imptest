# imptest

An IMPure function TEST tool.

I find that most test suites are written assuming pure functions: Inputs -> Outputs. 

Impure functions, on the other hand, are characterized by calls to _other_ functions. The whole point of some functions is that they coordinate calls to other functions. We often don't want to validate what those other functions _do_, as we already have tests for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we can use integration tests or end-to-end testing. This library is here to help where we really do just want to test that the function-under-test makes the calls it's supposed to, in the right order, shuffling inputs and outputs between them correctly.

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

You can choose any dependency injection method you like. For simplicity, the following examples will assume a package level struct that stores the functions which we want to test for.

```go
// DoThings now calls functions from a package-level dependencies struct
func DoThings() {
    pkgDeps.thing1()
    pkgDeps.thing2()
    pkgDeps.thing3()
}

// The package dependencies are initialized with the original functions
var pkgDeps = struct{
    thing1 func()
    thing2 func()
    thing3 func()
}{thing1, thing2, thing3}

// The test replaces those functions in order to test they are called
func TestDoThingsRunsExpectedFuncsInOrder(t *testing.T) {
    // Given pkg deps replaced
    relay := imptest.NewRelay()
    // WrapFunc returns a function of the same signature, but which:
    // * puts the given function on the relay for test validation
    // * waits for the test to tell it to return before returning
    // It also returns an ID, to compare against, because go does not allow    
    // us to compare functions.
    pkgDeps.thing1, id1 = relay.WrapFunc(thing1)
    pkgDeps.thing2, id2 = relay.WrapFunc(thing2)
    pkgDeps.thing3, id3 = relay.WrapFunc(thing3)

    // When DoThings is started
    done := false
    go func() {
        DoThings()
        // record when the func is done so we can test that, too
        done = true
    }()

    // Then thing1 is called 
    if relay.NextCallID() != id1 {
        t.Fail()
    }

    // When thing1 returns 
    relay.Return()

    // Then thing2 is called
    if relay.NextCallID() != id2 {
        t.Fail()
    }

    // When thing2 returns 
    relay.Return() 

    // Then thing3 is called
    if relay.NextCallID() != id3 {
        t.Fail()
    }

    // When thing3 returns
    relay.Return()

    // Then there are no more calls
    if relay.NextCallID() != nil {
        t.Fail()
    }

    // And the function returns
    if !done {
        t.Fail()
    }
}
```

A slightly less verbose version of the test is generally helpful for parsing and understanding, so Imptest provides some syntactic sugar.

TODO: also wrap the function under test into the tester.

```go
// The test replaces those functions in order to test they are called
func TestDoThingsRunsExpectedFuncsInOrder(t *testing.T) {
    // Given pkg deps replaced
    tester := imptest.NewTester(t, DoThings)
    // WrapFunc returns a function of the same signature, but which:
    // * puts the given function on the relay for test validation
    // * waits for the test to tell it to return before returning
    // It also returns an ID, to compare against, because go does not allow    
    // us to compare functions.
    pkgDeps.thing1, id1 = tester.WrapFunc(thing1)
    pkgDeps.thing2, id2 = tester.WrapFunc(thing2)
    pkgDeps.thing3, id3 = tester.WrapFunc(thing3)

    // When DoThings is started
    tester.Start()

    // Then the functions are called in the following order
    tester.AssertCalled(id1).Return()
    tester.AssertCalled(id2).Return()
    tester.AssertCalled(id3).Return()

    // Then the function returned
    tester.AssertReturned()
}
```

The `AssertReturned()` function has a partner `Returned()` func that simply returns the returned values in an array for you to inspect.

Let's explore a more complex example, where we have a more interesting function, which acts on returns from the subfunctions.

```go
func DoThings() {
    pkgDeps.thing1()
    if pkgDeps.thing2() {
        pkgDeps.thing3() 
    }
}
```

Now we would like two tests:

* when `thing2` returns `false`, we should only expect to call `thing1` and `thing2` before returning.
* when `thing2` returns `true`, we should expect to call all three functions before returning.

```go
// The test replaces those functions in order to test they are called
func TestDoThingsAvoidsThings3IfThings2ReturnsFalse(t *testing.T) {
    // Given pkg deps replaced
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
    tester.Start(DoThings)

    // Then the functions are called in the following order
    tester.AssertCalled(id1).Return()
    tester.AssertCalled(id2).Return(false)

    // Then the function is done
    tester.AssertReturned()
}

func TestDoThingsCallsThings3IfThings2ReturnsTrue(t *testing.T) {
    // Given pkg deps replaced
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
    tester.Start(DoThings)

    // Then the functions are called in the following order
    tester.AssertCalled(id1).Return()
    tester.AssertCalled(id2).Return(true)
    tester.AssertCalled(id3).Return()

    // Then the function is done
    tester.AssertReturned()
}
```

Adding arguments and more returns is fairly trivial.

```go
func DoThings(int x) int {
    y := pkgDeps.thing1(x)
    return pkgDeps.thing2(y) {
}

func TestDoThingsRunsExpectedFuncsInOrder(t *testing.T) {
    // Given pkg deps replaced
    tester := imptest.NewTester(t)
    // WrapFunc returns a function of the same signature, but which:
    // * puts the given function on the relay for test validation
    // * waits for the test to tell it to return before returning
    // It also returns an ID, to compare against, because go does not allow    
    // us to compare functions.
    pkgDeps.thing1, id1 = tester.WrapFunc(thing1)
    pkgDeps.thing2, id2 = tester.WrapFunc(thing2)

    // When DoThings is started
    tester.Start(func(){DoThings(1)})

    // Then the functions are called in the following order
    tester.AssertCalled(id1, 1).Return(2)
    tester.AssertCalled(id2, 2).Return(3)

    // Then the function returned as expected
    tester.AssertReturned(3)
}
```

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
    // Given pkg deps replaced
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
    // Given pkg deps replaced
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
    // Given pkg deps replaced
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

* `Return()` is unnecessary. If another tester/relay call is made, it will automatically apply `Return()` if no `Return(...interface{})` call was already made.
* `SetGoroutines(0)` will allow an arbitrary number of goroutines, if you don't care to track their count.
* `Timeout(duration)` will set the timeout for all future assertions/next calls. The default is 1s.

Using this library to validate your dependencies are called correctly, and their results are used correctly, requires some kind of dependency injection to be used. ImpTest works by mocking, mostly, but instead of supplying a single mocked version of the function, we supply a version of the function that pipes the call through a relay, so that mid-call it can be inspected by the test. The test then injects an appropriate return value, which the relay then picks up from within the mocked function, and returns.


The general flow of events for building an ImpTest:

* decide which calls you want to test for (you want to test that these calls are made, without _actually_ calling them in the test)
* inject them into your function via dependency injection, rather than as direct calls 
* 
Example of use:
```go
func TestRepeatedCalls(t *testing.T) {
	t.Parallel()

	// Given test needs
	relay := NewCallRelay()
	tester := &RelayTester{T: t, Relay: relay} //nolint: exhaustruct
	// nobody else would be able to fill in private fields
	// Given inputs
	superSum := func(deps superSumDeps, a, b int) int {
		return deps.sum(a, a) +
			deps.sum(b, b) +
			deps.sum(a, b) +
			deps.sum(b, a)
	}
	deps := &superSumTestDeps{relay: relay}

	// When the func is run
	tester.Start(superSum, deps, 2, 3)

	// Then the internal sum is called 4x with different args
	tester.AssertNextCallIs(deps.sum, 2, 2).InjectReturns(4)
	tester.AssertNextCallIs(deps.sum, 3, 3).InjectReturns(6)
	tester.AssertNextCallIs(deps.sum, 2, 3).InjectReturns(5)
	tester.AssertNextCallIs(deps.sum, 3, 2).InjectReturns(5)

	// Then the relay is shut down
	tester.AssertDoneWithin(time.Second)

	// Then the result is as expected
	tester.AssertReturned(20)
}

type superSumDeps interface {
	sum(a, b int) int
}

type superSumTestDeps struct {
	relay *CallRelay
}

func (d *superSumTestDeps) sum(a, b int) int {
	var result int

	d.relay.PutCall(d.sum, a, b).FillReturns(&result)

	return result
}
```
# alternatives/inspirations
Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?

In the straightforward use cases, you only get to specify simple call/return behavior, with no guarantees about ordering, and you need to unset handlers for repeated calls for the same function.

On the other hand, there's https://github.com/stretchr/testify/issues/741, which calls some of this out, and which is answered by the author with some additional syntax and functionality. I still found this somewhat confusing, overly verbose, and non-obvious that the functionality even existed, so I set out to see if I could do any better, mostly out of curiosity, and to learn. Having done so, I'm happy with what I came up with.
