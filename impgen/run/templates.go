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

	// Framework package names (always use these constants instead of hardcoding)
	PkgTesting string // "_testing"
	PkgFmt     string // "_fmt"
	PkgImptest string // "_imptest"
	PkgTime    string // "_time"
	PkgReflect string // "_reflect"

	// Framework packages are always imported with underscore prefix to avoid conflicts.
	// User's package (Qualifier/PkgPath) is imported without alias.
	NeedsFmt     bool // Whether fmt import is needed for Sprintf
	NeedsReflect bool // Whether reflect import is needed for DeepEqual
	NeedsImptest bool // Whether imptest import is needed for matchers

	// Additional imports needed for external types used in method signatures
	AdditionalImports []importInfo
}

// callbackParam holds metadata about a callback function parameter.
type callbackParam struct {
	ParamName        string               // Original parameter name (e.g., "fn")
	ParamType        string               // Full type signature (e.g., "func(string, fs.DirEntry, error) error")
	CamelName        string               // CamelCase version for type names (e.g., "Fn")
	RequestType      string               // Request struct type name (e.g., "TreeWalkerImpWalkCallFnRequest")
	ResponseType     string               // Response struct type name (e.g., "TreeWalkerImpWalkCallFnResponse")
	ResultType       string               // Result struct type name (e.g., "TreeWalkerImpWalkCallFnCallbackResult")
	InvokeMethodName string               // Method name for invoking (e.g., "InvokeFn")
	Params           []callbackParamField // Callback function parameters
	Results          []callbackParamField // Callback function results
	HasResults       bool                 // Whether callback has return values
}

// callbackParamField holds metadata about a single callback parameter or result field.
type callbackParamField struct {
	Name  string // Field name (e.g., "Path", "Result0")
	Type  string // Field type (e.g., "string", "error")
	Index int    // Zero-based index
}

// importInfo holds information about an additional import needed for external types.
type importInfo struct {
	Alias string // Package alias/name (e.g., "io", "os")
	Path  string // Full import path (e.g., "io", "os", "github.com/dave/dst")
}

// methodWrapperData holds template data for a single interface method wrapper.
type methodWrapperData struct {
	MethodName        string        // Interface method name (e.g., "Log")
	WrapName          string        // Internal wrapper constructor (e.g., "wrapWrapLoggerWrapperLog")
	WrapperType       string        // Wrapper type (e.g., "WrapLoggerWrapperLogWrapper")
	ReturnsType       string        // Returns type (e.g., "WrapLoggerWrapperLogReturns")
	Params            string        // Full parameter list (e.g., "msg string")
	ParamNames        string        // Comma-separated parameter names (e.g., "msg")
	ParamFields       []paramField  // Parameter fields for call record
	ParamFieldsStruct string        // Struct field definitions for inline struct (e.g., "Msg string")
	Results           string        // Full results list (e.g., "(int, error)")
	HasResults        bool          // Whether method has return values
	ResultVars        string        // Comma-separated result vars (e.g., "r0, r1")
	ReturnAssignments string        // Struct assignments (e.g., "Result0: r0, Result1: r1")
	ResultReturnList  string        // Return type list (e.g., "(int, error)")
	ResultFields      []resultField // Result fields for Returns struct
	ResultChecks      []resultCheck // Result comparison checks
	WaitMethodName    string        // "WaitForResponse" or "WaitForCompletion"
	ExpectedParams    string        // Expected parameters for assertions
	MatcherParams     string        // Matcher parameters for assertions
}

// paramField holds info about a single parameter field for args structs.
type paramField struct {
	Name  string // Field name (e.g., "A", "B", "Key")
	Type  string // Field type (e.g., "int", "string")
	Index int    // Zero-based index in args array
}

// resultCheck holds data for a single result comparison check.
type resultCheck struct {
	Field    string // Field name (e.g., "R1")
	Expected string // Expected parameter name (e.g., "expected1")
	Index    int    // 1-based index for error messages
}

// resultField holds data for a single result struct field.
type resultField struct {
	Name string // Field name (e.g., "R1")
	Type string // Field type (e.g., "int")
}

// resultVar holds info about a single result variable.
type resultVar struct {
	Name  string
	Type  string
	Index int
}

