package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	go_types "go/types"
	"strings"
)

var errUnsupportedEmbeddedType = errors.New("unsupported embedded type")

// Entry Point - Public

// generateImplementationCode generates the complete mock implementation code for an interface.
func generateImplementationCode(
	astFiles []*ast.File,
	info generatorInfo,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (string, error) {
	gen, err := newCodeGenerator(astFiles, info, fset, typesInfo, pkgImportPath, pkgLoader, ifaceWithDetails)
	if err != nil {
		return "", err
	}

	code, err := gen.generate()
	if err != nil {
		return "", err
	}

	return code, nil
}

// newCodeGenerator initializes a codeGenerator with common properties and performs initial setup.
func newCodeGenerator(
	astFiles []*ast.File,
	info generatorInfo,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (*codeGenerator, error) {
	impName := info.impName

	var (
		pkgPath, qualifier string
		err                error
	)
	if pkgImportPath != "." {
		pkgPath, qualifier, err = GetPackageInfo(
			info.interfaceName,
			pkgLoader,
			info.pkgName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get interface package info: %w", err)
		}
	}

	gen := &codeGenerator{
		baseGenerator: newBaseGenerator(

			fset, info.pkgName, impName, pkgPath, qualifier, ifaceWithDetails.typeParams, typesInfo,
		),

		mockName: impName + "Mock", callName: impName + "Call",

		expectCallIsName: impName + "ExpectCallIs", timedName: impName + "Timed",

		identifiedInterface: ifaceWithDetails.iface, astFiles: astFiles,

		pkgImportPath: pkgImportPath, pkgLoader: pkgLoader,
	}

	methodNames, err := interfaceCollectMethodNames(ifaceWithDetails.iface, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return nil, err
	}

	gen.methodNames = methodNames

	return gen, nil
}

// generate orchestrates the code generation process after initialization.
func (gen *codeGenerator) generate() (string, error) {
	// Pre-scan to determine if reflect import is needed
	gen.checkIfReflectNeeded()
	// Pre-scan to determine if imptest import is needed
	gen.checkIfImptestNeeded()
	// Pre-scan to see if qualifier is needed
	gen.checkIfQualifierNeeded()

	err := gen.checkIfValidForExternalUsage()
	if err != nil {
		return "", err
	}

	gen.generateHeader()
	gen.generateMockStruct()
	gen.generateMainStruct()
	gen.generateMethodStructs()
	gen.generateMockMethods()
	gen.generateCallStruct()
	gen.generateExpectCallIsStruct()
	gen.generateMethodBuilders()
	gen.generateTimedStruct()
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
	baseGenerator

	mockName            string
	callName            string
	expectCallIsName    string
	timedName           string
	identifiedInterface *ast.InterfaceType
	astFiles            []*ast.File
	pkgImportPath       string
	pkgLoader           PackageLoader
	methodNames         []string
}

// codeGenerator Methods

// checkIfReflectNeeded pre-scans all interface methods to determine if reflect import is needed.
func (gen *codeGenerator) checkIfReflectNeeded() {
	gen.forEachMethod(func(_ string, ftype *ast.FuncType) {
		if ftype.Params == nil {
			return
		}

		for _, param := range ftype.Params.List {
			if !isComparableExpr(param.Type, gen.typesInfo) {
				gen.needsReflect = true
				return // Early exit once we know reflect is needed
			}
		}
	})
}

// checkIfImptestNeeded pre-scans all interface methods to determine if imptest import is needed.
// imptest is needed when any method has parameters (for ExpectArgsShould).
func (gen *codeGenerator) checkIfImptestNeeded() {
	gen.forEachMethod(func(_ string, ftype *ast.FuncType) {
		if ftype.Params != nil && len(ftype.Params.List) > 0 {
			gen.needsImptest = true
			return // Early exit once we know imptest is needed
		}
	})
}

// checkIfQualifierNeeded pre-scans to determine if the package qualifier is needed.
func (gen *codeGenerator) checkIfQualifierNeeded() {
	gen.forEachMethod(func(_ string, ftype *ast.FuncType) {
		gen.baseGenerator.checkIfQualifierNeeded(ftype)
	})
}

// checkIfValidForExternalUsage checks if the interface can be mocked from an external package.
func (gen *codeGenerator) checkIfValidForExternalUsage() error {
	var validationErr error

	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		if validationErr != nil {
			return
		}

		err := gen.baseGenerator.checkIfValidForExternalUsage(ftype)
		if err != nil {
			validationErr = fmt.Errorf("method '%s': %w", methodName, err)
		}
	})

	return validationErr
}

