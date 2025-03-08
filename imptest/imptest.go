// Package imptest provides impure function testing functionality.
package imptest

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// Error philosophy:
//
// Failures: conditions which signal expected failures the user is testing for (this is a test library), should
// trigger a test failure.
//
// Panics: conditions which signal an error which it is not generally reasonable to expect a caller to recover from,
// which instead imply programmer intervention is necessary to resolve, should trigger an explanatory panic for the
// programmer to track down.
//
// Errors: all other error conditions should trigger an error with sufficient detail to enable the caller to take
// corrective action

// ==L1 Exported Types==

// FuncActivity is essentially a union struct representing the various activities the function under test can perform:
// * panic
// * return
// * call a dependency.
type FuncActivity struct {
	Type       ActivityType
	PanicVal   any
	ReturnVals []any
	Call       *Call
}

// ActivityType represents the known types of activities the function under test can perform.
type ActivityType int

const (
	ActivityTypeUnset ActivityType = iota
	ActivityTypeReturn
	ActivityTypePanic
	ActivityTypeCall
)

// Call represents a call to a dependency, as well as providing the channel to send the response that
// the depenency should perform.
type Call struct {
	ID           string
	Args         []any
	ResponseChan chan CallResponse
	Type         reflect.Type
	t            Tester
}

// CallResponse is essentially a union struct representing the various responses a mimicked dependency call
// can be asked to perform:
// * panic
// * return.
type CallResponse struct {
	Type         ResponseType
	PanicValue   any
	ReturnValues []any
}

// ResponseType represents the known types of responses a mimicked dependency call can perform.
type ResponseType int

const (
	ResponseTypeUnset ResponseType = iota
	ResponseTypePanic
	ResponseTypeReturn
)

// MimicOptionModifier is a type representing a function that takes a MimicOptions struct and returns a modified
// version of that struct.
type MimicOptionModifier func(MimicOptions) MimicOptions

// MimicOptions is a struct of unexported data to be used by the MimicDependency function.
type MimicOptions struct {
	name string
}

// ==L1 Exported Methods==

// SendReturn sends a return response to the mimicked dependency, which will cause it to return with the given values.
// SendReturn checks for the correct # of values as well as their assignability ot the dependency call's return types.
// SendReturn closes and clears the response channel for the dependency call when it is done.
func (c *Call) SendReturn(returnVals ...any) {
	// TODO: migrate this back into the function mimic in the first place. Then we wouldn't need to pass Type or t
	// around.
	c.t.Helper()
	// clean up after this call
	defer func() {
		// close the response channel
		close(c.ResponseChan)
		// nil the response channel out
		c.ResponseChan = nil
	}()
	// make sure these are at least the right number of returns
	expectedNumReturns := c.Type.NumOut()
	if len(returnVals) != expectedNumReturns {
		c.t.Fatalf(
			"%d returns were pushed, but %s only returns %d values",
			len(returnVals), c.ID, expectedNumReturns,
		)
	}
	// make sure these are at least assignable
	for rvi := range returnVals {
		actual := reflect.TypeOf(returnVals[rvi])
		expected := c.Type.Out(rvi)

		if actual != nil && !actual.AssignableTo(expected) {
			c.t.Fatalf(
				"unable to push return value %d for the call to %s: a value of type %v was pushed, "+
					"but that is unassignable to the expected type (%v)",
				rvi, c.ID, actual, expected,
			)
		}
	}
	// send the response
	c.ResponseChan <- CallResponse{
		Type:         ResponseTypeReturn,
		PanicValue:   nil,
		ReturnValues: returnVals,
	}
}

// SendPanic sends a panic response to the mimicked dependency, which will cause it to panic with the given value.
// SendPanic closes and clears the response channel for the dependency call when it is done.
func (c *Call) SendPanic(panicVal any) {
	// clean up after this call
	defer func() {
		// close the response channel
		close(c.ResponseChan)
		// nil the response channel out
		c.ResponseChan = nil
	}()
	// send the response
	c.ResponseChan <- CallResponse{
		Type:         ResponseTypePanic,
		PanicValue:   panicVal,
		ReturnValues: nil,
	}
}

// ==L1 Exported Funcs==

