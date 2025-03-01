package imptest_test

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/toejough/imptest/imptest"
)

// DoThings now calls functions from a dependencies struct.
func DoThings(deps doThingsDeps) {
	deps.Thing1()
	deps.Thing2()
}

type doThingsDeps struct {
	Thing1 func()
	Thing2 func()
	Thing3 func()
	Thing4 func() bool
	Thing5 func(int) string
	Thing6 func(string) int
	Thing7 func(bool)
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
	calls := make(chan imptest.YieldedValue)

	// Given the dependencies are replaced by functions which place their calls on the channel

	// WrapFunc returns a function of the same signature, but which:
	// * puts the given function on the calls channel for test validation
	// * waits for the test to tell it to return before returning
	// It also returns an ID, to compare against, because go does not allow us
	// to compare functions.
	var (
		id1, id2 string
		deps     doThingsDeps
	)

	deps.Thing1, id1 = imptest.WrapFunc(t, thing1, calls)
	deps.Thing2, id2 = imptest.WrapFunc(t, thing2, calls)

	// When DoThings is started
	go func() {
		// record when the func is done so we can test that, too
		defer close(calls)
		DoThings(deps)
	}()

	// Then thing1 is called
	funcCall1 := <-calls
	if funcCall1.Type != imptest.YieldedCall {
		t.Fail()
	}

	if funcCall1.Call.ID != id1 {
		t.Fail()
	}

	// When thing1 returns
	funcCall1.Call.Return()

	// Then thing2 is called
	funcCall2 := <-calls
	if funcCall2.Type != imptest.YieldedCall {
		t.Fail()
	}

	if funcCall2.Call.ID != id2 {
		t.Fail()
	}

	// When thing2 returns
	funcCall2.Call.Return()

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
	deps := new(doThingsDeps)
	imp := imptest.NewImp(t, deps)

	// When DoThings is started
	imp.Start(DoThings, *deps)

	// Then the functions are called in the following order
	imp.ExpectCall("Thing1").ExpectArgs().PushReturns()
	imp.ExpectCall("Thing2").ExpectArgs().PushReturns()

	// Then the function returned
	imp.ExpectReturns()
}

// The test replaces those functions in order to test they are called.
func TestNoMoreCalls(t *testing.T) {
	t.Parallel()

	mockTester := newMockedTestingT()

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		// Given deps replaced
		var (
			id1, id2 string
			deps     doThingsDeps
		)

		deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)
		deps.Thing2, id2 = imptest.WrapFunc(t, thing2, tester.OutputChan)

		// When DoThings is started
		tester.Start(DoThings, deps)

		// Then the functions are called in the following order
		tester.AssertCalled(id1).Return()
		tester.AssertCalled(id2).Return()
		tester.Called()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "Expected a call, but none was found"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func DoThingsWithBranch(deps doThingsDeps) {
	deps.Thing1()

	if deps.Thing4() {
		deps.Thing2()
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

	deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)
	deps.Thing4, id4 = imptest.WrapFunc(t, thing4, tester.OutputChan)

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

	deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)
	deps.Thing2, id2 = imptest.WrapFunc(t, thing2, tester.OutputChan)
	deps.Thing4, id4 = imptest.WrapFunc(t, thing4, tester.OutputChan)

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
	y := deps.Thing5(x)
	return deps.Thing6(y)
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

	deps.Thing5, id5 = imptest.WrapFunc(t, thing5, tester.OutputChan)
	deps.Thing6, id6 = imptest.WrapFunc(t, thing6, tester.OutputChan)

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

	deps.Thing1()

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

	deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)

	// When DoThings is started
	tester.Start(DoThingsWithPanic, deps)

	// Then id7 is called. When it panics...
	tester.AssertCalled(id1).Panic("omg what?")

	// Then the function returns the panic message
	tester.AssertReturned("omg what?")
}

