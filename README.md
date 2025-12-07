# imptest

An IMPure function TEST tool.

There are plenty of test tools written to facilitate testing pure functions: Inputs -> Outputs.

Impure functions, on the other hand, are characterized by calls to _other_ functions. The whole point of some (_most_?) functions is that they coordinate calls to other functions.

We often don't want to validate what those other functions _do_, as we already have tests for them, or they're 3rd parties that we trust. If we _do_ care about the end-to-end functionality, we can use integration tests or end-to-end testing.

This library is here to help where we really do just want to test that the function-under-test makes the calls it's supposed to, in the right order, shuffling inputs and outputs between them correctly.

Let's look at the tests to see how this really works.

```go
package run_test

import (
	"testing"

	"github.com/toejough/imptest"
	"github.com/toejough/imptest/UAT/run"
)

//go:generate go run ../../generator/main.go run.IntOps --name IntOpsImp

func Test_PrintSum_Auto(t *testing.T) {
	t.Parallel()
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// Given: the generated implementation of IntOps
	imp := NewIntOpsImp(t)

	// When: the function under test is started with some args and the mocked dependencies...
	inputA := 10
	inputB := 32
	printSumImp := imptest.Start(t, run.PrintSum, inputA, inputB, imp.Mock)

	// Then: we expect the calls to the methods of IntOps to be made in the correct order and with the correct arguments
	// sum := deps.Add(a, b)
	normalAddResult := inputA + inputB
	imp.ExpectCallTo.Add(inputA, inputB).InjectResult(normalAddResult)
	// formatted := deps.Format(sum)
	normalFormatResult := "42"
	imp.ExpectCallTo.Format(normalAddResult).InjectResult(normalFormatResult)
	// deps.Print(formatted)
	imp.ExpectCallTo.Print(normalFormatResult).Resolve()

	// Then: we expect the function under test to return the correct values
	// return a, b, formatted
	printSumImp.ExpectReturnedValues(inputA, inputB, normalFormatResult)
}
```

## What's going on here?

The basic idea is to find a way to treat impure function call activity as pure input/output data, thereby allowing us to write fast, repeatable tests for it, just like we do for pure functions.

This library:
* generates a mock implementation of a dependency interface (go:generate ...)
* starts the function under test in a goroutine (imptest.Start...)
* intercepts dependency calls, and pushes them onto a channel
* provides compile-time-safe methods for validating those calls and args (imp.ExpectCallTo)
* provides a way for the test to _interactively_ inject the result (.InjectResult...)
* intercepts the function under test's response (either returns or panics) and validate those (.ExpectReturnedValues)

It's this call & response via channels that is one of the key distinguishing features of this library vs other mock libraries - each test instruction is interacting synchronously with the function under test. If you want to make sure calls happen in a certain order, you check for them in that order. If you want to know what happens if a dependency returns true 30 times and then false the 31st time, you can control that in the test code itself, rather than having to write yet another mock implementation.

Oops, buried the lede there - that's one of the best things about this library: at _most_ you have to write a single mock for the dependencies, and you don't even have to implement any logic - all Imptest needs is the interface!

## API's 

TBD

## alternatives/inspirations
Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?

https://github.com/stretchr/testify/issues/741, highlights some challenges, and is answered by the author with some additional syntax and functionality. I still found myself wondering if something with what I considered simpler syntax was possible, mostly out of curiosity, and to learn. 

A couple (gulp) _years_ later, I'm pretty happy with what I've come up with. I hope you will be, too!
