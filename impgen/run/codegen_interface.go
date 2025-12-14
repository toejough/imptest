package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
)

var errFunctionNotFound = errors.New("function not found")

// Structs

// codeGenerator holds state for code generation.
type codeGenerator struct {
	codeWriter

	pkgName             string
	impName             string
	mockName            string
	callName            string
	expectCallToName    string
	timedName           string
	identifiedInterface *ast.InterfaceType
	methodNames         []string
}

// Methods on codeGenerator

// forEachMethod iterates over interface methods and calls the callback for each.
func (gen *codeGenerator) forEachMethod(callback func(methodName string, ftype *ast.FuncType)) {
	forEachInterfaceMethod(gen.identifiedInterface, callback)
}

// templateData returns common template data for this generator.
func (gen *codeGenerator) templateData() templateData {
	return templateData{
		ImpName:          gen.impName,
		MockName:         gen.mockName,
		CallName:         gen.callName,
		ExpectCallToName: gen.expectCallToName,
		TimedName:        gen.timedName,
		PkgName:          gen.pkgName,
		MethodNames:      gen.methodNames,
	}
}

// methodTemplateData returns template data for a specific method.
func (gen *codeGenerator) methodTemplateData(methodCallName string) methodTemplateData {
	return methodTemplateData{
		templateData:   gen.templateData(),
		MethodCallName: methodCallName,
	}
}

// callStructTemplateData returns template data for generating the call struct.
func (gen *codeGenerator) callStructData() callStructTemplateData {
	methods := make([]callStructMethodData, len(gen.methodNames))
	for i, methodName := range gen.methodNames {
		methods[i] = callStructMethodData{
			Name:     methodName,
			CallName: gen.methodCallName(methodName),
		}
	}

	return callStructTemplateData{
		templateData: gen.templateData(),
		Methods:      methods,
	}
}

// generateCallStruct generates the union call struct that can hold any method call.
func (gen *codeGenerator) generateCallStruct() {
	gen.ps(executeTemplate(callStructTemplate, gen.callStructData()))
}

// generateHeader writes the package declaration and imports for the generated file.
func (gen *codeGenerator) generateHeader() {
	gen.ps(executeTemplate(headerTemplate, gen.templateData()))
}

// generateMockStruct generates the mock struct that wraps the implementation.
func (gen *codeGenerator) generateMockStruct() {
	gen.ps(executeTemplate(mockStructTemplate, gen.templateData()))
}

// generateMainStruct generates the main implementation struct that handles test call tracking.
func (gen *codeGenerator) generateMainStruct() {
	gen.ps(executeTemplate(mainStructTemplate, gen.templateData()))
}

// methodCallName returns the call struct name for a method (e.g. "MyImpDoSomethingCall").
func (gen *codeGenerator) methodCallName(methodName string) string {
	return gen.impName + methodName + "Call"
}

// writeMethodSignature writes the method name and parameters (e.g., "MethodName(a int, b string)").
func (gen *codeGenerator) writeMethodSignature(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.pf("%s(", methodName)
	gen.writeMethodParams(ftype, paramNames)
	gen.pf(")")
}

// generateMethodStructs generates the call and response structs for each interface method.
func (gen *codeGenerator) generateMethodStructs() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateMethodCallStruct(methodName, ftype)
		gen.generateMethodResponseStruct(methodName, ftype)
		gen.generateMethodResponseMethods(methodName, ftype)
	})
}

// generateMethodCallStruct generates the call struct for a specific method, which tracks the method call parameters.
func (gen *codeGenerator) generateMethodCallStruct(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	gen.pf(`type %s struct {
	responseChan chan %sResponse
	done bool
`, callName, callName)

	if hasParams(ftype) {
		gen.generateCallStructParamFields(ftype)
	}

	gen.pf("}\n\n")
}

