// Package imptest provides impure function testing functionality.
package imptest

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
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

	calls := make(chan FuncActivity, 100)

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

			close(t.OutputChan)
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
	tester2 := &Tester2{ft: ftester, Concurrency: 1}

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
	ft          *FuncTester
	Concurrency int
}

func (t *Tester2) Start(f any, args ...any) *Tester2 {
	// start the test
	t.ft.Start(f, args...)

	return t
}

func (t *Tester2) ReceiveCall(expectedCallID string, expectedArgs ...any) *Call {
	t.ft.T.Helper()
	t.ft.T.Logf("receiving call")

	errors := []string{}
	activities := []FuncActivity{}

	for {
		t.ft.T.Logf("looping in call")
		activity := <-t.ft.OutputChan
		if activity.Type != DependencyCallActivityType {
			errors = append(errors, "Expected to receive a dependency call but instead received...FIXME")
			activities = append(activities, activity)

			if len(errors) >= t.Concurrency {
				t.ft.T.Fatalf(strings.Join(errors, "\n"))
			}

			continue
		}

		// and the dependency call is to the mimicked dependency
		if activity.DependencyCall.ID != expectedCallID {
			errors = append(errors, fmt.Sprintf("Expected to receive a call to %s, but instead received a call to %s", expectedCallID, activity.DependencyCall.ID))
			activities = append(activities, activity)

			if len(errors) >= t.Concurrency {
				t.ft.T.Fatalf(strings.Join(errors, "\n"))
			}

			continue
		}

		expected := expectedArgs
		actual := activity.DependencyCall.Args

		if !reflect.DeepEqual(actual, expected) {
			errors = append(errors, fmt.Sprintf("mismatched args. actual: %#v\nexpected: %#v", actual, expected))
			activities = append(activities, activity)

			if len(errors) >= t.Concurrency {
				t.ft.T.Fatalf(strings.Join(errors, "\n"))
			}

			continue
		}

		for i := range activities {
			t.ft.T.Logf("putting an activity back on the queue")
			t.ft.OutputChan <- activities[i]
		}

		return activity.DependencyCall
	}
}

func (t *Tester2) ReceiveReturn(returned ...any) {
	t.ft.T.Helper()
	t.ft.T.Logf("receiving return")

	errors := []string{}
	activities := []FuncActivity{}

	for {
		t.ft.T.Logf("looping in return")
		activity := <-t.ft.OutputChan
		if activity.Type != ReturnActivityType {
			errors = append(errors, fmt.Sprintf("Expected to receive a return activity but instead received...FIXME"))
			activities = append(activities, activity)

			if len(errors) >= t.Concurrency {
				t.ft.T.Fatalf(strings.Join(errors, "\n"))
			}

			continue
		}

		expected := returned
		actual := activity.ReturnVals

		if !reflect.DeepEqual(actual, expected) {
			errors = append(errors, fmt.Sprintf("Mismatched returns. actual: %#v\nexpected: %#v", actual, expected))
			activities = append(activities, activity)

			if len(errors) >= t.Concurrency {
				t.ft.T.Fatalf(strings.Join(errors, "\n"))
			}

			continue
		}

		for i := range activities {
			t.ft.T.Logf("putting an activity back on the queue")
			t.ft.OutputChan <- activities[i]
		}

		return
	}
}

func (t *Tester2) ReceivePanic(panicValue any) {
	t.ft.T.Helper()
	t.ft.T.Logf("receiving panic")

	errors := []string{}
	activities := []FuncActivity{}

	for {
		t.ft.T.Logf("looping in panic")
		activity := <-t.ft.OutputChan
		if activity.Type != PanicActivityType {
			errors = append(errors, fmt.Sprintf("Expected to receive a panic activity but instead received...FIXME"))
			activities = append(activities, activity)

			if len(errors) >= t.Concurrency {
				t.ft.T.Fatalf(strings.Join(errors, "\n"))
			}

			continue
		}

		expected := panicValue
		actual := activity.PanicVal

		if !reflect.DeepEqual(actual, expected) {
			errors = append(errors, fmt.Sprintf("Mismatched panic value. actual: %#v\nexpected: %#v", actual, expected))
			activities = append(activities, activity)

			if len(errors) >= t.Concurrency {
				t.ft.T.Fatalf(strings.Join(errors, "\n"))
			}

			continue
		}

		for i := range activities {
			t.ft.T.Logf("putting an activity back on the queue")
			t.ft.OutputChan <- activities[i]
		}

		return
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
