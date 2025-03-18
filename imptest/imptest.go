package imptest

// Package imptest provides impure function testing functionality.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/akedrou/textdiff"
	"github.com/davecgh/go-spew/spew"
)

// ==Error philosophy==
// Failures: conditions which signal expected failures the user is testing for (this is a test library), should
// trigger a test failure.

// Panics: conditions which signal an error which it is not generally reasonable to expect a caller to recover from,
// which instead imply programmer intervention is necessary to resolve, should trigger an explanatory panic for the
// programmer to track down.

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
	PanicStack string
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

//go:generate stringer -type ActivityType

// Call represents a call to a dependency, as well as providing the channel to send the response that
// the depenency should perform.
type Call struct {
	ID           string            `json:"id"`
	Args         []any             `json:"args"`
	ResponseChan chan CallResponse `json:"-"`
	Type         reflect.Type      `json:"-"`
	t            Tester            `json:"-"`
}

// CallResponse is essentially a union struct representing the various responses a mimicked dependency call
// can be asked to perform:
// * panic
// * return.
type CallResponse struct {
	// TODO: type isn't really necessary, is it?
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

// TODO move these to L2 exported methods - these weren't added till we moved to an L2 API
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

func (c *Call) String() string {
	pretty, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return fmt.Sprintf("couldn't json marshal: %#v (%s)", c, err.Error())
	}

	return string(pretty)
}

func (fa *FuncActivity) String() string {
	switch fa.Type {
	case ActivityTypeReturn:
		return "Return:\n" + prettyString(fa.ReturnVals)
	case ActivityTypePanic:
		return "Panic:\n" + fmt.Sprint(fa.PanicVal) + "\nPanic Trace:\n" + fa.PanicStack
	case ActivityTypeCall:
		return "Call:\n" + fa.Call.String()
	case ActivityTypeUnset:
		panic("unset activity type")
	default:
		panic("unknown activity type")
	}
}

// ==L1 Exported Funcs==

// MimicDependency mimics a given dependency. Instead of a call performing the dependency's logic, the returned mimic
// sends the call signature to the dependency on the given activity channel, waits for a response command, and then
// executes that command by either returning or panicking with the given values.
func MimicDependency[T any](
	tester Tester, function T, activityChan chan FuncActivity, modifiers ...MimicOptionModifier,
) (T, string) {
	name := getDependencyName(function, modifiers)
	funcType := reflect.TypeOf(function)
	mimicAsValue := makeMimicAsValue(tester, funcType, name, activityChan)
	// Ignore the type assertion lint check - we are depending on reflect.MakeFunc to
	// return the correct type, as documented. If it fails to, the only thing
	// we'd do is panic anyway.
	typedMimic := mimicAsValue.Interface().(T) //nolint:forcetypeassert

	// returns both the wrapped func and the ID
	return typedMimic, name
}

// WithName is a modifier which sets the dependency's name.
func WithName(name string) MimicOptionModifier {
	return func(wo MimicOptions) MimicOptions {
		wo.name = name
		return wo
	}
}

// ==L1 Unexported Helpers==.
func getDependencyName(function function, modifiers []MimicOptionModifier) string {
	options := MimicOptions{name: getFuncName(function)}
	for _, modify := range modifiers {
		options = modify(options)
	}

	return options.name
}