// MimicDependency mimics a given dependency. Instead of calling performing the dependency's logic, the mimic sends the
// call signature to the dependency on the given activity channel, waits for a response command, and then executes that
// command by either returning or panicking with the given values.
func MimicDependency[T any](
	tester Tester, function T, activityChan chan FuncActivity, modifiers ...MimicOptionModifier,
) (T, string) {
	options := MimicOptions{name: getFuncName(function)}
	for _, modify := range modifiers {
		options = modify(options)
	}

	name := options.name

	funcType := reflect.TypeOf(function)

	funcAsValue := makeFunc(tester, funcType, name, activityChan)
	// Ignore the type assertion lint check - we are depending on MakeFunc to
	// return the correct type, as documented. If it fails to, the only thing
	// we'd do is panic anyway.
	typedMimic := funcAsValue.Interface().(T) //nolint:forcetypeassert

	// returns both the wrapped func and the ID
	return typedMimic, options.name
}

func makeFunc(tester Tester, funcType reflect.Type, name string, activityChan chan FuncActivity) reflect.Value {
	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	reflectedMimic := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		responseChan := make(chan CallResponse)

		// Submit this call to the calls channel
		activityChan <- FuncActivity{
			ActivityTypeCall,
			nil,
			nil,
			&Call{
				name,
				unreflectValues(args),
				responseChan,
				funcType,
				tester,
			},
		}

		// wait for a response
		responseToActivity := <-responseChan

		// handle the response
		switch responseToActivity.Type {
		case ResponseTypeReturn:
			// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
			return convertReturnValues(responseToActivity)
		// if we're supposed to panic, do.
		case ResponseTypePanic:
			panic(responseToActivity.PanicValue)
		// for unknown or unset response types, panic
		case ResponseTypeUnset:
			panic("imptest failure - a ResponseTypeUnset was received")
		default:
			panic("imptest failure - unrecognized response type was received")
		}
	}

	// Make the new function
	return reflect.MakeFunc(funcType, reflectedMimic)
}

// WithName is a modifier which sets the dependency's name.
func WithName(name string) MimicOptionModifier {
	return func(wo MimicOptions) MimicOptions {
		wo.name = name
		return wo
	}
}

// ==L1 Unexported Helpers==

func convertReturnValues(outputV CallResponse) []reflect.Value {
	returnValues := make([]reflect.Value, len(outputV.ReturnValues))

	for i, a := range outputV.ReturnValues {
		returnValues[i] = reflect.ValueOf(a)
	}

	return returnValues
}

// getFuncName gets the function's name.
func getFuncName(f function) string {
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	// this suffix gets appended sometimes. It's unimportant, as far as I can tell.
	name = strings.TrimSuffix(name, "-fm")

	return name
}

// unreflectValues returns the actual values of the reflected values.
func unreflectValues(rArgs []reflect.Value) []any {
	// tricking nilaway with repeated appends till this issue is closed
	// https://github.com/uber-go/nilaway/pull/60
	// args := make([]any, len(rArgs))
	if len(rArgs) == 0 {
		return nil
	}

	args := []any{}

	for i := range rArgs {
		// args[i] = rArgs[i].Interface()
		args = append(args, rArgs[i].Interface())
	}

	return args
}

// function is here to help us distinguish functions internally, because there is no single
// function _type_ in go.
type function any

// ==L2 Exported Types==.
type Imp struct {
	concurrency     atomic.Int64
	expectationChan chan expectation
	ActivityChan    chan FuncActivity
	T               Tester
}

type Tester interface {
	Helper()
	Fatal(args ...any)
	Fatalf(message string, args ...any)
	Logf(message string, args ...any)
}

// ==L2 Exported Methods==

func (t *Imp) Close() {
	close(t.ActivityChan)
	close(t.expectationChan)
}

func (t *Imp) Start(f any, args ...any) *Imp {
	go t.startFunctionUnderTest(f, args)
	go t.matchActivitiesToExpectations()

	return t
}

