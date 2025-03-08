// Package imptest provides impure function testing functionality.
package imptest

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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

// ==L1 Types==

type FuncActivity struct {
	Type       ActivityType
	PanicVal   any
	ReturnVals []any
	Call       *Call
}

type ActivityType int

const (
	UnsetActivityType  ActivityType = iota
	ReturnActivityType ActivityType = iota
	PanicActivityType  ActivityType = iota
	CallActivityType   ActivityType = iota
)

type Call struct {
	ID           string
	Args         []any
	ResponseChan chan CallResponse
	Type         reflect.Type
	t            Tester
}

type CallResponse struct {
	Type         ResponseType
	ReturnValues []any
	PanicValue   any
}

type ResponseType int

const (
	ReturnResponseType ResponseType = iota
	PanicResponseType  ResponseType = iota
)

type MimicOption func(MimicOptions) MimicOptions

type MimicOptions struct {
	name string
}

// ==L1 Methods==

// Return returns the given values in the func call.
func (c *Call) Return(returnVals ...any) {
	c.t.Helper()
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
	c.ResponseChan <- CallResponse{
		ReturnResponseType,
		returnVals,
		nil,
	}
	close(c.ResponseChan)
}

// Panic makes the func call result in a panic with the given value.
func (c *Call) Panic(panicVal any) {
	c.ResponseChan <- CallResponse{
		PanicResponseType,
		nil,
		panicVal,
	}
	close(c.ResponseChan)
}

func (c *Call) SendReturn(returns ...any) {
	c.t.Helper()
	c.Return(returns...)
}

func (c *Call) SendPanic(panicValue any) {
	c.t.Helper()
	c.Panic(panicValue)
}

// ==L1 Funcs==

// MimicDependency mimics a given dependency. Instead of calling performing the dependency's logic, the mimic sends the
// call signature to the dependency on the given activity channel, waits for a response command, and then executes that
// command by either returning or panicking with the given values.
func MimicDependency[T any](
	tester Tester, function T, activityChan chan FuncActivity, options ...MimicOption,
) (T, string) {
	opts := MimicOptions{name: getFuncName(function)}
	for _, o := range options {
		opts = o(opts)
	}

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := reflect.TypeOf(function)

	reflectedMimic := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		responseChan := make(chan CallResponse)

		// Submit this call to the calls channel
		activityChan <- FuncActivity{
			CallActivityType,
			nil,
			nil,
			&Call{
				opts.name,
				unreflectValues(args),
				responseChan,
				funcType,
				tester,
			},
		}

		outputV := <-responseChan

		switch outputV.Type {
		case ReturnResponseType:
			returnValues := make([]reflect.Value, len(outputV.ReturnValues))

			// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
			for i, a := range outputV.ReturnValues {
				returnValues[i] = reflect.ValueOf(a)
			}

			return returnValues
		// if we're supposed to panic, do.
		case PanicResponseType:
			panic(outputV.PanicValue)
		default:
			panic("imptest failure - unrecognized outputValue type was passed")
		}
	}

	// Make a function of the right type.
	// Ignore the type assertion lint check - we are depending on MakeFunc to
	// return the correct type, as documented. If it fails to, the only thing
	// we'd do is panic anyway.
	typedMimic := reflect.MakeFunc(funcType, reflectedMimic).Interface().(T) //nolint:forcetypeassert

	// returns both the wrapped func and the ID
	return typedMimic, opts.name
}

func WithName(name string) MimicOption {
	return func(wo MimicOptions) MimicOptions {
		wo.name = name
		return wo
	}
}

// ==L1 Helpers==

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

// ==L2 types==.
type Tester2 struct {
	ft              *FuncTester
	concurrency     atomic.Int64
	expectationChan chan expectation
}

type Tester interface {
	Helper()
	Fatal(args ...any)
	Fatalf(message string, args ...any)
	Logf(message string, args ...any)
}

// ==L2 methods==

func (t *Tester2) Close() {
	close(t.ft.OutputChan)
	close(t.expectationChan)
}

func (t *Tester2) Start(f any, args ...any) *Tester2 {
	// start the test
	t.ft.Start(f, args...)

	go t.matchActivitiesToExpectations()

	return t
}