func DoThingsConcurrently(deps doThingsDeps) {
	go deps.Thing3()
	go func() {
		z := deps.Thing4()
		deps.Thing7(z)
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

	deps.Thing3, id3 = imptest.WrapFunc(t, thing3, tester.OutputChan)
	deps.Thing4, id4 = imptest.WrapFunc(t, thing4, tester.OutputChan)
	deps.Thing7, id7 = imptest.WrapFunc(t, thing7, tester.OutputChan)

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
	go deps.Thing3()
	go func() {
		z := deps.Thing4()
		deps.Thing7(z)

		go deps.Thing1()
		go deps.Thing2()
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

	deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)
	deps.Thing2, id2 = imptest.WrapFunc(t, thing2, tester.OutputChan)
	deps.Thing3, id3 = imptest.WrapFunc(t, thing3, tester.OutputChan)
	deps.Thing4, id4 = imptest.WrapFunc(t, thing4, tester.OutputChan)
	deps.Thing7, id7 = imptest.WrapFunc(t, thing7, tester.OutputChan)

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

func TestCallAfterDonePanics(t *testing.T) {
	t.Parallel()

	tester := imptest.NewFuncTester(t)

	var (
		deps          doThingsDeps
		id1, id2, id3 string
	)

	deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)
	deps.Thing2, id2 = imptest.WrapFunc(t, thing2, tester.OutputChan)
	deps.Thing3, id3 = imptest.WrapFunc(t, thing3, tester.OutputChan)

	orphanReleaseChan := make(chan struct{})
	testReleaseChan := make(chan struct{})

	DoThingsWithOrphanMadeAfterDone := func(deps doThingsDeps) {
		deps.Thing1()
		deps.Thing2()
		deps.Thing3()

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
			deps.Thing3()
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

		deps.Thing3, id3 = imptest.WrapFunc(t, thing3, tester.OutputChan)
		deps.Thing4, id4 = imptest.WrapFunc(t, thing4, tester.OutputChan)
		deps.Thing7, id7 = imptest.WrapFunc(t, thing7, tester.OutputChan)

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

	expected := "Found expected call"
	expected2 := "but the args differed"
	expected3 := "expected: []interface {}{true}"
	actual := mockTester.Failure()

	if !(strings.Contains(actual, expected) &&
		strings.Contains(actual, expected2) &&
		strings.Contains(actual, expected3)) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

const NOTFOUNDMESSAGE = "it was not found"

func DoThings3xSync(deps doThingsDeps) {
	deps.Thing1()
	deps.Thing1()
	deps.Thing1()
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

		deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)

		// When DoThings is started
		tester.Start(DoThings3xSync, deps)

		tester.AssertCalled(id1).Return()
		tester.AssertReturned()
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "Expected a return, but none was found"
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

		deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)

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

	expected := `Failed to find expected call to .*thing1`
	actual := mockTester.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func DoThings3xAsync(deps doThingsDeps) {
	go deps.Thing1()
	go deps.Thing1()
	go deps.Thing1()
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

		deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)

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

		deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)

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

	expected := "outputs channel was already closed"
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
	tester := imptest.NewFuncTester(t)
	tester.Timeout = 10 * time.Second

	// Given deps replaced
	var (
		id6  string
		deps doThingsDeps
	)

	deps.Thing5, _ = imptest.WrapFunc(t,
		thing5,
		tester.OutputChan,
		imptest.WithName("thing5"),
	)
	deps.Thing6, id6 = imptest.WrapFunc(t, thing6, tester.OutputChan)

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

		deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)

		// When DoThings is started
		tester.Start(DoThings3xSync, deps)

		tester.AssertCalled(id1).Return()
		tester.AssertCalled(id1).Return()
		tester.AssertReturned()
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "Expected a return, but none was found"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}

	expected = `(?s:.*Yielded outputs.*call to.*thing1.*with args.*\[])`
	actual = mockTester.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected regexp match for '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func TestWrongReturnFails(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	testFunc := func() int { return 5 }

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		tester.Start(testFunc)

		// should fail - this is the wrong return
		tester.AssertReturned(4)
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

func TestWrongPanicFails(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	testFunc := func() { panic("a message") }

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		tester.Start(testFunc)

		// should fail - this is the wrong return
		tester.AssertPanicked("a different message")
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "panicked with \"a message\" instead"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func TestPanicInsteadOfReturn(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	testFunc := func() int { panic(5) }

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		tester.Start(testFunc)

		// should fail - this is the wrong return
		tester.AssertReturned(4)
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

func TestReturnInsteadOfPanic(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	mockTester := newMockedTestingT()

	testFunc := func() int { return 5 }

	mockTester.Wrap(func() {
		// convenience test wrapper
		tester := imptest.NewFuncTester(mockTester)

		tester.Start(testFunc)

		// should fail - this is the wrong return
		tester.AssertPanicked(4)
		tester.Close()
	})

	if !mockTester.Failed() {
		t.Fatal("Test didn't fail, but we expected it to.")
	}

	expected := "panic"
	actual := mockTester.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't fail with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected, actual,
		)
	}
}

func TestPanicCustom(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	testFunc := func() { panic("a message") }

	// convenience test wrapper
	tester := imptest.NewFuncTester(t)

	tester.Start(testFunc)

	// type assertion failure will just fail the test, it's fine
	actual := tester.Panicked().(string) //nolint:forcetypeassert
	tester.Close()

	expected := "a message"

	if !strings.Contains(actual, expected) {
		t.Fatalf("Test didn't panic with the expected message.\n"+
			"Expected '%s'.\n"+
			"Got '%s'",
			expected,
			// type assertion failure wit's fine
			tester.Panicked().(string), //nolint:forcetypeassert
		)
	}
}

func TestPanicTimeout(t *testing.T) {
	t.Parallel()

	holdChan := make(chan struct{})

	// Given convenience test wrapper
	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		tester := imptest.NewFuncTester(mockedT)
		tester.Timeout = 1 * time.Microsecond
		tester.Start(func() { <-holdChan })
		tester.AssertPanicked(nil)
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to time out instead of validating the panic")
	}

	expected := "timed out"
	actual := mockedT.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
	holdChan <- struct{}{}
}

