package imptest_test

import (
	"testing"

	"github.com/toejough/protest/imptest"
)

func TestFillReturnWrongTypeFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillWrongType) int {
		// Then test fails with wrong return type
		defer expectPanicWith(mockedt, "wrong return type")
		return deps.FillWrongType()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// and a good return is injected
		tester.AssertNextCallIs(tdm.FillWrongType).InjectReturns(5)

		// and we wait for the test to complete
		tester.AssertFinishes()
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsFillWrongType interface{ FillWrongType() int }

func (tdm *testDepsMock) FillWrongType() int {
	var (
		goodR int
		badR  string
	)

	tdm.tester.PutCall(tdm.FillWrongType).FillReturns(&badR)

	return goodR
}

func TestFillReturnWrongNumberFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillWrongNumber) int {
		// Then test fails with wrong return type
		defer expectPanicWith(mockedt, "Too many returns")
		return deps.FillWrongNumber()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// And a good return is injected
		tester.AssertNextCallIs(tdm.FillWrongNumber).InjectReturns(5)

		// and we wait for the test to complete
		tester.AssertFinishes()
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsFillWrongNumber interface{ FillWrongNumber() int }

func (tdm *testDepsMock) FillWrongNumber() int {
	var (
		goodR int
		badR  string
	)

	tdm.tester.PutCall(tdm.FillWrongNumber).FillReturns(&goodR, &badR)

	return goodR
}

func TestFillNonPointerFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillNonPointer) int {
		// Then test fails with wrong return type
		defer expectPanicWith(mockedt, "cannot fill value into non-pointer")
		return deps.FillNonPointer()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// and a good return is injected
		tester.AssertNextCallIs(tdm.FillNonPointer).InjectReturns(5)

		// and we wait for the test to complete
		tester.AssertFinishes()
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsFillNonPointer interface{ FillNonPointer() int }

func (tdm *testDepsMock) FillNonPointer() int {
	var goodR int

	tdm.tester.PutCall(tdm.FillNonPointer).FillReturns(goodR)

	return goodR
}

func TestFillNeverCalledFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillNeverCalled) int {
		return deps.FillNeverCalled()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)
		call := tester.AssertNextCallIs(tdm.FillNeverCalled)

		// Then test fails with fill never called
		defer expectPanicWith(mockedt, "fill was not called")
		call.InjectReturns(5)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsFillNeverCalled interface{ FillNeverCalled() int }

func (tdm *testDepsMock) FillNeverCalled() int {
	var goodR int

	tdm.tester.PutCall(tdm.FillNeverCalled)

	return goodR
}

// TODO: figure out if there's a way to do some property-based testing
