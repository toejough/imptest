package protest_test

import (
	"strings"
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestAssertDoneFailsIfNotDone(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	lockchan := make(chan struct{})
	wait := func() {
		<-lockchan
	}

	// release the lock at the end of the test
	defer close(lockchan)

	// When the func is run
	tester.Start(wait)

	// And we wait for it to finish
	// FIXME: this depends on actual wall time, and for test purposes, we really should
	// have the timer as an injected dependency
	tester.AssertDoneWithin(time.Second)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal("The test should've failed due to FUT not ending, but it didn't")
	}

	// Then the test has the right error message
	if !strings.Contains(mockedt.Failure(), "waiting for shutdown") {
		t.Fatalf(
			"The test should've failed due to the FUT not ending, but it didn't. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertDonePassesIfDone(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	wait := func() {}

	// When the func is run
	tester.Start(wait)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// Then the test is marked as failed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed due to FUT ending immediately, but it didn't: %s",
			mockedt.Failure(),
		)
	}
}
