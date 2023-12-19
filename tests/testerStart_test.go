package protest_test

import (
	"strings"
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestStartRunsFUTInGoroutine(t *testing.T) {
	t.Parallel()

	// Given test needs
	tester := protest.NewTester(t)
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

func TestStartFailsCleanlyWithTooFewArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)

	// Given FUT
	argFunc := func(_, _, _ int) {}

	// When the func is run with the wrong number of args
	tester.Start(argFunc)
	// And we wait for shutdown
	tester.AssertDoneWithin(time.Second)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal("The test should've failed due to too few args, but it didn't")
	}

	// Then the test has the right error message
	if !strings.Contains(mockedt.Failure(), "too few input arguments") {
		t.Fatalf(
			"The test should've failed due to too few args, but it didn't. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestStartFailsCleanlyWithTooManyArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	// When the func is run with the wrong number of args
	tester.Start(argFunc, 1, 2, 3)
	// And we wait for shutdown
	tester.AssertDoneWithin(time.Second)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal("The test should've failed due to too few args, but it didn't")
	}

	// Then the test has the right error message
	if !strings.Contains(mockedt.Failure(), "too many input arguments") {
		t.Fatalf(
			"The test should've failed due to too many args, but it didn't. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestStartFailsCleanlyWithWrongArgTypes(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	// When the func is run with the wrong number of args
	tester.Start(argFunc, "1")
	// And we wait for shutdown
	tester.AssertDoneWithin(time.Second)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal("The test should've failed due to wrong arg type, but it didn't")
	}

	// Then the test has the right error message
	if !strings.Contains(mockedt.Failure(), "using string as type int") {
		t.Fatalf(
			"The test should've failed due wrong arg type, but it didn't. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}
