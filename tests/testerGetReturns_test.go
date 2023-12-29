package imptest_test

import (
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
)

func TestGetReturnsPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsGet) int {
		return deps.Get()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	mockedt.Wrap(func() {
		tester.Start(returns, tdm)
		tester.AssertNextCallIs(tdm.Get).InjectReturns(5)
		tester.AssertDoneWithin(time.Second)
	})

	returnVals := tester.GetReturns()

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}

	// Then the return vals are as expected
	if len(returnVals) != 1 {
		t.Fatalf("The test returned %d returns. 1 was expected", len(returnVals))
	}
	// don't care if the type assertion fails. that'll fail the test, too
	if returnVals[0].(int) != 5 { //nolint:forcetypeassert
		t.Fatalf("The test returned %d. expected 5", returnVals[0].(int)) //nolint:forcetypeassert
	}
}

type testDepsGet interface{ Get() int }

func (tdm *testDepsMock) Get() int {
	var r int

	tdm.tester.PutCall(tdm.Get).FillReturns(&r)

	return r
}
