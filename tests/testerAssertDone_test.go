package imptest_test

import (
	"strings"
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
)

func TestAssertDoneFailsIfNotDone(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)
	// Given inputs
	lockchan := make(chan struct{})
	wait := func() {
		<-lockchan
	}

	// release the lock at the end of the test
	defer close(lockchan)

	// FIXME: this depends on actual wall time, and for test purposes, we really should
	// have the timer as an injected dependency
	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(wait)

		// And we wait for it to finish
		tester.AssertDoneWithin(time.Second)
	})

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
	tester := imptest.NewTester(mockedt)
	// Given inputs
	wait := func() {}

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(wait)

		// And we wait for it to finish
		tester.AssertDoneWithin(time.Second)
	})

	// Then the test is marked as failed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed due to FUT ending immediately, but it didn't: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertDoneWithQueuedCallFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)
	// Given inputs
	wait := func(deps testDepsQueuedCall) {
		deps.QueuedCall()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	mockedt.Wrap(func() {
		tester.Start(wait, tdm)

		// And we wait for it to finish
		tester.AssertDoneWithin(time.Second)
	})

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal("The test should've failed due to a queued call, but it didn't")
	}

	// Then the test has the right error message
	if !strings.Contains(mockedt.Failure(), "had a call queued") {
		t.Fatalf(
			"The test should've failed due to a queued call, but it didn't. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsQueuedCall interface{ QueuedCall() }

func (tdm *testDepsMock) QueuedCall() {
	tdm.tester.PutCall(tdm.QueuedCall)
}
