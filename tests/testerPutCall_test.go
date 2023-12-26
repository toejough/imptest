package protest_test

import (
	"strings"
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestPutCallTooFewArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutTooFew) {
		// Then the next call fails with too few args
		defer expectPanicWith(t, "too few args")
		deps.PutTooFew(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertDoneWithin(time.Second)
}

type testDepsPutTooFew interface{ PutTooFew(i int, s string) }

func (tdm *testDepsMock) PutTooFew(i int, _ string) { tdm.tester.PutCall(tdm.PutTooFew, i) }

func TestPutCallTooManyArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutTooMany) {
		// Then the next call fails with too many args
		defer expectPanicWith(t, "too many args")
		deps.PutTooMany(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertDoneWithin(time.Second)
}

type testDepsPutTooMany interface{ PutTooMany(i int, s string) }

func (tdm *testDepsMock) PutTooMany(i int, s string) {
	tdm.tester.PutCall(tdm.PutTooMany, i, s, "THIS ONE IS TOO MANY")
}

func TestPutCallWrongTypesFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutWrongTypes) {
		// Then the next call fails with too many args
		defer expectPanicWith(t, "wrong arg type")
		deps.PutWrongTypes(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertDoneWithin(time.Second)
}

type testDepsPutWrongTypes interface{ PutWrongTypes(i int, s string) }

func (tdm *testDepsMock) PutWrongTypes(_ int, s string) {
	tdm.tester.PutCall(tdm.PutWrongTypes, "THIS ONE IS THE WRONG TYPE", s)
}

func TestNoPutCallFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	lockchan := make(chan struct{})
	returns := func(deps testDepsNoPut) {
		deps.NoPut()
		<-lockchan
	}
	tdm := newTestDepsMock(tester)

	// release the lock at the end of the test
	defer close(lockchan)

	// When the func is run
	tester.Start(returns, tdm)

	// Then the assert next call fails waiting for that call
	defer expectPanicWith(t, "waiting for a call")
	// FIXME: this depends on actual wall time, and for test purposes, we really should
	// have the timer as an injected dependency
	tester.AssertNextCallIs(tdm.NoPut)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal(
			"The test should've failed with relay already shut down. Instead the test passed!",
		)
	}
	// Then the error calls out wrong value
	if !strings.Contains(mockedt.Failure(), "already shut down") {
		t.Fatalf(
			"The test should've failed with relay already shut down. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsNoPut interface{ NoPut() }

func (tdm *testDepsMock) NoPut() {}
