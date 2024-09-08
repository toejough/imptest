package imptest_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

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
	thing7 func(bool)
}

func thing1()           {}
func thing2()           {}
func thing3()           {}
func thing4() bool      { return false }
func thing5(int) string { return "" }
func thing6(string) int { return 0 }
func thing7(bool)       {}

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

func DoThingsWithBranch(deps doThingsDeps) {
	deps.thing1()

	if deps.thing4() {
		deps.thing2()
	}
}

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

func DoThingsThatPanic() {
	panic("on purpose?!")
}

// The test replaces those functions in order to test they are called.
func TestDoThingsThatPanic(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	tester := imptest.NewFuncTester(t)

	// When DoThings is started
	tester.Start(DoThingsThatPanic)
	tester.AssertPanicked("on purpose?!")
}

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

// not in the README - move to a different test file.
func DoThingsConcurrentlyNested(deps doThingsDeps) {
	go deps.thing3()
	go func() {
		z := deps.thing4()
		deps.thing7(z)

		go deps.thing1()
		go deps.thing2()
	}()
}

func TestNestedConcurrentlies(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	// convenience test wrapper
	tester := imptest.NewFuncTester(t)

	var (
		deps                    doThingsDeps
		id1, id2, id3, id4, id7 string
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.CallChan)
	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.CallChan)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.CallChan)
	deps.thing7, id7 = imptest.WrapFunc(thing7, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThingsConcurrentlyNested, deps)

	// Then the functions are called in any order
	tester.Concurrently(func() {
		tester.AssertCalled(id3).Return()
	}, func() {
		tester.AssertCalled(id4).Return(true)
		tester.AssertCalled(id7, true).Return()
		tester.Concurrently(func() {
			tester.AssertCalled(id1).Return()
		}, func() {
			tester.AssertCalled(id2).Return()
		})
	}, func() {
		tester.AssertReturned()
	})
	tester.Close()
}

// TODO: put return/panic on own channels and select between
// TODO: allow own comparison func to be set as an option on the tester
// those are all positive cases. What about negative cases? What do the error
// messages from this library look like when things go wrong?
// orphaned calls from sync calls
// orphaned calls from concurrent calls
// more calls made early in a concurrent run than the test expected
// does the testing scale beyond a handful of calls?

func TestCallAfterDonePanics(t *testing.T) {
	t.Parallel()

	tester := imptest.NewFuncTester(t)

	var (
		deps          doThingsDeps
		id1, id2, id3 string
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.CallChan)
	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.CallChan)

	orphanReleaseChan := make(chan struct{})
	testReleaseChan := make(chan struct{})

	DoThingsWithOrphanMadeAfterDone := func(deps doThingsDeps) {
		deps.thing1()
		deps.thing2()
		deps.thing3()

		go func() {
			defer func() {
				panicVal := recover()
				if panicVal == nil {
					t.Fatal("Expected a 'send on closed channel' panic, but instead got no panic") //nolint:govet
				}

				if e, ok := panicVal.(error); ok && e.Error() == "send on closed channel" {
					// the govet concerns about calling t from a non-test goroutine are mitigated by us waiting for test release here.
					testReleaseChan <- struct{}{}
					return
				}

				t.Fatalf("Expected a 'send on closed channel' panic, but instead got a panic of %#v", panicVal) //nolint:govet
			}()
			// wait for the test to think it's done
			<-orphanReleaseChan

			// now call thing3 again
			deps.thing3()
		}()
	}

	tester.Start(DoThingsWithOrphanMadeAfterDone, deps)

	tester.AssertCalled(id1).Return()
	tester.AssertCalled(id2).Return()
	tester.AssertCalled(id3).Return()
	tester.AssertReturned()

	// assert no orphans
	tester.Close()

	// let the orphan go
	orphanReleaseChan <- struct{}{}

	// wait for the test release from the orphan defer
	<-testReleaseChan
}

func TestDoThingsConcurrentlyFails(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

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
			tester.AssertCalled(id4).Return(false)
			// expect this call to trigger a failure, as we are looking for
			// true, but we just made id4 return false, and the function
			// under test doesn't do the inversion
			tester.AssertCalled(id7, true).Return()
		}, func() {
			tester.AssertReturned()
		})
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "thing7\n  with args [false]"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func DoThings3xSync(deps doThingsDeps) {
	deps.thing1()
	deps.thing1()
	deps.thing1()
}

