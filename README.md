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
import (
	"spacer/dev/protest"
	"testing"
	"time"
)

type mockRunDeps struct {
	relay *protest.CallRelay
}

func (rd *mockRunDeps) pretest() bool {
	var b bool

	rd.relay.PutCall(rd.pretest).FillReturns(&b)

	return b
}

func (rd *mockRunDeps) testMutations() bool {
	var success bool

	rd.relay.PutCall(rd.testMutations).FillReturns(&success)

	return success
}

func (rd *mockRunDeps) exit(code int) {
	rd.relay.PutCallNoReturn(rd.exit, code)
}

func TestRunHappyPath(t *testing.T) {
	t.Parallel()

	// Given test needs
	relay := protest.NewCallRelay()
	tester := &protest.RelayTester{T: t, Relay: relay}
	// Given inputs
	deps := &mockRunDeps{relay: relay}

	// When the func is run
	tester.Start(run, deps)

	// Then the pretest is run
	tester.AssertNextCallIs(deps.pretest).InjectReturns(true)
	// Then the mutation testing is run
	tester.AssertNextCallIs(deps.testMutations).InjectReturns(true)
	// Then the program exits with 0
	tester.AssertNextCallIs(deps.exit, 0)

	// Then the relay is shut down
	tester.AssertDoneWithin(time.Second)
}
```
