package imptest

// Start is a stub implementation for test scaffolding. It returns a dummy struct with ExpectReturnedValues and GetCurrentEvent methods.
func Start(fn interface{}, args ...interface{}) *TestInvocation {
	return &TestInvocation{}
}

type TestInvocation struct{}

func (t *TestInvocation) ExpectReturnedValues(vals ...interface{}) {}
func (t *TestInvocation) GetCurrentEvent() *TestEvent              { return &TestEvent{} }

type TestEvent struct{}

func (e *TestEvent) Type() string          { return "stub" }
func (e *TestEvent) AsReturn() *TestReturn { return &TestReturn{} }

// Only TestReturn remains as a stub for return events

type TestReturn struct {
	Ret0, Ret1 int
	Ret2       string
}

// EventType is an enum for event types in imptest
//
type EventType string

const (
	CallEvent   EventType = "CallEvent"
	ReturnEvent EventType = "ReturnEvent"
	PanicEvent  EventType = "PanicEvent"
)
