package protest_test

import (
	"strings"
	"testing"
	"time"

	protest "github.com/toejough/protest/api"
)

func TestAssertReturnPassesWithCorrectValue(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func() int {
		return 5
	}

	// When the func is run
	tester.Start(returns)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// And we expect it to return the right value
	tester.AssertReturned(5)

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertReturnPassesWithCorrectValues(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func() (int, string) {
		return 5, "five" //nolint: goconst // similarity across functions is a detail, not a feature to be abstracted
	}

	// When the func is run
	tester.Start(returns)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// And we expect it to return the right value
	tester.AssertReturned(5, "five")

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertReturnPassesWithNoValues(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func() {}

	// When the func is run
	tester.Start(returns)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// And we expect it to return no value
	tester.AssertReturned()

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertReturnFailsWithTooFewReturns(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func() (int, string) {
		return 5, "five"
	}

	// When the func is run
	tester.Start(returns)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// And we expect it to return the right value
	tester.AssertReturned(5)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal(
			"The test should've failed with too few returns. Instead the test passed!",
		)
	}
	// Then the error calls out too few
	if !strings.Contains(mockedt.Failure(), "too few") {
		t.Fatalf(
			"The test should've failed with too few returns. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertReturnFailsWithTooManyReturns(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func() (int, string) {
		return 5, "five"
	}

	// When the func is run
	tester.Start(returns)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// And we expect it to return the right value
	tester.AssertReturned(5, "five", 0x5)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal(
			"The test should've failed with too many returns. Instead the test passed!",
		)
	}
	// Then the error calls out too many
	if !strings.Contains(mockedt.Failure(), "too many") {
		t.Fatalf(
			"The test should've failed with too many returns. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertReturnFailsWithWrongTypes(t *testing.T) {
	t.Parallel()

	// Given test needs
	mockedt := newMockedTestingT()
	tester := protest.NewTester(mockedt)
	// Given inputs
	returns := func() int {
		return 5
	}

	// When the func is run
	tester.Start(returns)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// And we expect it to return the right value
	tester.AssertReturned("five")

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal(
			"The test should've failed with wrong types. Instead the test passed!",
		)
	}
	// Then the error calls out wrong type
	if !strings.Contains(mockedt.Failure(), "wrong type") {
		t.Fatalf(
			// FIXME: the failure messages should assume that the test expectations are correct. Right now they assume that the function under test is correct and the test is wrong.
			"The test should've failed with wrong types returned. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

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
// TODO: refactor protest out into separate files
