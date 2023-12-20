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
	returns := func(deps testDeps) {
		deps.Func()
	}
	tdm := newTestDepsMockWrongFunc(tester)

	// When the func is run
	tester.Start(returns, tdm)
	// and nice cleanup is scheduled
	defer tester.AssertDoneWithin(time.Second)

	// Then the next call is to the func
	tester.AssertNextCallIs(tdm.Func)

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

type testDepsMockWrongFunc struct{ tester *protest.RelayTester }

func newTestDepsMockWrongFunc(t *protest.RelayTester) *testDepsMockWrongFunc {
	return &testDepsMockWrongFunc{tester: t}
}

func (tdm *testDepsMockWrongFunc) Func() { tdm.tester.PutCall(wrongFunc) }

func wrongFunc() {}

// TODO: test that AssertNextCallIs fails if the call is wrong
// TODO: test that AssertNextCallIs fails if the args are the wrong type
// TODO: test that AssertNextCallIs fails if the args are the wrong number
// TODO: test that AssertNextCallIs fails if the args are the wrong value
// TODO: test that InjectReturns passes if the args are the right type and number
// TODO: test that InjectReturns fails if the args are the wrong type
// TODO: test that InjectReturns fails if the args are the wrong number
// TODO: test that PutCall passes if the args are the right type and number for the call
// TODO: test that PutCall fails if the args are the wrong type for the call
// TODO: test that FillReturns passes if the args are the right type and number for the call
// TODO: test that FillReturns fails if the args are the wrong type for the call
// TODO: test that FillReturns fails if the args are the wrong number for the call
// TODO: test parallel calls
// TODO: refactor protest out into separate files
