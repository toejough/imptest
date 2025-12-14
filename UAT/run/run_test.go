package run_test

import (
	"testing"

	"github.com/toejough/imptest/UAT/run"
)

//go:generate go run ../../impgen/main.go run.IntOps --name IntOpsImp
//go:generate go run ../../impgen/main.go run.PrintSum --name PrintSumImp --call

const (
	addMethod       = "Add"
	formatMethod    = "Format"
	printMethod     = "Print"
	returnEventType = "ReturnEvent"
	panicEventType  = "PanicEvent"
)

func Test_PrintSum_Auto(t *testing.T) {
	t.Parallel()
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// Given: the generated implementation of IntOps
	imp := NewIntOpsImp(t)

	// When: the function under test is started with some args and the mocked dependencies...
	inputA := 10
	inputB := 32
	printSumImp := NewPrintSumImp(t, run.PrintSum).Start(inputA, inputB, imp.Mock)

	// Then: expect add to be called with a & b
	// When: we return normalAddResult
	normalAddResult := inputA + inputB
	imp.ExpectCallTo.Add(inputA, inputB).InjectResult(normalAddResult)
	// Then: expect format to be called with normalAddResult
	// When: we return normalFormatResult
	normalFormatResult := "42"
	imp.ExpectCallTo.Format(normalAddResult).InjectResult(normalFormatResult)
	// Then: expect print to be called with normalFormatResult
	// When: we resolve the print call
	imp.ExpectCallTo.Print(normalFormatResult).Resolve()

	// Then: expect the function under test to return inputA, inputB, normalFormatResult
	printSumImp.ExpectReturnedValues(inputA, inputB, normalFormatResult)
}

func Test_PrintSum_Manual(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()
	// disabling the cyclop linter because this is not actually that complex, but also
	// because whatever complexity is here is here by design, to demonstrate how to manually
	// validate imptest events

	// we want to validate that run.PrintSum calls the methods of IntOps correctly
	// get the generated implementation of IntOps
	imp := NewIntOpsImp(t)

	// call the function under test
	inputA := 10
	inputB := 32
	printSumImp := NewPrintSumImp(t, run.PrintSum).Start(inputA, inputB, imp.Mock)

	// sum := deps.Add(a, b)
	normalAddResult := inputA + inputB
	// imp.ExpectCallTo.Add(inputA, inputB).InjectResult(normalAddResult)
	// manually:
	// get the current call
	call := imp.GetCurrentCall()
	// validate it's Add
	if call.Name() != addMethod {
		t.Fatalf("expected call to Add, got %s", call.Name())
	}
	// get the args
	add := call.AsAdd()
	// validate args are inputA and inputB
	if add.a != inputA || add.b != inputB {
		t.Fatalf(
			"expected args %d, %d; got %d, %d",
			inputA, inputB, add.a, add.b,
		)
	}
	// inject the result normalAddResult
	add.InjectResult(normalAddResult)
	// it is up to the caller to perform any timeout-based retries in the above, if they want, in order to handle any
	// expected concurrency. GetCurrentEvent will always block until the next event is available, with an optional
	// timeout arg. The default is 1s.

	// formatted := deps.Format(sum)
	normalFormatResult := "42"
	// imp.ExpectCallTo.Format(normalAddResult).InjectResult(normalFormatResult)
	// get the current call
	call = imp.GetCurrentCall()
	if call.Name() != formatMethod {
		t.Fatalf("expected call to Format, got %s", call.Name())
	}

	format := call.AsFormat()
	if format.i != normalAddResult {
		t.Fatalf("expected arg %d; got %d", normalAddResult, format.i)
	}

	format.InjectResult(normalFormatResult)

	// deps.Print(formatted)
	// imp.ExpectCallTo.Print(normalFormatResult)
	// get the current call
	call = imp.GetCurrentCall()
	if call.Name() != printMethod {
		t.Fatalf("expected call to Print, got %s", call.Name())
	}

	printCall := call.AsPrint()
	if printCall.s != normalFormatResult {
		t.Fatalf("expected arg %q; got %q", normalFormatResult, printCall.s)
	}

	printCall.Resolve() // there's no return value to inject, but we do need to mark this event as resolved

	// return a, b, formatted
	// printSumImp.ExpectReturnedValues(inputA, inputB, normalFormatResult)
	response := printSumImp.GetResponse()
	if response.Type() != returnEventType {
		t.Fatalf("expected ReturnEvent, got %v", response.Type())
	}

	ret := response.AsReturn()
	if len(ret) != 3 {
		t.Fatalf("expected 3 returned values, got %d", len(ret))
	}

	if ret[0] != inputA || ret[1] != inputB || ret[2] != normalFormatResult {
		t.Fatalf("expected returned values %d, %d, %q; got %v, %v, %v",
			inputA, inputB, normalFormatResult,
			ret[0], ret[1], ret[2])
	}
}

func Test_PrintSum_Panic(t *testing.T) {
	t.Parallel()
	// we want to validate that run.PrintSum calls the methods of IntOps correctly
	// get the generated implementation of IntOps
	imp := NewIntOpsImp(t) // if passed multiple in the generate call, this should return multiple imps in the same order

	// call the function under test
	inputA := 10
	inputB := 32
	printSumImp := NewPrintSumImp(t, run.PrintSum).Start(inputA, inputB, imp.Mock)

	// sum := deps.Add(a, b)
	imp.ExpectCallTo.Add(inputA, inputB).InjectPanic("mock panic")

	// panic with message
	printSumImp.ExpectPanicWith("mock panic")
}

func Test_PrintSum_WithDuration(t *testing.T) {
	t.Parallel()
	// we want to validate that run.PrintSum calls the methods of IntOps correctly
	// get the generated implementation of IntOps
	imp := NewIntOpsImp(t) // if passed multiple in the generate call, this should return multiple imps in the same order

	// call the function under test
	inputA := 10
	inputB := 32
	printSumImp := NewPrintSumImp(t, run.PrintSum).Start(inputA, inputB, imp.Mock)

	// sum := deps.Add(a, b)
	imp.ExpectCallTo.Add(inputA, inputB).InjectPanic("mock panic")

	// panic with message
	printSumImp.ExpectPanicWith("mock panic")
}