func (t *Imp) startFunctionUnderTest(function any, args []any) {
	var rVals []any

	// TODO: push this down into callFunc?
	defer func() {
		panicVal := recover()
		if panicVal != nil {
			t.ActivityChan <- FuncActivity{
				ActivityTypePanic,
				panicVal,
				nil,
				nil,
			}
		} else {
			t.ActivityChan <- FuncActivity{
				ActivityTypeReturn,
				nil,
				rVals,
				nil,
			}
		}
	}()

	rVals = callFunc(function, args)
}

func (t *Imp) ReceiveCall(expectedCallID string, expectedArgs ...any) *Call {
	t.T.Helper()
	// t.T.Logf("receiving call")

	expected := expectation{
		FuncActivity{
			ActivityTypeCall,
			nil,
			nil,
			&Call{
				expectedCallID,
				expectedArgs,
				nil,
				nil,
				nil,
			},
		},
		make(chan expectationResponse),
	}

	t.expectationChan <- expected
	response := <-expected.responseChan

	if response.match == nil {
		t.T.Fatalf("expected %v, but got %v", expected.activity, response.misses)
	}

	return response.match.Call
}

func (t *Imp) ReceiveReturn(returned ...any) {
	t.T.Helper()
	t.T.Logf("receiving return")

	expected := expectation{
		FuncActivity{
			ActivityTypeReturn,
			nil,
			returned,
			nil,
		},
		make(chan expectationResponse),
	}

	t.expectationChan <- expected
	response := <-expected.responseChan

	if response.match == nil {
		t.T.Fatalf("expected %v, but got %v", expected.activity, response.misses)
	}
}

func (t *Imp) ReceivePanic(panicValue any) {
	t.T.Helper()
	t.T.Logf("receiving panic")

	expected := expectation{
		FuncActivity{
			ActivityTypePanic,
			panicValue,
			nil,
			nil,
		},
		make(chan expectationResponse),
	}

	t.expectationChan <- expected
	response := <-expected.responseChan

	if response.match == nil {
		t.T.Fatalf("expected %v, but got %v", expected.activity, response.misses)
	}
}

func (t *Imp) Concurrently(funcs ...func()) {
	// set up waitgroup
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(funcs))
	defer waitGroup.Wait()

	// run the expected call flows
	for index := range funcs {
		go func() {
			defer waitGroup.Done()
			defer func() { t.concurrency.Add(-1) }()

			t.concurrency.Add(1)

			funcs[index]()
		}()
	}
}

// ==L2 funcs==

// NewImp creates a new imp to help you test without being so verbose.
func NewImp(tester Tester, funcStructs ...any) *Imp {
	tester2 := &Imp{
		concurrency:     atomic.Int64{},
		expectationChan: make(chan expectation),
		ActivityChan:    make(chan FuncActivity),
		T:               tester,
	}

	for _, fs := range funcStructs {
		// get all methods of the funcStructs
		fsType := reflect.ValueOf(fs).Elem().Type()
		fsValue := reflect.ValueOf(fs).Elem()
		numFields := fsType.NumField()
		fields := make([]fieldPair, numFields)

		for i := range numFields {
			fields[i].Type = fsType.Field(i)
			fields[i].Value = fsValue.Field(i)
		}

		// reduce to fields that are functions
		functionFields := []fieldPair{}

		for i := range numFields {
			if fields[i].Type.Type.Kind() != reflect.Func {
				continue
			}

			functionFields = append(functionFields, fields[i])
		}

		// intercept them all
		for i := range functionFields {
			wrapFuncField(tester, functionFields[i], tester2.ActivityChan)
		}
	}

	return tester2
}

// ==L2 Unexported Helpers==.
func (t *Imp) matchActivitiesToExpectations() {
	activities := []FuncActivity{}
	expectations := []expectation{}

	for {
		// select on either expectation or action chans
		//   put it on the buffer
		var done bool
		expectations, activities, done = t.updateActivitiesAndExpectations(expectations, activities)

		if done {
			return
		}
		// check against eachother
		// if any activity matches
		expectationIndex, activityIndex, matched := matchBuffers(expectations, activities)

		// if not matched & either buffer is full & there's at least one in each buffer, fail all expectations
		activityBufferFull := len(activities) >= int(t.concurrency.Load())
		expectationBufferFull := len(expectations) >= int(t.concurrency.Load())
		eitherBufferFull := activityBufferFull || expectationBufferFull
		shouldBeAMatch := eitherBufferFull && len(activities) > 0 && len(expectations) > 0

		if !matched && shouldBeAMatch {
			failExpectations(expectations, activities)

			continue
		}

		// just no match, go back to start
		if !matched {
			continue
		}

		// there was a match - remove the matches from the buffers & respond to the expectation
		expectation := expectations[expectationIndex]
		activity := activities[activityIndex]
		activities = removeFromSlice(activities, activityIndex)
		expectations = removeFromSlice(expectations, expectationIndex)
		//   respond to the receive event with success
		expectation.responseChan <- expectationResponse{match: &activity, misses: nil}
	}
}