// generateCallStructParamFields generates the parameter fields for a call struct.
func (gen *codeGenerator) generateCallStructParamFields(ftype *ast.FuncType) {
	totalParams := countFields(ftype.Params)
	unnamedIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(gen.fset, param.Type)

		if len(param.Names) > 0 {
			gen.generateNamedParamFields(param, paramType, unnamedIndex, totalParams)
		} else {
			gen.generateUnnamedParamField(param, paramType, unnamedIndex, totalParams)
			unnamedIndex++
		}
	}
}

// generateNamedParamFields generates fields for named parameters.
func (gen *codeGenerator) generateNamedParamFields(param *ast.Field, paramType string, unnamedIndex, totalParams int) {
	for i := range param.Names {
		fieldName := getParamFieldName(param, i, unnamedIndex, paramType, totalParams)
		gen.pf("\t%s %s\n", fieldName, paramType)
	}
}

// generateUnnamedParamField generates a field for an unnamed parameter.
func (gen *codeGenerator) generateUnnamedParamField(param *ast.Field, paramType string, unnamedIndex, totalParams int) {
	fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
	gen.pf("\t%s %s\n", fieldName, paramType)
}

// generateMethodResponseStruct generates the response struct for a method, which holds return values or panic data.
func (gen *codeGenerator) generateMethodResponseStruct(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	gen.pf(`type %sResponse struct {
	Type string // "return", "panic", or "resolve"
`, callName)

	if hasResults(ftype) {
		gen.generateResponseStructResultFields(ftype)
	}

	gen.pf(`	PanicValue any
}

`)
}

// generateResponseStructResultFields generates the result fields for a response struct.
func (gen *codeGenerator) generateResponseStructResultFields(ftype *ast.FuncType) {
	for _, r := range extractResults(gen.fset, ftype) {
		gen.pf("\t%s %s\n", r.Name, r.Type)
	}
}

// generateMethodResponseMethods generates the InjectResult, InjectResults, InjectPanic, and Resolve methods
// for a call struct.
func (gen *codeGenerator) generateMethodResponseMethods(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)

	if hasResults(ftype) {
		totalReturns := countFields(ftype.Results)

		if totalReturns == 1 {
			gen.generateInjectResultMethod(callName, ftype)
		} else {
			gen.generateInjectResultsMethod(callName, ftype)
		}

		gen.generateInjectPanicMethod(callName)
	} else {
		gen.generateResolveMethod(callName)
		gen.generateInjectPanicMethod(callName)
	}

	gen.pf("\n")
}

// generateInjectResultMethod generates the InjectResult method for methods with a single return value.
func (gen *codeGenerator) generateInjectResultMethod(methodCallName string, ftype *ast.FuncType) {
	resultType := exprToString(gen.fset, ftype.Results.List[0].Type)
	gen.pf(`func (c *%s) InjectResult(result %s) {
	c.done = true
	c.responseChan <- %sResponse{Type: "return"`, methodCallName, resultType, methodCallName)

	if len(ftype.Results.List[0].Names) > 0 {
		gen.pf(", %s: result", ftype.Results.List[0].Names[0].Name)
	} else {
		gen.pf(", Result0: result")
	}

	gen.pf(`}
}
`)
}

// generateInjectResultsMethod generates the InjectResults method for methods with multiple return values.
func (gen *codeGenerator) generateInjectResultsMethod(methodCallName string, ftype *ast.FuncType) {
	gen.pf("func (c *%s) InjectResults(", methodCallName)

	returnParamNames := gen.writeInjectResultsParams(ftype)

	gen.pf(`) {
	c.done = true
	resp := %sResponse{Type: "return"`, methodCallName)

	gen.writeInjectResultsResponseFields(ftype, returnParamNames)

	gen.pf(`}
	c.responseChan <- resp
}
`)
}

