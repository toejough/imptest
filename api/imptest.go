// Package imptest provides impure function testing functionality.
package imptest

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// RelayTester.
func NewTester(t Tester) *RelayTester {
	return &RelayTester{
		T:        t,
		Relay:    NewCallRelay(),
		function: nil,
		returns:  nil,
	}
}

type RelayTester struct {
	T        Tester
	Relay    *CallRelay
	function Function
	returns  []reflect.Value
}

// RelayTester methods.
func (rt *RelayTester) Start(function Function, args ...any) {
	rt.function = function

	go func() {
		// get args as reflect.Values
		rArgs := make([]reflect.Value, len(args))
		for i := range args {
			rArgs[i] = reflect.ValueOf(args[i])
		}

		// define some cleanup
		defer func() {
			// catch and handle bad args
			if r := recover(); r != nil {
				rt.T.Fatalf("failed to call %s with args (%v): %v", GetFuncName(function), rArgs, r)
			}

			// always shutdown afterwards
			rt.Relay.Shutdown()
		}()

		// actually call the function
		rt.returns = reflect.ValueOf(function).Call(rArgs)
	}()
}

func (rt *RelayTester) AssertDoneWithin(d time.Duration) {
	rt.T.Helper()
	AssertRelayShutsDownWithin(rt.T, rt.Relay, d)
}

func (rt *RelayTester) AssertReturned(assertedReturns ...any) {
	lenReturnsAsserted := len(assertedReturns)

	reflectedFunc := reflect.TypeOf(rt.function)
	numFunctionReturns := reflectedFunc.NumOut()

	if numFunctionReturns > lenReturnsAsserted {
		panic(fmt.Sprintf("Too few return values asserted. The func (%s) returns %d values,"+
			" but only %d were asserted",
			GetFuncName(rt.function),
			numFunctionReturns,
			lenReturnsAsserted,
		))
	} else if numFunctionReturns < lenReturnsAsserted {
		panic(fmt.Sprintf("Too many return values asserted. The func (%s) only returns %d values,"+
			" but %d were asserted",
			GetFuncName(rt.function),
			numFunctionReturns,
			lenReturnsAsserted,
		))
	}

	for index := range assertedReturns {
		returned := rt.returns[index].Interface()
		returnAsserted := assertedReturns[index]
		// if the func type is a pointer and the passed Arg is nil, that's ok, too.
		if returnAsserted == nil && reflectedFunc.Out(index).Kind() == reflect.Pointer {
			continue
		}
		// TODO: investigate undefined types. pointers don't seem to have a name.
		returnType := reflectedFunc.Out(index).Name()
		assertedType := reflect.TypeOf(returnAsserted).Name()

		if returnType != assertedType {
			panic(fmt.Sprintf("Wrong return type asserted. The return at index %d from func (%s) is %s,"+
				" but a value of type %s was asserted",
				index,
				GetFuncName(rt.function),
				returnType,
				assertedType,
			))
		}

		if !reflect.DeepEqual(returned, returnAsserted) {
			rt.T.Fatalf(
				"The func returned the wrong value for a return: "+
					"the return value at index %d was expected to be %#v, "+
					"but it was %#v",
				index, returnAsserted, returned,
			)

			return
		}
	}
}

func (rt *RelayTester) PutCall(f Function, a ...any) *Call { return rt.Relay.PutCall(f, a...) }

func (rt *RelayTester) GetNextCall() *Call {
	call, err := rt.Relay.Get()
	if err != nil {
		rt.T.Fatalf(err.Error())
		return nil
	}

	return call
}

func (rt *RelayTester) AssertNextCallIs(function Function, args ...any) *Call {
	rt.T.Helper()
	panicIfNotFunc(function, rt.AssertNextCallIs)

	call := rt.GetNextCall()

	if rt.T.Failed() {
		return nil
	}

	return AssertCallIs(rt.T, call, function, args...)
}

func (rt *RelayTester) GetReturns() []reflect.Value { return rt.returns }

type (
	Tester interface {
		Helper()
		// FIXME: this really needs to panic, but the testing mock doesn't, and that causes all kinds of other
		// awkward logic in this library...
		Fatalf(message string, args ...any)
		Failed() bool
	}
	CallRelay struct {
		callChan chan *Call
	}
	Call struct {
		function Function
		args     []any
		returns  chan []any
	}
	Function    any
	DelayedCall struct{}
)

var (
	errCallRelayNotShutDown     = errors.New("call relay was not shut down")
	errCallRelayShutdownTimeout = errors.New("call relay timed out waiting for shutdown")
	errCallAfterShutDown        = errors.New("expected a call, but the relay was already shut down")
)

