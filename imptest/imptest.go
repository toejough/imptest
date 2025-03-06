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

// =The Core types and functions that let any of this happen=

// MimicDependency mocks the given function, creating a named coroutine from it, to be used in testing.
// The coroutine will yield a Call onto the given YieldedValue channel whenever it is called.
// This Call can be checked for its name (ID) and args. The calling function will wait for the
// Call.Return or Call.Panic methods to be called. Calling either of those methods will cause the
// coroutine to return or panic with the values as passed.
func MimicDependency[T any](tester Tester, function T, calls chan FuncActivity, options ...WrapOption) (T, string) {
	opts := WrapOptions{name: getFuncName(function)}
	for _, o := range options {
		opts = o(opts)
	}

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := reflect.TypeOf(function)

	relayer := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		injectedValueChan := make(chan DependencyResponse)

		// Submit this call to the calls channel
		calls <- FuncActivity{
			DependencyCallActivityType,
			nil,
			nil,
			&Call{
				opts.name,
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
	wrapped := reflect.MakeFunc(funcType, relayer).Interface().(T) //nolint:forcetypeassert

	// returns both the wrapped func and the ID
	return wrapped, opts.name
}

type FuncActivity struct {
	Type           YieldType
	PanicVal       any
	ReturnVals     []any
	DependencyCall *Call
}

type YieldType int

const (
	noActivityType             YieldType = iota
	ReturnActivityType         YieldType = iota
	PanicActivityType          YieldType = iota
	DependencyCallActivityType YieldType = iota
)

type Call struct {
	ID           string
	Args         []any
	ResponseChan chan DependencyResponse
	Type         reflect.Type
	t            Tester
}

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
	c.ResponseChan <- DependencyResponse{
		ReturnResponseType,
		returnVals,
		nil,
	}
	close(c.ResponseChan)
}

// Panic makes the func call result in a panic with the given value.
func (c *Call) Panic(panicVal any) {
	c.ResponseChan <- DependencyResponse{
		PanicResponseType,
		nil,
		panicVal,
	}
	close(c.ResponseChan)
}

type DependencyResponse struct {
	Type         injectionType
	ReturnValues []any
	PanicValue   any
}

type injectionType int

const (
	ReturnResponseType injectionType = iota
	PanicResponseType  injectionType = iota
)

type WrapOption func(WrapOptions) WrapOptions

type WrapOptions struct {
	name string
}

func WithName(name string) WrapOption {
	return func(wo WrapOptions) WrapOptions {
		wo.name = name
		return wo
	}
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

type Tester interface {
	Helper()
	Fatal(args ...any)
	Fatalf(message string, args ...any)
	Logf(message string, args ...any)
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

type fieldPair struct {
	Type  reflect.StructField
	Value reflect.Value
}

type Tester2 struct {
	ft              *FuncTester
	concurrency     atomic.Int64
	expectationChan chan expectation
}

type expectation struct {
	activity     FuncActivity
	responseChan chan expectationResponse
}

type expectationResponse struct {
	match  *FuncActivity
	misses []FuncActivity
}

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
		case DependencyCallActivityType:
			// check ID
			if activity.DependencyCall.ID != expectedActivity.DependencyCall.ID {
				continue
			}

			// check args
			expected = expectedActivity.DependencyCall.Args
			actual = activity.DependencyCall.Args

		case ReturnActivityType:
			// check values
			expected = expectedActivity.ReturnVals
			actual = activity.ReturnVals

		case PanicActivityType:
			// check value
			expected = expectedActivity.PanicVal
			actual = activity.PanicVal
		case noActivityType:
			return -1
		}

		if !reflect.DeepEqual(actual, expected) {
			continue
		}

		return index
	}

	return -1
}

func (t *Tester2) ReceiveCall(expectedCallID string, expectedArgs ...any) *Call {
	t.ft.T.Helper()
	// t.ft.T.Logf("receiving call")

	expected := expectation{
		FuncActivity{
			DependencyCallActivityType,
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

	return response.match.DependencyCall
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

func (c *Call) SendReturn(returns ...any) {
	c.t.Helper()
	c.Return(returns...)
}

func (c *Call) SendPanic(panicValue any) {
	c.t.Helper()
	c.Panic(panicValue)
}

func wrapFuncField(tester Tester, funcField fieldPair, calls chan FuncActivity) {
	name := funcField.Type.Name

	// create the function, that when called:
	// * puts its ID and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	funcType := funcField.Type.Type

	relayer := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		injectedValueChan := make(chan DependencyResponse)

		// Submit this call to the calls channel
		calls <- FuncActivity{
			DependencyCallActivityType,
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
