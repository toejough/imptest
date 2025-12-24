package run

import (
	"strings"
	"text/template"
)

// unexported variables.
//
//nolint:gochecknoglobals,lll // Template variables are inherently global; long lines in templates are acceptable
var (
	callStructTemplate = mustParse("callStruct", `// {{.CallName}} represents a captured call to any method.
// Only one method field is non-nil at a time, indicating which method was called.
// Use Name() to identify the method and As{Method}() to access typed call details.
type {{.CallName}}{{.TypeParamsDecl}} struct {
{{range .Methods}}	{{.Name}} *{{.CallName}}{{.TypeParamsUse}}
{{end}}}

// Name returns the name of the method that was called.
// Returns an empty string if the call struct is invalid.
func (c *{{.CallName}}{{.TypeParamsUse}}) Name() string {
{{range .Methods}}	if c.{{.Name}} != nil {
		return "{{.Name}}"
	}
{{end}}	return ""
}

// Done returns true if the call has been completed (response injected).
// Used internally to track call state.
func (c *{{.CallName}}{{.TypeParamsUse}}) Done() bool {
{{range .Methods}}	if c.{{.Name}} != nil {
		return c.{{.Name}}.done
	}
{{end}}	return false
}

{{range .Methods}}// As{{.Name}} returns the call cast to {{.CallName}} for accessing call details.
// Returns nil if the call was not to {{.Name}}.
func (c *{{$.CallName}}{{.TypeParamsUse}}) As{{.Name}}() *{{.CallName}}{{.TypeParamsUse}} {
	return c.{{.Name}}
}

{{end}}`)
	callableAsReturnMethodTemplate = mustParse("callableAsReturnMethod",
		`// AsReturn converts the return values to a slice of any for generic processing.
// Returns nil if the response was a panic or if there are no return values.
func (r *{{.ImpName}}Response{{.TypeParamsUse}}) AsReturn() []any {
{{if .HasReturns}}	if r.ReturnVal == nil {
		return nil
	}
	return []any{ {{- range $i, $_ := .ReturnFields}}{{if $i}}, {{end}}r.ReturnVal.Result{{$i}}{{end -}} }
{{else}}	return nil
{{end}}}

`)
	callableConstructorTemplate = mustParse("callableConstructor",
		`// New{{.ImpName}} creates a new wrapper for testing the callable function.
// Pass the function to test and a testing.TB to enable assertion failures.
//
// Example:
//
//	wrapper := New{{.ImpName}}(t, myFunction)
//	wrapper.Start(args...).ExpectReturnedValuesAre(expectedVals...)
func New{{.ImpName}}{{.TypeParamsDecl}}(t {{if .TestingAlias}}{{.TestingAlias}}{{else}}testing{{end}}.TB, callable func({{.CallableSignature}}){{.CallableReturns}}) *{{.ImpName}}{{.TypeParamsUse}} {
	return &{{.ImpName}}{{.TypeParamsUse}}{
		CallableController: {{if .ImptestAlias}}{{.ImptestAlias}}{{else}}imptest{{end}}.NewCallableController[{{.ReturnType}}](t),
		callable:           callable,
	}
}

`)
	callableExpectPanicWithTemplate = mustParse("callableExpectPanicWith",
		`// ExpectPanicWith asserts the callable panicked with a value matching the expectation.
// Use imptest.Any() to match any panic value, or imptest.Satisfies(fn) for custom matching.
// Fails the test if the callable returned normally or panicked with a different value.
func (s *{{.ImpName}}{{.TypeParamsUse}}) ExpectPanicWith(expected any) {
	s.T.Helper()
	s.WaitForResponse()

	if s.Panicked != nil {
		ok, msg := imptest.MatchValue(s.Panicked, expected)
		if !ok {
			s.T.Fatalf("panic value: %s", msg)
		}
		return
	}

	s.T.Fatalf("expected function to panic, but it returned")
}

`)
	callableExpectReturnedValuesAreTemplate = mustParse("callableExpectReturnedValuesAre",
		`// ExpectReturnedValuesAre asserts the callable returned with exactly the specified values.
// Fails the test if the values don't match exactly or if the callable panicked.
// Uses == for comparison, so reference types must be the same instance.
func (s *{{.ImpName}}{{.TypeParamsUse}}) ExpectReturnedValuesAre({{.ResultParams}}) {
	s.T.Helper()
	s.WaitForResponse()

	if s.Returned != nil {
{{.ResultComparisons}}		return
	}

	s.T.Fatalf("expected function to return, but it panicked with: %v", s.Panicked)
}

`)
	callableExpectReturnedValuesShouldTemplate = mustParse("callableExpectReturnedValuesShould",
		`// ExpectReturnedValuesShould asserts return values match the given matchers.
// Use imptest.Any() to match any value, or imptest.Satisfies(fn) for custom matching.
// Fails the test if any matcher fails or if the callable panicked.
func (s *{{.ImpName}}{{.TypeParamsUse}}) ExpectReturnedValuesShould({{.ResultParamsAny}}) {
	s.T.Helper()
	s.WaitForResponse()

	if s.Returned != nil {
{{if .HasReturns}}		var ok bool
		var msg string
{{.ResultMatchers}}{{end}}		return
	}

	s.T.Fatalf("expected function to return, but it panicked with: %v", s.Panicked)
}

`)
	callableGetResponseMethodTemplate = mustParse("callableGetResponseMethod",
		`// GetResponse waits for and returns the callable's response.
// Use this when you need to inspect the response without asserting specific values.
// The response indicates whether the callable returned or panicked.
func (s *{{.ImpName}}{{.TypeParamsUse}}) GetResponse() *{{.ImpName}}Response{{.TypeParamsUse}} {
	s.WaitForResponse()

	if s.Returned != nil {
		return &{{.ImpName}}Response{{.TypeParamsUse}}{
			EventType: "ReturnEvent",
{{if .HasReturns}}			ReturnVal: s.Returned,
{{end}}		}
	}

	return &{{.ImpName}}Response{{.TypeParamsUse}}{
		EventType: "PanicEvent",
		PanicVal:  s.Panicked,
	}
}

`)
	// Callable generator templates.

	callableHeaderTemplate = mustParse("callableHeader", `// Code generated by impgen. DO NOT EDIT.

package {{.PkgName}}

import (
	{{if .ImptestAlias}}{{.ImptestAlias}} {{end}}"github.com/toejough/imptest/imptest"
	{{if .TestingAlias}}{{.TestingAlias}} {{end}}"testing"
{{if .NeedsQualifier}}	{{.Qualifier}} "{{.PkgPath}}"
{{end}})

`)
	callableMainStructTemplate = mustParse("callableMainStruct", `// {{.ImpName}} wraps a callable function for testing.
// Create with New{{.ImpName}}(t, yourFunction), call Start() to execute,
// then use ExpectReturnedValuesAre/Should() or ExpectPanicWith() to verify behavior.
type {{.ImpName}}{{.TypeParamsDecl}} struct {
	*imptest.CallableController[{{.ReturnType}}]
	callable func({{.CallableSignature}}){{.CallableReturns}}
}

`)
	callableResponseStructTemplate = mustParse("callableResponseStruct",
		`// {{.ImpName}}Response represents the response from the callable (either return or panic).
// Check EventType to determine if the callable returned normally or panicked.
// Use AsReturn() to get return values as a slice, or access PanicVal directly.
type {{.ImpName}}Response{{.TypeParamsDecl}} struct {
	EventType string // "return" or "panic"
{{if .HasReturns}}	ReturnVal *{{.ImpName}}Return{{.TypeParamsUse}}
{{end}}	PanicVal  any
}

`)
	callableResponseTypeMethodTemplate = mustParse("callableResponseTypeMethod",
		`// Type returns the event type: "return" for normal returns, "panic" for panics.
func (r *{{.ImpName}}Response{{.TypeParamsUse}}) Type() string {
	return r.EventType
}

`)
	callableReturnStructTemplate = mustParse("callableReturnStruct",
		`{{if .HasReturns}}// {{.ImpName}}Return holds the return values from the callable function.
// Access individual return values via Result0, Result1, etc. fields.
type {{.ImpName}}Return{{.TypeParamsDecl}} struct {
{{range .ReturnFields}}	Result{{.Index}} {{.Type}}
{{end}}}

{{end}}`)
	callableStartMethodTemplate = mustParse("callableStartMethod",
		`// Start begins execution of the callable in a goroutine with the provided arguments.
// Returns the wrapper for method chaining with expectation methods.
// Captures both normal returns and panics for verification.
//
// Example:
//
//	wrapper.Start(arg1, arg2).ExpectReturnedValuesAre(expectedResult)
func (s *{{.ImpName}}{{.TypeParamsUse}}) Start({{.CallableSignature}}) *{{.ImpName}}{{.TypeParamsUse}} {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.PanicChan <- r
			}
		}()

{{if .HasReturns}}		{{.ReturnVars}} := s.callable({{.ParamNames}})
		s.ReturnChan <- {{.ImpName}}Return{{.TypeParamsUse}}{
{{range .ReturnFields}}			Result{{.Index}}: {{.Name}},
{{end}}		}
{{else}}		s.callable({{.ParamNames}})
		s.ReturnChan <- struct{}{}
{{end}}	}()
	return s
}

`)
	constructorTemplate = mustParse("constructor",
		`// New{{.ImpName}} creates a new test controller for mocking the interface.
// The returned controller manages mock expectations and response injection.
// Pass t to enable automatic test failure on unexpected calls or timeouts.
//
// Example:
//
//	imp := New{{.ImpName}}(t)
//	go codeUnderTest(imp.Mock)
//	imp.ExpectCallIs.Method().ExpectArgsAre(...).InjectResult(...)
func New{{.ImpName}}{{.TypeParamsDecl}}(t *{{if .TestingAlias}}{{.TestingAlias}}{{else}}testing{{end}}.T) *{{.ImpName}}{{.TypeParamsUse}} {
	imp := &{{.ImpName}}{{.TypeParamsUse}}{
		Controller: {{if .ImptestAlias}}{{.ImptestAlias}}{{else}}imptest{{end}}.NewController[*{{.CallName}}{{.TypeParamsUse}}](t),
	}
	imp.Mock = &{{.MockName}}{{.TypeParamsUse}}{imp: imp}
	imp.ExpectCallIs = &{{.ExpectCallIsName}}{{.TypeParamsUse}}{imp: imp}
	return imp
}

`)
	expectCallIsStructTemplate = mustParse("expectCallIsStruct",
		`// {{.ExpectCallIsName}} provides methods to set expectations for specific method calls.
// Each method returns a builder for fluent expectation configuration.
// Use Within() on the parent {{.ImpName}} to configure timeouts.
type {{.ExpectCallIsName}}{{.TypeParamsDecl}} struct {
	imp *{{.ImpName}}{{.TypeParamsUse}}
	timeout {{if .TimeAlias}}{{.TimeAlias}}{{else}}time{{end}}.Duration
}

`)
	getCurrentCallMethodTemplate = mustParse("getCurrentCallMethod",
		`// GetCurrentCall returns the current call being processed.
// If no call is pending, waits indefinitely for the next call.
// Returns the existing current call if it hasn't been completed yet.
func (i *{{.ImpName}}{{.TypeParamsUse}}) GetCurrentCall() *{{.CallName}}{{.TypeParamsUse}} {
	if i.currentCall != nil && !i.currentCall.Done() {
		return i.currentCall
	}
	i.currentCall = i.GetCall(0, func(c *{{.CallName}}{{.TypeParamsUse}}) bool { return true })
	return i.currentCall
}

`)
	headerTemplate = mustParse("header", `// Code generated by impgen. DO NOT EDIT.

package {{.PkgName}}

import {{if .ImptestAlias}}{{.ImptestAlias}} {{end}}"github.com/toejough/imptest/imptest"
{{if .NeedsReflect}}import {{if .ReflectAlias}}{{.ReflectAlias}} {{end}}"reflect"
{{end}}import {{if .TestingAlias}}{{.TestingAlias}} {{end}}"testing"
import {{if .TimeAlias}}{{.TimeAlias}} {{end}}"time"
{{if .NeedsQualifier}}import {{.Qualifier}} "{{.PkgPath}}"
{{end}}
`)
	interfaceVerificationTemplate = mustParse("interfaceVerification",
		`var (
	// Compile-time verification that {{.MockName}} implements {{.InterfaceName}}.
	_ {{.InterfaceName}}{{.TypeParamsUse}} = (*{{.MockName}}{{.TypeParamsUse}})(nil)
)

`)
	injectPanicMethodTemplate = mustParse("injectPanic",
		`// InjectPanic causes the mocked method to panic with the given value.
// Use this to test panic handling in code under test.
// The panic occurs in the goroutine where the mock was called.
func (c *{{.MethodCallName}}{{.TypeParamsUse}}) InjectPanic(msg any) {
	c.done = true
	c.responseChan <- {{.MethodCallName}}Response{{.TypeParamsUse}}{Type: "panic", PanicValue: msg}
}
`)
	mainStructTemplate = mustParse("mainStruct", `// {{.ImpName}} is the test controller for mocking the interface.
// Create with New{{.ImpName}}(t), then use Mock field to get the mock implementation
// and ExpectCallIs field to set expectations for method calls.
//
// Example:
//
//	imp := New{{.ImpName}}(t)
//	go codeUnderTest(imp.Mock)
//	imp.ExpectCallIs.MethodName().ExpectArgsAre(...).InjectResult(...)
type {{.ImpName}}{{.TypeParamsDecl}} struct {
	*imptest.Controller[*{{.CallName}}{{.TypeParamsUse}}]
	Mock *{{.MockName}}{{.TypeParamsUse}}
	ExpectCallIs *{{.ExpectCallIsName}}{{.TypeParamsUse}}
	currentCall *{{.CallName}}{{.TypeParamsUse}}
}

`)
	mockStructTemplate = mustParse("mockStruct", `// {{.MockName}} provides the mock implementation of the interface.
// Pass {{.MockName}} to code under test that expects the interface implementation.
// Use the parent {{.ImpName}} controller to set expectations and inject responses.
type {{.MockName}}{{.TypeParamsDecl}} struct {
	imp *{{.ImpName}}{{.TypeParamsUse}}
}

`)
	resolveMethodTemplate = mustParse("resolve",
		`// Resolve completes a void method call without error.
// Use this to unblock the mock method and allow execution to continue.
// Only applicable to methods with no return values.
func (c *{{.MethodCallName}}{{.TypeParamsUse}}) Resolve() {
	c.done = true
	c.responseChan <- {{.MethodCallName}}Response{{.TypeParamsUse}}{Type: "resolve"}
}
`)
	timedStructTemplate = mustParse("timedStruct", `// {{.TimedName}} provides timeout-configured expectation methods.
// Access via {{.ImpName}}.Within(duration) to set a timeout for expectations.
type {{.TimedName}}{{.TypeParamsDecl}} struct {
	ExpectCallIs *{{.ExpectCallIsName}}{{.TypeParamsUse}}
}

// Within configures a timeout for expectations and returns a {{.TimedName}} for method chaining.
// The timeout applies to subsequent expectation calls.
//
// Example:
//
//	imp.Within(100*{{if .TimeAlias}}{{.TimeAlias}}{{else}}time{{end}}.Millisecond).ExpectCallIs.Method().ExpectArgsAre(...)
func (i *{{.ImpName}}{{.TypeParamsUse}}) Within(d {{if .TimeAlias}}{{.TimeAlias}}{{else}}time{{end}}.Duration) *{{.TimedName}}{{.TypeParamsUse}} {
	return &{{.TimedName}}{{.TypeParamsUse}}{
		ExpectCallIs: &{{.ExpectCallIsName}}{{.TypeParamsUse}}{imp: i, timeout: d},
	}
}

`)
)

