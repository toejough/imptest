package imptest_test

import (
	"reflect"
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
const (
	anyString = "literally anything"
	anyInt    = 42 // literally anything
)

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
			Type:       imptest.ActivityTypeReturn,
			PanicVal:   nil,
			ReturnVals: []any{returnVal},
			Call:       nil,
		}
	}()

	// Then the first activity in the funcActivitychannel is a dependency call
	activity1 := <-funcActivityChan
	if activity1.Type != imptest.ActivityTypeCall {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if activity1.Call.ID != depMimicID {
		t.Fail()
	}

	// When we push a return string
	activity1.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.ResponseTypeReturn,
		ReturnValues: []any{returnString},
		PanicValue:   nil,
	}

	// Then the next activity from the function under test is its return
	activity2 := <-funcActivityChan
	if activity2.Type != imptest.ActivityTypeReturn {
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
					Type:       imptest.ActivityTypePanic,
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
	if activity1.Type != imptest.ActivityTypeCall {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if activity1.Call.ID != depMimicID {
		t.Fail()
	}

	// When we push a panic string
	activity1.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.ResponseTypePanic,
		ReturnValues: nil,
		PanicValue:   panicString,
	}

	// Then the next activity from the function under test is its panic
	activity2 := <-funcActivityChan
	if activity2.Type != imptest.ActivityTypePanic {
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
			Type:       imptest.ActivityTypeReturn,
			PanicVal:   nil,
			ReturnVals: nil,
			Call:       nil,
		}
	}()

	// Then we get 100 calls to ping
	pingCallCount := 0
	for pingCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.ActivityTypeCall {
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
			Type:         imptest.ResponseTypeReturn,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then we get 100 calls to pong
	pongCallCount := 0
	for pongCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.ActivityTypeCall {
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
			Type:         imptest.ResponseTypeReturn,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then we ping once
	pingActivity := <-funcActivityChan
	if pingActivity.Type != imptest.ActivityTypeCall {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if pingActivity.Call.ID != pingMimicID {
		t.Fail()
	}

	// When we push a return
	pingActivity.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.ResponseTypeReturn,
		ReturnValues: []any{true},
		PanicValue:   nil,
	}

	// Then we pong once
	pongActivity := <-funcActivityChan
	if pongActivity.Type != imptest.ActivityTypeCall {
		t.Fail()
	}

	// and the dependency call is to the mimicked dependency
	if pongActivity.Call.ID != pongMimicID {
		t.Fail()
	}

	// When we push a return
	pongActivity.Call.ResponseChan <- imptest.CallResponse{
		Type:         imptest.ResponseTypeReturn,
		ReturnValues: []any{false},
		PanicValue:   nil,
	}

	// Then we get 100 more calls to ping
	pingCallCount = 0
	for pingCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.ActivityTypeCall {
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
			Type:         imptest.ResponseTypeReturn,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then we get 100 more calls to pong
	pongCallCount = 0
	for pongCallCount < 100 {
		activity := <-funcActivityChan
		if activity.Type != imptest.ActivityTypeCall {
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
			Type:         imptest.ResponseTypeReturn,
			ReturnValues: []any{true},
			PanicValue:   nil,
		}
	}

	// Then the next activity from the function under test is its return
	returnActivity := <-funcActivityChan
	if returnActivity.Type != imptest.ActivityTypeReturn {
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

// TestL2ReturnNil tests returning nil, which is tricky from a reflection standpoint.
func TestL2ReturnNil(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct1) *string {
		deps.Dep1()
		return nil
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
	// When we push a return string
	imp.ReceiveCall("Dep1").SendReturn(returnString)
	// Then the next thing the function under test does is return values matching our expectations
	imp.ReceiveReturn(nil)
}

// TestL2PushNil tests pushing nil return, which is tricky from a reflection standpoint.
func TestL2PushNil(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct2) {
		deps.Dep1()
	}
	// and a struct of dependencies to mimic
	depsToMimic := depStruct2{} //nolint:exhaustruct
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	defer imp.Close()

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)
	// Then the next thing the function under test does is make a call matching our expectations
	// When we push a return string
	imp.ReceiveCall("Dep1").SendReturn(nil)
	// Then the next thing the function under test does is return values matching our expectations
	imp.ReceiveReturn()
}

type depStruct2 struct {
	Dep1 func() *string
}

// TestL2OutOfOrderActivityTypesConcurrency tests when concurrency means function activity calls are out of order, but
// the test still works as expected.
func TestL2OutOfOrderActivityTypesConcurrency(t *testing.T) {
	t.Parallel()

	// Given a function to test
	callBlockChan := make(chan struct{})
	returnBlockChan := make(chan struct{})
	expectationCallBlockChan := make(chan struct{})
	expectationReturnBlockChan := make(chan struct{})
	funcToTest := func(deps depStruct1) error {
		go func() {
			// won't call dep1 till we tell it to
			<-callBlockChan
			deps.Dep1()
		}()

		<-returnBlockChan

		return nil
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
	// expect the call and the return concurrently
	// allow the call expectation
	// allow the return activity
	// once we're sure they have been queued up in conflict...
	// allow return expectation
	// allow call activity
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		imp.Concurrently(func() {
			// Then the next thing the function under test does is make a call matching our expectations
			// When we push a return string
			<-expectationCallBlockChan
			imp.ReceiveCall("Dep1").SendReturn(returnString)
		}, func() {
			// Then the next thing the function under test does is return values matching our expectations
			<-expectationReturnBlockChan
			imp.ReceiveReturn(nil)
		})
	}()
	go func() {
		defer waitGroup.Done()
		expectationReturnBlockChan <- struct{}{}
		callBlockChan <- struct{}{}
		expectationCallBlockChan <- struct{}{}
		returnBlockChan <- struct{}{}
	}()
	waitGroup.Wait()
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

// TestL2ReceiveWrongReturnType tests returning an incorrect type from a dependency.
func TestL2ReceiveWrongReturnType(t *testing.T) {
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
		returnInt := anyInt

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		call := imp.ReceiveCall("Dep1")
		// THIS IS WHAT WE EXPECT TO TRIGGER A FAILURE
		call.SendReturn(returnInt)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ReceiveReturn(returnInt)
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

// TestL2PushWrongReturnType tests returning an incorrect type from a dependency.
func TestL2PushWrongReturnType(t *testing.T) {
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
		returnInt := anyInt

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		call := imp.ReceiveCall("Dep1")
		// THIS IS WHAT WE EXPECT TO TRIGGER A FAILURE
		call.SendReturn(returnInt)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ReceiveReturn(returnInt)
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := "unable to push"
	actual := mockedT.Failure()

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}

// ==Mixed L1/L2 tests demonstrating finer control==.
func TestL2L1MixReceiveCallSendReturn(t *testing.T) {
	t.Parallel()

	// ==L2 stuff, which is easier to read but gives you less control==
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

	// ==L1 stuff, for when you need more control over matching and failure messaging than L2 allows==
	// Then the next thing the function under test does is return values matching our expectations
	functionReturned := <-imp.ActivityChan
	if functionReturned.Type != imptest.ActivityTypeReturn {
		t.Fatal("expected a return action but got something else")
	}

	returns := functionReturned.ReturnVals
	if len(returns) != 1 {
		t.Fatalf("Expected only one return but got %d", len(returns))
	}

	// if this is not a string, the imp would've already complained
	// in general, if you are asking  the test _should_ panic.
	retString := returns[0].(string) //nolint:forcetypeassert
	if !strings.HasPrefix(retString, returnString) {
		t.Fatalf(
			"expected the return string to have a prefix of the sent return from the dependency call (%s),"+
				"but it didn't. Instead it was just '%s'.",
			returnString, retString,
		)
	}
}

// ==Exploration tests & sanity checks==.
func TestNilEquality(t *testing.T) {
	t.Parallel()

	if !reflect.DeepEqual(nil, nil) { //nolint:gocritic
		t.Fatal("nil doesn't equal nil...")
	}
}

func TestTypedNilEquality(t *testing.T) {
	t.Parallel()

	var actual any

	var expected *string

	actual = nil
	expected = nil

	if reflect.DeepEqual(expected, actual) {
		t.Fatal("typed nil shouldn't equal nil...")
	}

	if expected != nil {
		t.Fatal("typed nil should equal nil...")
	}

	if actual != nil {
		t.Fatal("untyped nil should equal nil...")
	}

	if reflect.ValueOf(actual).IsValid() {
		t.Fatal("expected invalid value for actual")
	}

	// TODO: loop through comparison arrays, and if an expected value is not valid, and the actual value is nil, pass.
	if !reflect.ValueOf(expected).IsValid() {
		t.Fatal("expected valid value for expected")
	}

	if !reflect.ValueOf(expected).IsNil() {
		t.Fatal("expected nil value for expected")
	}
}

func TestNilSliceEquality(t *testing.T) {
	t.Parallel()

	if !reflect.DeepEqual([]any{nil}, []any{nil}) { //nolint:gocritic
		t.Fatal("nil doesn't equal nil...")
	}
}

func TestNilAnySliceEquality(t *testing.T) {
	t.Parallel()

	var expected, actual any

	expected = nil
	actual = nil

	if !reflect.DeepEqual([]any{expected}, []any{actual}) {
		t.Fatal("nil doesn't equal nil...")
	}
}

func TestTypedNilSliceEquality(t *testing.T) {
	t.Parallel()

	var actual any

	var expected *string

	actual = nil
	expected = nil

	newValue := reflect.New(reflect.TypeOf(expected))
	converted := newValue.Elem().Interface()

	if reflect.DeepEqual([]any{expected}, []any{actual}) {
		t.Fatal("string nil shouldn't equal any nil...")
	}

	if !reflect.DeepEqual([]any{expected}, []any{converted}) {
		t.Fatal("string nil didn't equal string nil...")
	}

	// this works...
	asssertedActual, _ := actual.(*string)
	if !reflect.DeepEqual([]any{expected}, []any{asssertedActual}) {
		t.Fatal("string nil didn't equal string nil...")
	}

	// but... this still doesn't work
	asssertedExpected := any(expected)
	if reflect.DeepEqual([]any{asssertedExpected}, []any{actual}) {
		t.Fatal("string nil shouldn't equal any nil...")
	}

	// neither does this
	actualArr, _ := any([]any{actual}).([]any)
	expectedArr, _ := any([]any{expected}).([]any)

	if reflect.DeepEqual(expectedArr[0], actualArr[0]) {
		t.Fatal("string nil shouldn't equal any nil...")
	}

	// maybe this?
	var expectedAny any = expected
	expectedArr = []any{expectedAny}

	var eaAny any = expectedArr

	arrAgain, _ := eaAny.([]any)

	if reflect.DeepEqual(arrAgain[0], actualArr[0]) {
		t.Fatal("string nil shouldn't equal any nil...")
	}
}
