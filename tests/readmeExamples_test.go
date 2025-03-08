package imptest_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/toejough/imptest/imptest"
)

// Things Imptest does:
// track activity from function under test
// match activity to expectations
// respond to activity
// support concurrency

// Level 1:
// track activity from function under test: wrap dependency funcs to track their calls, manually track return/panic
// match activity to expectations: receive activity from chan, manually check type, args
// respond to activity: manually send response type & args to chan
// support concurrency: manually send activity back to chan if not the one we wanted

// Level 2:
// track activity from function under test: wrap dep structs of funcs to track their calls, auto track return/panic
// match activity to expectations: receive activity & check type, args via simple sugar funcs
// respond to activity: send response type & args via simple sugar funcs
// support concurrency: auto track and compare expectations to activity

// Level 3 (not implemented yet):
// track activity from function under test: generate dep structs of funcs to track their calls
// match activity to expectations: complex matchers?
// respond to activity: ???
// support concurrency: ???

// ===L1 Tests===.
const anyString = "literally anything"

// TestL1ReceiveDependencyCallSendReturn tests receiving a dependency call and sending a return.
// ignore the linter error about how this test is too long.he point behind the L2 API.
func TestL1ReceiveDependencyCallSendReturn(t *testing.T) { //nolint:funlen
	t.Parallel()

	// Given a function to test
	funcToTest := func(dependency func() string) string {
		return dependency()
	}
	// and a dependency to mimic
	depToMimic := func() string { panic("not implemented") }
	// and a channel to put function activity onto
	funcActivityChan := make(chan imptest.FuncActivity)
	defer close(funcActivityChan)
	// and a dependency mimic that pushes its calls onto the func activity chan
	depMimic, depMimicID := imptest.MimicDependency(t, depToMimic, funcActivityChan)
	// and a string to return from the dependency call
	returnString := anyString

	// When we run the function to test with the mimicked dependency
	go func() {
		// call the function to test
		returnVal := funcToTest(depMimic)
		// record what the func returns as its final activity
		funcActivityChan <- imptest.FuncActivity{
			Type:       imptest.ReturnActivityType,
			PanicVal:   nil,
			ReturnVals: []any{returnVal},
			Call:       nil,
		}
	}()

	// Then the first activity in the funcActivitychannel is a dependency call
	activity1 := <-funcActivityChan
	if activity1.Type != imptest.CallActivityType {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if activity1.Call.ID != depMimicID {
		t.Fail()
	}

	// When we push a return string
	activity1.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.ReturnResponseType,
		ReturnValues: []any{returnString},
		PanicValue:   nil,
	}

	// Then the next activity from the function under test is its return
	activity2 := <-funcActivityChan
	if activity2.Type != imptest.ReturnActivityType {
		t.Fail()
	}

	// and the returned data is only one item
	if len(activity2.ReturnVals) != 1 {
		t.Fail()
	}

	// and the returned value is the returnString
	returnedString, ok := activity2.ReturnVals[0].(string)
	if !ok || returnedString != returnString {
		t.Fail()
	}
}

func namedDependencyFunc() string { panic("not implemented") }

// TestMimicCallID verifies the call ID is the func name.
func TestL1MimicCallID(t *testing.T) {
	t.Parallel()

	// Given a channel to put function activity onto
	funcActivityChan := make(chan imptest.FuncActivity)
	defer close(funcActivityChan)
	// and an expected func name
	expectedName := "github.com/toejough/imptest/tests_test.namedDependencyFunc"

	// When the dependency is mimicked
	_, depMimicID := imptest.MimicDependency(t, namedDependencyFunc, funcActivityChan)

	// Then the func ID is the function name
	if depMimicID != expectedName {
		t.Fatalf("expected the mimic ID to be %s but instead it was %s", expectedName, depMimicID)
	}
}

// TestMimicCallIDOverrideOption verifies the call ID is the overridden name.
func TestL1MimicCallIDOverrideOption(t *testing.T) {
	t.Parallel()

	// Given a channel to put function activity onto
	funcActivityChan := make(chan imptest.FuncActivity)
	defer close(funcActivityChan)
	// and an expected func name
	expectedName := "overriddenName"

	// When the dependency is mimicked
	_, depMimicID := imptest.MimicDependency(
		t,
		namedDependencyFunc,
		funcActivityChan,
		imptest.WithName(expectedName),
	)

	// Then the func ID is the function name
	if depMimicID != expectedName {
		t.Fatalf("expected the mimic ID to be %s but instead it was %s", expectedName, depMimicID)
	}
}

