package imptest_test

import (
	"testing"
	"time"

	imptest "github.com/toejough/protest/api"
)

func TestFillReturnWrongTypeFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillWrongType) int {
		// Then test fails with wrong return type
		defer expectPanicWith(t, "wrong return type")
		return deps.FillWrongType()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertNextCallIs(tdm.FillWrongType).InjectReturns(5)
	tester.AssertDoneWithin(time.Second)
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
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillWrongNumber) int {
		// Then test fails with wrong return type
		defer expectPanicWith(t, "wrong number of returns")
		return deps.FillWrongNumber()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertNextCallIs(tdm.FillWrongNumber).InjectReturns(5)
	tester.AssertDoneWithin(time.Second)
}

type testDepsFillWrongNumber interface{ FillWrongNumber() int }

func (tdm *testDepsMock) FillWrongNumber() int {
	var (
		goodR int
		badR  string
	)

	tdm.tester.PutCall(tdm.FillWrongType).FillReturns(&goodR, &badR)

	return goodR
}

func TestFillNonPointerFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillNonPointer) int {
		// Then test fails with wrong return type
		defer expectPanicWith(t, "cannot fill value into non-pointer")
		return deps.FillNonPointer()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertNextCallIs(tdm.FillWrongType).InjectReturns(5)
	tester.AssertDoneWithin(time.Second)
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
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillNeverCalled) int {
		return deps.FillNeverCalled()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	call := tester.AssertNextCallIs(tdm.FillNeverCalled)

	// Then test fails with fill never called
	defer expectPanicWith(t, "fill was not called")
	call.InjectReturns(5)

	tester.AssertDoneWithin(time.Second)
}

type testDepsFillNeverCalled interface{ FillNeverCalled() int }

func (tdm *testDepsMock) FillNeverCalled() int {
	var goodR int

	tdm.tester.PutCall(tdm.FillNeverCalled)

	return goodR
}

// TODO: refactor imptest out into separate files
// TODO: figure out if there's a way to do some property-based testing
