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
func (e *TestEvent) AsCall() *TestCall     { return &TestCall{} }
func (e *TestEvent) AsReturn() *TestReturn { return &TestReturn{} }

// Stub types for call/return events

type TestCall struct{}

func (c *TestCall) Name() string          { return "stub" }
func (c *TestCall) AsAdd() *TestAdd       { return &TestAdd{} }
func (c *TestCall) AsFormat() *TestFormat { return &TestFormat{} }
func (c *TestCall) AsPrint() *TestPrint   { return &TestPrint{} }

type TestAdd struct{ A, B int }

func (a *TestAdd) InjectResult(res int)   {}
func (a *TestAdd) InjectPanic(msg string) {}

type TestFormat struct{ Input int }

func (f *TestFormat) InjectResult(res string) {}

type TestPrint struct{ S string }

func (p *TestPrint) Resolve() {}

type TestReturn struct {
	Ret0, Ret1 int
	Ret2       string
}