// TestReceiveDependencyCallSendPanic tests receiving a dependency call and sending a panic.
// ignore the linter error about how this test is too longthe point behind the L2 API.
func TestL1ReceiveDependencyCallSendPanic(t *testing.T) { //nolint:funlen
	t.Parallel()

	// Given a function to test
	funcToTest := func(dependency func() string) string {
		return dependency()
	}
	// and a dependency to mimic
	depToMimic := func() string { panic("not implemented") }
	// and a channel to put function activity onto
	funcActivityChan := make(chan imptest.FuncActivity)
	defer close(funcActivityChan)
	// and a dependency mimic that pushes its calls onto the func activity chan
	depMimic, depMimicID := imptest.MimicDependency(t, depToMimic, funcActivityChan)
	// and a string to panic from the dependency call
	panicString := anyString

	// When we run the function to test with the mimicked dependency
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// record what the func panicked as its final activity
				funcActivityChan <- imptest.FuncActivity{
					Type:       imptest.PanicActivityType,
					PanicVal:   r,
					ReturnVals: nil,
					Call:       nil,
				}
			}
		}()
		// call the function to test
		funcToTest(depMimic)
	}()

	// Then the first activity in the funcActivitychannel is a dependency call
	activity1 := <-funcActivityChan
	if activity1.Type != imptest.CallActivityType {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if activity1.Call.ID != depMimicID {
		t.Fail()
	}

	// When we push a panic string
	activity1.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.PanicResponseType,
		ReturnValues: nil,
		PanicValue:   panicString,
	}

	// Then the next activity from the function under test is its panic
	activity2 := <-funcActivityChan
	if activity2.Type != imptest.PanicActivityType {
		t.Fail()
	}

	// and the panicked value is the panicString
	panickedString, ok := activity2.PanicVal.(string)
	if !ok || panickedString != panicString {
		t.Fail()
	}
}

