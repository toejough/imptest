package imptest_test

import (
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
)

func TestPutCallTooFewArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutTooFew) {
		// Then the next call fails with too few args
		defer expectPanicWith(t, "Too few args")
		deps.PutTooFew(5, "six")
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// and the func is done
		tester.AssertDoneWithin(time.Second)
	})
}

type testDepsPutTooFew interface{ PutTooFew(i int, s string) }

func (tdm *testDepsMock) PutTooFew(i int, _ string) { tdm.tester.PutCall(tdm.PutTooFew, i) }

func TestPutCallTooManyArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutTooMany) {
		// Then the next call fails with too many args
		defer expectPanicWith(t, "Too many args")
		deps.PutTooMany(5, "six")
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run and completes
		tester.Start(returns, tdm)
		tester.AssertDoneWithin(time.Second)
	})
}

type testDepsPutTooMany interface{ PutTooMany(i int, s string) }

func (tdm *testDepsMock) PutTooMany(i int, s string) {
	tdm.tester.PutCall(tdm.PutTooMany, i, s, "THIS ONE IS TOO MANY")
}

func TestPutCallWrongTypesFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutWrongTypes) {
		// Then the next call fails with too many args
		defer expectPanicWith(t, "Wrong arg type")
		deps.PutWrongTypes(5, "six")
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run and completes
		tester.Start(returns, tdm)
		tester.AssertDoneWithin(time.Second)
	})
}

type testDepsPutWrongTypes interface{ PutWrongTypes(i int, s string) }

func (tdm *testDepsMock) PutWrongTypes(_ int, s string) {
	tdm.tester.PutCall(tdm.PutWrongTypes, "THIS ONE IS THE WRONG TYPE", s)
}

func TestNoPutCallFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)
	// Given inputs
	lockchan := make(chan struct{})
	returns := func(deps testDepsNoPut) {
		deps.NoPut()
		<-lockchan
	}
	tdm := newTestDepsMock(tester)

	// release the lock at the end of the test
	defer close(lockchan)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// Then the assert next call fails waiting for that call
		defer expectPanicWith(t, "waiting for a call")
		tester.AssertNextCallWithin(time.Second, tdm.NoPut)
	})
}

type testDepsNoPut interface{ NoPut() }

func (tdm *testDepsMock) NoPut() {}
