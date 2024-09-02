package imptest_test

import (
	"strings"
	"testing"

	"github.com/toejough/protest/imptest"
)

// TODO: context deps?
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
	// Given pkg deps replaced
	calls := make(chan imptest.FuncCall)

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

	// Given pkg deps replaced
	var (
		id1, id2, id3 string
		deps          doThingsDeps
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.Calls)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.Calls)
	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.Calls)

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

	// Given pkg deps replaced
	var (
		id1, id4 string
		deps     doThingsDeps
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.Calls)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.Calls)

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

	// Given pkg deps replaced
	var (
		id1, id2, id4 string
		deps          doThingsDeps
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.Calls)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.Calls)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.Calls)

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

	// Given pkg deps replaced
	var (
		id5, id6 string
		deps     doThingsDeps
	)

	deps.thing5, id5 = imptest.WrapFunc(thing5, tester.Calls)
	deps.thing6, id6 = imptest.WrapFunc(thing6, tester.Calls)

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

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.Calls)

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
	// Given pkg deps replaced
	t.Parallel()

	// convenience test wrapper
	tester := imptest.NewFuncTester(t)

	var (
		deps          doThingsDeps
		id3, id4, id7 string
	)

	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.Calls)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.Calls)
	deps.thing7, id7 = imptest.WrapFunc(thing7, tester.Calls)

	// When DoThings is started
	tester.Start(DoThingsConcurrently, deps)

	// Then the functions are called in any order
	// TODO randomize order asserted
	// TODO add a way to verify that nothing _unexpected_ landed in the call
	// queue after we finish our tests. tester.AssertNoOrphanedCalls()?
	tester.Concurrently(func() {
		// we expect this call _concurrently_, acking that this is not
		// necessarily the order the calls will actually come through because
		// of this, within a call to "Concurrently", calls are pulled until
		// there's a match
		tester.AssertCalled(id3).Return()
	}, func() {
		// yet unmatched calls will be walked through in order for this set. If
		// id7 is called before id4, it will be a test failure because by
		// the time we match id4, id7 will be in the no-match queue we
		// already walked through, and won't be walked again within this
		// call to "Concurrently"
		tester.AssertCalled(id4).Return(true)
		tester.AssertCalled(id7, true).Return()
	}, func() {
		tester.AssertReturned()
	})
	tester.AssertNoOrphans()
}

func DoThingsConcurrentlyNested(deps doThingsDeps) {
	go deps.thing3()
	go func() {
		z := deps.thing4()
		deps.thing7(z)

		go deps.thing1()
		go deps.thing2()
	}()
}

// TODO: test a more complex nested case.
func TestNestedConcurrentlies(t *testing.T) {
	// Given pkg deps replaced
	t.Parallel()

	// convenience test wrapper
	tester := imptest.NewFuncTester(t)

	var (
		deps                    doThingsDeps
		id1, id2, id3, id4, id7 string
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.Calls)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.Calls)
	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.Calls)
	deps.thing4, id4 = imptest.WrapFunc(thing4, tester.Calls)
	deps.thing7, id7 = imptest.WrapFunc(thing7, tester.Calls)

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
	tester.AssertNoOrphans()
}

// those are all positive cases. What about negative cases? What do the error
// messages from this library look like when things go wrong?
// orphaned calls from sync calls
// orphaned calls from concurrent calls
// more calls made early in a concurrent run than the test expected
// does the testing scale beyond a handful of calls?

func TestOrphanMadeAfterDonePanics(t *testing.T) {
	t.Parallel()

	tester := imptest.NewFuncTester(t)

	var (
		deps          doThingsDeps
		id1, id2, id3 string
	)

	deps.thing1, id1 = imptest.WrapFunc(thing1, tester.Calls)
	deps.thing2, id2 = imptest.WrapFunc(thing2, tester.Calls)
	deps.thing3, id3 = imptest.WrapFunc(thing3, tester.Calls)

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
	tester.AssertNoOrphans()

	// let the orphan go
	orphanReleaseChan <- struct{}{}

	// wait for the test release from the orphan defer
	<-testReleaseChan
}

// This test fails the race detector, due to the second thing1 not being read from
// the call queue before AssertNoOrphans is called.
//
//	func TestOrphanMadeBeforeDone(t *testing.T) {
//		t.Parallel()
//
//		tester := imptest.NewFuncTester(t)
//
//		var (
//			deps          doThingsDeps
//			id1, id2, id3 string
//		)
//
//		deps.thing1, id1 = imptest.WrapFunc(thing1, tester.Calls)
//		deps.thing2, id2 = imptest.WrapFunc(thing2, tester.Calls)
//		deps.thing3, id3 = imptest.WrapFunc(thing3, tester.Calls)
//
//		DoThingsWithOrphanMadeAfterDone := func(deps doThingsDeps) {
//			go deps.thing1()
//			go deps.thing1()
//			go deps.thing2()
//			go deps.thing3()
//		}
//
//		tester.Start(DoThingsWithOrphanMadeAfterDone, deps)
//
//		tester.Concurrently(
//			func() { tester.AssertCalled(id1).Return() },
//			func() { tester.AssertCalled(id2).Return() },
//			func() { tester.AssertCalled(id3).Return() },
//			func() { tester.AssertReturned() },
//		)
//
//		// assert no orphans
//		tester.AssertNoOrphans()

func TestDoThingsConcurrentlyFails(t *testing.T) {
	// Given pkg deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		var (
			deps          doThingsDeps
			id3, id4, id7 string
		)

		deps.thing3, id3 = imptest.WrapFunc(thing3, tester.Calls)
		deps.thing4, id4 = imptest.WrapFunc(thing4, tester.Calls)
		deps.thing7, id7 = imptest.WrapFunc(thing7, tester.Calls)

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
		tester.AssertNoOrphans()
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

// }
