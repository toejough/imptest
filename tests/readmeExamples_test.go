package imptest_test

import (
	"testing"

	"github.com/toejough/protest/imptest"
)

// DoThings now calls functions from a package-level dependencies struct.
func DoThings() {
	pkgDeps.thing1()
	pkgDeps.thing2()
	pkgDeps.thing3()
}

func thing1() {}
func thing2() {}
func thing3() {}

// The package dependencies are initialized with the original functions.
var pkgDeps = struct {
	thing1 func()
	thing2 func()
	thing3 func()
}{thing1, thing2, thing3}

// The test replaces those functions in order to test they are called.
func TestDoThingsRunsExpectedFuncsInOrder(t *testing.T) {
	// Given pkg deps replaced
	calls := make(chan imptest.FuncCall)
	// WrapFunc returns a function of the same signature, but which:
	// * puts the given function on the relay for test validation
	// * waits for the test to tell it to return before returning
	// It also returns an ID, to compare against, because go does not allow
	// us to compare functions.
	var id1, id2, id3 string
	pkgDeps.thing1, id1 = imptest.WrapFunc(thing1, calls)
	pkgDeps.thing2, id2 = imptest.WrapFunc(thing2, calls)
	pkgDeps.thing3, id3 = imptest.WrapFunc(thing3, calls)

	// When DoThings is started
	go func() {
		// record when the func is done so we can test that, too
		defer close(calls)
		DoThings()
	}()

	// Then thing1 is called
	funcCall1 := <-calls
	if funcCall1.ID != id1 {
		t.Fail()
	}

	// When thing1 returns
	funcCall1.Out <- []any{}

	// Then thing2 is called
	funcCall2 := <-calls
	if funcCall2.ID != id2 {
		t.Fail()
	}

	// When thing2 returns
	funcCall2.Out <- []any{}

	// Then thing3 is called
	funcCall3 := <-calls
	if funcCall3.ID != id3 {
		t.Fail()
	}

	// When thing3 returns
	funcCall3.Out <- []any{}

	// Then there are no more calls
	_, open := <-calls
	if open {
		t.Fail()
	}
}
