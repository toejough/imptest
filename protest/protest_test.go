package protest_test

import (
	"testing"
	"time"

	"github.com/toejough/protest/protest"
)

func TestStartRunsFUTInGoroutine(t *testing.T) {
	t.Parallel()

	// Given test needs
	relay := protest.NewCallRelay()
	tester := &protest.RelayTester{T: t, Relay: relay}
	// TODO: just make a NewTester(t) func.
	// Given inputs
	lockchan := make(chan struct{})
	waitchan := make(chan struct{})
	wait := func() {
		<-lockchan
	}

	// release the lock at the end of the test
	defer close(lockchan)

	// When the func is run
	go func() {
		tester.Start(wait)
		close(waitchan)
	}()

	// Then the return from waitchan should be immediate
	select {
	case <-waitchan:
	case <-time.After(time.Second):
		t.Error("waitchan never closed, indicating function was run synchronously instead of in a goroutine.")
	}
}

func TestStartFailsCleanlyWithWrongNumArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)

	// Given FUT
	argFunc := func(_, _, _ int) {}

	// When the func is run with the wrong number of args
	tester.Start(argFunc)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal("The test should've failed due to wrong num args, but it didn't")
	}
}

// MockedTestingT
func newMockedTestingT() *mockedTestingT { return &mockedTestingT{} }

type mockedTestingT struct{ failed bool }

func (mt *mockedTestingT) Fatalf(message string, args ...any) {}
func (mt *mockedTestingT) Helper()                            {}
func (mt *mockedTestingT) Failed() bool                       { return mt.failed }

// func TestRepeatedCalls(t *testing.T) {
// 	t.Parallel()
//
// 	// Given test needs
// 	relay := NewCallRelay()
// 	tester := &RelayTester{T: t, Relay: relay} //nolint: exhaustruct
// 	// nobody else would be able to fill in private fields
// 	// Given inputs
// 	superSum := func(a, b int, deps superSumDeps) int {
// 		return deps.sum(a, a) +
// 			deps.sum(b, b) +
// 			deps.sum(a, b) +
// 			deps.sum(b, a)
// 	}
// 	deps := &superSumTestDeps{relay: relay}
//
// 	// When the func is run
// 	tester.Start(superSum, 2, 3, deps)
//
// 	// Then the internal sum is called 4x with different args
// 	tester.AssertNextCallIs(deps.sum, 2, 2).InjectReturns(4)
// 	tester.AssertNextCallIs(deps.sum, 3, 3).InjectReturns(6)
// 	tester.AssertNextCallIs(deps.sum, 2, 3).InjectReturns(5)
// 	tester.AssertNextCallIs(deps.sum, 3, 2).InjectReturns(5)
//
// 	// Then the relay is shut down
// 	tester.AssertDoneWithin(time.Second)
//
// 	// Then the result is as expected
// 	tester.AssertReturned(20)
// }
//
// type superSumDeps interface {
// 	sum(a, b int) int
// }
//
// type superSumTestDeps struct {
// 	relay *CallRelay
// }
//
// func (d *superSumTestDeps) sum(a, b int) int {
// 	var result int
//
// 	d.relay.PutCall(d.sum, a, b).FillReturns(&result)
//
// 	return result
// }

// TODO: test that start starts the func in a goroutine
// TODO: test that assert done within passes if the goroutine is done
// TODO: test that assert done within fails if the goroutine isn't done
// TODO: test that assert return passes if the return is correct
// TODO: test that assert return fails if the return is wrong
// TODO: test that AssertNextCallIs passes if the call & args match
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
