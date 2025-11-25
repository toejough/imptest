package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/toejough/imptest/POC-gen-from-black-box-test/run"
)

//go:generate go run generate.go run.PrintSum
// TODO: allow another arg for generate to name the implementation
// TODO: pull this generate function out into its own package
// TODO: require another arg for generate to name the runner

func Test_PrintSum_Auto(t *testing.T) {
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewPrintSumImp(t)

	// call the function under test
	inputA := 10
	inputB := 32
	imp.Start(inputA, inputB)

	// sum := deps.Add(a, b)
	normalAddResult := inputA + inputB
	imp.ExpectCallTo.Add(inputA, inputB).InjectResult(normalAddResult)

	// formatted := deps.Format(sum)
	normalFormatResult := strings.Itoa(normalAddResult)
	imp.ExpectCallTo.Format(normalAddResult).InjectResult(normalFormatResult)

	// deps.Print(formatted)
	imp.ExpectCallTo.Print(normalFormatResult)

	// return a, b, formatted
	imp.ExpectReturnedValues(inputA, inputB, normalFormatResult)
}

func Test_PrintSum_Manual(t *testing.T) {
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewPrintSumIntOpsImp(t)

	// call the function under test
	inputA := 10
	inputB := 32
	imp.Start(inputA, inputB)

	// sum := deps.Add(a, b)
	normalAddResult := inputA + inputB
	// imp.ExpectCallTo.Add(inputA, inputB).InjectResult(normalAddResult)
	// manually:
	// get the current event
	event := imp.GetCurrentEvent()
	// validate it's a call to a method
	if event.Type() != imptest.CallEvent {
		t.Fatalf("expected CallEvent, got %v", event.Type)
	}
	// get which method
	call := event.AsCall()
	// validate it's Add
	if call.Name() != "Add" {
		t.Fatalf("expected call to Add, got %s", call.Name())
	}
	// get the args
	add := call.AsAdd()
	// validate args are inputA and inputB
	if add.A != inputA || add.B != inputB {
		t.Fatalf("expected args %d, %d; got %d, %d", inputA, inputB, add.A, add.B)
	}
	// inject the result normalAddResult
	add.InjectResult(normalAddResult)
	imp.Resolve(event)
	// it is up to the caller to perform any timeout-based retries in the above, if they want, in order to handle any expected concurrency. GetCurrentEvent will always block until the next event is available, with an optional timeout arg. The default is 1s.

	// formatted := deps.Format(sum)
	normalFormatResult := strings.Itoa(normalAddResult)
	// imp.ExpectCallTo.Format(normalAddResult).InjectResult(normalFormatResult)
	event = imp.GetCurrentEvent()
	if event.Type() != imptest.CallEvent {
		t.Fatalf("expected CallEvent, got %v", event.Type)
	}
	call = event.AsCall()
	if call.Name() != "Format" {
		t.Fatalf("expected call to Format, got %s", call.Name())
	}
	format := call.AsFormat()
	if format.Input != normalAddResult {
		t.Fatalf("expected arg %d; got %d", normalAddResult, format.Input)
	}
	format.InjectResult(normalFormatResult)
	imp.Resolve(event)

	// deps.Print(formatted)
	// imp.ExpectCallTo.Print(normalFormatResult)
	event = imp.GetCurrentEvent()
	if event.Type() != imptest.CallEvent {
		t.Fatalf("expected CallEvent, got %v", event.Type())
	}
	call = event.AsCall()
	if call.Name() != "Print" {
		t.Fatalf("expected call to Print, got %s", call.Name())
	}
	print := call.AsPrint()
	if print.S != normalFormatResult {
		t.Fatalf("expected arg %q; got %q", normalFormatResult, print.S)
	}
	imp.Resolve(event)

	// return a, b, formatted
	imp.ExpectReturnedValues(inputA, inputB, normalFormatResult)
	event = imp.GetCurrentEvent()
	if event.Type() != imptest.ReturnEvent {
		t.Fatalf("expected ReturnEvent, got %v", event.Type())
	}
	ret := event.AsReturn()
	if ret.Ret0 != inputA || ret.Ret1 != inputB || ret.Ret2 != normalFormatResult {
		t.Fatalf("expected returned values %d, %d, %q; got %d, %d, %q",
			inputA, inputB, normalFormatResult,
			ret.Ret0, ret.Ret1, ret.Ret2)
	}
	imp.Resolve(event)

	// verify done manually
	done := imp.IsDone()
	if !done {
		t.Fatalf("expected imp to be done after resolving return event")
	}
}

func Test_PrintSum_Panic(t *testing.T) {
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewPrintSumIntOpsImp(t)

	// call the function under test
	inputA := 10
	inputB := 32
	imp.Start(inputA, inputB)

	// sum := deps.Add(a, b)
	imp.ExpectCallTo.Add(inputA, inputB).InjectPanic("mock panic")

	// panic with message
	imp.ExpectPanicWith("mock panic")
}

func Test_PrintSum_WithDuration(t *testing.T) {
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewPrintSumIntOpsImp(t)
	imp.SetDefaultTimeout(500 * time.Millisecond)

	// call the function under test
	inputA := 10
	inputB := 32
	imp.Start(inputA, inputB)

	// sum := deps.Add(a, b)
	imp.ExpectCallTo.Add(inputA, inputB).InjectPanic("mock panic")

	// panic with message
	imp.ExpectPanicWith("mock panic")
}
