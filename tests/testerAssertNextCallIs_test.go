package protest_test

import (
	"strings"
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestAssertNextCallIsNoArgsPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDeps) {
		deps.Func()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	// and nice cleanup is scheduled
	defer tester.AssertDoneWithin(time.Second)

	// Then the next call is to the func
	tester.AssertNextCallIs(tdm.Func)

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDeps interface{ Func() }

type testDepsMock struct{ tester *protest.RelayTester }

func newTestDepsMock(t *protest.RelayTester) *testDepsMock {
	return &testDepsMock{tester: t}
}

func (tdm *testDepsMock) Func() { tdm.tester.PutCall(tdm.Func) }

func TestAssertNextCallIsWrongFuncFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsWrongFunc) {
		deps.WrongFunc()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	// and nice cleanup is scheduled
	defer tester.AssertDoneWithin(time.Second)

	// Then the next call is to the func
	tester.AssertNextCallIs(tdm.WrongFunc)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal(
			"The test should've failed with wrong func called. Instead the test passed!",
		)
	}
	// Then the error calls out wrong value
	if !strings.Contains(mockedt.Failure(), "wrong func") {
		t.Fatalf(
			"The test should've failed with wrong func called. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsWrongFunc interface{ WrongFunc() }

func (tdm *testDepsMock) WrongFunc() { tdm.tester.PutCall(tdm.Func) }

func TestAssertNextCallIsTooFewArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)

	// Then the next call fails with too few args
	defer expectPanicWith(t, "too few args")
	tester.AssertNextCallIs(tdm.SomeArgs, 5)
}

func TestAssertNextCallIsTooManyArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)

	// Then the next call fails with too few args
	defer expectPanicWith(t, "too many args")
	tester.AssertNextCallIs(tdm.SomeArgs, 5, "six", 0x7)
}

func TestAssertNextCallIsWrongTypeFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	// returns(tdm)

	// Then the next call fails with wrong arg type
	defer expectPanicWith(t, "Wrong type")
	tester.AssertNextCallIs(tdm.SomeArgs, 5, 6)
}

func TestAssertNextCallIsWrongValuesFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	// and nice cleanup is scheduled
	defer tester.AssertDoneWithin(time.Second)

	// Then the next call is to the func
	tester.AssertNextCallIs(tdm.SomeArgs, 5, "seven")

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal(
			"The test should've failed with wrong values. Instead the test passed!",
		)
	}
	// Then the error calls out wrong value
	if !strings.Contains(mockedt.Failure(), "wrong values") {
		t.Fatalf(
			"The test should've failed with wrong values. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsSomeArgs interface{ SomeArgs(i int, s string) }

func (tdm *testDepsMock) SomeArgs(i int, s string) { tdm.tester.PutCall(tdm.SomeArgs, i, s) }
