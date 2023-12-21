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
		return 5, "five"
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
	returns := func() int {
		return 5
	}

	// When the func is run
	tester.Start(returns)

	// And we wait for it to finish
	tester.AssertDoneWithin(time.Second)

	// And we expect it to return the right value
	defer expectPanicWith(t, "too few return values asserted")
	tester.AssertReturned(5, "five")
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
	tester.AssertReturned(5)

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
			"The test should've failed with wrong types returned. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestAssertReturnFailsWithWrongValues(t *testing.T) {
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
	tester.AssertReturned(6)

	// Then the test is marked as failed
	if !mockedt.Failed() {
		t.Fatal(
			"The test should've failed with wrong values. Instead the test passed!",
		)
	}
	// Then the error calls out wrong value
	if !strings.Contains(mockedt.Failure(), "wrong value") {
		t.Fatalf(
			"The test should've failed with wrong value returned. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}
