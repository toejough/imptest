# protest

A PROcedure-focused-TEST tool.

In some circles, a "procedure" is side-effect-ful, whereas a "function" does not. In other words, a procedure
is an "impure" function. I could also have called this "imptest", then, but "protest" is a real word and didn't seem
obviously taken.

Of course, if I'm doing SEO someday, I should probably take the option that isn't a real word. hmmm...

The general idea is to allow a test to be written _just_ for what we're testing. That sounds obvious, but I find
that most test suites are written assuming pure functions. Inputs -> Outputs. Procedures, or impure functions, on
the other hand, are characterized by calls to _other_ functions. The whole point, or the only point, of some functions,
is that they call others. We often don't want to validate what those other functions _do_, as we already have tests
for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we will often
use integration tests or end-to-end testing. But that leaves a hole where we really do just wan to test that the
function-under-test makes the calls it's supposed to with the given inputs.

That's what this library is about.

Example of use:
```go
func TestRepeatedCalls(t *testing.T) {
	t.Parallel()

	// Given test needs
	relay := NewCallRelay()
	tester := &RelayTester{T: t, Relay: relay} //nolint: exhaustruct
	// nobody else would be able to fill in private fields
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
```
