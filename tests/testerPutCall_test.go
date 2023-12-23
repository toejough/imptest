package protest_test

import (
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestPutCallTooFewArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutTooFew) {
		// Then the next call fails with too few args
		defer expectPanicWith(t, "too few args")
		deps.PutTooFew(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertDoneWithin(time.Second)
}

type testDepsPutTooFew interface{ PutTooFew(i int, s string) }

func (tdm *testDepsMock) PutTooFew(i int, _ string) { tdm.tester.PutCall(tdm.PutTooFew, i) }

func TestPutCallTooManyArgsFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutTooMany) {
		// Then the next call fails with too many args
		defer expectPanicWith(t, "too many args")
		deps.PutTooMany(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertDoneWithin(time.Second)
}

type testDepsPutTooMany interface{ PutTooMany(i int, s string) }

func (tdm *testDepsMock) PutTooMany(i int, s string) {
	tdm.tester.PutCall(tdm.PutTooMany, i, s, "THIS ONE IS TOO MANY")
}

func TestPutCallWrongTypesFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsPutWrongTypes) {
		// Then the next call fails with too many args
		defer expectPanicWith(t, "wrong arg type")
		deps.PutWrongTypes(5, "six")
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertDoneWithin(time.Second)
}

type testDepsPutWrongTypes interface{ PutWrongTypes(i int, s string) }

func (tdm *testDepsMock) PutWrongTypes(_ int, s string) {
	tdm.tester.PutCall(tdm.PutWrongTypes, "THIS ONE IS THE WRONG TYPE", s)
}

// TODO: test that InjectReturns passes if the args are the right type and number
// TODO: test that InjectReturns fails if the args are the wrong type
// TODO: test that InjectReturns fails if the args are the wrong number
// TODO: test that FillReturns passes if the args are the right type and number for the call
// TODO: test that FillReturns fails if the args are the wrong type for the call
// TODO: test that FillReturns fails if the args are the wrong number for the call
// TODO: test parallel calls
// TODO: refactor protest out into separate files