// writeInjectResultsParams writes the parameter list for InjectResults method and returns the result names.
func (gen *codeGenerator) writeInjectResultsParams(ftype *ast.FuncType) []string {
	results := extractResults(gen.fset, ftype)
	names := make([]string, len(results))

	for resultIdx, result := range results {
		if resultIdx > 0 {
			gen.pf(", ")
		}

		gen.pf("%s %s", result.Name, result.Type)
		names[resultIdx] = result.Name
	}

	return names
}

// writeInjectResultsResponseFields writes the response struct field assignments for InjectResults.
func (gen *codeGenerator) writeInjectResultsResponseFields(ftype *ast.FuncType, returnParamNames []string) {
	for resultIdx, result := range extractResults(gen.fset, ftype) {
		gen.pf(", %s: %s", result.Name, returnParamNames[resultIdx])
	}
}

// generateInjectPanicMethod generates the InjectPanic method for simulating panics.
func (gen *codeGenerator) generateInjectPanicMethod(methodCallName string) {
	gen.ps(executeTemplate(injectPanicMethodTemplate, gen.methodTemplateData(methodCallName)))
}

// generateResolveMethod generates the Resolve method for methods with no return values.
func (gen *codeGenerator) generateResolveMethod(methodCallName string) {
	gen.ps(executeTemplate(resolveMethodTemplate, gen.methodTemplateData(methodCallName)))
}

// generateMockMethods generates the mock methods that implement the interface on the mock struct.
func (gen *codeGenerator) generateMockMethods() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateMockMethod(methodName, ftype)
	})
}

// generateMockMethod generates a single mock method that creates a call, sends it to the imp, and handles the response.
func (gen *codeGenerator) generateMockMethod(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	paramNames := extractParamNames(ftype)

	gen.pf("func (m *%s) ", gen.mockName)
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.pf("%s", renderFieldList(gen.fset, ftype.Results))
	gen.pf(` {
	responseChan := make(chan %sResponse, 1)

	call := &%s{
		responseChan: responseChan,
`, callName, callName)
	gen.writeCallStructFields(ftype, paramNames)
	gen.pf(`	}

	callEvent := &%s{
		%s: call,
	}

	m.imp.callChan <- callEvent

	resp := <-responseChan

	if resp.Type == "panic" {
		panic(resp.PanicValue)
	}

`, gen.callName, methodName)

	gen.writeReturnStatement(ftype)
	gen.pf("}\n\n")
}

// writeMethodParams writes the method parameters in the form "name type, name2 type2".
func (gen *codeGenerator) writeMethodParams(ftype *ast.FuncType, paramNames []string) {
	if !hasParams(ftype) {
		return
	}

	paramNameIndex := 0

	for i, param := range ftype.Params.List {
		if i > 0 {
			gen.pf(", ")
		}

		paramType := exprToString(gen.fset, param.Type)
		paramNameIndex = gen.writeParamForField(param, paramType, paramNames, paramNameIndex)
	}
}

// writeParamForField writes parameters for a single field (which may contain multiple names).
func (gen *codeGenerator) writeParamForField(
	param *ast.Field, paramType string, paramNames []string, paramNameIndex int,
) int {
	if len(param.Names) > 0 {
		return gen.writeNamedParams(param, paramType, paramNameIndex)
	}

	gen.pf("%s %s", paramNames[paramNameIndex], paramType)

	return paramNameIndex + 1
}

// writeNamedParams writes multiple named parameters of the same type.
func (gen *codeGenerator) writeNamedParams(param *ast.Field, paramType string, paramNameIndex int) int {
	for j, name := range param.Names {
		if j > 0 {
			gen.pf(", ")
		}

		gen.pf("%s %s", name.Name, paramType)

		paramNameIndex++
	}

	return paramNameIndex
}

// writeCallStructFields writes the field assignments for initializing a call struct.
func (gen *codeGenerator) writeCallStructFields(ftype *ast.FuncType, paramNames []string) {
	if !hasParams(ftype) {
		return
	}

	totalParams := countFields(ftype.Params)
	paramNameIndex := 0
	unnamedIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(gen.fset, param.Type)
		paramNameIndex, unnamedIndex = gen.writeCallStructField(
			param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams,
		)
	}
}

