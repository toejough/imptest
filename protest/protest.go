// Package protest provides procedure testing functionality.
// Why not https://github.com/stretchr/testify/blob/master/README.md#mock-package?
// You only get to specify simple call/return behavior, with no guarantees about ordering, and you need to unset
// handlers for repeated calls for the same function.
// On the other hand, there's https://github.com/stretchr/testify/issues/741.  Is this necessary?
// maybe this whole suite is pointless?
// A win over testify mocked: when the there's a test failure, we don't panic.
// a win over testify.Mocked: we fail if the number of returns doesn't match
package protest

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// RelayTester methods.
func (rt *RelayTester) Start(function Function, args ...any) {
	go func() {
		rArgs := make([]reflect.Value, len(args))
		for i := range args {
			rArgs[i] = reflect.ValueOf(args[i])
		}

		rt.returns = reflect.ValueOf(function).Call(rArgs)

		rt.Relay.Shutdown()
	}()
}

type (
	RelayTester struct {
		T       Tester
		Relay   *CallRelay
		returns []reflect.Value
	}
	Tester interface {
		Helper()
		Fatalf(message string, args ...any)
	}
	CallRelay struct {
		callChan chan *Call
	}
	Call struct {
		function Function
		args     []any
		returns  chan []any
	}
	Function any
)

var (
	errCallRelayNotShutDown     = errors.New("call relay was not shut down")
	errCallRelayShutdownTimeout = errors.New("call relay timed out waiting for shutdown")
)

// Public helpers.
func AssertNextCallIs(t Tester, r *CallRelay, name string, expectedArgs ...any) *Call {
	t.Helper()

	c := r.Get()
	assertCalledNameIs(t, c, name)
	assertArgsAre(t, c, expectedArgs...)

	return c
}

func AssertRelayShutsDownWithin(t Tester, relay *CallRelay, waitTime time.Duration) {
	t.Helper()

	if err := relay.WaitForShutdown(waitTime); err != nil {
		t.Fatalf("the relay has not shut down yet: %s", err)
	}
}

func NewCall(f Function, args ...any) *Call {
	panicIfNotFunc(f, NewCall)
	return &Call{function: f, args: args, returns: make(chan []any)}
}

func NewCallNoReturn(f Function, args ...any) *Call {
	panicIfNotFunc(f, NewCallNoReturn)
	return &Call{function: f, args: args, returns: nil}
}

func NewCallRelay() *CallRelay {
	return &CallRelay{callChan: make(chan *Call)}
}

// Private helpers.
func assertCalledNameIs(t Tester, c *Call, expectedName string) {
	t.Helper()

	if c.Name() != expectedName {
		t.Fatalf("the called function was expected to be %s, but was %s instead", expectedName, c.Name())
	}
}

func assertArgsAre(tester Tester, theCall *Call, expectedArgs ...any) {
	tester.Helper()

	if theCall.Args() == nil && expectedArgs != nil {
		tester.Fatalf(
			"the function %s was expected to be called with %#v, but was called without args",
			theCall.Name(),
			expectedArgs,
		)
	}

	if theCall.Args() != nil && expectedArgs == nil {
		tester.Fatalf(
			"the function %s was expected to be called without args, but was called with %#v",
			theCall.Name(),
			theCall.Args(),
		)
	}

	if !reflect.DeepEqual(theCall.Args(), expectedArgs) {
		tester.Fatalf("the function %s was expected to be called with %#v but was called with %#v",
			theCall.Name(), expectedArgs, theCall.Args(),
		)
	}
}

func getFuncName(f Function) string {
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	name = strings.TrimSuffix(name, "-fm")

	return name
}

func panicIfNotFunc(evaluate Function, from Function) {
	kind := reflect.ValueOf(evaluate).Kind()
	if kind != reflect.Func {
		panic(fmt.Sprintf("must pass a function as the first argument to %s. received a %s instead.",
			getFuncName(from),
			kind.String(),
		))
	}
}

// CallRelay Methods.
func (cr *CallRelay) Get() *Call {
	select {
	case c, ok := <-cr.callChan:
		if !ok {
			panic("expected a call, but the relay was already shut down")
		}

		return c
	case <-time.After(time.Second):
		panic("testing timeout waiting for a call")
	}
}

func (cr *CallRelay) Put(c *Call) *Call {
	cr.callChan <- c
	return c
}

func (cr *CallRelay) Shutdown() {
	close(cr.callChan)
}