// forEachMethod iterates over interface methods and calls the callback for each.
// This is safe to call without error checking because we already validated the interface
// structure during method name collection in generateImplementationCode. If an error occurs
// here, it indicates a programming error and will cause a panic in the underlying function.
func (gen *codeGenerator) forEachMethod(callback func(methodName string, ftype *ast.FuncType)) {
	// Ignore error - interface was already validated during method name collection
	_ = forEachInterfaceMethod(
		gen.identifiedInterface,
		gen.astFiles,
		gen.fset,
		gen.pkgImportPath,
		gen.pkgLoader,
		callback,
	)
}

// templateData returns common template data for this generator.
func (gen *codeGenerator) templateData() templateData {
	return templateData{
		baseTemplateData: baseTemplateData{
			PkgName:        gen.pkgName,
			ImpName:        gen.impName,
			PkgPath:        gen.pkgPath,
			Qualifier:      gen.qualifier,
			NeedsQualifier: gen.needsQualifier,
			TypeParamsDecl: gen.formatTypeParamsDecl(),
			TypeParamsUse:  gen.formatTypeParamsUse(),
		},
		MockName:         gen.mockName,
		CallName:         gen.callName,
		ExpectCallIsName: gen.expectCallIsName,
		TimedName:        gen.timedName,
		MethodNames:      gen.methodNames,
		NeedsReflect:     gen.needsReflect,
		NeedsImptest:     gen.needsImptest,
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
			Name:          methodName,
			CallName:      gen.methodCallName(methodName),
			TypeParamsUse: gen.formatTypeParamsUse(),
		}
	}

	return callStructTemplateData{
		templateData: gen.templateData(),
		Methods:      methods,
	}
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

// methodBuilderName returns the builder struct name for a method (e.g. "MyImpAddBuilder").
func (gen *codeGenerator) methodBuilderName(methodName string) string {
	return gen.impName + methodName + "Builder"
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
	gen.pf(`type %s%s struct {
	responseChan chan %sResponse%s
	done bool
`, callName, gen.formatTypeParamsDecl(), callName, gen.formatTypeParamsUse())

	if hasParams(ftype) {
		gen.generateCallStructParamFields(ftype)
	}

	gen.pf("}\n\n")
}