// writeCallStructField writes a single field assignment for a call struct initialization.
func (gen *codeGenerator) writeCallStructField(
	param *ast.Field, paramType string, paramNames []string, paramNameIndex, unnamedIndex, totalParams int,
) (int, int) {
	if len(param.Names) > 0 {
		for i, name := range param.Names {
			fieldName := getParamFieldName(param, i, unnamedIndex, paramType, totalParams)
			gen.pf("\t\t%s: %s,\n", fieldName, name.Name)

			paramNameIndex++
		}

		return paramNameIndex, unnamedIndex
	}

	fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
	gen.pf("\t\t%s: %s,\n", fieldName, paramNames[paramNameIndex])

	return paramNameIndex + 1, unnamedIndex + 1
}

// writeReturnStatement writes the return statement for a mock method.
func (gen *codeGenerator) writeReturnStatement(ftype *ast.FuncType) {
	if !hasResults(ftype) {
		gen.pf("\treturn\n")
		return
	}

	gen.pf("\treturn")
	gen.writeReturnValues(ftype)
	gen.pf("\n")
}

// writeReturnValues writes all return values from the response struct.
func (gen *codeGenerator) writeReturnValues(ftype *ast.FuncType) {
	for i, r := range extractResults(gen.fset, ftype) {
		if i > 0 {
			gen.pf(",")
		}

		gen.pf(" resp.%s", r.Name)
	}
}

// generateExpectCallToStruct generates the struct for expecting specific method calls.
func (gen *codeGenerator) generateExpectCallToStruct() {
	gen.ps(executeTemplate(expectCallToStructTemplate, gen.templateData()))
}

// generateExpectCallToMethods generates expectation methods for each interface method.
func (gen *codeGenerator) generateExpectCallToMethods() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateExpectCallToMethod(methodName, ftype)
	})
}

// generateExpectCallToMethod generates a single expectation method that validates and returns a call.
func (gen *codeGenerator) generateExpectCallToMethod(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	paramNames := extractParamNames(ftype)

	gen.pf("func (e *%s) ", gen.expectCallToName)
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.pf(" *%s {\n", callName)

	gen.generateValidatorFunction(methodName, ftype, paramNames)

	gen.pf(`	call := e.imp.GetCall(e.timeout, validator)
	return call.As%s()
}

`, methodName)
}

// generateValidatorFunction generates a validator closure that checks method name and parameters.
func (gen *codeGenerator) generateValidatorFunction(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.pf(`	validator := func(c *%s) bool {
		if c.Name() != %q {
			return false
		}
`, gen.callName, methodName)

	if hasParams(ftype) {
		gen.pf("		methodCall := c.As%s()\n", methodName)
		gen.writeValidatorChecks(ftype, paramNames)
	}

	gen.pf(`		return true
	}

`)
}

// writeValidatorChecks writes parameter validation checks for an expectation method.
func (gen *codeGenerator) writeValidatorChecks(ftype *ast.FuncType, paramNames []string) {
	totalParams := countFields(ftype.Params)
	paramNameIndex := 0
	unnamedIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(gen.fset, param.Type)
		paramNameIndex, unnamedIndex = gen.writeValidatorCheck(
			param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams,
		)
	}
}

// writeValidatorCheck writes a single parameter validation check.
func (gen *codeGenerator) writeValidatorCheck(
	param *ast.Field, paramType string, paramNames []string, paramNameIndex, unnamedIndex, totalParams int,
) (int, int) {
	if len(param.Names) > 0 {
		for i, name := range param.Names {
			fieldName := getParamFieldName(param, i, unnamedIndex, paramType, totalParams)
			gen.pf(`		if methodCall.%s != %s {
			return false
		}
`, fieldName, name.Name)

			paramNameIndex++
		}

		return paramNameIndex, unnamedIndex
	}

	fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
	gen.pf(`		if methodCall.%s != %s {
			return false
		}
`, fieldName, paramNames[paramNameIndex])

	return paramNameIndex + 1, unnamedIndex + 1
}

