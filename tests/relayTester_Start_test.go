package imptest_test

import (
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
	"pgregory.net/rapid"
)

func TestStartRunsFUTInGoroutine(t *testing.T) {
	t.Parallel()

	// Given test needs
	tester := imptest.NewRelayTester(t)
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

func testStartPanicsWithNonFunction(rapidT *rapid.T) {
	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := 5
	// TODO: use rapid.Custom to generate arbitrary non-function types from reflect.
	// https://pkg.go.dev/pgregory.net/rapid#Custom
	// https://pkg.go.dev/reflect#Type
	// https://pkg.go.dev/reflect#New

	mockedt.Wrap(func() {
		// When the func is run with something that isn't a function
		defer expectPanicWith(mockedt, "must pass a function")
		tester.Start(argFunc)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		rapidT.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestStartPanicsWithNonFunction(t *testing.T) {
	t.Parallel()
	rapid.Check(t, testStartPanicsWithNonFunction)
}

func TestStartPanicsWithTooFewArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := func(_, _, _ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(mockedt, "Too few args")
		tester.Start(argFunc)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestStartPanicsWithTooManyArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(mockedt, "Too many args")
		tester.Start(argFunc, 1, 2, 3)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestStartPanicsWithWrongArgTypes(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(mockedt, "Wrong arg type")
		tester.Start(argFunc, "1")
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}
