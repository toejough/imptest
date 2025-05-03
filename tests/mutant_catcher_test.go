package imptest_test

import (
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/toejough/imptest/imptest"
)

// ===L1 Tests===.

// TestL1ReceiveDependencyCallSendReturn tests receiving a dependency call and sending a return.
// ignore the linter error about how this test is too long.he point behind the L2 API.
func TestL1ReceiveDependencyCallSendReturn(t *testing.T) {
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
func TestL1ReceiveDependencyCallSendPanic(t *testing.T) {
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

// TestL1UnorderedConcurrency tests that we can verify unordered concurrency. That is, we are launching some function
// calls, and not caring whether they happened sequentially, so long as they all happen.
// This test does not require that the calls are truly happening concurrently.
func TestL1UnorderedConcurrency(t *testing.T) { //nolint:funlen,gocognit,gocyclo,cyclop,maintidx
	t.Parallel()

	// Given a function to test
	funcToTest := func(d1, d2, d3, d4, d5 func()) {
		d1()
		d2()
		d3()
		d4()
		d5()
	}
	// and functions to mimic
	depToMimic1 := func() { panic("not implemented") }
	depToMimic2 := func() { panic("not implemented") }
	depToMimic3 := func() { panic("not implemented") }
	depToMimic4 := func() { panic("not implemented") }
	depToMimic5 := func() { panic("not implemented") }
	// and a channel to put function activity onto
	funcActivityChan := make(chan imptest.FuncActivity)
	// and mimics of those dependencies, which send notifications of calls to themselves on the activity channel
	depMimic1, depMimic1ID := imptest.MimicDependency(
		t,
		depToMimic1,
		funcActivityChan,
		imptest.WithName("dep1"),
	)
	depMimic2, depMimic2ID := imptest.MimicDependency(
		t,
		depToMimic2,
		funcActivityChan,
		imptest.WithName("dep2"),
	)
	depMimic3, depMimic3ID := imptest.MimicDependency(
		t,
		depToMimic3,
		funcActivityChan,
		imptest.WithName("dep3"),
	)
	depMimic4, depMimic4ID := imptest.MimicDependency(
		t,
		depToMimic4,
		funcActivityChan,
		imptest.WithName("dep4"),
	)
	depMimic5, depMimic5ID := imptest.MimicDependency(
		t,
		depToMimic5,
		funcActivityChan,
		imptest.WithName("dep5"),
	)
	// and a channel to put function activity onto
	expectationsChan := make(chan imptest.FuncActivity, 6)
	defer close(expectationsChan)

	// When we run the function to test with the mimicked dependencies
	go func() {
		defer close(funcActivityChan)
		// call the function to test
		funcToTest(depMimic1, depMimic2, depMimic3, depMimic4, depMimic5)
		// record what the func returns as its final activity
		funcActivityChan <- imptest.FuncActivity{
			Type: imptest.ActivityTypeReturn,
		}
	}()

	// When we set expectations for the function calls
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic5ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic2ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic4ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeReturn,
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic1ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic3ID},
	}

	// When we get the first activity from the function under test
	activity := <-funcActivityChan

	// Then we expect it to match one of the expectations
	matched := false

	// search 6 expectations for a match
	for range 6 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity.Type == imptest.ActivityTypeCall {
			activity.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the expectations")
	}

	// When we get the second activity
	activity = <-funcActivityChan

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 5 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity.Type == imptest.ActivityTypeCall {
			activity.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// When we get the next activity
	activity = <-funcActivityChan

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 4 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity.Type == imptest.ActivityTypeCall {
			activity.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// When we get the next activity
	activity = <-funcActivityChan

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 3 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity.Type == imptest.ActivityTypeCall {
			activity.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// When we get the next activity
	activity = <-funcActivityChan

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 2 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity.Type == imptest.ActivityTypeCall {
			activity.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// When we get the next activity
	activity = <-funcActivityChan

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 1 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity.Type == imptest.ActivityTypeCall {
			activity.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}
}

// TestL1RequiredConcurrency tests that when function activity happens concurrently, we can expect
// that with the L1 API.
func TestL1RequiredConcurrency(t *testing.T) { //nolint:funlen,gocognit,gocyclo,cyclop,maintidx
	t.Parallel()

	// Given a function to test
	funcToTest := func(d1, d2, d3, d4, d5 func()) {
		go d1()
		go d2()
		go d3()
		go d4()
		go d5()
	}
	// and functions to mimic
	depToMimic1 := func() { panic("not implemented") }
	depToMimic2 := func() { panic("not implemented") }
	depToMimic3 := func() { panic("not implemented") }
	depToMimic4 := func() { panic("not implemented") }
	depToMimic5 := func() { panic("not implemented") }
	// and a channel to put function activity onto
	funcActivityChan := make(chan imptest.FuncActivity)
	// and mimics of those dependencies, which send notifications of calls to themselves on the activity channel
	depMimic1, depMimic1ID := imptest.MimicDependency(
		t,
		depToMimic1,
		funcActivityChan,
		imptest.WithName("dep1"),
	)
	depMimic2, depMimic2ID := imptest.MimicDependency(
		t,
		depToMimic2,
		funcActivityChan,
		imptest.WithName("dep2"),
	)
	depMimic3, depMimic3ID := imptest.MimicDependency(
		t,
		depToMimic3,
		funcActivityChan,
		imptest.WithName("dep3"),
	)
	depMimic4, depMimic4ID := imptest.MimicDependency(
		t,
		depToMimic4,
		funcActivityChan,
		imptest.WithName("dep4"),
	)
	depMimic5, depMimic5ID := imptest.MimicDependency(
		t,
		depToMimic5,
		funcActivityChan,
		imptest.WithName("dep5"),
	)
	// and a channel to put function activity onto
	expectationsChan := make(chan imptest.FuncActivity, 6)
	defer close(expectationsChan)

	// When we run the function to test with the mimicked dependencies
	go func() {
		// call the function to test
		funcToTest(depMimic1, depMimic2, depMimic3, depMimic4, depMimic5)
		// record what the func returns as its final activity
		funcActivityChan <- imptest.FuncActivity{
			Type: imptest.ActivityTypeReturn,
		}
	}()

	// When we set expectations for the function calls
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic5ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic2ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic4ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeReturn,
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic1ID},
	}
	expectationsChan <- imptest.FuncActivity{
		Type: imptest.ActivityTypeCall,
		Call: &imptest.Call{ID: depMimic3ID},
	}

	// When we get the concurrent activities from the function under test
	activity1 := <-funcActivityChan
	activity2 := <-funcActivityChan
	activity3 := <-funcActivityChan
	activity4 := <-funcActivityChan
	activity5 := <-funcActivityChan
	activity6 := <-funcActivityChan

	// Then we expect it to match one of the expectations
	matched := false

	// search 6 expectations for a match
	for range 6 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity1.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity1.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity1.Type == imptest.ActivityTypeCall {
			activity1.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the expectations")
	}

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 5 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity2.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity2.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity2.Type == imptest.ActivityTypeCall {
			activity2.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 4 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity3.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity3.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity3.Type == imptest.ActivityTypeCall {
			activity3.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 3 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity4.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity4.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity4.Type == imptest.ActivityTypeCall {
			activity4.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 2 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity5.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity5.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity5.Type == imptest.ActivityTypeCall {
			activity5.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}

	// Then we expect it to match one of the remaining expectations
	matched = false

	// search the remaining expectations
	for range 1 {
		// get the next expectation
		expectation := <-expectationsChan
		// if no match, push it back onto the channel
		if expectation.Type != activity6.Type {
			expectationsChan <- expectation
			continue
		}
		// if no match, push it back onto the channel
		if expectation.Type == imptest.ActivityTypeCall && expectation.Call.ID != activity6.Call.ID {
			expectationsChan <- expectation
			continue
		}

		// record the match
		matched = true

		// if it's a call, push a response
		if activity6.Type == imptest.ActivityTypeCall {
			activity6.Call.ResponseChan <- imptest.CallResponse{
				Type:         imptest.ResponseTypeReturn,
				PanicValue:   nil,
				ReturnValues: nil,
			}
		}

		break
	}

	if !matched {
		t.Fatal("expected to match one of the remaining expectations")
	}
}

// ===L2 Tests===

// TestL2ReturnNil tests returning nil, which is tricky from a reflection standpoint.
func TestL2ReturnNil(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct1) *string {
		deps.Dep1()
		return nil
	}
	// and a struct of dependencies to mimic
	depsToMimic := depStruct1{}
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	defer imp.Close()
	// and a string to return from the dependency call
	returnString := anyString

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)
	// Then the next thing the function under test does is make a call matching our expectations
	// When we push a return string
	imp.ExpectCall("Dep1").SendReturn(returnString)
	// Then the next thing the function under test does is return values matching our expectations
	imp.ExpectReturn(nil)
}

// TestL2PushNil tests pushing nil return, which is tricky from a reflection standpoint.
func TestL2PushNil(t *testing.T) {
	t.Parallel()

	// Given a function to test
	funcToTest := func(deps depStruct5) {
		deps.Dep1()
	}
	// and a struct of dependencies to mimic
	depsToMimic := depStruct5{}
	// and a helpful test imp
	imp := imptest.NewImp(t, &depsToMimic)
	defer imp.Close()

	// When we run the function to test with the mimicked dependencies
	imp.Start(funcToTest, depsToMimic)
	// Then the next thing the function under test does is make a call matching our expectations
	// When we push a return string
	imp.ExpectCall("Dep1").SendReturn(nil)
	// Then the next thing the function under test does is return values matching our expectations
	imp.ExpectReturn()
}

type depStruct5 struct {
	Dep1 func() *string
}

// TestL2ConcurrentlyRuns verifies that the L2 concurrently API actually does run the checks contained within it.
func TestL2ConcurrentlyRuns(t *testing.T) {
	t.Parallel()

	// Given
	funcToTest := func() {}
	counter := atomic.Int32{}
	imp := imptest.NewImp(t)

	// When
	imp.Start(funcToTest)
	imp.Concurrently(
		func() { counter.Add(1) },
		func() { counter.Add(1) },
		func() { counter.Add(1) },
		func() { counter.Add(1) },
	)

	// Then
	actual := int(counter.Load())
	expected := 4

	if actual != expected {
		t.Fatalf("Expected the counter to show %d calls, but it only showed %d", expected, actual)
	}
}

func TestL1PrettyPrintFailure(t *testing.T) {
	t.Parallel()

	// Given call that can't be json encoded
	c := &imptest.Call{ID: "yo", Args: []any{make(chan string)}}

	// When we try to make it a string
	actual := c.String()

	// Then we expect to find a json error inside it
	expected := `(?s)couldn't json marshal:.*json: unsupported type.*`

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf(
			"expected string conversion to result in a string matching %s, but it resulted in %s instead",
			expected, actual,
		)
	}
}

// ==Failure Tests==

type depStruct4 struct {
	D1 func(*int, int, int)
}

// TestL2ReturnEmptyArrayIsNotNil verifies that returning an empty array and expecting nil fails.
func TestL2ReturnEmptyArrayIsNotNil(t *testing.T) {
	t.Parallel()

	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		// Given
		funcToTest := func() []struct{} {
			return []struct{}{}
		}
		imp := imptest.NewImp(mockedT)

		// When
		imp.Start(funcToTest)

		// Then
		imp.ExpectReturn(nil)
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := `(?s)expected.*nil.*but got.*\[\[]].*`
	actual := mockedT.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}

// TestL2ReturnEmptyArrayIsNotNil verifies that returning a nil and expecting an empty arrayyyyyyyy fails.
func TestL2ReturnNilIsNotEmptyArray(t *testing.T) {
	t.Parallel()

	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		// Given
		funcToTest := func() []struct{} {
			return nil
		}
		imp := imptest.NewImp(mockedT)

		// When
		imp.Start(funcToTest)

		// Then
		imp.ExpectReturn([]struct{}{})
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := `(?s)expected.*\[\[]].*but got.*nil.*`
	actual := mockedT.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}

// TestL2ReceiveWrongArgValues tests the failure message when receiving wrong arg values.
func TestL2ReceiveWrongArgValues(t *testing.T) {
	t.Parallel()

	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		// Given
		funcToTest := func(x *int, y, z int, deps depStruct4) {
			deps.D1(x, y, z)
		}
		depsToMimic := depStruct4{}
		imp := imptest.NewImp(mockedT, &depsToMimic)

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, nil, 4, 3, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		// EXPECT THE 2 TO CAUSE A PROBLEM
		imp.ExpectCall("D1", nil, 4, 2).SendReturn()
		// And again

		// Then the next thing the function under test does is return values matching our expectations
		imp.ExpectReturn()
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := `(?s)expected.*args.*2.*args.*3.*`
	actual := mockedT.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}

// TestL2ReceiveWrongType tests the failure message when receiving wrong type.
func TestL2ReceiveWrongType(t *testing.T) {
	t.Parallel()

	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		// Given
		funcToTest := func(x *int) {
			panic(x)
		}
		imp := imptest.NewImp(mockedT)

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, nil)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ExpectReturn()
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := `(?s)expected.*Return.*Panic.*`
	actual := mockedT.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}

func TestCallPanicString(t *testing.T) {
	t.Parallel()

	defer func() {
		p := recover()
		if p == nil {
			t.Fatal("expected to panic")
		}
	}()

	_ = (&imptest.FuncActivity{Type: imptest.ActivityTypeUnset}).String()
}

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
		depsToMimic := depStruct1{}
		// and a helpful test imp
		imp := imptest.NewImp(mockedT, &depsToMimic)
		// and a string to return from the dependency call
		returnString := anyString

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		imp.ExpectCall("Dep1").SendReturn(returnString)
		// And again
		// THIS IS WHAT WE EXPECT TO TRIGGER A FAILURE
		imp.ExpectCall("Dep1").SendReturn(returnString)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ExpectReturn(returnString)
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := `.*expected Call.*`
	actual := mockedT.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("expected test to fail with %s, but it failed with %s instead", expected, actual)
	}
}

