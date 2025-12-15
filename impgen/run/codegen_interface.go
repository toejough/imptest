package run

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"text/template"
)

// Entry Point - Public

// generateImplementationCode generates the complete mock implementation code for an interface.
func generateImplementationCode(
	astFiles []*ast.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
) (string, error) {
	identifiedInterface, err := getMatchingInterfaceFromAST(astFiles, info.localInterfaceName, pkgImportPath)
	if err != nil {
		return "", err
	}

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
		methodNames:         interfaceCollectMethodNames(identifiedInterface),
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

// Types

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

// codeGenerator Methods

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

// execTemplate executes a template and writes the result to the buffer.
func (gen *codeGenerator) execTemplate(tmpl *template.Template, data any) {
	gen.ps(executeTemplate(tmpl, data))
}

// generateCallStruct generates the union call struct that can hold any method call.
func (gen *codeGenerator) generateCallStruct() {
	gen.execTemplate(callStructTemplate, gen.callStructData())
}

// generateHeader writes the package declaration and imports for the generated file.
func (gen *codeGenerator) generateHeader() {
	gen.execTemplate(headerTemplate, gen.templateData())
}

// generateMockStruct generates the mock struct that wraps the implementation.
func (gen *codeGenerator) generateMockStruct() {
	gen.execTemplate(mockStructTemplate, gen.templateData())
}

// generateMainStruct generates the main implementation struct that handles test call tracking.
func (gen *codeGenerator) generateMainStruct() {
	gen.execTemplate(mainStructTemplate, gen.templateData())
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
	visitParams(gen.fset, ftype, func(
		param *ast.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		if len(param.Names) > 0 {
			gen.writeNamedParamFields(param, paramType, unnamedIndex, totalParams)
			return paramNameIndex + len(param.Names), unnamedIndex
		}

		gen.writeUnnamedParamField(param, paramType, unnamedIndex, totalParams)

		return paramNameIndex + 1, unnamedIndex + 1
	})
}

// writeNamedParamFields writes fields for named parameters.
func (gen *codeGenerator) writeNamedParamFields(param *ast.Field, paramType string, unnamedIndex, totalParams int) {
	for i := range param.Names {
		fieldName := interfaceGetParamFieldName(param, i, unnamedIndex, paramType, totalParams)
		gen.pf("\t%s %s\n", fieldName, paramType)
	}
}

// writeUnnamedParamField writes a field for an unnamed parameter.
func (gen *codeGenerator) writeUnnamedParamField(param *ast.Field, paramType string, unnamedIndex, totalParams int) {
	fieldName := interfaceGetParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
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
	gen.execTemplate(injectPanicMethodTemplate, gen.methodTemplateData(methodCallName))
}

// generateResolveMethod generates the Resolve method for methods with no return values.
func (gen *codeGenerator) generateResolveMethod(methodCallName string) {
	gen.execTemplate(resolveMethodTemplate, gen.methodTemplateData(methodCallName))
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
	paramNames := interfaceExtractParamNames(gen.fset, ftype)

	gen.writeMockMethodSignature(methodName, ftype, paramNames)
	gen.writeMockMethodCallCreation(callName, ftype, paramNames)
	gen.writeMockMethodEventDispatch(methodName)
	gen.writeMockMethodResponseHandling()
	gen.writeReturnStatement(ftype)
	gen.pf("}\n\n")
}

// writeMockMethodSignature writes the mock method signature and opening brace.
func (gen *codeGenerator) writeMockMethodSignature(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.pf("func (m *%s) ", gen.mockName)
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.pf("%s", interfaceRenderFieldList(gen.fset, ftype.Results))
	gen.pf(" {\n")
}

// writeMockMethodCallCreation writes the response channel and call struct creation.
func (gen *codeGenerator) writeMockMethodCallCreation(callName string, ftype *ast.FuncType, paramNames []string) {
	gen.pf("\tresponseChan := make(chan %sResponse, 1)\n\n", callName)
	gen.pf("\tcall := &%s{\n", callName)
	gen.pf("\t\tresponseChan: responseChan,\n")
	gen.writeCallStructFields(ftype, paramNames)
	gen.pf("\t}\n\n")
}

// writeMockMethodEventDispatch writes the call event creation and dispatch to the imp.
func (gen *codeGenerator) writeMockMethodEventDispatch(methodName string) {
	gen.pf("\tcallEvent := &%s{\n", gen.callName)
	gen.pf("\t\t%s: call,\n", methodName)
	gen.pf("\t}\n\n")
	gen.pf("\tm.imp.callChan <- callEvent\n\n")
}

// writeMockMethodResponseHandling writes the response reception and panic handling.
func (gen *codeGenerator) writeMockMethodResponseHandling() {
	gen.pf("\tresp := <-responseChan\n\n")
	gen.pf("\tif resp.Type == \"panic\" {\n")
	gen.pf("\t\tpanic(resp.PanicValue)\n")
	gen.pf("\t}\n\n")
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
	visitParams(gen.fset, ftype, func(
		param *ast.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		return gen.writeCallStructField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams)
	})
}