func makeMimicAsValue(tester Tester, funcType reflect.Type, name string, activityChan chan FuncActivity) reflect.Value {
	// create the function, that when called:
	// * puts its name and args onto the call channel along with a return channel
	// * waits until the return channel has something, and then returns that
	reflectedMimic := func(args []reflect.Value) []reflect.Value {
		// Create a channel to receive injected output values on
		responseChan := make(chan CallResponse)

		// Submit this call to the calls channel
		activityChan <- FuncActivity{
			ActivityTypeCall,
			nil,
			"",
			nil,
			&Call{
				// TODO: turn ID into name
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
			return convertReturnValues(responseToActivity, funcType)
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

func convertReturnValues(outputV CallResponse, funcType reflect.Type) []reflect.Value {
	returnValues := make([]reflect.Value, len(outputV.ReturnValues))

	for i, a := range outputV.ReturnValues {
		returnValues[i] = reflect.ValueOf(a)
		// if any of these are nil, make them typed nils - otherwise makefunc panics
		if a == nil {
			returnValues[i] = reflect.Zero(funcType.Out(i))
		}
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
	args := make([]any, len(rArgs))

	for i := range rArgs {
		args[i] = rArgs[i].Interface()
	}

	return args
}

// function is here to help us distinguish functions internally, because there is no single
// function _type_ in go.
type function any

// I generally don't like using inits, but this is literally updating global config when this module gets loaded.
// There's no other place to do this besides in another function, which would then need to be protected by mutex's.
func init() { //nolint:gochecknoinits
	spew.Config.SortKeys = true
	spew.Config.SpewKeys = true
	spew.Config.DisablePointerAddresses = true
	spew.Config.DisableCapacities = true
}

func prettyString(a any) string {
	return spew.Sprint(a)
}

// ==L2 Exported Types==.
type Imp struct {
	ActivityChan          chan FuncActivity
	T                     Tester
	ActivityReadMutex     sync.Mutex
	ResolutionMaxDuration time.Duration
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
}

func (t *Imp) Start(f any, args ...any) *Imp {
	go t.startFunctionUnderTest(f, args)

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
				string(debug.Stack()),
				nil,
				nil,
			}
		} else {
			t.ActivityChan <- FuncActivity{
				ActivityTypeReturn,
				nil,
				"",
				rVals,
				nil,
			}
		}
	}()

	rVals = callFunc(function, args)
}

func (t *Imp) ExpectCall(expectedCallID string, expectedArgs ...any) *Call {
	t.T.Helper()

	expected := FuncActivity{
		ActivityTypeCall,
		nil,
		"",
		nil,
		&Call{
			expectedCallID,
			expectedArgs,
			nil,
			nil,
			nil,
		},
	}

	return t.resolveExpectations(expected).Call
}

func (t *Imp) ExpectReturn(returned ...any) {
	t.T.Helper()

	expected := FuncActivity{
		ActivityTypeReturn,
		nil,
		"",
		returned,
		nil,
	}

	t.resolveExpectations(expected)
}

func (t *Imp) ExpectPanic(panicValue any) {
	t.T.Helper()

	expected := FuncActivity{
		ActivityTypePanic,
		panicValue,
		"",
		nil,
		nil,
	}

	t.resolveExpectations(expected)
}

func (t *Imp) Concurrently(funcs ...func()) {
	waitGroup := sync.WaitGroup{}

	// start all the flows
	for index := range funcs {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()
			funcs[index]()
		}()
	}

	waitGroup.Wait()
}

// ==L2 funcs==

// NewImp creates a new imp to help you test without being so verbose.
func NewImp(tester Tester, funcStructs ...any) *Imp {
	tester2 := &Imp{
		ActivityChan:          make(chan FuncActivity, constDefaultActivityBufferSize),
		T:                     tester,
		ResolutionMaxDuration: defaultResolutionMaxDuration,
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

		for index := range numFields {
			if fields[index].Type.Type.Kind() != reflect.Func {
				continue
			}

			functionFields = append(functionFields, fields[index])
		}

		// intercept them all
		// TODO: simplify - all I do with the func fields is this - I probably don't need another loop here
		for i := range functionFields {
			replaceFuncFieldWithMimic(tester, functionFields[i], tester2.ActivityChan)
		}
	}

	return tester2
}

// ==L2 Unexported Helpers==.
const (
	constDefaultActivityBufferSize = 100 // should be enough concurrency for anyone?
	constInvalidIndex              = -1
	defaultResolutionMaxDuration   = 100 * time.Millisecond
)

func matchActivity(activity, expectedActivity FuncActivity) bool {
	if activityTypeMismatch(activity, expectedActivity) {
		return false
	}

	if callIDMismatch(expectedActivity, activity) {
		return false
	}

	if activityValueMismatch(expectedActivity, activity) {
		return false
	}

	return true
}

func activityValueMismatch(expectedActivity FuncActivity, activity FuncActivity) bool {
	expected, actual := extractActivityValues(expectedActivity, activity)

	if valueLenMismatch(expected, actual) {
		return true
	}

	if valueMismatch(actual, expected) {
		return true
	}

	return false
}

func valueMismatch(actual []any, expected []any) bool {
	for index := range actual {
		if isNil(actual[index]) && isNil(expected[index]) {
			continue
		}

		if reflect.DeepEqual(actual[index], expected[index]) {
			continue
		}

		return true
	}

	return false
}

func valueLenMismatch(expected []any, actual []any) bool {
	return len(expected) != len(actual)
}

func extractActivityValues(expectedActivity FuncActivity, activity FuncActivity) ([]any, []any) {
	var expected, actual []any

	switch expectedActivity.Type {
	case ActivityTypeCall:
		expected = expectedActivity.Call.Args
		actual = activity.Call.Args
	case ActivityTypeReturn:
		expected = expectedActivity.ReturnVals
		actual = activity.ReturnVals
	case ActivityTypePanic:
		expected = []any{expectedActivity.PanicVal}
		actual = []any{activity.PanicVal}
	case ActivityTypeUnset:
		panic("tried to match against an unset activity type, and that should never happen")
	default:
		panic("tried to match against an unknown activity type, and that should never happen")
	}

	return expected, actual
}

func callIDMismatch(expectedActivity FuncActivity, activity FuncActivity) bool {
	switch expectedActivity.Type {
	case ActivityTypeCall:
		if activity.Call.ID != expectedActivity.Call.ID {
			return true
		}

	case ActivityTypeReturn:
	case ActivityTypePanic:
	case ActivityTypeUnset:
		panic("tried to match against an unset activity type, and that should never happen")
	}

	return false
}

func activityTypeMismatch(activity FuncActivity, expectedActivity FuncActivity) bool {
	return activity.Type != expectedActivity.Type
}

func (t *Imp) resolveExpectations(expectation FuncActivity) FuncActivity {
	t.ActivityReadMutex.Lock()
	defer t.ActivityReadMutex.Unlock()

	expectedActivity := expectation
	activities := []FuncActivity{}
	maxWaitChan := time.After(t.ResolutionMaxDuration)

Loop:
	for {
		select {
		case activity := <-t.ActivityChan:
			if !matchActivity(activity, expectedActivity) {
				activities = append(activities, activity)
			} else {
				for i := range activities {
					t.ActivityChan <- activities[i]
				}

				return activity
			}

		case <-maxWaitChan:
			break Loop
		}
	}

	failureMessage := fmt.Sprintf("expected %s, but got different function activity:", prettyString(expectedActivity))
	diffs := make([]string, len(activities))

	for index := range activities {
		diffs[index] = textdiff.Unified(
			"expected",
			"actual",
			prettyString(expectedActivity)+"\n",
			prettyString(activities[index])+"\n",
		)
		failureMessage += "\n" + fmt.Sprintf("diff %d: \n%s", index, diffs[index])
	}

	t.T.Fatal(failureMessage + "\n" + "no function activity matched the expectation")
	panic("should never get here, due to the preceding Fatal call")
}

type ExpectationResponse struct {
	Match  *FuncActivity
	Misses []FuncActivity
}

// callFunc calls the given function with the given args, and returns the return values from that callFunc.
func callFunc(f function, args []any) []any {
	rf := reflect.ValueOf(f)
	rArgs := reflectValuesOf(args, reflect.TypeOf(f))
	rReturns := rf.Call(rArgs)

	return unreflectValues(rReturns)
}

// reflectValuesOf returns reflected values for all of the values.
func reflectValuesOf(args []any, funcType reflect.Type) []reflect.Value {
	rArgs := make([]reflect.Value, len(args))

	for i := range args {
		arg := args[i]
		if arg == nil {
			rArgs[i] = reflect.Zero(funcType.In(i))
		} else {
			rArgs[i] = reflect.ValueOf(args[i])
		}
	}

	return rArgs
}

type fieldPair struct {
	Type  reflect.StructField
	Value reflect.Value
}

func isNil(thing any) (toReturn bool) {
	defer func() {
		// don't bother checking - this is just to catch the panic and prevent it from bubbling up.
		// if no panic, then we're returning whatever we were told to.
		// if panic, returning cleanly with the default toReturn value (false).
		// we onlyot.
		recover() //nolint:errcheck
	}()

	value := reflect.ValueOf(thing)

	if !value.IsValid() {
		return true
	}

	if value.IsNil() {
		return true
	}

	return
}

// replaceFuncFieldWithMimic mimics a given dependency struct func field and replaces it. Instead of a call performing
// the dependency's logic, calling now runs the mimic, which sends the call signature to the dependency on the given
// activity channel, waits for a response command, and then executes that command by either returning or panicking with
// the given values.
func replaceFuncFieldWithMimic(tester Tester, funcField fieldPair, activityChan chan FuncActivity) {
	name := funcField.Type.Name
	funcType := funcField.Type.Type
	mimicAsValue := makeMimicAsValue(tester, funcType, name, activityChan)

	funcField.Value.Set(mimicAsValue)
}
