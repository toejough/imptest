package imptest_test

import (
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
}

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

	// since WrapFunc returns a function of the same signature, we don't even
	// need a concrete function to emulate, we can just use the zero value
	// functions from the dependency struct to build the test function from
	deps.thing1, id1 = imptest.WrapFunc(deps.thing1, calls)
	deps.thing2, id2 = imptest.WrapFunc(deps.thing2, calls)
	deps.thing3, id3 = imptest.WrapFunc(deps.thing3, calls)

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
