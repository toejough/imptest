package imptest_test

import (
	"strings"
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
)

func TestAssertNextCallIsNoArgsPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDeps) {
		deps.Func()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// Then the next call is to the func
		tester.AssertNextCallIs(tdm.Func)

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

type testDeps interface{ Func() }

type testDepsMock struct{ tester *imptest.RelayTester }

func newTestDepsMock(t *imptest.RelayTester) *testDepsMock {
	return &testDepsMock{tester: t}
}

func (tdm *testDepsMock) Func() { tdm.tester.PutCall(tdm.Func) }

func TestAssertNextCallIsWrongFuncFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsWrongFunc) {
		deps.WrongFunc()
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// Then the next call is to the func
		tester.AssertNextCallIs(tdm.WrongFunc)

		// And we wait for the test to complete
		tester.AssertFinishes()
	})

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
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// Then the next call fails with too few args
		defer expectPanicWith(mockedt, "Too few args")
		tester.AssertNextCallIs(tdm.SomeArgs, 5)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertNextCallIsTooManyArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)

		// Then the next call fails with too few args
		defer expectPanicWith(mockedt, "Too many args")
		tester.AssertNextCallIs(tdm.SomeArgs, 5, "six", 0x7)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertNextCallIsWrongTypeFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)
		// returns(tdm)

		// Then the next call fails with wrong arg type
		defer expectPanicWith(mockedt, "Wrong arg type")
		tester.AssertNextCallIs(tdm.SomeArgs, 5, 6)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertNextCallIsWrongValuesFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	mockedt.Wrap(func() {
		tester.Start(returns, tdm)

		// Then the next call is to the func
		tester.AssertNextCallIs(tdm.SomeArgs, 5, "seven")

		// and we wait for the test to complete
		tester.AssertFinishes()
	})

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

func TestAssertNextCallIsAfterDoneFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsAfterDone) {}
	tdm := newTestDepsMock(tester)

	// When the func is run
	mockedt.Wrap(func() {
		tester.Start(returns, tdm)
		// and nice cleanup happens
		tester.AssertFinishes()

		// Then the assertion on the next call fails
		tester.AssertNextCallIs(tdm.AfterDone)

		// and we wait for the test to complete
		tester.AssertFinishes()
	})

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

type testDepsAfterDone interface{ AfterDone() }

func (tdm *testDepsMock) AfterDone() { tdm.tester.PutCall(tdm.AfterDone) }

func TestAssertNextCallIsWithNonFunction(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)
	// Given inputs
	returns := func(deps testDepsSomeArgs) {
		deps.SomeArgs(5, "six")
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, tdm)
		// returns(tdm)

		// Then the next call fails with wrong arg type
		defer expectPanicWith(mockedt, "must pass a function")
		tester.AssertNextCallIs(time.Second, "SomeArgs", 5, 6)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}
