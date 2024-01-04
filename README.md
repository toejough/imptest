# imptest

An IMPure function TEST tool.

The general idea is to allow a test to be written _just_ for what we're testing. 

Hopefully that sounds obvious, but I find that most test suites are written assuming pure functions: Inputs -> Outputs. 

Impure functions, on the other hand, are characterized by calls to _other_ functions. The whole point, or the _only_ point, of some functions, is that they coordinate calls to other functions. We often don't want to validate what those other functions _do_, as we already have tests for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we will often use integration tests or end-to-end testing. But that leaves a hole where we really do just want to test that the function-under-test makes the calls it's supposed to, in the right order, shuffling inputs and outputs between them correctly.

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
	superSum := func(deps superSumDeps, a, b int) int {
		return deps.sum(a, a) +
			deps.sum(b, b) +
			deps.sum(a, b) +
			deps.sum(b, a)
	}
	deps := &superSumTestDeps{relay: relay}

	// When the func is run
	tester.Start(superSum, deps, 2, 3)

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
# alternatives/inspirations
Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?

In the straightforward use cases, you only get to specify simple call/return behavior, with no guarantees about ordering, and you need to unset handlers for repeated calls for the same function.

On the other hand, there's https://github.com/stretchr/testify/issues/741, which calls some of this out, and which is answered by the author with some additional syntax and functionality. I still found this somewhat confusing, overly verbose, and non-obvious that the functionality even existed, so I set out to see if I could do any better, mostly out of curiosity, and to learn. Having done so, I'm happy with what I came up with.