// generateCallStructParamFields generates the parameter fields for a call struct.
func (gen *codeGenerator) generateCallStructParamFields(ftype *ast.FuncType) {
	visitParams(ftype, gen.typeWithQualifier, func(
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
	structType := normalizeVariadicType(paramType)

	for i := range param.Names {
		fieldName := interfaceGetParamFieldName(param, i, unnamedIndex, structType, totalParams)
		gen.pf("\t%s %s\n", fieldName, structType)
	}
}

// writeUnnamedParamField writes a field for an unnamed parameter.
func (gen *codeGenerator) writeUnnamedParamField(param *ast.Field, paramType string, unnamedIndex, totalParams int) {
	structType := normalizeVariadicType(paramType)

	fieldName := interfaceGetParamFieldName(param, 0, unnamedIndex, structType, totalParams)
	gen.pf("\t%s %s\n", fieldName, structType)
}

// generateMethodResponseStruct generates the response struct for a method, which holds return values or panic data.
func (gen *codeGenerator) generateMethodResponseStruct(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	gen.pf(`type %sResponse%s struct {
	Type string // "return", "panic", or "resolve"
`, callName, gen.formatTypeParamsDecl())

	if hasResults(ftype) {
		gen.generateResponseStructResultFields(ftype)
	}

	gen.pf("	PanicValue any\n}\n\n")
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
	resultType := gen.typeWithQualifier(ftype.Results.List[0].Type)
	gen.pf(`func (c *%s%s) InjectResult(result %s) {
	c.done = true
	c.responseChan <- %sResponse%s{Type: "return"`,
		methodCallName, gen.formatTypeParamsUse(), resultType, methodCallName, gen.formatTypeParamsUse())

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
	gen.pf("func (c *%s%s) InjectResults(", methodCallName, gen.formatTypeParamsUse())

	returnParamNames := gen.writeInjectResultsParams(ftype)

	gen.pf(`) {
	c.done = true
	resp := %sResponse%s{Type: "return"`, methodCallName, gen.formatTypeParamsUse())

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

		paramName := fmt.Sprintf("r%d", resultIdx)
		gen.pf("%s %s", paramName, result.Type)
		names[resultIdx] = paramName
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
	gen.pf("func (m *%s%s) ", gen.mockName, gen.formatTypeParamsUse())
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.pf("%s", gen.renderFieldList(ftype.Results))
	gen.pf(" {\n")
}

// writeMockMethodCallCreation writes the response channel and call struct creation.
func (gen *codeGenerator) writeMockMethodCallCreation(callName string, ftype *ast.FuncType, paramNames []string) {
	gen.pf("\tresponseChan := make(chan %sResponse%s, 1)\n\n", callName, gen.formatTypeParamsUse())
	gen.pf("\tcall := &%s%s{\n", callName, gen.formatTypeParamsUse())
	gen.pf("\t\tresponseChan: responseChan,\n")
	gen.writeCallStructFields(ftype, paramNames)
	gen.pf("\t}\n\n")
}

// writeMockMethodEventDispatch writes the call event creation and dispatch to the imp.
func (gen *codeGenerator) writeMockMethodEventDispatch(methodName string) {
	gen.pf("\tcallEvent := &%s%s{\n", gen.callName, gen.formatTypeParamsUse())
	gen.pf("\t\t%s: call,\n", methodName)
	gen.pf("\t}\n\n")
	gen.pf("\tm.imp.CallChan <- callEvent\n\n")
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

		paramType := gen.typeWithQualifier(param.Type)
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

// forEachParamField iterates over parameter fields, handling both named and unnamed parameters.
// It calls the action callback for each field with the computed field name and parameter name.
func forEachParamField(
	param *ast.Field,
	paramType string,
	paramNames []string,
	paramNameIndex, unnamedIndex, totalParams int,
	action func(fieldName, paramName string),
) (int, int) {
	if len(param.Names) > 0 {
		for i, name := range param.Names {
			fieldName := interfaceGetParamFieldName(param, i, unnamedIndex, paramType, totalParams)
			action(fieldName, name.Name)

			paramNameIndex++
		}

		return paramNameIndex, unnamedIndex
	}

	fieldName := interfaceGetParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
	action(fieldName, paramNames[paramNameIndex])

	return paramNameIndex + 1, unnamedIndex + 1
}

// writeCallStructFields writes the field assignments for initializing a call struct.
func (gen *codeGenerator) writeCallStructFields(ftype *ast.FuncType, paramNames []string) {
	visitParams(ftype, gen.typeWithQualifier, func(
		param *ast.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		return gen.writeCallStructField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams)
	})
}

// writeCallStructField writes a single field assignment for a call struct initialization.
func (gen *codeGenerator) writeCallStructField(
	param *ast.Field, paramType string, paramNames []string, paramNameIndex, unnamedIndex, totalParams int,
) (int, int) {
	return forEachParamField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams,
		func(fieldName, paramName string) {
			gen.pf("\t\t%s: %s,\n", fieldName, paramName)
		})
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

// generateExpectCallIsStruct generates the struct for expecting specific method calls.
func (gen *codeGenerator) generateExpectCallIsStruct() {
	gen.execTemplate(expectCallIsStructTemplate, gen.templateData())
}

// generateMethodBuilders generates builder structs and methods for each interface method.
func (gen *codeGenerator) generateMethodBuilders() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateMethodBuilder(methodName, ftype)
	})
}

