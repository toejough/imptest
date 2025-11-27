package imptest

import (
	"fmt"
	"reflect"
)

// Start validates that fn is a function, validates that args match the function signature,
// calls the function in a goroutine, and stores the return values for retrieval.
func Start(fn interface{}, args ...interface{}) *TestInvocation {
	// Validate that fn is a function
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		panic(fmt.Sprintf("Start: fn must be a function, got %T", fn))
	}

	fnType := fnValue.Type()

	// Validate that the number of args matches the function signature
	if fnType.NumIn() != len(args) {
		panic(fmt.Sprintf("Start: function expects %d arguments, got %d", fnType.NumIn(), len(args)))
	}

	// Validate that each arg type matches the function parameter type
	argValues := make([]reflect.Value, len(args))
	for i := 0; i < len(args); i++ {
		argType := fnType.In(i)
		argValue := reflect.ValueOf(args[i])
		if !argValue.Type().AssignableTo(argType) {
			panic(fmt.Sprintf("Start: argument %d: expected %v, got %v", i, argType, argValue.Type()))
		}
		argValues[i] = argValue
	}

	inv := &TestInvocation{
		returnChan: make(chan TestReturn, 1),
		panicChan:  make(chan interface{}, 1),
	}

	// Call the function in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				inv.panicChan <- r
			}
		}()

		// Call the function with the validated args
		returnValues := fnValue.Call(argValues)

		// Convert return values to TestReturn ([]any)
		ret := make(TestReturn, len(returnValues))
		for i, v := range returnValues {
			ret[i] = v.Interface()
		}

		inv.returnChan <- ret
	}()

	return inv
}

type TestInvocation struct {
	returnChan chan TestReturn
	panicChan  chan interface{}
	returned   *TestReturn
	panicked   interface{}
}

func (t *TestInvocation) ExpectReturnedValues(vals ...interface{}) {}
func (t *TestInvocation) GetCurrentEvent() *TestEvent {
	// Check if we already have a return value or panic
	if t.returned != nil {
		return &TestEvent{
			eventType: ReturnEvent,
			returnVal: t.returned,
		}
	}
	if t.panicked != nil {
		return &TestEvent{
			eventType: PanicEvent,
			panicVal:  t.panicked,
		}
	}

	// Wait for either return or panic
	select {
	case ret := <-t.returnChan:
		t.returned = &ret
		return &TestEvent{
			eventType: ReturnEvent,
			returnVal: &ret,
		}
	case p := <-t.panicChan:
		t.panicked = p
		return &TestEvent{
			eventType: PanicEvent,
			panicVal:  p,
		}
	}
}

type TestEvent struct {
	eventType EventType
	returnVal *TestReturn
	panicVal  interface{}
}

func (e *TestEvent) Type() string {
	if e.eventType == "" {
		return "stub"
	}
	return string(e.eventType)
}

func (e *TestEvent) AsReturn() TestReturn {
	if e.returnVal != nil {
		return *e.returnVal
	}
	return TestReturn{}
}

// Only TestReturn remains as a stub for return events

type TestReturn []any

// EventType is an enum for event types in imptest
type EventType string

const (
	CallEvent   EventType = "CallEvent"
	ReturnEvent EventType = "ReturnEvent"
	PanicEvent  EventType = "PanicEvent"
)
