package run

// unexported variables.
// All template definitions have been migrated to quicktemplate (.qtpl files).
// The generated Go code is in *_qtpl.go files.

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
	templateData

	Methods []callStructMethodData
}

// callableTemplateData holds data for callable wrapper templates.
type callableTemplateData struct {
	baseTemplateData

	HasReturns bool
	ReturnType string // "{ImpName}Return" or "struct{}"
	NumReturns int
}

// methodTemplateData holds data for method-specific templates.
type methodTemplateData struct {
	templateData

	MethodName     string
	MethodCallName string
}

// Types

// templateData holds common data passed to templates.
type templateData struct {
	baseTemplateData

	MockName         string
	CallName         string
	ExpectCallIsName string
	TimedName        string
	InterfaceName    string // Full interface name for compile-time verification
	MethodNames      []string
	NeedsReflect     bool // Whether reflect import is needed for DeepEqual
	NeedsImptest     bool // Whether imptest import is needed for matchers
}