// generateMethodBuilder generates the builder struct and all its methods for a single interface method.
func (gen *codeGenerator) generateMethodBuilder(methodName string, ftype *ast.FuncType) {
	builderName := gen.methodBuilderName(methodName)
	callName := gen.methodCallName(methodName)

	// Generate builder struct
	gen.pf("type %s%s struct {\n", builderName, gen.formatTypeParamsDecl())
	gen.pf("\timp     *%s%s\n", gen.impName, gen.formatTypeParamsUse())
	gen.pf("\ttimeout time.Duration\n")
	gen.pf("}\n\n")

	// Generate ExpectCallIs.MethodName() -> returns builder
	gen.pf("func (e *%s%s) %s() *%s%s {\n",
		gen.expectCallIsName, gen.formatTypeParamsUse(), methodName, builderName, gen.formatTypeParamsUse())
	gen.pf("\treturn &%s%s{imp: e.imp, timeout: e.timeout}\n", builderName, gen.formatTypeParamsUse())
	gen.pf("}\n\n")

	// Generate ExpectArgsAre (type-safe)
	gen.generateExpectArgsAre(methodName, ftype, builderName, callName)

	// Generate ExpectArgsShould (matcher-based)
	gen.generateExpectArgsShould(methodName, ftype, builderName, callName)

	// Generate shortcut InjectResult/InjectPanic/Resolve
	gen.generateBuilderShortcuts(methodName, ftype, builderName, callName)
}

// generateExpectArgsAre generates the type-safe ExpectArgsAre method on the builder.
func (gen *codeGenerator) generateExpectArgsAre(methodName string, ftype *ast.FuncType, builderName, callName string) {
	paramNames := interfaceExtractParamNames(gen.fset, ftype)

	// Method signature
	gen.pf("func (bldr *%s%s) ExpectArgsAre(", builderName, gen.formatTypeParamsUse())
	gen.writeMethodParams(ftype, paramNames)
	gen.pf(") *%s%s {\n", callName, gen.formatTypeParamsUse())

	// Validator function
	gen.pf("\tvalidator := func(c *%s%s) bool {\n", gen.callName, gen.formatTypeParamsUse())
	gen.pf("\t\tif c.Name() != %q {\n", methodName)
	gen.pf("\t\t\treturn false\n")
	gen.pf("\t	}\n")

	if hasParams(ftype) {
		gen.pf("\t\tmethodCall := c.As%s()\n", methodName)
		gen.writeExpectArgsAreChecks(ftype, paramNames)
	}

	gen.pf("\t\treturn true\n")
	gen.pf("\t}\n\n")

	// GetCall and return
	gen.pf("\tcall := bldr.imp.GetCall(bldr.timeout, validator)\n")
	gen.pf("\treturn call.As%s()\n", methodName)
	gen.pf("}\n\n")
}

// writeExpectArgsAreChecks writes parameter equality checks for ExpectArgsAre.
func (gen *codeGenerator) writeExpectArgsAreChecks(ftype *ast.FuncType, paramNames []string) {
	visitParams(ftype, gen.typeWithQualifier, func(
		param *ast.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		isComparable := isComparableExpr(param.Type, gen.typesInfo)

		return forEachParamField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams,
			func(fieldName, paramName string) {
				gen.writeEqualityCheck(fieldName, paramName, isComparable)
			})
	})
}

// writeEqualityCheck writes an equality check (== or DeepEqual).
func (gen *codeGenerator) writeEqualityCheck(fieldName, expectedName string, isComparable bool) {
	if isComparable {
		gen.pf("\t\tif methodCall.%s != %s {\n", fieldName, expectedName)
	} else {
		gen.needsReflect = true
		gen.pf("\t\tif !reflect.DeepEqual(methodCall.%s, %s) {\n", fieldName, expectedName)
	}

	gen.pf("\t\t\treturn false\n")
	gen.pf("\t\t}\n")
}

