# imptest

An IMPure function TEST tool.

There are plenty of test tools written to facilitate testing pure functions: Inputs -> Outputs. 

Impure functions, on the other hand, are characterized by calls to _other_ functions. The whole point of some functions is that they coordinate calls to other functions. 

We often don't want to validate what those other functions _do_, as we already have tests for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we can use integration tests or end-to-end testing. 

This library is here to help where we really do just want to test that the function-under-test makes the calls it's supposed to, in the right order, shuffling inputs and outputs between them correctly.

Let's look at the tests to see how this really works. Ignore the `L2` prefixes, we'll come back to that later.

```go
// TestL2ReceiveCallSendReturn tests matching a dependency call and pushing a return more simply, with a
// dependency struct.
func TestL2ReceiveCallSendReturn(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct1) string {
		return deps.Dep1()
	}
	// and a struct of dependenc mimics
	depsToMimic := depStruct1{} //nolint:exhaustruct
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	defer imp.Close()
	// and a string to return from the dependency call
	returnString := anyString

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)
	// Then the next thing the function under test does is make a call matching our expectations
	// When we push a return string
	imp.ReceiveCall("Dep1").SendReturn(returnString)
	// Then the next thing the function under test does is return values matching our expectations
	imp.ReceiveReturn(returnString)
}

// TestL2ReceiveCallSendPanic tests matching a dependency call and pushing a panic more simply, with a
// dependency struct.
func TestL2ReceiveCallSendPanic(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct1) string {
		return deps.Dep1()
	}
	// and a struct of dependencies to mimic
	depsToMimic := depStruct1{} //nolint:exhaustruct
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	defer imp.Close()
	// and a string to panic from the dependency call
	panicString := anyString

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)

	// Then the next thing the function under test does is make a call matching our expectations
	// When we push a return string
	imp.ReceiveCall("Dep1").SendPanic(panicString)

	// Then the next thing the function under test does is return values matching our expectations
	imp.ReceivePanic(panicString)
}

type pingPongDeps struct {
	Ping, Pong func() bool
}

// TestL2PingPongConcurrently tests using the funcActivityChan with a funcToTest that calls ping-pong dependencies
// concurrently.
func TestL2PingPongConcurrently(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := pingPong
	// and dependencies to mimic
	// ignore exhaustruct - the zero value for pingpong deps is fine
	depsToMimic := pingPongDeps{} //nolint: exhaustruct
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	defer imp.Close()

	// When we run the function to test with the mimicked dependency
	imp.Start(funcToTest, depsToMimic.Ping, depsToMimic.Pong)

	// When we set concurrency to 2
	imp.Concurrently(func() {
		// Then we get 100 calls to ping
		pingCallCount := 0
		for pingCallCount < 100 {
			imp.ReceiveCall("Ping").SendReturn(true)

			pingCallCount++
		}
		// Then we ping once
		imp.ReceiveCall("Ping").SendReturn(true)

		// Then we get 100 more calls to ping
		pingCallCount = 0
		for pingCallCount < 100 {
			imp.ReceiveCall("Ping").SendReturn(true)

			pingCallCount++
		}
	}, func() {
		// Then we get 100 calls to pong
		pongCallCount := 0
		for pongCallCount < 100 {
			imp.ReceiveCall("Pong").SendReturn(true)

			pongCallCount++
		}
		// Then we pong once
		imp.ReceiveCall("Pong").SendReturn(false)

		// Then we get 100 more calls to pong
		pongCallCount = 0
		for pongCallCount < 100 {
			imp.ReceiveCall("Pong").SendReturn(true)

			pongCallCount++
		}
	})

	// Then the next activity from the function under test is its return
	imp.ReceiveReturn()
}
```

TODO: ADD SOMETHING ABOUT THE ACTUAL PING PONG FUNCTION BEING TESTED

TODO: Unsurprisingly, concurrency is hard, and the concurrency checks are currently where the flaky test and mutation
test failures are occurring. I'm considering a full rewrite of the way I validate concurrency. Again. :lolsob:

## Customization

TODO: TALK ABOUT L1 API HERE

## alternatives/inspirations
Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?

https://github.com/stretchr/testify/issues/741, highlights some challenges, and is answered by the author with some additional syntax and functionality. I still found myself wondering if something with what I considered simpler syntax was possible, mostly out of curiosity, and to learn.
