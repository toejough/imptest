package protest_test

import (
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestInjectReturnPasses(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertNextCallIs(tdm.Inject).InjectReturns(5)
	tester.AssertDoneWithin(time.Second)
	tester.AssertReturned(5)

	// Then the test passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

type testDepsInject interface{ Inject() int }

func (tdm *testDepsMock) Inject() int {
	var r int

	tdm.tester.PutCall(tdm.Inject).FillReturns(&r)

	return r
}

func TestInjectReturnWrongTypeFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	call := tester.AssertNextCallIs(tdm.Inject)

	defer expectPanicWith(t, "wrong return type")
	call.InjectReturns("five")
	tester.AssertDoneWithin(time.Second)
}

func TestInjectReturnWrongNumberFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsInject) int {
		return deps.Inject()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	call := tester.AssertNextCallIs(tdm.Inject)

	defer expectPanicWith(t, "wrong number of returns")
	call.InjectReturns(5, "five")
	tester.AssertDoneWithin(time.Second)
}

// func TestPutCallTooFewArgsFails(t *testing.T) {
// 	t.Parallel()
//
// 	// Given test needs
// 	mockedt := newMockedTestingT()
// 	tester := protest.NewTester(mockedt)
// 	// Given inputs
// 	returns := func(deps testDepsPutTooFew) {
// 		// Then the next call fails with too few args
// 		defer expectPanicWith(t, "too few args")
// 		deps.PutTooFew(5, "six")
// 	}
// 	tdm := newTestDepsMock(tester)
//
// 	// When the func is run
// 	tester.Start(returns, tdm)
// 	tester.AssertDoneWithin(time.Second)
// }
//
// type testDepsPutTooFew interface{ PutTooFew(i int, s string) }
//
// func (tdm *testDepsMock) PutTooFew(i int, _ string) { tdm.tester.PutCall(tdm.PutTooFew, i) }

// func TestPutCallTooManyArgsFails(t *testing.T) {
// 	t.Parallel()
//
// 	// Given test needs
// 	mockedt := newMockedTestingT()
// 	tester := protest.NewTester(mockedt)
// 	// Given inputs
// 	returns := func(deps testDepsPutTooMany) {
// 		// Then the next call fails with too many args
// 		defer expectPanicWith(t, "too many args")
// 		deps.PutTooMany(5, "six")
// 	}
// 	tdm := newTestDepsMock(tester)
//
// 	// When the func is run
// 	tester.Start(returns, tdm)
// 	tester.AssertDoneWithin(time.Second)
// }
//
// type testDepsPutTooMany interface{ PutTooMany(i int, s string) }
//
// func (tdm *testDepsMock) PutTooMany(i int, s string) {
// 	tdm.tester.PutCall(tdm.PutTooMany, i, s, "THIS ONE IS TOO MANY")
// }
//
// func TestPutCallWrongTypesFails(t *testing.T) {
// 	t.Parallel()
//
// 	// Given test needs
// 	mockedt := newMockedTestingT()
// 	tester := protest.NewTester(mockedt)
// 	// Given inputs
// 	returns := func(deps testDepsPutWrongTypes) {
// 		// Then the next call fails with too many args
// 		defer expectPanicWith(t, "wrong arg type")
// 		deps.PutWrongTypes(5, "six")
// 	}
// 	tdm := newTestDepsMock(tester)
//
// 	// When the func is run
// 	tester.Start(returns, tdm)
// 	tester.AssertDoneWithin(time.Second)
// }
//
// type testDepsPutWrongTypes interface{ PutWrongTypes(i int, s string) }
//
// func (tdm *testDepsMock) PutWrongTypes(_ int, s string) {
// 	tdm.tester.PutCall(tdm.PutWrongTypes, "THIS ONE IS THE WRONG TYPE", s)
// }

// TODO: test that FillReturns fails if the args are the wrong type for the call
// TODO: test that FillReturns fails if the args are the wrong number for the call
// TODO: test parallel calls
// TODO: refactor protest out into separate files
