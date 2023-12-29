package imptest_test

import (
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
)

func TestInjectReturnPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertNextCallIs(tdm.Inject).InjectReturns(5)
	tester.AssertDoneWithin(time.Second)
	tester.AssertReturned(5)

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
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInjectNil) *int {
		return deps.InjectNil()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)
		tester.AssertNextCallIs(tdm.InjectNil).InjectReturns(nil)
		tester.AssertDoneWithin(time.Second)
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
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	call := tester.AssertNextCallIs(tdm.Inject)

	// Then test fails with wrong return type
	defer expectPanicWith(t, "Wrong return type")
	call.InjectReturns("five")
	tester.AssertDoneWithin(time.Second)
}

func TestInjectReturnWrongNumberFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	call := tester.AssertNextCallIs(tdm.Inject)

	// Then test fails with wrong number of returns
	defer expectPanicWith(t, "Too many returns")
	call.InjectReturns(5, "five")
	tester.AssertDoneWithin(time.Second)
}