// TestL2ReceiveNothing tests the error presented when no func activity is found.
func TestL2ReceiveNothing(t *testing.T) {
	t.Parallel()

	mockedT := newMockedTestingT()
	mockedT.Wrap(func() {
		// Given a function to test
		funcToTest := func(deps depStruct1) string {
			return deps.Dep1()
		}
		// and a struct of dependenc mimics
		depsToMimic := depStruct1{}
		// and a helpful test imp
		imp := imptest.NewImp(mockedT, &depsToMimic)
		// and a string to return from the dependency call
		returnString := anyString

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		imp.ExpectCall("Dep1").SendReturn(returnString)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ExpectReturn(returnString)

		// THIS IS WHAT WE EXPECT TO TRIGGER A FAILURE
		// shouldn't return a second time... there shouldn't be anything
		imp.ExpectReturn(returnString)
	})

	if !mockedT.Failed() {
		t.Fatalf("expected to fail instead of passing")
	}

	expected := `.*but got no function activity.*`
	actual := mockedT.Failure()

	if !regexp.MustCompile(expected).MatchString(actual) {
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
		depsToMimic := depStruct1{}
		// and a helpful test imp
		imp := imptest.NewImp(mockedT, &depsToMimic)
		// and a string to return from the dependency call
		returnInt := anyInt

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		call := imp.ExpectCall("Dep1")
		// THIS IS WHAT WE EXPECT TO TRIGGER A FAILURE
		call.SendReturn(returnInt)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ExpectReturn(returnInt)
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
		depsToMimic := depStruct1{}
		// and a helpful test imp
		imp := imptest.NewImp(mockedT, &depsToMimic)
		// and a string to return from the dependency call
		returnInt := anyInt

		// When we run the function to test with the mimicked dependencies
		imp.Start(funcToTest, depsToMimic)

		// Then the next thing the function under test does is make a call matching our expectations
		// When we push a return string
		call := imp.ExpectCall("Dep1")
		// THIS IS WHAT WE EXPECT TO TRIGGER A FAILURE
		call.SendReturn(returnInt)

		// Then the next thing the function under test does is return values matching our expectations
		imp.ExpectReturn(returnInt)
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

	if reflect.ValueOf(actual).IsValid() {
		t.Fatal("expected invalid value for actual")
	}

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