// Public helpers.
func AssertCallIs(tester Tester, called *Call, function Function, expectedArgs ...any) *Call {
	tester.Helper()

	name := GetFuncName(function)
	assertCalledNameIs(tester, called, name)

	reflectedFunc := reflect.TypeOf(function)
	supportedNumArgs := reflectedFunc.NumIn()
	expectedNumArgs := len(expectedArgs)

	if expectedNumArgs < supportedNumArgs {
		panic(fmt.Sprintf(
			"too few args in the expected argument list (%d)"+
				// I want to keep these error messages independent
				" compared to the number of arguments (%s) supports (%d)",
			expectedNumArgs,
			GetFuncName(function),
			supportedNumArgs,
		))
	} else if expectedNumArgs > supportedNumArgs {
		panic(fmt.Sprintf(
			"too many args in the expected argument list (%d)"+
				" compared to the number of arguments (%s) supports (%d)",
			expectedNumArgs,
			GetFuncName(function),
			supportedNumArgs,
		))
	}

	for index := range expectedArgs {
		argAsserted := expectedArgs[index]
		assertedType := reflect.TypeOf(argAsserted).Name()
		actualType := reflectedFunc.In(index).Name()

		if assertedType != actualType {
			panic(fmt.Sprintf(
				"Wrong type."+
					"The type asserted for the arg at index %d is %s,"+
					"but the actual type for that arg for function %s is %s",
				index,
				assertedType,
				GetFuncName(function),
				actualType,
			))
		}
	}

	assertArgsAre(tester, called, expectedArgs...)

	return called
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

func NewCallRelay() *CallRelay {
	return &CallRelay{callChan: make(chan *Call)}
}

// Private helpers.
func assertCalledNameIs(t Tester, c *Call, expectedName string) {
	t.Helper()

	if c.Name() != expectedName {
		t.Fatalf("wrong func called: the called function was expected to be %s, but was %s instead", expectedName, c.Name())
	}
}

func assertArgsAre(tester Tester, theCall *Call, expectedArgs ...any) {
	tester.Helper()

	if !reflect.DeepEqual(theCall.Args(), expectedArgs) {
		tester.Fatalf("wrong values: the function %s was expected to be called with %#v but was called with %#v",
			theCall.Name(), expectedArgs, theCall.Args(),
		)
	}
}

func GetFuncName(f Function) string {
	// docs say to use UnsafePointer explicitly instead of Pointer()
	// https://pkg.Pgo.dev/reflect@go1.21.1#Value.Pointer
	name := runtime.FuncForPC(uintptr(reflect.ValueOf(f).UnsafePointer())).Name()
	// this suffix gets appended sometimes. It's unimportant, as far as I can tell.
	name = strings.TrimSuffix(name, "-fm")

	return name
}

func panicIfNotFunc(evaluate Function, from Function) {
	kind := reflect.ValueOf(evaluate).Kind()
	if kind != reflect.Func {
		panic(fmt.Sprintf("must pass a function as the first argument to %s. received a %s instead.",
			GetFuncName(from),
			kind.String(),
		))
	}
}

// CallRelay Methods.
func (cr *CallRelay) Get() (*Call, error) {
	select {
	case c, ok := <-cr.callChan:
		if !ok {
			return nil, errCallAfterShutDown
		}

		return c, nil
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

	if expectedNumArgs < supportedNumArgs {
		panic(fmt.Sprintf(
			"too few args: the length of the expected argument list (%d)"+
				" is less than the length of the arguments (%s) supports (%d)",
			expectedNumArgs,
			GetFuncName(function),
			supportedNumArgs,
		))
	}

	if expectedNumArgs > supportedNumArgs {
		panic(fmt.Sprintf(
			"too many args: the length of the expected argument list (%d)"+
				" is greater than the length of the arguments (%s) supports (%d)",
			expectedNumArgs,
			GetFuncName(function),
			supportedNumArgs,
		))
	}

	for index := range args {
		passedArg := args[index]
		passedArgType := reflect.TypeOf(passedArg).Name()
		expectedArgType := reflect.TypeOf(function).In(index).Name()

		if passedArgType != expectedArgType {
			panic(fmt.Sprintf(
				"wrong arg type: arg %d was type %s, but func (%s) wants type %s",
				index,
				passedArgType,
				GetFuncName(function),
				expectedArgType,
			))
		}
	}

	return cr.Put(NewCall(function, args...))
}

// Call methods.
func (c Call) Name() string {
	return GetFuncName(c.function)
}

func (c Call) Args() []any {
	return c.args
}

func (c Call) InjectReturns(returnValues ...any) {
	supportedNumReturns := reflect.TypeOf(c.function).NumOut()
	injectedNumReturns := len(returnValues)

	if injectedNumReturns != supportedNumReturns {
		panic(fmt.Sprintf(
			"wrong number of returns: the length of the injected return list (%d)"+
				" does not equal the length of the returns (%s) supports (%d)",
			injectedNumReturns,
			GetFuncName(c.function),
			supportedNumReturns,
		))
	}

	for index := range returnValues {
		passedArg := returnValues[index]

		// if the func type is a pointer and the passed Arg is nil, that's ok, too.
		if passedArg == nil && reflect.TypeOf(c.function).Out(index).Kind() == reflect.Pointer {
			continue
		}

		passedArgType := reflect.TypeOf(passedArg).Name()
		expectedArgType := reflect.TypeOf(c.function).Out(index).Name()

		if passedArgType != expectedArgType {
			panic(fmt.Sprintf(
				"wrong return type: return value %d was type %s, but func (%s) returns type %s",
				index,
				passedArgType,
				GetFuncName(c.function),
				expectedArgType,
			))
		}
	}

	select {
	case c.returns <- returnValues:
		return
	case <-time.After(1 * time.Second):
		panic("fill was not called: timed out waiting for " + c.Name() + " to read the injected return values")
	}
}

func (c Call) FillReturns(returnPointers ...any) {
	returnValues := <-c.returns

	if len(returnPointers) != len(returnValues) {
		panic(fmt.Sprintf(
			"wrong number of returns: "+
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

		returnPointerType := reflect.TypeOf(returnPointerValue.Elem().Interface()).Name()
		expectedArgType := reflect.TypeOf(c.function).Out(index).Name()

		if returnPointerType != expectedArgType {
			panic(fmt.Sprintf(
				"wrong return type: return value %d to be filled was type %s, but func (%s) returns type %s",
				index,
				returnPointerType,
				GetFuncName(c.function),
				expectedArgType,
			))
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
