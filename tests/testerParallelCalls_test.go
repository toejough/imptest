package imptest_test

import (
	"sync"
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
)

func TestParallelCallsPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := imptest.NewDefaultRelayTester(mockedt)
	// Given inputs
	returns := func(arg int, deps testDepsParallel) (int, int) {
		var aResult, bResult int

		waitgroup := &sync.WaitGroup{}
		waitgroup.Add(2)

		go func() {
			defer waitgroup.Done()

			aResult = deps.ParallelA(arg)
		}()
		go func() {
			defer waitgroup.Done()

			bResult = deps.ParallelB(arg)
		}()

		waitgroup.Wait()

		return aResult, bResult
	}
	tdm := newTestDepsMock(tester)

	mockedt.Wrap(func() {
		// When the func is run
		tester.Start(returns, 6, tdm)

		// And the parallel calls are made
		call1 := tester.GetNextCallWithin(time.Second)
		if call1.Name() == imptest.GetFuncName(tdm.ParallelA) {
			imptest.AssertCallIs(t, call1, tdm.ParallelA, 6)
			call1.InjectReturnsWithin(time.Second, 1)
			tester.AssertNextCallWithin(time.Second, tdm.ParallelB, 6).InjectReturnsWithin(time.Second, 2)
		} else {
			imptest.AssertCallIs(t, call1, tdm.ParallelB, 6)
			call1.InjectReturnsWithin(time.Second, 2)
			tester.AssertNextCallWithin(time.Second, tdm.ParallelA, 6).InjectReturnsWithin(time.Second, 1)
		}

		// And the test completes
		tester.AssertDoneWithin(time.Second)
		tester.AssertReturned(1, 2)
	})

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