// generateExpectArgsShould generates the matcher-based ExpectArgsShould method on the builder.
func (gen *codeGenerator) generateExpectArgsShould(
	methodName string, ftype *ast.FuncType, builderName, callName string,
) {
	paramNames := interfaceExtractParamNames(gen.fset, ftype)

	// Method signature - all params are 'any'
	gen.pf("func (bldr *%s%s) ExpectArgsShould(", builderName, gen.formatTypeParamsUse())
	gen.writeMethodParamsAsAny(ftype, paramNames)
	gen.pf(") *%s%s {\n", callName, gen.formatTypeParamsUse())

	// Validator function
	gen.pf("\tvalidator := func(c *%s%s) bool {\n", gen.callName, gen.formatTypeParamsUse())
	gen.pf("\t\tif c.Name() != %q {\n", methodName)
	gen.pf("\t\t\treturn false\n")
	gen.pf("\t	}\n")

	if hasParams(ftype) {
		gen.pf("\t\tmethodCall := c.As%s()\n", methodName)
		gen.pf("\t\tvar ok bool\n")
		gen.writeExpectArgsShouldChecks(ftype, paramNames)
	}

	gen.pf("\t\treturn true\n")
	gen.pf("\t}\n\n")

	// GetCall and return
	gen.pf("\tcall := bldr.imp.GetCall(bldr.timeout, validator)\n")
	gen.pf("\treturn call.As%s()\n", methodName)
	gen.pf("}\n\n")
}

// writeExpectArgsShouldChecks writes matcher-based checks for ExpectArgsShould.
func (gen *codeGenerator) writeExpectArgsShouldChecks(ftype *ast.FuncType, paramNames []string) {
	visitParams(ftype, gen.typeWithQualifier, func(
		param *ast.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		return forEachParamField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams,
			func(fieldName, paramName string) {
				gen.writeMatcherCheck(fieldName, paramName)
			})
	})
}

// writeMatcherCheck writes a MatchValue check.
func (gen *codeGenerator) writeMatcherCheck(fieldName, expectedName string) {
	gen.pf("\tok, _ = imptest.MatchValue(methodCall.%s, %s)\n", fieldName, expectedName)
	gen.pf("\t\tif !ok {\n")
	gen.pf("\t\t\treturn false\n")
	gen.pf("\t\t}\n")
}

// generateBuilderShortcuts generates InjectResult/InjectPanic/Resolve shortcut methods on the builder.
func (gen *codeGenerator) generateBuilderShortcuts(
	methodName string, ftype *ast.FuncType, builderName, callName string,
) {
	// Validator that only checks method name
	validatorCode := fmt.Sprintf(`validator := func(c *%s%s) bool {
		return c.Name() == %q
	}

	call := bldr.imp.GetCall(bldr.timeout, validator)
	methodCall := call.As%s()
`, gen.callName, gen.formatTypeParamsUse(), methodName, methodName)

	if hasResults(ftype) {
		// Generate InjectResult shortcut
		if len(ftype.Results.List) == 1 {
			resultType := exprToString(gen.fset, ftype.Results.List[0].Type)
			gen.pf("func (bldr *%s%s) InjectResult(result %s) *%s%s {\n",
				builderName, gen.formatTypeParamsUse(), resultType, callName, gen.formatTypeParamsUse())
			gen.pf("\t%s", validatorCode)
			gen.pf("\tmethodCall.InjectResult(result)\n")
			gen.pf("\treturn methodCall\n")
			gen.pf("}\n\n")
		} else {
			// Multiple return values - InjectResults
			gen.pf("func (bldr *%s%s) InjectResults(", builderName, gen.formatTypeParamsUse())
			gen.writeInjectResultsParams(ftype)
			gen.pf(") *%s%s {\n", callName, gen.formatTypeParamsUse())
			gen.pf("\t%s", validatorCode)
			gen.pf("\tmethodCall.InjectResults(")
			gen.writeInjectResultsArgs(ftype)
			gen.pf(")\n")
			gen.pf("\treturn methodCall\n")
			gen.pf("}\n\n")
		}
	} else {
		// No results - generate Resolve shortcut
		gen.pf("func (bldr *%s%s) Resolve() *%s%s {\n",
			builderName, gen.formatTypeParamsUse(), callName, gen.formatTypeParamsUse())
		gen.pf("\t%s", validatorCode)
		gen.pf("\tmethodCall.Resolve()\n")
		gen.pf("\treturn methodCall\n")
		gen.pf("}\n\n")
	}

	// Generate InjectPanic shortcut (always available)
	gen.pf("func (bldr *%s%s) InjectPanic(msg any) *%s%s {\n",
		builderName, gen.formatTypeParamsUse(), callName, gen.formatTypeParamsUse())
	gen.pf("\t%s", validatorCode)
	gen.pf("\tmethodCall.InjectPanic(msg)\n")
	gen.pf("\treturn methodCall\n")
	gen.pf("}\n\n")
}