// writeCallStructField writes a single field assignment for a call struct initialization.
func (gen *codeGenerator) writeCallStructField(
	param *ast.Field, paramType string, paramNames []string, paramNameIndex, unnamedIndex, totalParams int,
) (int, int) {
	if len(param.Names) > 0 {
		for i, name := range param.Names {
			fieldName := interfaceGetParamFieldName(param, i, unnamedIndex, paramType, totalParams)
			gen.pf("\t\t%s: %s,\n", fieldName, name.Name)

			paramNameIndex++
		}

		return paramNameIndex, unnamedIndex
	}

	fieldName := interfaceGetParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
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
	gen.execTemplate(expectCallToStructTemplate, gen.templateData())
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
	paramNames := interfaceExtractParamNames(gen.fset, ftype)

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
	visitParams(gen.fset, ftype, func(
		param *ast.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		return gen.writeValidatorCheck(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams)
	})
}

// writeValidatorCheck writes a single parameter validation check.
func (gen *codeGenerator) writeValidatorCheck(
	param *ast.Field, paramType string, paramNames []string, paramNameIndex, unnamedIndex, totalParams int,
) (int, int) {
	if len(param.Names) > 0 {
		for i, name := range param.Names {
			fieldName := interfaceGetParamFieldName(param, i, unnamedIndex, paramType, totalParams)
			gen.pf(`		if methodCall.%s != %s {
			return false
		}
`, fieldName, name.Name)

			paramNameIndex++
		}

		return paramNameIndex, unnamedIndex
	}

	fieldName := interfaceGetParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
	gen.pf(`		if methodCall.%s != %s {
			return false
		}
`, fieldName, paramNames[paramNameIndex])

	return paramNameIndex + 1, unnamedIndex + 1
}

// generateTimedStruct generates the struct and method for timed call expectations.
func (gen *codeGenerator) generateTimedStruct() {
	gen.execTemplate(timedStructTemplate, gen.templateData())
}

// generateGetCallMethod generates the GetCall method that retrieves matching calls from queue or channel.
func (gen *codeGenerator) generateGetCallMethod() {
	gen.execTemplate(getCallMethodTemplate, gen.templateData())
}

// generateGetCurrentCallMethod generates the GetCurrentCall method that returns the current or next call.
func (gen *codeGenerator) generateGetCurrentCallMethod() {
	gen.execTemplate(getCurrentCallMethodTemplate, gen.templateData())
}

// generateConstructor generates the New{ImpName} constructor function.
func (gen *codeGenerator) generateConstructor() {
	gen.execTemplate(constructorTemplate, gen.templateData())
}

// Private Functions

// interfaceCollectMethodNames extracts all method names from an interface.
func interfaceCollectMethodNames(iface *ast.InterfaceType) []string {
	var methodNames []string

	forEachInterfaceMethod(iface, func(methodName string, _ *ast.FuncType) {
		methodNames = append(methodNames, methodName)
	})

	return methodNames
}

// forEachInterfaceMethod iterates over interface methods and calls the callback for each.
func forEachInterfaceMethod(iface *ast.InterfaceType, callback func(methodName string, ftype *ast.FuncType)) {
	for _, field := range iface.Methods.List {
		interfaceProcessFieldMethods(field, callback)
	}
}

// interfaceProcessFieldMethods processes all method names in a field and calls the callback for each valid method.
func interfaceProcessFieldMethods(field *ast.Field, callback func(methodName string, ftype *ast.FuncType)) {
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

// interfaceGetParamFieldName returns the struct field name for a parameter.
// For named params, returns the name. For unnamed params, generates a name based on type/index.
func interfaceGetParamFieldName(
	param *ast.Field, nameIdx int, unnamedIdx int, paramType string, totalParams int,
) string {
	if len(param.Names) > 0 {
		return param.Names[nameIdx].Name
	}

	return interfaceGenerateParamName(unnamedIdx, paramType, totalParams)
}

// interfaceExtractParamNames extracts or generates parameter names from a function type.
func interfaceExtractParamNames(fset *token.FileSet, ftype *ast.FuncType) []string {
	params := extractParams(fset, ftype)
	names := make([]string, len(params))

	for i, p := range params {
		names[i] = p.Name
	}

	return names
}

// interfaceRenderFieldList renders a *ast.FieldList as Go code for return types.
func interfaceRenderFieldList(fset *token.FileSet, fieldList *ast.FieldList) string {
	if fieldList == nil || len(fieldList.List) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("(")

	for i, field := range fieldList.List {
		if i > 0 {
			buf.WriteString(", ")
		}

		interfaceRenderField(fset, field, &buf)
	}

	buf.WriteString(")")

	return buf.String()
}

// interfaceRenderField renders a single field with its name and type.
func interfaceRenderField(fset *token.FileSet, field *ast.Field, buf *bytes.Buffer) {
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

// interfaceGenerateParamName generates a field name for an unnamed parameter
// Uses common conventions: single string -> "S", single int -> "Input", multiple -> "A", "B", "C", etc.
func interfaceGenerateParamName(index int, paramType string, totalParams int) string {
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
