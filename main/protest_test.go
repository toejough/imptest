package main

import (
	"testing"
	"time"
)

func TestNoCalls(t *testing.T) {
	t.Parallel()

	// Given test needs
	relay := NewCallRelay()
	tester := &RelayTester{T: t, Relay: relay} //nolint: exhaustruct // nobody else would be able to fill in private fields
	// Given inputs
	sum := func(a, b int) int {
		return a + b
	}

	// When the func is run
	tester.Start(sum, 2, 3)

	// Then the relay is shut down
	tester.AssertDoneWithin(time.Second)

	// Then the result is as expected
	tester.AssertReturned(5)
}

func TestRepeatedCalls(t *testing.T) {
	t.Parallel()

	// Given test needs
	relay := NewCallRelay()
	tester := &RelayTester{T: t, Relay: relay} //nolint: exhaustruct // nobody else would be able to fill in private fields
	// Given inputs
	superSum := func(a, b int, deps superSumDeps) int {
		return deps.sum(a, a) +
			deps.sum(b, b) +
			deps.sum(a, b) +
			deps.sum(b, a)
	}
	deps := &superSumTestDeps{relay: relay}

	// When the func is run
	tester.Start(superSum, 2, 3, deps)

	// Then the internal sum is called 4x with different args
	tester.AssertNextCallIs(deps.sum, 2, 2).InjectReturns(4)
	tester.AssertNextCallIs(deps.sum, 3, 3).InjectReturns(6)
	tester.AssertNextCallIs(deps.sum, 2, 3).InjectReturns(5)
	tester.AssertNextCallIs(deps.sum, 3, 2).InjectReturns(5)

	// Then the relay is shut down
	tester.AssertDoneWithin(time.Second)

	// Then the result is as expected
	tester.AssertReturned(20)
}

type superSumDeps interface {
	sum(a, b int) int
}

type superSumTestDeps struct {
	relay *CallRelay
}

func (d *superSumTestDeps) sum(a, b int) int {
	var result int

	d.relay.PutCall(d.sum, a, b).FillReturns(&result)

	return result
}

// TODO: move linting to check for todos after everything else
// TODO: test that start starts the func in a goroutine
// TODO: test that assert done within passes if the goroutine is done
// TODO: test that assert done within fails if the goroutine isn't done
// TODO: test that assert return passes if the return is correct
// TODO: test that assert return fails if the return is wrong
// TODO: test that AssertNextCallIs passes if the call & args match
// TODO: test that AssertNextCallIs fails if the call is wrong
// TODO: test that AssertNextCallIs fails if the args are the wrong type
// TODO: test that AssertNextCallIs fails if the args are the wrong number
// TODO: test that InjectReturns passes if the args are the right type and number
// TODO: test that InjectReturns fails if the args are the wrong type
// TODO: test that InjectReturns fails if the args are the wrong number
// TODO: test that PutCall passes if the args are the right type and number for the call
// TODO: test that PutCall fails if the args are the wrong type for the call
// TODO: test that PutCall fails if the args are the wrong number for the call
// TODO: test that FillReturns passes if the args are the right type and number for the call
// TODO: test that FillReturns fails if the args are the wrong type for the call
// TODO: test that FillReturns fails if the args are the wrong number for the call