func DoThingsConcurrentlyNested2(deps doThingsDeps) {
	deps.Thing1()

	go func() {
		z := deps.Thing4()
		deps.Thing7(z)

		go func() {
			a := deps.Thing5(2)
			deps.Thing6(a)
		}()

		deps.Thing5(4)
		deps.Thing5(5)
	}()
	deps.Thing2()

	go deps.Thing3()
}

func TestNestedConcurrentlies2(t *testing.T) {
	// Given deps replaced
	t.Parallel()

	// convenience test wrapper
	tester := imptest.NewFuncTester(t)

	var (
		deps                              doThingsDeps
		id1, id2, id3, id4, id5, id6, id7 string
	)

	deps.Thing1, id1 = imptest.WrapFunc(t, thing1, tester.OutputChan)
	deps.Thing2, id2 = imptest.WrapFunc(t, thing2, tester.OutputChan)
	deps.Thing3, id3 = imptest.WrapFunc(t, thing3, tester.OutputChan)
	deps.Thing4, id4 = imptest.WrapFunc(t, thing4, tester.OutputChan)
	deps.Thing5, id5 = imptest.WrapFunc(t, thing5, tester.OutputChan)
	deps.Thing6, id6 = imptest.WrapFunc(t, thing6, tester.OutputChan)
	deps.Thing7, id7 = imptest.WrapFunc(t, thing7, tester.OutputChan)

	// When DoThings is started
	tester.Start(DoThingsConcurrentlyNested2, deps)

	// Then the functions are called in any order
	tester.AssertCalled(id1).Return()
	tester.Concurrently(func() {
		tester.AssertCalled(id4).Return(true)
		tester.AssertCalled(id7, true).Return()
		tester.Concurrently(func() {
			tester.AssertCalled(id5, 2).Return("two")
		}, func() {
			tester.AssertCalled(id6, "two").Return(2)
		}, func() {
			tester.AssertCalled(id5, 4).Return("four")
			tester.AssertCalled(id5, 5).Return("five")
		})
	}, func() {
		tester.AssertCalled(id2).Return()
		tester.Concurrently(func() {
			tester.AssertCalled(id3).Return()
		}, func() {
			tester.AssertReturned()
		})
	})
	tester.Close()
}

func TestDefaultTimeout(t *testing.T) {
	t.Parallel()

	tester := imptest.NewFuncTester(t)
	actual := tester.Timeout
	expected := 500 * time.Millisecond

	if actual != expected {
		t.Fatalf("Expected the default timeout to be %v, but it was %v instead", expected, actual)
	}
}

func customDiffer(a, b any) (string, error) {
	// expect a and b are arrays of args
	aarray, aarrayok := a.([]any)
	barray, barrayok := b.([]any)

	// if not, not equal
	if !(aarrayok && barrayok) {
		panic(fmt.Sprintf("arguments were not arrays of args: diffing %#v against %#v", aarray, barray))
	}

	// if not same length, not equal
	if len(aarray) != len(barray) {
		return fmt.Sprintf("arrays were different lengths: %d vs %d", len(aarray), len(barray)), nil
	}

	return customDiffArrays(aarray, barray), nil
}

func customDiffArrays(aarray []any, barray []any) string {
	// check equality of all args
	for argNum := range aarray {
		// compare strings
		astring, astringok := aarray[argNum].(string)
		bstring, bstringok := barray[argNum].(string)

		if astringok && bstringok {
			// if they're not equal, but they are case-insensitive equal, say that.
			if astring == bstring {
				continue
			}

			lowAString := strings.ToLower(astring)
			lowBString := strings.ToLower(bstring)

			if lowAString == lowBString {
				return fmt.Sprintf("capitalization mismatch: %s vs %s", astring, bstring)
			}
		}

		// compare ints
		aint, aintok := aarray[argNum].(int)
		bint, bintok := barray[argNum].(int)

		if aintok && bintok && aint == bint {
			continue
		}

		// anything else is not equal
		return fmt.Sprintf("%#v and %#v are not equal", aarray[argNum], barray[argNum])
	}

	// if we made it through, including if array is zero length, call it equal
	return ""
}

func TestDoThingsWithCustomDiffer(t *testing.T) {
	t.Parallel()

	// Given convenience test wrapper
	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		tester := imptest.NewFuncTester(mockedT)
		tester.Differ = customDiffer

		// Given deps replaced
		var (
			id5, id6 string
			deps     doThingsDeps
		)

		deps.Thing5, id5 = imptest.WrapFunc(t, thing5, tester.OutputChan)
		deps.Thing6, id6 = imptest.WrapFunc(t, thing6, tester.OutputChan)

		// When DoThings is started
		tester.Start(DoThingsWithArgs, 1, deps)

		// Then the functions are called in the following order
		tester.AssertCalled(id5, 1).Return("hi")
		// Fail with a custom case-sensitivity message
		tester.AssertCalled(id6, "Hi").Return(2)
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := "capitalization mismatch"
	actual := mockedT.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}