// writeMethodParamsAsAny writes method parameters with all types as 'any'.
func (gen *codeGenerator) writeMethodParamsAsAny(ftype *ast.FuncType, paramNames []string) {
	if !hasParams(ftype) {
		return
	}

	paramNameIndex := 0
	first := true

	visitParams(ftype, gen.typeWithQualifier, func(
		param *ast.Field, _ string, _, _, _ int,
	) (int, int) {
		if len(param.Names) > 0 {
			for _, name := range param.Names {
				if !first {
					gen.pf(", ")
				}

				first = false

				gen.pf("%s any", name.Name)

				paramNameIndex++
			}
		} else {
			if !first {
				gen.pf(", ")
			}

			first = false

			gen.pf("%s any", paramNames[paramNameIndex])
			paramNameIndex++
		}

		return 0, 0 // Not used in this context
	})
}

// writeInjectResultsArgs writes the argument list for InjectResults call.
func (gen *codeGenerator) writeInjectResultsArgs(ftype *ast.FuncType) {
	results := extractResults(gen.fset, ftype)
	for resultIndex := range results {
		if resultIndex > 0 {
			gen.pf(", ")
		}

		gen.pf("r%d", resultIndex)
	}
}

// generateTimedStruct generates the struct and method for timed call expectations.
func (gen *codeGenerator) generateTimedStruct() {
	gen.execTemplate(timedStructTemplate, gen.templateData())
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

// interfaceCollectMethodNames collects all method names from an interface, including embedded ones.
func interfaceCollectMethodNames(
	iface *ast.InterfaceType, astFiles []*ast.File, fset *token.FileSet, pkgImportPath string, pkgLoader PackageLoader,
) ([]string, error) {
	var methodNames []string

	err := forEachInterfaceMethod(
		iface, astFiles, fset, pkgImportPath, pkgLoader,
		func(methodName string, _ *ast.FuncType) {
			methodNames = append(methodNames, methodName)
		},
	)
	if err != nil {
		return nil, err
	}

	return methodNames, nil
}

// forEachInterfaceMethod iterates over interface methods and calls the callback for each,
// expanding embedded interfaces.
func forEachInterfaceMethod(
	iface *ast.InterfaceType,
	astFiles []*ast.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	callback func(methodName string, ftype *ast.FuncType),
) error {
	for _, field := range iface.Methods.List {
		err := interfaceProcessFieldMethods(field, astFiles, fset, pkgImportPath, pkgLoader, callback)
		if err != nil {
			return err
		}
	}

	return nil
}

// interfaceProcessFieldMethods handles a single field in an interface's method list.
func interfaceProcessFieldMethods(
	field *ast.Field,
	astFiles []*ast.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	callback func(methodName string, ftype *ast.FuncType),
) error {
	// Handle embedded interfaces (they have no names)
	if len(field.Names) == 0 {
		return interfaceExpandEmbedded(field.Type, astFiles, fset, pkgImportPath, pkgLoader, callback)
	}

	// Skip non-function types (shouldn't happen in a valid interface, but be safe)
	ftype, ok := field.Type.(*ast.FuncType)
	if !ok {
		return nil
	}

	// Process each method name with the same function type
	for _, methodName := range field.Names {
		callback(methodName.Name, ftype)
	}

	return nil
}

// interfaceExpandEmbedded expands an embedded interface by loading its definition and recursively processing methods.
func interfaceExpandEmbedded(
	embeddedType ast.Expr,
	astFiles []*ast.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	callback func(methodName string, ftype *ast.FuncType),
) error {
	var (
		embeddedInterfaceName string
		embeddedPkgPath       string
	)

	// Determine if it's a local interface or external

	switch typ := embeddedType.(type) {
	case *ast.Ident:
		// Local interface (e.g., "Reader")
		embeddedInterfaceName = typ.Name
		embeddedPkgPath = pkgImportPath
	case *ast.SelectorExpr:
		// External interface (e.g., "io.Reader")
		pkgIdent, ok := typ.X.(*ast.Ident)
		if !ok {
			return fmt.Errorf("%w: %T", errUnsupportedEmbeddedType, typ.X)
		}

		// Find the import path for this package
		importPath, err := findImportPath(astFiles, pkgIdent.Name, pkgLoader)
		if err != nil {
			return fmt.Errorf("failed to find import path for embedded interface %s.%s: %w", pkgIdent.Name, typ.Sel.Name, err)
		}

		embeddedInterfaceName = typ.Sel.Name
		embeddedPkgPath = importPath
	default:
		return fmt.Errorf("%w: %T", errUnsupportedEmbeddedType, embeddedType)
	}

	// Load the embedded interface definition
	var (
		embeddedAstFiles []*ast.File
		embeddedFset     *token.FileSet
		err              error
	)

	if embeddedPkgPath == pkgImportPath {
		// Same package - reuse existing AST files
		embeddedAstFiles = astFiles
		embeddedFset = fset
	} else {
		// Different package - need to load it
		// We now HAVE a PackageLoader, so we can support external embedded interfaces!
		embeddedAstFiles, embeddedFset, _, err = pkgLoader.Load(embeddedPkgPath)
		if err != nil {
			return fmt.Errorf("failed to load external embedded interface package %s: %w", embeddedPkgPath, err)
		}
	}

	// Find the embedded interface in the AST
	embeddedInterfaceWithDetails, err := getMatchingInterfaceFromAST(
		embeddedAstFiles, embeddedInterfaceName, embeddedPkgPath,
	)
	if err != nil {
		return fmt.Errorf("failed to find embedded interface %s: %w", embeddedInterfaceName, err)
	}

	// Recursively process the embedded interface's methods
	return forEachInterfaceMethod(
		embeddedInterfaceWithDetails.iface, embeddedAstFiles, embeddedFset, embeddedPkgPath, pkgLoader, callback,
	)
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

// renderFieldList renders a *ast.FieldList as Go code for return types.
func (gen *codeGenerator) renderFieldList(fieldList *ast.FieldList) string {
	if fieldList == nil || len(fieldList.List) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("(")

	for i, field := range fieldList.List {
		if i > 0 {
			buf.WriteString(", ")
		}

		gen.renderField(field, &buf)
	}

	buf.WriteString(")")

	return buf.String()
}

// renderField renders a single field with its name and type.
func (gen *codeGenerator) renderField(field *ast.Field, buf *bytes.Buffer) {
	// Names
	buf.WriteString(joinWith(field.Names, func(n *ast.Ident) string { return n.Name }, ", "))

	// Type
	if len(field.Names) > 0 {
		buf.WriteString(" ")
	}

	buf.WriteString(gen.typeWithQualifier(field.Type))
}