// baseTemplateData holds common fields shared by all template data structs.
type baseTemplateData struct {
	PkgName        string
	ImpName        string
	PkgPath        string // Import path for the package being mocked/wrapped
	Qualifier      string // Package qualifier (e.g., "basic")
	NeedsQualifier bool   // Whether the package qualifier is actually used
	TypeParamsDecl string // Type parameters with constraints, e.g., "[T any, U comparable]"
	TypeParamsUse  string // Type parameters for instantiation, e.g., "[T, U]"

	// Aliases for stdlib packages when they conflict with the user's package qualifier.
	// Empty string means no alias needed (no conflict).
	TimeAlias    string // "_time" if qualifier conflicts with "time"
	TestingAlias string // "_testing" if qualifier conflicts with "testing"
	ReflectAlias string // "_reflect" if qualifier conflicts with "reflect"
	ImptestAlias string // "_imptest" if qualifier conflicts with "imptest"
}

// callStructMethodData holds data for generating call struct methods with method field info.
type callStructMethodData struct {
	Name          string // Method name (e.g., "DoSomething")
	CallName      string // Full call struct name (e.g., "MyImpDoSomethingCall")
	TypeParamsUse string
}

// callStructTemplateData holds data for generating the call struct and its methods.
type callStructTemplateData struct {
	templateData //nolint:unused // Used in templates

	Methods []callStructMethodData
}

