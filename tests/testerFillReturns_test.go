package protest_test

import (
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestFillReturnWrongTypeFails(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func(deps testDepsFillWrongType) int {
		// Then test fails with wrong return type
		defer expectPanicWith(t, "wrong return type")
		return deps.FillWrongType()
	}
	tdm := newTestDepsMock(tester)

	// When the func is run
	tester.Start(returns, tdm)
	tester.AssertNextCallIs(tdm.FillWrongType).InjectReturns(5)
	tester.AssertDoneWithin(time.Second)
}

type testDepsFillWrongType interface{ FillWrongType() int }

func (tdm *testDepsMock) FillWrongType() int {
	var r int
	var badR string

	tdm.tester.PutCall(tdm.FillWrongType).FillReturns(&badR)

	return r
}

// TODO: test that FillReturns fails if the args are the wrong number for the call
// TODO: test parallel calls
// TODO: refactor protest out into separate files
// TODO: rename to imptest
