package imptest_test

import (
	"testing"

	"github.com/toejough/protest/imptest"
)

func TestInjectReturnPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)

	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// and a good return is injected
		tester.AssertNextCallIs(tdm.Inject).InjectReturns(5)

		// and we wait for done
		tester.AssertFinishes()

		// Then we get a good return from the FUT
		tester.AssertReturned(5)
	})

	// Then the test passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsInject interface{ Inject() int }

func (tdm *testDepsMock) Inject() int {
	var r int

	tdm.tester.PutCall(tdm.Inject).FillReturns(&r)

	return r
}

func TestInjectReturnNilPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)

	// Given inputs
	returns := func(deps testDepsInjectNil) *int {
		return deps.InjectNil()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// and a nil is injected
		tester.AssertNextCallIs(tdm.InjectNil).InjectReturns(nil)

		// and we wait for the function to return
		tester.AssertFinishes()

		// then a nil is returned
		tester.AssertReturned(nil)
	})

	// Then the test passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsInjectNil interface{ InjectNil() *int }

func (tdm *testDepsMock) InjectNil() *int {
	var r *int

	tdm.tester.PutCall(tdm.InjectNil).FillReturns(&r)

	return r
}

func TestInjectReturnWrongTypeFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// and the call is returned
		call := tester.AssertNextCallIs(tdm.Inject)

		// Then test fails with wrong return type
		defer expectPanicWith(t, "Wrong return type")
		call.InjectReturns("five")
	})
}

func TestInjectReturnWrongNumberFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// and the next call is returned
		call := tester.AssertNextCallIs(tdm.Inject)

		// Then test fails with wrong number of returns
		defer expectPanicWith(t, "Too many returns")
		call.InjectReturns(5, "five")
	})
}
