package imptest_test

import (
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
)

func TestStartRunsFUTInGoroutine(t *testing.T) {
	t.Parallel()

	// Given test needs
	tester := imptest.NewTester(t)
	// Given inputs
	lockchan := make(chan struct{})
	waitchan := make(chan struct{})
	wait := func() {
		<-lockchan
	}

	// release the lock at the end of the test
	defer close(lockchan)

	// When the func is run
	go func() {
		tester.Start(wait)
		close(waitchan)
	}()

	// Then the return from waitchan should be immediate
	select {
	case <-waitchan:
	case <-time.After(time.Second):
		t.Error("waitchan never closed, indicating function was run synchronously instead of in a goroutine.")
	}
}

func TestStartPanicsWithNonFunction(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)

	// Given FUT
	argFunc := 5

	mockedt.Wrap(func() {
		// When the func is run with something that isn't a function
		defer expectPanicWith(t, "must pass a function")
		tester.Start(argFunc)
	})
}

func TestStartPanicsWithTooFewArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)

	// Given FUT
	argFunc := func(_, _, _ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(t, "Too few args")
		tester.Start(argFunc)
	})
}

func TestStartPanicsWithTooManyArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(t, "Too many args")
		tester.Start(argFunc, 1, 2, 3)
	})
}

func TestStartPanicsWithWrongArgTypes(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(t, "Wrong arg type")
		tester.Start(argFunc, "1")
	})
}