func (cr *CallRelay) WaitForShutdown(waitTime time.Duration) error {
	select {
	case thisCall, ok := <-cr.callChan:
		if !ok {
			// channel is closed
			return nil
		}

		return fmt.Errorf("had a call queued: %v: %w", thisCall, errCallRelayNotShutDown)
	case <-time.After(waitTime):
		return errCallRelayShutdownTimeout
	}
}

func (cr *CallRelay) PutCall(function Function, args ...any) *Call {
	supportedNumArgs := reflect.TypeOf(function).NumIn()
	expectedNumArgs := len(args)

	if expectedNumArgs != supportedNumArgs {
		panic(fmt.Sprintf(
			"the length of the expected argument list (%d)"+
				" does not equal the length of the arguments (%s) supports (%d)",
			expectedNumArgs,
			getFuncName(function),
			supportedNumArgs,
		))
	}

	return cr.Put(NewCall(function, args...))
}

func (cr *CallRelay) PutCallNoReturn(f Function, args ...any) *Call {
	return cr.Put(NewCallNoReturn(f, args...))
}

// Call methods.
func (c Call) Name() string {
	return getFuncName(c.function)
}

func (c Call) Args() []any {
	return c.args
}

func (c Call) InjectReturns(returnValues ...any) {
	if c.returns == nil {
		panic("cannot inject a return on a call with no returns")
	}

	supportedNumReturns := reflect.TypeOf(c.function).NumOut()
	injectedNumReturns := len(returnValues)

	if injectedNumReturns != supportedNumReturns {
		panic(fmt.Sprintf(
			"the length of the injected return list (%d)"+
				" does not equal the length of the returns (%s) supports (%d)",
			injectedNumReturns,
			getFuncName(c.function),
			supportedNumReturns,
		))
	}

	select {
	case c.returns <- returnValues:
		return
	case <-time.After(1 * time.Second):
		panic("timed out waiting for " + c.Name() + " to read the injected return values")
	}
}

func (c Call) GetReturns() []any {
	return <-c.returns
}

func (c Call) FillReturns(returnPointers ...any) {
	returnValues := <-c.returns

	if len(returnPointers) != len(returnValues) {
		panic(fmt.Sprintf(
			"the length of the pointer array to fill with return values (%d) does not match the "+
				" length of the return value array injected by the test (%d)",
			len(returnPointers),
			len(returnValues),
		))
	}

	for index := range returnValues {
		// USEFUL SNIPPETS FROM JSON.UNMARSHAL
		// if returnPointerValue.Kind() != reflect.Pointer || returnPointerValue.IsNil() {
		// 	return &InvalidUnmarshalError{reflect.TypeOf(v)}
		// }
		// v.Set(reflect.ValueOf(oi))
		returnPointerValue := reflect.ValueOf(returnPointers[index])
		if returnPointerValue.Kind() != reflect.Pointer || returnPointerValue.IsNil() {
			panic("cannot fill value into non-pointer")
		}

		returnedValue := reflect.ValueOf(returnValues[index])
		// handle nils
		if !returnedValue.IsValid() {
			returnPointerValue.Elem().SetZero()
			continue
		}
		// Use Elem instead of directly using Set for setting pointers
		returnPointerValue.Elem().Set(returnedValue)
	}
}

func (rt *RelayTester) AssertNextCallIs(function Function, args ...any) *Call {
	rt.T.Helper()
	panicIfNotFunc(function, AssertNextCallIs)

	supportedNumArgs := reflect.TypeOf(function).NumIn()
	expectedNumArgs := len(args)

	if expectedNumArgs != supportedNumArgs {
		panic(fmt.Sprintf(
			"the length of the expected argument list (%d)"+
				" does not equal the length of the arguments (%s) supports (%d)",
			expectedNumArgs,
			getFuncName(function),
			supportedNumArgs,
		))
	}

	return AssertNextCallIs(rt.T, rt.Relay, getFuncName(function), args...)
}

func (rt *RelayTester) AssertDoneWithin(d time.Duration) {
	rt.T.Helper()
	AssertRelayShutsDownWithin(rt.T, rt.Relay, d)
}

func (rt *RelayTester) AssertReturned(args ...any) {
	lenReturns := len(rt.returns)
	lenArgs := len(args)

	if lenReturns != lenArgs {
		rt.T.Fatalf("The function returned %d values, but the test asserted %d returns", lenReturns, lenArgs)
	}

	for i := range args {
		if !reflect.DeepEqual(rt.returns[i].Interface(), args[i]) {
			rt.T.Fatalf("the return value at index %d was expected to be %#v but it was %#v",
				i, args[i], rt.returns[i],
			)
		}
	}
}
