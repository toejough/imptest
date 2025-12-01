package main

import (
	"strings"
	"testing"

	"github.com/toejough/imptest/POC-imptest-API/UAT/run"
	"github.com/toejough/imptest/POC-imptest-API/imptest"
)

//go:generate go run ../imptest/main.go run.IntOps --name IntOpsImp
// TODO: pull this generate function out into its own package
// TODO: allow another arg for generate to name the runner
// TODO: allow a function imp, which just allows static compile-time checking of args and return values

func Test_PrintSum_Auto(t *testing.T) {
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewIntOpsImp(t) // if passed multiple in the generate call, this should return multiple imps in the same order

	// call the function under test
	inputA := 10
	inputB := 32
	printSumImp := imptest.Start(run.PrintSum, inputA, inputB, imp.Mock)

	// sum := deps.Add(a, b)
	normalAddResult := inputA + inputB
	imp.ExpectCallTo.Add(inputA, inputB).InjectResult(normalAddResult)

	// formatted := deps.Format(sum)
	normalFormatResult := strings.Itoa(normalAddResult)
	imp.ExpectCallTo.Format(normalAddResult).InjectResult(normalFormatResult)

	// deps.Print(formatted)
	imp.ExpectCallTo.Print(normalFormatResult)

	// return a, b, formatted
	printSumImp.ExpectReturnedValues(inputA, inputB, normalFormatResult)
}

func Test_PrintSum_Manual(t *testing.T) {
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewIntOpsImp(t)

	// call the function under test
	inputA := 10
	inputB := 32
	printSumImp := imptest.Start(run.PrintSum, inputA, inputB, imp.Mock)

	// sum := deps.Add(a, b)
	normalAddResult := inputA + inputB
	// imp.ExpectCallTo.Add(inputA, inputB).InjectResult(normalAddResult)
	// manually:
	// get the current call
	call := imp.GetCurrentCall()
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
	// it is up to the caller to perform any timeout-based retries in the above, if they want, in order to handle any expected concurrency. GetCurrentEvent will always block until the next event is available, with an optional timeout arg. The default is 1s.

	// formatted := deps.Format(sum)
	normalFormatResult := strings.Itoa(normalAddResult)
	// imp.ExpectCallTo.Format(normalAddResult).InjectResult(normalFormatResult)
	// get the current call
	call = imp.GetCurrentCall()
	if call.Name() != "Format" {
		t.Fatalf("expected call to Format, got %s", call.Name())
	}
	format := call.AsFormat()
	if format.Input != normalAddResult {
		t.Fatalf("expected arg %d; got %d", normalAddResult, format.Input)
	}
	format.InjectResult(normalFormatResult)

	// deps.Print(formatted)
	// imp.ExpectCallTo.Print(normalFormatResult)
	// get the current call
	call = imp.GetCurrentCall()
	if call.Name() != "Print" {
		t.Fatalf("expected call to Print, got %s", call.Name())
	}
	print := call.AsPrint()
	if print.S != normalFormatResult {
		t.Fatalf("expected arg %q; got %q", normalFormatResult, print.S)
	}
	print.Resolve() // there's no return value to inject, but we do need to mark this event as resolved

	// return a, b, formatted
	// printSumImp.ExpectReturnedValues(inputA, inputB, normalFormatResult)
	response := printSumImp.GetResponse()
	if response.Type() != imptest.ReturnEvent {
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
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewIntOpsImp(t) // if passed multiple in the generate call, this should return multiple imps in the same order

	// call the function under test
	inputA := 10
	inputB := 32
	printSumImp := imptest.Start(run.PrintSum, inputA, inputB, imp.Mock)

	// sum := deps.Add(a, b)
	imp.ExpectCallTo.Add(inputA, inputB).InjectPanic("mock panic")

	// panic with message
	printSumImp.ExpectPanicWith("mock panic")
}

func Test_PrintSum_WithDuration(t *testing.T) {
	// we want to validate that run.PrintSum calls the methods of IntOps correctly

	// get the generated implementation of IntOps
	imp := NewIntOpsImp(t) // if passed multiple in the generate call, this should return multiple imps in the same order

	// call the function under test
	inputA := 10
	inputB := 32
	printSumImp := imptest.Start(run.PrintSum, inputA, inputB, imp.Mock)

	// sum := deps.Add(a, b)
	imp.ExpectCallTo.Add(inputA, inputB).InjectPanic("mock panic")

	// panic with message
	printSumImp.ExpectPanicWith("mock panic")
}
func Test_MultiMulti(t *testing.T) {
	// we want to validate a complex scenario with multiple functions and multiple imps and concurrency

	// Test a ping pong match between two players. Each player randomly hits or misses the ball.
	// gameTracker := new GameTracker(randomizer)
	// ping := NewPlayer("Ping", gameTracker, randomizer)
	// pong := NewPlayer("Pong", gameTracker, randomizer)
	// go ping.Play()
	// go pong.Play()
	// expect ping to tell gameTracker it is ready
	// expect pong to tell gameTracker it is ready
	// expect gameTracker to start the game
	// expect gameTracker to pick a player to serve
	// expect that player to serve through gameTracker
	// expect the other player to receive through gameTracker
	// expect the other player to randomly hit or miss
	// now expect a series of hits and misses until one player wins
	// expect gameTracker to announce the winner
}