func TestMoreSyncCallsFails(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		var (
			deps doThingsDeps
			id1  string
		)

		deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)

		// When DoThings is started
		tester.Start(DoThings3xSync, deps)

		tester.AssertCalled(id1).Return()
		tester.AssertReturned()
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "thing1\n  with args []"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func TestFewerSyncCallsFails(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		var (
			deps doThingsDeps
			id1  string
		)

		deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)

		// When DoThings is started
		tester.Start(DoThings3xSync, deps)

		tester.AssertCalled(id1).Return()
		tester.AssertCalled(id1).Return()
		tester.AssertCalled(id1).Return()
		tester.AssertCalled(id1).Return()
		tester.AssertReturned()
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "thing1\n  with args []"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func DoThings3xAsync(deps doThingsDeps) {
	go deps.thing1()
	go deps.thing1()
	go deps.thing1()
}

func TestFewerAsyncCallsTimesOut(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		var (
			deps doThingsDeps
			id1  string
		)

		deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)

		// When DoThings is started
		tester.Start(DoThings3xAsync, deps)

		tester.Concurrently(
			func() { tester.AssertCalled(id1).Return() },
			func() { tester.AssertCalled(id1).Return() },
			func() { tester.AssertCalled(id1).Return() },
			func() { tester.AssertCalled(id1).Return() },
			func() { tester.AssertReturned() },
		)
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "timed out"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func TestAssertAfterReturnFails(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		var (
			deps doThingsDeps
			id1  string
		)

		deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)

		// When DoThings is started
		tester.Start(DoThings3xAsync, deps)

		tester.Concurrently(
			func() { tester.AssertCalled(id1).Return() },
			func() { tester.AssertCalled(id1).Return() },
			func() { tester.AssertCalled(id1).Return() },
			func() { tester.AssertReturned() },
		)
		tester.Close()
		tester.AssertCalled(id1).Return()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "calls channel was already closed"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func TestDoThingsCustom(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	tester := imptest.NewFuncTester(
		t,
		imptest.WithTimeout(10*time.Second),
	)

	// Given deps replaced
	var (
		id6  string
		deps doThingsDeps
	)

	deps.thing5, _ = imptest.WrapFunc(
		thing5,
		tester.CallChan,
		imptest.WithName("thing5"),
	)
	deps.thing6, id6 = imptest.WrapFunc(thing6, tester.CallChan)

	// When DoThings is started
	tester.Start(DoThingsWithArgs, 5, deps)

	// Then the functions are called in the following order
	called := tester.Called()
	if called.ID != "thing5" {
		t.Fatalf(
			"Expected the function 'thing5' to be called, but '%s' was called instead, with args %v",
			called.ID,
			called.Args,
		)
	}

	if !reflect.DeepEqual(called.Args, []any{5}) {
		t.Fatalf("Expected args to be empty, but they were %v instead", called.Args)
	}

	called.Return("five")
	tester.AssertCalled(id6, "five").Return(6)

	// Then the function returned
	if !reflect.DeepEqual(tester.Returned(), []any{6}) {
		t.Fatalf("Expected returns to be []any{6}, but were %v instead", tester.Returned())
	}
}

func TestDoThingsThatPanicCustom(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	tester := imptest.NewFuncTester(t)

	// When DoThings is started
	tester.Start(DoThingsThatPanic)

	if !reflect.DeepEqual(tester.Panicked(), "on purpose?!") {
		t.Fatalf("Expected panic with 'on purpose?!', but panicked with %v instead", tester.Panicked())
	}
}

func TestEarlyReturnFails(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		var (
			deps doThingsDeps
			id1  string
		)

		deps.thing1, id1 = imptest.WrapFunc(thing1, tester.CallChan)

		// When DoThings is started
		tester.Start(DoThings3xSync, deps)

		tester.AssertCalled(id1).Return()
		tester.AssertCalled(id1).Return()
		tester.Returned()
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "return"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}