func (t *Imp) updateActivitiesAndExpectations(
	expectations []expectation, activities []FuncActivity,
) ([]expectation, []FuncActivity, bool) {
	select {
	case expectation, ok := <-t.expectationChan:
		if !ok {
			return nil, nil, true
		}

		expectations = append(expectations, expectation)
	case activity, ok := <-t.ActivityChan:
		if !ok {
			return nil, nil, true
		}

		activities = append(activities, activity)
	}

	return expectations, activities, false
}

type expectation struct {
	activity     FuncActivity
	responseChan chan expectationResponse
}

type expectationResponse struct {
	match  *FuncActivity
	misses []FuncActivity
}

// callFunc calls the given function with the given args, and returns the return values from that callFunc.
func callFunc(f function, args []any) []any {
	rf := reflect.ValueOf(f)
	rArgs := reflectValuesOf(args)
	rReturns := rf.Call(rArgs)

	return unreflectValues(rReturns)
}

// reflectValuesOf returns reflected values for all of the values.
func reflectValuesOf(args []any) []reflect.Value {
	rArgs := make([]reflect.Value, len(args))
	for i := range args {
		rArgs[i] = reflect.ValueOf(args[i])
	}

	return rArgs
}

type fieldPair struct {
	Type  reflect.StructField
	Value reflect.Value
}

func failExpectations(expectationBuffer []expectation, activityBuffer []FuncActivity) {
	for i := range expectationBuffer {
		expectationBuffer[i].responseChan <- expectationResponse{match: nil, misses: activityBuffer}
	}
}

func matchBuffers(expectationBuffer []expectation, activityBuffer []FuncActivity) (int, int, bool) {
	expectationIndex := -1
	activityIndex := -1
	matched := false

	for index := range expectationBuffer {
		expectation := expectationBuffer[index]

		activityIndex = matchActivity(expectation.activity, activityBuffer)
		if activityIndex >= 0 {
			expectationIndex = index
			matched = true

			break
		}
	}

	return expectationIndex, activityIndex, matched
}

func removeFromSlice[T any](slice []T, index int) []T {
	slice = append(slice[0:index], slice[index+1:]...)
	return slice
}

func matchActivity(expectedActivity FuncActivity, activityBuffer []FuncActivity) int {
	for index := range activityBuffer {
		activity := activityBuffer[index]

		if activity.Type != expectedActivity.Type {
			continue
		}

		var expected, actual any

		switch expectedActivity.Type {
		case ActivityTypeCall:
			// check ID
			if activity.Call.ID != expectedActivity.Call.ID {
				continue
			}

			// check args
			expected = expectedActivity.Call.Args
			actual = activity.Call.Args

		case ActivityTypeReturn:
			// check values
			expected = expectedActivity.ReturnVals
			actual = activity.ReturnVals

		case ActivityTypePanic:
			// check value
			expected = expectedActivity.PanicVal
			actual = activity.PanicVal
		case ActivityTypeUnset:
			return -1
		}

		if !reflect.DeepEqual(actual, expected) {
			continue
		}

		return index
	}

	return -1
}

// TODO: use wrapDependencyFunc?
func wrapFuncField(tester Tester, funcField fieldPair, calls chan FuncActivity) {
	name := funcField.Type.Name

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := funcField.Type.Type

	funcAsValue := makeFunc(tester, funcType, name, calls)

	// Make a function of the right type.
	// Ignore the type assertion lint check - we are depending on MakeFunc to
	// return the correct type, as documented. If it fails to, the only thing
	// we'd do is panic anyway.

	funcField.Value.Set(funcAsValue)
}