// TestL1PingPongConcurrency tests using the funcActivityChan with a funcToTest that calls ping-pong dependencies
// concurrently.
// ignore the linter error about how this test is too long. It's kind of the point behind the L2 API.
// ignore the linter error about how this test is too complex. It's kind of the point behind the L2 API.
func TestL1PingPongConcurrency(t *testing.T) { //nolint:funlen,cyclop
	t.Parallel()

	// Given a function to test
	funcToTest := pingPong
	// and dependencies to mimic
	pingToMimic := func() bool { panic("not implemented") }
	pongToMimic := func() bool { panic("not implemented") }
	// and a channel to put function activity onto
	// give it a buffer of 2 - we expect to need to put 2 goroutine's worth of activity onto it before reading.
	funcActivityChan := make(chan imptest.FuncActivity, 2)
	defer close(funcActivityChan)
	// and dependency mimics that push their calls onto the func activity chan
	pingMimic, pingMimicID := imptest.MimicDependency(
		t,
		pingToMimic,
		funcActivityChan,
		imptest.WithName("ping"),
	)
	pongMimic, pongMimicID := imptest.MimicDependency(
		t,
		pongToMimic,
		funcActivityChan,
		imptest.WithName("pong"),
	)

	// When we run the function to test with the mimicked dependency
	go func() {
		// call the function to test
		funcToTest(pingMimic, pongMimic)
		// record that the func returned as its final activity
		funcActivityChan <- imptest.FuncActivity{
			Type:       imptest.ReturnActivityType,
			PanicVal:   nil,
			ReturnVals: nil,
			Call:       nil,
		}
	}()

	// Then we get 100 calls to ping
	pingCallCount := 0
	for pingCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.CallActivityType {
			t.Fail()
		}

		// and the dependency call is to the mimicked dependency
		if activity.Call.ID != pingMimicID {
			// if not, push it back on the queue and try again
			funcActivityChan <- activity
			continue
		}

		pingCallCount++

		// When we push a return
		activity.Call.ResponseChan <- imptest.CallResponse{
			Type:         imptest.ReturnResponseType,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then we get 100 calls to pong
	pongCallCount := 0
	for pongCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.CallActivityType {
			t.Fail()
		}

		// and the dependency call is to the mimicked dependency
		if activity.Call.ID != pongMimicID {
			// if not, push it back on the queue and try again
			funcActivityChan <- activity
			continue
		}

		pongCallCount++

		// When we push a return
		activity.Call.ResponseChan <- imptest.CallResponse{
			Type:         imptest.ReturnResponseType,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then we ping once
	pingActivity := <-funcActivityChan
	if pingActivity.Type != imptest.CallActivityType {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if pingActivity.Call.ID != pingMimicID {
		t.Fail()
	}

	// When we push a return
	pingActivity.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.ReturnResponseType,
		ReturnValues: []any{true},
		PanicValue:   nil,
	}

	// Then we pong once
	pongActivity := <-funcActivityChan
	if pongActivity.Type != imptest.CallActivityType {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if pongActivity.Call.ID != pongMimicID {
		t.Fail()
	}

	// When we push a return
	pongActivity.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.ReturnResponseType,
		ReturnValues: []any{false},
		PanicValue:   nil,
	}

	// Then we get 100 more calls to ping
	pingCallCount = 0
	for pingCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.CallActivityType {
			t.Fail()
		}

		// and the dependency call is to the mimicked dependency
		if activity.Call.ID != pingMimicID {
			// if not, push it back on the queue and try again
			funcActivityChan <- activity
			continue
		}

		pingCallCount++

		// When we push a return
		activity.Call.ResponseChan <- imptest.CallResponse{
			Type:         imptest.ReturnResponseType,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then we get 100 more calls to pong
	pongCallCount = 0
	for pongCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.CallActivityType {
			t.Fail()
		}

		// and the dependency call is to the mimicked dependency
		if activity.Call.ID != pongMimicID {
			// if not, push it back on the queue and try again
			funcActivityChan <- activity
			continue
		}

		pongCallCount++

		// When we push a return
		activity.Call.ResponseChan <- imptest.CallResponse{
			Type:         imptest.ReturnResponseType,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then the next activity from the function under test is its return
	returnActivity := <-funcActivityChan
	if returnActivity.Type != imptest.ReturnActivityType {
		t.Fail()
	}

	// and the returned data is 0 items
	if len(returnActivity.ReturnVals) != 0 {
		t.Fail()
	}
}

func pingPong(ping, pong func() bool) {
	pingChan := make(chan bool)
	defer close(pingChan)

	pongChan := make(chan bool)
	defer close(pongChan)

	wgPingPong := sync.WaitGroup{}
	pingLoop := func() {
		// unsynced calls
		for range 100 {
			ping()
		}
		// synced calls
		shouldPing := true
		for shouldPing {
			pingChan <- ping()

			shouldPing = <-pongChan
		}
		pingChan <- false
		// unsynced calls
		for range 100 {
			ping()
		}

		wgPingPong.Done()
	}
	pongLoop := func() {
		// unsynced calls
		for range 100 {
			pong()
		}
		// synced calls
		shouldPong := <-pingChan
		for shouldPong {
			pongChan <- pong()

			shouldPong = <-pingChan
		}
		// unsynced calls
		for range 100 {
			pong()
		}

		wgPingPong.Done()
	}

	wgPingPong.Add(2)

	go pingLoop()
	go pongLoop()

	wgPingPong.Wait()
}

// ===L2 Tests===

type depStruct1 struct {
	Dep1 func() string
}

// TestL2ReceiveCallSendReturn tests matching a dependency call and pushing a return more simply, with a
// dependency struct.
func TestL2ReceiveCallSendReturn(t *testing.T) {
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
	// and a string to return from the dependency call
	returnString := anyString

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)
	// Then the next thing the function under test does is make a call matching our expectations
	call := imp.ReceiveCall("Dep1")
	// When we push a return string
	call.SendReturn(returnString)
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
	// (and then When we push a panic value...)
	imp.ReceiveCall("Dep1").SendPanic(panicString)

	// Then the next thing the function under test does is panic with a value matching our expectations
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

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic.Ping, depsToMimic.Pong)

	// then we get concurrent call flows...
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

// ==Failure Tests==

// TestL2ReceiveTooFewCalls tests matching a dependency call and pushing a return more simply, with a
// dependency struct.
func TestL2ReceiveTooFewCalls(t *testing.T) {
	t.Parallel()

	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		// Given a function to test
		funcToTest := func(deps depStruct1) string {
			return deps.Dep1()
		}
		// and a struct of dependenc mimics
		depsToMimic := depStruct1{} //nolint:exhaustruct
		// and a helpful test imp
		imp := imptest.NewImp(mockedT, &depsToMimic)
		// and a string to return from the dependency call
		returnString := anyString

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		imp.ReceiveCall("Dep1").SendReturn(returnString)
		// And again
		// THIS IS WHAT WE EXPECT TO TRIGGER A FAILURE
		imp.ReceiveCall("Dep1").SendReturn(returnString)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ReceiveReturn(returnString)
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := "expected"
	actual := mockedT.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}
