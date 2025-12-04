package imptest

import (
	"fmt"
	"reflect"
	"testing"
)

// Start validates that fn is a function, validates that args match the function signature,
// calls the function in a goroutine, and stores the return values for retrieval.
func Start(t *testing.T, fn any, args ...any) *TestInvocation {
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

	for i := range args {
		argType := fnType.In(i)

		argValue := reflect.ValueOf(args[i])

		if !argValue.Type().AssignableTo(argType) {
			panic(fmt.Sprintf("Start: argument %d: expected %v, got %v", i, argType, argValue.Type()))
		}

		argValues[i] = argValue
	}

	inv := &TestInvocation{
		t:          t,
		returnChan: make(chan TestReturn, 1),
		panicChan:  make(chan any, 1),
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
	t          *testing.T
	returnChan chan TestReturn
	panicChan  chan any
	returned   *TestReturn
	panicked   any
}

func (t *TestInvocation) ExpectReturnedValues(vals ...any) {
	resp := t.GetResponse()
	if resp.Type() != ReturnEvent {
		t.t.Fatalf("expected ReturnEvent, got %v", resp.Type())
	}

	ret := resp.AsReturn()
	if len(ret) != len(vals) {
		t.t.Fatalf("expected %d returned values, got %d", len(vals), len(ret))
	}

	for i, val := range vals {
		if !reflect.DeepEqual(ret[i], val) {
			t.t.Fatalf("expected returned value %d to be %v, got %v", i, val, ret[i])
		}
	}
}

func (t *TestInvocation) ExpectPanicWith(expected any) {
	resp := t.GetResponse()
	if resp.Type() != PanicEvent {
		t.t.Fatalf("expected PanicEvent, got %v", resp.Type())
	}

	if !reflect.DeepEqual(resp.panicVal, expected) {
		t.t.Fatalf("expected panic with %v, got %v", expected, resp.panicVal)
	}
}

func (t *TestInvocation) GetResponse() *TestResponse {
	// Check if we already have a return value or panic
	if t.returned != nil {
		return &TestResponse{
			eventType: ReturnEvent,
			returnVal: t.returned,
		}
	}

	if t.panicked != nil {
		return &TestResponse{
			eventType: PanicEvent,
			panicVal:  t.panicked,
		}
	}

	// Wait for either return or panic
	select {
	case ret := <-t.returnChan:
		t.returned = &ret

		return &TestResponse{
			eventType: ReturnEvent,
			returnVal: &ret,
		}
	case p := <-t.panicChan:
		t.panicked = p

		return &TestResponse{
			eventType: PanicEvent,
			panicVal:  p,
		}
	}
}

type TestResponse struct {
	eventType EventType
	returnVal *TestReturn
	panicVal  any
}

func (e *TestResponse) Type() EventType {
	if e.eventType == "" {
		return "stub"
	}

	return e.eventType
}

func (e *TestResponse) AsReturn() TestReturn {
	if e.returnVal != nil {
		return *e.returnVal
	}

	return TestReturn{}
}

// Only TestReturn remains as a stub for return events

type TestReturn []any

// EventType is an enum for event types in imptest.
type EventType string

const (
	CallEvent   EventType = "CallEvent"
	ReturnEvent EventType = "ReturnEvent"
	PanicEvent  EventType = "PanicEvent"
)