func (t *Tester2) matchActivitiesToExpectations() {
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

func (t *Tester2) updateActivitiesAndExpectations(
	expectations []expectation, activities []FuncActivity,
) ([]expectation, []FuncActivity, bool) {
	select {
	case expectation, ok := <-t.expectationChan:
		if !ok {
			return nil, nil, true
		}

		expectations = append(expectations, expectation)
	case activity, ok := <-t.ft.OutputChan:
		if !ok {
			return nil, nil, true
		}

		activities = append(activities, activity)
	}

	return expectations, activities, false
}

func (t *Tester2) ReceiveCall(expectedCallID string, expectedArgs ...any) *Call {
	t.ft.T.Helper()
	// t.ft.T.Logf("receiving call")

	expected := expectation{
		FuncActivity{
			CallActivityType,
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
		t.ft.T.Fatalf("expected %v, but got %v", expected.activity, response.misses)
	}

	return response.match.Call
}

func (t *Tester2) ReceiveReturn(returned ...any) {
	t.ft.T.Helper()
	t.ft.T.Logf("receiving return")

	expected := expectation{
		FuncActivity{
			ReturnActivityType,
			nil,
			returned,
			nil,
		},
		make(chan expectationResponse),
	}

	t.expectationChan <- expected
	response := <-expected.responseChan

	if response.match == nil {
		t.ft.T.Fatalf("expected %v, but got %v", expected.activity, response.misses)
	}
}

func (t *Tester2) ReceivePanic(panicValue any) {
	t.ft.T.Helper()
	t.ft.T.Logf("receiving panic")

	expected := expectation{
		FuncActivity{
			PanicActivityType,
			panicValue,
			nil,
			nil,
		},
		make(chan expectationResponse),
	}

	t.expectationChan <- expected
	response := <-expected.responseChan

	if response.match == nil {
		t.ft.T.Fatalf("expected %v, but got %v", expected.activity, response.misses)
	}
}

func (t *Tester2) Concurrently(funcs ...func()) {
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
func NewImp(tester Tester, funcStructs ...any) *Tester2 {
	ftester := NewFuncTester(tester)
	tester2 := &Tester2{ft: ftester, concurrency: atomic.Int64{}, expectationChan: make(chan expectation)}

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
			wrapFuncField(tester, functionFields[i], ftester.OutputChan)
		}
	}

	return tester2
}

// ==L2 helpers==.
type expectation struct {
	activity     FuncActivity
	responseChan chan expectationResponse
}

type expectationResponse struct {
	match  *FuncActivity
	misses []FuncActivity
}

// =Test Simplification and readability abstractions=

// NewFuncTester returns a newly initialized FuncTester.
func NewFuncTester(tester Tester) *FuncTester {
	tester.Helper()

	calls := make(chan FuncActivity)

	funcTester := new(FuncTester)
	funcTester.T = tester
	funcTester.OutputChan = calls
	// I want this to be a magic number, it's half a second
	funcTester.Timeout = 500 * time.Millisecond //nolint:mnd,gomnd
	funcTester.Differ = func(actual, expected any) (string, error) {
		if !reflect.DeepEqual(actual, expected) {
			return fmt.Sprintf("actual: %#v\nexpected: %#v", actual, expected), nil
		}

		return "", nil
	}

	return funcTester
}

// Tester contains the *testing.T and the chan FuncCall.
type FuncTester struct {
	T          Tester
	OutputChan chan FuncActivity
	Timeout    time.Duration
	Differ     Differ
}

// Start starts the function.
func (t *FuncTester) Start(function any, args ...any) {
	// record when the func is done so we can test that, too
	go func() {
		var rVals []any

		defer func() {
			panicVal := recover()
			if panicVal != nil {
				t.OutputChan <- FuncActivity{
					PanicActivityType,
					panicVal,
					nil,
					nil,
				}
			} else {
				t.OutputChan <- FuncActivity{
					ReturnActivityType,
					nil,
					rVals,
					nil,
				}
			}
		}()

		rVals = callFunc(function, args)
	}()
}

type Differ func(any, any) (string, error)

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
		case CallActivityType:
			// check ID
			if activity.Call.ID != expectedActivity.Call.ID {
				continue
			}

			// check args
			expected = expectedActivity.Call.Args
			actual = activity.Call.Args

		case ReturnActivityType:
			// check values
			expected = expectedActivity.ReturnVals
			actual = activity.ReturnVals

		case PanicActivityType:
			// check value
			expected = expectedActivity.PanicVal
			actual = activity.PanicVal
		case UnsetActivityType:
			return -1
		}

		if !reflect.DeepEqual(actual, expected) {
			continue
		}

		return index
	}

	return -1
}

func wrapFuncField(tester Tester, funcField fieldPair, calls chan FuncActivity) {
	name := funcField.Type.Name

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := funcField.Type.Type

	relayer := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		injectedValueChan := make(chan CallResponse)

		// Submit this call to the calls channel
		calls <- FuncActivity{
			CallActivityType,
			nil,
			nil,
			&Call{
				name,
				unreflectValues(args),
				injectedValueChan,
				funcType,
				tester,
			},
		}

		outputV := <-injectedValueChan

		switch outputV.Type {
		case ReturnResponseType:
			returnValues := make([]reflect.Value, len(outputV.ReturnValues))

			// Convert return values to reflect.Values, to meet the required reflect.MakeFunc signature
			for rvi, a := range outputV.ReturnValues {
				// special casing to avoid a completely nil return hosing things up
				v := reflect.ValueOf(a)
				if !v.IsValid() {
					v = reflect.Zero(funcType.Out(rvi))
				}

				returnValues[rvi] = v
			}

			return returnValues
		// if we're supposed to panic, do.
		case PanicResponseType:
			panic(outputV.PanicValue)
		default:
			panic("imptest failure - unrecognized outputValue type was passed")
		}
	}

	// Make a function of the right type.
	// Ignore the type assertion lint check - we are depending on MakeFunc to
	// return the correct type, as documented. If it fails to, the only thing
	// we'd do is panic anyway.
	wrappedValue := reflect.MakeFunc(funcType, relayer)

	funcField.Value.Set(wrappedValue)
}
