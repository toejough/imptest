package protest_test

import (
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestParallelCallsPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(arg int, deps testDepsParallel) (int, int) {
		var aResult, bResult int
		go func() { aResult = deps.ParallelA(arg) }()
		go func() { bResult = deps.ParallelB(arg) }()
		return aResult, bResult
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, 6, tdm)
	// And unordered calls are queueud
	tester.QueueUnordered(tdm.ParallelB, 6).InjectReturns(2)
	tester.QueueUnordered(tdm.ParallelA, 6).InjectReturns(1)
	// And the queue is asserted
	tester.AssertUnordered()
	// And the test completes
	tester.AssertDoneWithin(time.Second)
	tester.AssertReturned(1, 2)

	// Then the test passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsParallel interface {
	ParallelA(i int) int
	ParallelB(i int) int
}

func (tdm *testDepsMock) ParallelA(i int) int {
	var r int

	tdm.tester.PutCall(tdm.ParallelA, i).FillReturns(&r)

	return r
}

func (tdm *testDepsMock) ParallelB(i int) int {
	var r int

	tdm.tester.PutCall(tdm.ParallelB, i).FillReturns(&r)

	return r
}