// Types

// v2DepMethodTemplateData holds data for v2 dependency impl method template.
type v2DepMethodTemplateData struct {
	baseTemplateData //nolint:unused // Used by templates

	MethodName      string
	InterfaceType   string
	ImplName        string
	Params          string // Full parameter list string
	Results         string // Full result list string (including parens if multiple)
	HasVariadic     bool
	NonVariadicArgs string // Comma-separated non-variadic args
	VariadicArg     string // Name of variadic arg
	Args            string // Comma-separated all args (for non-variadic case)
	ArgNames        string // Comma-separated argument names
	HasResults      bool
	ResultVars      []resultVar
	ReturnList      string          // Comma-separated return variable names
	ReturnStatement string          // Return statement (e.g., "return r1, r2")
	Callbacks       []callbackParam // Callback function parameters
	HasCallbacks    bool            // Whether method has any callback parameters

	// Type-safe args support
	ParamFields    []paramField // Parameter fields for args struct
	HasParams      bool         // Whether method has parameters
	ArgsTypeName   string       // Args struct type name (e.g., "CalculatorAddArgs")
	CallTypeName   string       // Call wrapper type name (e.g., "CalculatorAddCall")
	MethodTypeName string       // Method wrapper type name (e.g., "CalculatorAddMethod")
	TypedParams    string       // Typed parameter list for ExpectCalledWithExactly (e.g., "a int, b int")

	// Type-safe return value support
	TypedReturnParams string // Typed return parameter list for InjectReturnValues (e.g., "result0 int, result1 error")
	ReturnParamNames  string // Comma-separated return parameter names (e.g., "result0, result1")
}

// v2DepTemplateData holds data for v2 dependency mock templates.
type v2DepTemplateData struct {
	baseTemplateData //nolint:unused // Used by templates

	MockName      string                    // Constructor function name (e.g., "MockOps")
	MockTypeName  string                    // Struct type name (e.g., "OpsMock")
	BaseName      string                    // Base interface name without "Mock" prefix
	InterfaceName string                    // Local interface name (e.g., "Ops")
	InterfaceType string                    // Qualified interface type (e.g., "basic.Ops")
	ImplName      string                    // Implementation struct name (e.g., "mockOpsImpl")
	MethodNames   []string                  // List of interface method names
	Methods       []v2DepMethodTemplateData // Full method data for generating wrappers
}

// v2InterfaceTargetTemplateData holds data for v2 interface target wrapper templates.
type v2InterfaceTargetTemplateData struct {
	baseTemplateData //nolint:unused // Used by templates

	WrapName      string              // Constructor function name (e.g., "WrapLogger")
	WrapperType   string              // Main wrapper struct type (e.g., "WrapLoggerWrapper")
	InterfaceName string              // Local interface name (e.g., "Logger")
	InterfaceType string              // Qualified interface type (e.g., "log.Logger")
	ImplName      string              // Real implementation field name (e.g., "impl")
	MethodNames   []string            // List of interface method names
	Methods       []methodWrapperData // Full method data for generating wrappers
}

// v2TargetTemplateData holds data for v2 target wrapper templates.
type v2TargetTemplateData struct {
	baseTemplateData //nolint:unused // Used by templates

	WrapName          string // Constructor function name (e.g., "WrapAdd")
	WrapperType       string // Wrapper struct type (e.g., "WrapAddWrapper")
	ReturnsType       string // Returns struct type (e.g., "WrapAddReturns")
	FuncSig           string // Full function signature
	Params            string // Function parameters for Start method
	ParamNames        string // Comma-separated parameter names
	HasResults        bool
	ResultVars        string        // Comma-separated result var declarations (e.g., "r1, r2")
	ReturnAssignments string        // Comma-separated return assignments (e.g., "R1: r1, R2: r2")
	WaitMethodName    string        // "WaitForResponse" or "WaitForCompletion"
	ExpectedParams    string        // Expected parameters for ExpectReturnsEqual (typed)
	MatcherParams     string        // Matcher parameters for ExpectReturnsMatch (all any)
	ResultChecks      []resultCheck // Result comparison checks
	ResultFields      []resultField // Result struct fields
}