// callableTemplateData holds data for callable wrapper templates.
type callableTemplateData struct {
	baseTemplateData //nolint:unused // Used in templates

	HasReturns bool
	ReturnType string // "{ImpName}Return" or "struct{}"
	NumReturns int
}

// methodTemplateData holds data for method-specific templates.
type methodTemplateData struct {
	templateData //nolint:unused // Used in templates

	MethodName     string
	MethodCallName string
}

// Types

// templateData holds common data passed to templates.
type templateData struct {
	baseTemplateData //nolint:unused // Used in templates

	MockName         string
	CallName         string
	ExpectCallIsName string
	TimedName        string
	InterfaceName    string // Full interface name for compile-time verification
	MethodNames      []string
	NeedsReflect     bool // Whether reflect import is needed for DeepEqual
	NeedsImptest     bool // Whether imptest import is needed for matchers
}

// executeTemplate executes a template and returns the result as a string.
func executeTemplate(tmpl *template.Template, data any) string {
	var buf strings.Builder

	err := tmpl.Execute(&buf, data)
	if err != nil {
		// Templates are compile-time validated, so this should never happen
		panic("template execution failed: " + err.Error())
	}

	return buf.String()
}

// Functions

// mustParse is a helper to parse template strings and panic on error.
func mustParse(name, text string) *template.Template {
	return template.Must(template.New(name).Parse(text))
}