// generateTimedStruct generates the struct and method for timed call expectations.
func (gen *codeGenerator) generateTimedStruct() {
	gen.ps(executeTemplate(timedStructTemplate, gen.templateData()))
}

// generateGetCallMethod generates the GetCall method that retrieves matching calls from queue or channel.
func (gen *codeGenerator) generateGetCallMethod() {
	gen.ps(executeTemplate(getCallMethodTemplate, gen.templateData()))
}

// generateGetCurrentCallMethod generates the GetCurrentCall method that returns the current or next call.
func (gen *codeGenerator) generateGetCurrentCallMethod() {
	gen.ps(executeTemplate(getCurrentCallMethodTemplate, gen.templateData()))
}

// generateConstructor generates the New{ImpName} constructor function.
func (gen *codeGenerator) generateConstructor() {
	gen.ps(executeTemplate(constructorTemplate, gen.templateData()))
}

// Functions - Public

// generateImplementationCode generates the complete mock implementation code for an interface.
func generateImplementationCode(
	identifiedInterface *ast.InterfaceType,
	info generatorInfo,
	fset *token.FileSet,
) (string, error) {
	impName := info.impName

	gen := &codeGenerator{
		codeWriter:          codeWriter{fset: fset},
		pkgName:             info.pkgName,
		impName:             impName,
		mockName:            impName + "Mock",
		callName:            impName + "Call",
		expectCallToName:    impName + "ExpectCallTo",
		timedName:           impName + "Timed",
		identifiedInterface: identifiedInterface,
		methodNames:         collectMethodNames(identifiedInterface),
	}

	gen.generateHeader()
	gen.generateMockStruct()
	gen.generateMainStruct()
	gen.generateMethodStructs()
	gen.generateMockMethods()
	gen.generateCallStruct()
	gen.generateExpectCallToStruct()
	gen.generateExpectCallToMethods()
	gen.generateTimedStruct()
	gen.generateGetCallMethod()
	gen.generateGetCurrentCallMethod()
	gen.generateConstructor()

	formatted, err := format.Source(gen.bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// Functions - Private

// collectMethodNames extracts all method names from an interface.
func collectMethodNames(iface *ast.InterfaceType) []string {
	var methodNames []string

	forEachInterfaceMethod(iface, func(methodName string, _ *ast.FuncType) {
		methodNames = append(methodNames, methodName)
	})

	return methodNames
}

// forEachInterfaceMethod iterates over interface methods and calls the callback for each.
func forEachInterfaceMethod(iface *ast.InterfaceType, callback func(methodName string, ftype *ast.FuncType)) {
	for _, field := range iface.Methods.List {
		processFieldMethods(field, callback)
	}
}

// processFieldMethods processes all method names in a field and calls the callback for each valid method.
func processFieldMethods(field *ast.Field, callback func(methodName string, ftype *ast.FuncType)) {
	// Skip embedded interfaces (they have no names)
	if len(field.Names) == 0 {
		return
	}

	// Skip non-function types (shouldn't happen in a valid interface, but be safe)
	ftype, ok := field.Type.(*ast.FuncType)
	if !ok {
		return
	}

	// Process each method name with the same function type
	for _, methodName := range field.Names {
		callback(methodName.Name, ftype)
	}
}

// getParamFieldName returns the struct field name for a parameter.
// For named params, returns the name. For unnamed params, generates a name based on type/index.
func getParamFieldName(param *ast.Field, nameIdx int, unnamedIdx int, paramType string, totalParams int) string {
	if len(param.Names) > 0 {
		return param.Names[nameIdx].Name
	}

	return generateParamName(unnamedIdx, paramType, totalParams)
}

// extractParamNames extracts or generates parameter names from a function type.
func extractParamNames(ftype *ast.FuncType) []string {
	paramNames := make([]string, 0)
	if !hasParams(ftype) {
		return paramNames
	}

	paramIndex := 0

	for _, param := range ftype.Params.List {
		paramNames, paramIndex = appendParamNames(param, paramNames, paramIndex)
	}

	return paramNames
}

// appendParamNames appends parameter names to the list, generating names for unnamed parameters.
func appendParamNames(param *ast.Field, paramNames []string, paramIndex int) ([]string, int) {
	if len(param.Names) > 0 {
		for _, name := range param.Names {
			paramNames = append(paramNames, name.Name)
		}

		return paramNames, paramIndex
	}

	paramName := fmt.Sprintf("param%d", paramIndex)
	paramNames = append(paramNames, paramName)

	return paramNames, paramIndex + 1
}

// renderFieldList renders a *ast.FieldList as Go code for return types.
func renderFieldList(fset *token.FileSet, fieldList *ast.FieldList) string {
	if fieldList == nil || len(fieldList.List) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("(")

	for i, field := range fieldList.List {
		if i > 0 {
			buf.WriteString(", ")
		}

		renderField(fset, field, &buf)
	}

	buf.WriteString(")")

	return buf.String()
}

// renderField renders a single field with its name and type.
func renderField(fset *token.FileSet, field *ast.Field, buf *bytes.Buffer) {
	// Names
	for j, name := range field.Names {
		if j > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(name.Name)
	}

	// Type
	if len(field.Names) > 0 {
		buf.WriteString(" ")
	}

	buf.WriteString(exprToString(fset, field.Type))
}

// generateParamName generates a field name for an unnamed parameter
// Uses common conventions: single string -> "S", single int -> "Input", multiple -> "A", "B", "C", etc.
func generateParamName(index int, paramType string, totalParams int) string {
	// Remove common prefixes/suffixes for comparison
	normalized := strings.TrimSpace(paramType)

	// Single parameter cases
	if totalParams == 1 {
		if normalized == "string" {
			return "S"
		}

		if normalized == "int" {
			return "I"
		}
	}

	// Multiple parameters - use A, B, C, etc.
	names := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	if index < len(names) {
		return names[index]
	}

	// Fallback
	return fmt.Sprintf("Arg%d", index)
}

// findFunctionInAST finds a function or method declaration in the AST files.
// funcName can be a plain function name like "PrintSum" or a method reference like "PingPongPlayer.Play".
//
//nolint:cyclop,nestif // AST searching requires some complexity
func findFunctionInAST(astFiles []*ast.File, funcName string, pkgImportPath string) (*ast.FuncDecl, error) {
	// Check if this is a method reference (TypeName.MethodName)
	typeName, methodName, isMethod := strings.Cut(funcName, ".")

	for _, file := range astFiles {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if isMethod {
				// Looking for a method - must have a receiver with matching type and method name
				if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
					continue
				}

				if funcDecl.Name.Name != methodName {
					continue
				}

				if matchesReceiverType(funcDecl.Recv.List[0].Type, typeName) {
					return funcDecl, nil
				}
			} else {
				// Looking for a plain function - must not have a receiver
				if funcDecl.Recv != nil {
					continue
				}

				if funcDecl.Name.Name == funcName {
					return funcDecl, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("%w: named %q in package %q", errFunctionNotFound, funcName, pkgImportPath)
}

// matchesReceiverType checks if the receiver type expression matches the given type name.
// Handles both value receivers (T) and pointer receivers (*T).
func matchesReceiverType(expr ast.Expr, typeName string) bool {
	switch recv := expr.(type) {
	case *ast.Ident:
		return recv.Name == typeName
	case *ast.StarExpr:
		// Pointer receiver - check the underlying type
		if ident, ok := recv.X.(*ast.Ident); ok {
			return ident.Name == typeName
		}
	}

	return false
}
