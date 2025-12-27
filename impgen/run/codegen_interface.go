package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/token"
	go_types "go/types"
	"sort"
	"strings"

	"github.com/dave/dst"
)

// unexported variables.
var (
	errUnsupportedEmbeddedType = errors.New("unsupported embedded type")
)

// Types

// codeGenerator holds state for code generation.
type codeGenerator struct {
	baseGenerator

	templates           *TemplateRegistry
	mockName            string
	callName            string
	expectCallIsName    string
	timedName           string
	interfaceName       string // Full interface name for compile-time verification
	identifiedInterface *dst.InterfaceType
	astFiles            []*dst.File
	pkgImportPath       string
	pkgLoader           PackageLoader
	methodNames         []string
	cachedTemplateData  *templateData // Cache to avoid redundant templateData() calls
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

// codeGenerator Methods

// checkIfFmtNeeded pre-scans all interface methods to determine if fmt import is needed.
// fmt is needed when any method has function-typed parameters (for callback validation).
func (gen *codeGenerator) checkIfFmtNeeded() {
	gen.forEachMethod(func(_ string, ftype *dst.FuncType) {
		funcParams := gen.extractFuncParams(ftype)
		if len(funcParams) > 0 {
			gen.needsFmt = true
			gen.needsReflect = true // Callbacks also need reflect for DeepEqual

			return // Early exit once we know fmt is needed
		}
	})
}

// checkIfImptestNeeded pre-scans all interface methods to determine if imptest import is needed.
// imptest is needed when any method has parameters (for ExpectArgsShould).
func (gen *codeGenerator) checkIfImptestNeeded() {
	gen.forEachMethod(func(_ string, ftype *dst.FuncType) {
		if ftype.Params != nil && len(ftype.Params.List) > 0 {
			gen.needsImptest = true
			return // Early exit once we know imptest is needed
		}
	})
}

// checkIfQualifierNeeded pre-scans to determine if the package qualifier is needed.
func (gen *codeGenerator) checkIfQualifierNeeded() {
	gen.forEachMethod(func(_ string, ftype *dst.FuncType) {
		gen.baseGenerator.checkIfQualifierNeeded(ftype)
	})
}

// checkIfReflectNeeded pre-scans all interface methods to determine if reflect import is needed.
func (gen *codeGenerator) checkIfReflectNeeded() {
	gen.forEachMethod(func(_ string, ftype *dst.FuncType) {
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

// checkIfValidForExternalUsage checks if the interface can be mocked from an external package.
func (gen *codeGenerator) checkIfValidForExternalUsage() error {
	var validationErr error

	gen.forEachMethod(func(methodName string, ftype *dst.FuncType) {
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

// collectAdditionalImports collects all external type imports needed for the interface methods.
// It walks through all method parameters and return types, collecting package references.
func (gen *codeGenerator) collectAdditionalImports() []importInfo {
	if gen.identifiedInterface == nil || len(gen.astFiles) == 0 {
		return nil
	}

	// Get source imports from the first AST file
	sourceImports := gen.getSourceImports()

	// Collect imports from all method signatures
	allImports := gen.collectMethodImports(sourceImports)

	// Convert map to slice and sort for deterministic output
	result := make([]importInfo, 0, len(allImports))
	for _, imp := range allImports {
		result = append(result, imp)
	}

	// Sort by import path for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// collectFieldListImports collects external imports from a field list (params or returns).
func (gen *codeGenerator) collectFieldListImports(
	fields *dst.FieldList,
	sourceImports []*dst.ImportSpec,
	allImports map[string]importInfo,
) {
	if fields == nil {
		return
	}

	for _, field := range fields.List {
		// Collect direct imports from the field type
		imports := collectExternalImports(field.Type, sourceImports)
		for _, imp := range imports {
			allImports[imp.Path] = imp
		}

		// If this is a type alias that resolves to a function type,
		// also collect imports from the underlying function's parameters and returns
		if funcType := resolveToFuncType(field.Type, gen.astFiles, sourceImports, gen.pkgLoader); funcType != nil {
			gen.collectFieldListImports(funcType.Params, sourceImports, allImports)
			gen.collectFieldListImports(funcType.Results, sourceImports, allImports)
		}
	}
}

// collectMethodImports walks through all interface methods and collects external imports.
func (gen *codeGenerator) collectMethodImports(sourceImports []*dst.ImportSpec) map[string]importInfo {
	allImports := make(map[string]importInfo) // Deduplicate by path

	for _, method := range gen.identifiedInterface.Methods.List {
		funcType, ok := method.Type.(*dst.FuncType)
		if !ok {
			continue
		}

		// Collect from parameters
		gen.collectFieldListImports(funcType.Params, sourceImports, allImports)
		// Collect from return types
		gen.collectFieldListImports(funcType.Results, sourceImports, allImports)
	}

	return allImports
}

// extractFuncParams returns information about function-typed parameters in a function.
// Supports inline function types, local type aliases, and external types.
func (gen *codeGenerator) extractFuncParams(ftype *dst.FuncType) []funcParamInfo {
	if !hasParams(ftype) {
		return nil
	}

	params := extractParams(nil, ftype)
	sourceImports := gen.getSourceImports()

	var funcParams []funcParamInfo

	for _, param := range params {
		// Resolve the parameter type to a function type if possible
		funcType := resolveToFuncType(param.Field.Type, gen.astFiles, sourceImports, gen.pkgLoader)
		if funcType != nil {
			funcParams = append(funcParams, funcParamInfo{
				Name:      param.Name,
				Index:     param.Index,
				FuncType:  funcType,
				FieldInfo: param,
			})
		}
	}

	return funcParams
}

// forEachMethod iterates over interface methods and calls the callback for each.
// This is safe to call without error checking because we already validated the interface
// structure during method name collection in generateImplementationCode. If an error occurs
// here, it indicates a programming error and will cause a panic in the underlying function.
func (gen *codeGenerator) forEachMethod(callback func(methodName string, ftype *dst.FuncType)) {
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

// generate orchestrates the code generation process after initialization.
func (gen *codeGenerator) generate() (string, error) {
	// Pre-scan to determine if reflect import is needed
	gen.checkIfReflectNeeded()
	// Pre-scan to determine if imptest import is needed
	gen.checkIfImptestNeeded()
	// Pre-scan to determine if fmt import is needed
	gen.checkIfFmtNeeded()
	// Pre-scan to see if qualifier is needed
	gen.checkIfQualifierNeeded()

	// If we have an interface name, we need the qualifier for interface verification.
	// Exception: when pkgPath is empty but qualifier is set (test package case), the import
	// already exists from baseGenerator, so we don't need to add it.
	if gen.interfaceName != "" && gen.qualifier != "" && gen.pkgPath != "" {
		gen.needsQualifier = true
	}

	err := gen.checkIfValidForExternalUsage()
	if err != nil {
		return "", err
	}

	gen.generateHeader()
	gen.generateMockStruct()
	gen.generateInterfaceVerification()
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

// generateBuilderShortcuts generates InjectResult/InjectPanic/Resolve shortcut methods on the builder.
func (gen *codeGenerator) generateBuilderShortcuts(
	methodName string, ftype *dst.FuncType, builderName, callName string,
) {
	// Validator that only checks method name
	validatorCode := fmt.Sprintf(`validator := func(callToCheck *%s%s) bool {
		return callToCheck.Name() == %q
	}

	call := bldr.imp.GetCall(bldr.timeout, validator)
	methodCall := call.As%s()
`, gen.callName, gen.formatTypeParamsUse(), methodName, methodName)

	if hasResults(ftype) {
		// Generate InjectResult shortcut
		if len(ftype.Results.List) == 1 {
			resultType := gen.typeWithQualifier(ftype.Results.List[0].Type)
			gen.pf("// InjectResult waits for a %s call and immediately injects the return value.\n", methodName)
			gen.pf("// This is a shortcut that combines waiting for the call with injecting the result.\n")
			gen.pf("// Returns the call object for further operations. Fails if no call arrives within the timeout.\n")
			gen.pf("func (bldr *%s%s) InjectResult(result %s) *%s%s {\n",
				builderName, gen.formatTypeParamsUse(), resultType, callName, gen.formatTypeParamsUse())
			gen.pf("\t%s", validatorCode)
			gen.pf("\tmethodCall.InjectResult(result)\n")
			gen.pf("\treturn methodCall\n")
			gen.pf("}\n\n")
		} else {
			// Multiple return values - InjectResults
			gen.pf("// InjectResults waits for a %s call and immediately injects the return values.\n", methodName)
			gen.pf("// This is a shortcut that combines waiting for the call with injecting multiple results.\n")
			gen.pf("// Returns the call object for further operations. Fails if no call arrives within the timeout.\n")
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
		gen.pf("// Resolve waits for a %s call and immediately completes it without error.\n", methodName)
		gen.pf("// This is a shortcut that combines waiting for the call with resolving it.\n")
		gen.pf("// Returns the call object for further operations. Fails if no call arrives within the timeout.\n")
		gen.pf("func (bldr *%s%s) Resolve() *%s%s {\n",
			builderName, gen.formatTypeParamsUse(), callName, gen.formatTypeParamsUse())
		gen.pf("\t%s", validatorCode)
		gen.pf("\tmethodCall.Resolve()\n")
		gen.pf("\treturn methodCall\n")
		gen.pf("}\n\n")
	}

	// Generate InjectPanic shortcut (always available)
	gen.pf("// InjectPanic waits for a %s call and causes it to panic with the given value.\n", methodName)
	gen.pf("// This is a shortcut that combines waiting for the call with injecting a panic.\n")
	gen.pf("// Use this to test panic handling in code under test. Returns the call object for further operations.\n")
	gen.pf("func (bldr *%s%s) InjectPanic(msg any) *%s%s {\n",
		builderName, gen.formatTypeParamsUse(), callName, gen.formatTypeParamsUse())
	gen.pf("\t%s", validatorCode)
	gen.pf("\tmethodCall.InjectPanic(msg)\n")
	gen.pf("\treturn methodCall\n")
	gen.pf("}\n\n")
}

// generateCallStruct generates the union call struct that can hold any method call.
func (gen *codeGenerator) generateCallStruct() {
	gen.templates.WriteCallStruct(&gen.buf, gen.callStructData())
}

// generateCallStructParamFields generates the parameter fields for a call struct.
func (gen *codeGenerator) generateCallStructParamFields(ftype *dst.FuncType) {
	visitParams(ftype, gen.typeWithQualifier, func(
		param *dst.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		if len(param.Names) > 0 {
			gen.writeNamedParamFields(param, paramType, unnamedIndex, totalParams)
			return paramNameIndex + len(param.Names), unnamedIndex
		}

		gen.writeUnnamedParamField(param, paramType, unnamedIndex, totalParams)

		return paramNameIndex + 1, unnamedIndex + 1
	})
}

// generateCallbackInvocationMethod generates type-safe InvokeFn and ExpectReturned methods for a callback parameter.
// For a callback like fn func(path string, d fs.DirEntry, err error) error, this generates:
//   - InvokeFn(path string, d fs.DirEntry, err error) - type-safe invocation
//   - ExpectReturned(result0 error) - type-safe result verification
//
//nolint:cyclop,funlen,varnamelen,wsl,gocognit // Code generation logic is inherently complex
func (gen *codeGenerator) generateCallbackInvocationMethod(
	_, callName string, fp funcParamInfo, requestTypeName, responseTypeName string,
) {
	capitalizedName := strings.ToUpper(fp.Name[:1]) + fp.Name[1:]
	resultTypeName := fmt.Sprintf("%s%sCallbackResult", callName, capitalizedName)

	// Generate the callback result type (holds response from the callback)
	gen.pf("// %s holds the result of invoking the %s callback.\n", resultTypeName, fp.Name)
	gen.pf("type %s struct {\n", resultTypeName)

	// Add private fields for each return value
	if hasResults(fp.FuncType) {
		results := extractResults(nil, fp.FuncType)
		for i := range results {
			gen.pf("\tresult%d ", i)
			gen.pf("%s\n", gen.typeWithQualifier(results[i].Field.Type))
		}
	}

	// Add panic field to track callback panics
	gen.pf("\tpanicked any\n")

	gen.pf("}\n\n")

	// Generate the ExpectReturned method with type-safe parameters
	gen.pf("// ExpectReturned verifies that the callback returned the expected values.\n")
	gen.pf("func (r *%s) ExpectReturned(", resultTypeName)

	// Generate type-safe parameters for ExpectReturned
	if hasResults(fp.FuncType) {
		results := extractResults(nil, fp.FuncType)
		for i, result := range results {
			if i > 0 {
				gen.pf(", ")
			}

			gen.pf("expected%d %s", i, gen.typeWithQualifier(result.Field.Type))
		}
	}

	gen.pf(") {\n")

	// Generate type-safe comparisons
	if hasResults(fp.FuncType) {
		results := extractResults(nil, fp.FuncType)
		for i, result := range results {
			resultType := gen.typeWithQualifier(result.Field.Type)
			// Use DeepEqual for non-comparable types (like functions, slices, maps)
			gen.pf("\tif !%s.DeepEqual(r.result%d, expected%d) {\n", pkgReflect, i, i)
			gen.pf("\t\tpanic(%s.Sprintf(\"callback result[%d] = %%v, expected %%v\", r.result%d, expected%d))\n",
				pkgFmt, i, i, i)
			gen.pf("\t}\n")

			_ = resultType // Keep for future optimizations (use == for comparable types)
		}
	}

	gen.pf("}\n\n")

	// Generate ExpectReturnedShould with matcher support
	gen.pf("// ExpectReturnedShould verifies callback return values match matchers.\n")
	gen.pf("// Use imptest.Any() to match any value, or imptest.Satisfies(fn) for custom matching.\n")
	gen.pf("func (r *%s) ExpectReturnedShould(", resultTypeName)

	if hasResults(fp.FuncType) {
		results := extractResults(nil, fp.FuncType)
		for i := range results {
			if i > 0 {
				gen.pf(", ")
			}

			gen.pf("expected%d any", i)
		}
	}

	gen.pf(") {\n")

	if hasResults(fp.FuncType) {
		results := extractResults(nil, fp.FuncType)
		for i := range results {
			gen.pf("\tok, msg := %s.MatchValue(r.result%d, expected%d)\n", pkgImptest, i, i)
			gen.pf("\tif !ok { panic(%s.Sprintf(\"callback result[%d]: %%s\", msg)) }\n", pkgFmt, i)
		}
	}

	gen.pf("}\n\n")

	// Generate ExpectPanicWith to verify callback panics
	gen.pf("// ExpectPanicWith verifies that the callback panicked with a value matching the expectation.\n")
	gen.pf("// Use imptest.Any() to match any panic value, or imptest.Satisfies(fn) for custom matching.\n")
	gen.pf("// Panics if the callback returned normally or panicked with a different value.\n")
	gen.pf("func (r *%s) ExpectPanicWith(expected any) {\n", resultTypeName)
	gen.pf("\tif r.panicked != nil {\n")
	gen.pf("\t\tok, msg := %s.MatchValue(r.panicked, expected)\n", pkgImptest)
	gen.pf("\t\tif !ok {\n")
	gen.pf("\t\t\tpanic(%s.Sprintf(\"callback panic value: %%s\", msg))\n", pkgFmt)
	gen.pf("\t\t}\n")
	gen.pf("\t\treturn\n")
	gen.pf("\t}\n")
	gen.pf("\tpanic(\"expected callback to panic, but it returned\")\n")
	gen.pf("}\n\n")

	// Generate the type-safe InvokeFn method
	methodNameFormatted := "Invoke" + capitalizedName
	gen.pf("// %s invokes the %s callback with the provided arguments.\n", methodNameFormatted, fp.Name)
	gen.pf("// Returns a result object that can verify the callback's return values.\n")
	gen.pf("func (c *%s) %s(", callName, methodNameFormatted)

	// Generate type-safe parameters for InvokeFn
	params := extractParams(nil, fp.FuncType)
	for i, param := range params {
		if i > 0 {
			gen.pf(", ")
		}

		paramName := param.Name
		if paramName == "" {
			paramName = fmt.Sprintf("arg%d", i)
		}

		gen.pf("%s %s", paramName, gen.typeWithQualifier(param.Field.Type))
	}

	gen.pf(") *%s {\n", resultTypeName)

	// Create result channel with typed response
	gen.pf("\tresultChan := make(chan %s)\n", responseTypeName)

	// Send typed request
	gen.pf("\tc.callback%sChan <- %s{\n", capitalizedName, requestTypeName)

	for i, param := range params {
		fieldName := fmt.Sprintf("Arg%d", i)
		if param.Name != "" {
			fieldName = strings.ToUpper(param.Name[:1]) + param.Name[1:]
		}

		paramName := param.Name
		if paramName == "" {
			paramName = fmt.Sprintf("arg%d", i)
		}

		gen.pf("\t\t%s: %s,\n", fieldName, paramName)
	}

	gen.pf("\t\tResultChan: resultChan,\n")
	gen.pf("\t}\n")

	// Receive typed response
	gen.pf("\tresp := <-resultChan\n")

	// Return typed result
	gen.pf("\treturn &%s{", resultTypeName)

	if hasResults(fp.FuncType) {
		results := extractResults(nil, fp.FuncType)
		for i := range results {
			gen.pf("result%d: resp.Result%d, ", i, i)
		}
	}

	gen.pf("panicked: resp.Panicked")

	gen.pf("}\n")

	gen.pf("}\n\n")
}

// generateCallbackHelper generates a helper function to invoke a callback with dynamic arguments.
// generateCallbackRequestResponseStructs generates type-safe request and response structs
// for a callback parameter. For a callback like fn func(path string, d fs.DirEntry, err error) error:
//   - TreeWalkerImpWalkCallFnRequest with fields for each parameter plus ResultChan
//   - TreeWalkerImpWalkCallFnResponse with fields for each return value
//
//nolint:varnamelen // Short parameter names are conventional in code generation
func (gen *codeGenerator) generateCallbackRequestResponseStructs(
	_, callName string, fp funcParamInfo,
) (requestTypeName, responseTypeName string) {
	capitalizedName := strings.ToUpper(fp.Name[:1]) + fp.Name[1:]
	requestTypeName = fmt.Sprintf("%s%sRequest", callName, capitalizedName)
	responseTypeName = fmt.Sprintf("%s%sResponse", callName, capitalizedName)

	// Generate request struct
	gen.pf("// %s carries callback invocation data for the %s parameter.\n", requestTypeName, fp.Name)
	gen.pf("type %s struct {\n", requestTypeName)

	// Add fields for each callback parameter
	params := extractParams(nil, fp.FuncType)
	for i, param := range params {
		fieldName := fmt.Sprintf("Arg%d", i)
		if param.Name != "" {
			// Capitalize the parameter name for the field
			fieldName = strings.ToUpper(param.Name[:1]) + param.Name[1:]
		}

		paramType := gen.typeWithQualifier(param.Field.Type)
		gen.pf("\t%s %s\n", fieldName, paramType)
	}

	// Add result channel
	gen.pf("\tResultChan chan %s\n", responseTypeName)
	gen.pf("}\n\n")

	// Generate response struct
	gen.pf("// %s carries callback return values for the %s parameter.\n", responseTypeName, fp.Name)
	gen.pf("type %s struct {\n", responseTypeName)

	// Add fields for each return value
	if hasResults(fp.FuncType) {
		results := extractResults(nil, fp.FuncType)
		for i, result := range results {
			resultType := gen.typeWithQualifier(result.Field.Type)
			gen.pf("\tResult%d %s\n", i, resultType)
		}
	}

	// Add panic field to capture callback panics
	gen.pf("\tPanicked any\n")

	gen.pf("}\n\n")

	return requestTypeName, responseTypeName
}

// generateConstructor generates the New{ImpName} constructor function.
func (gen *codeGenerator) generateConstructor() {
	gen.templates.WriteConstructor(&gen.buf, gen.templateData())
}

// generateExpectArgsAre generates the type-safe ExpectArgsAre method on the builder.
func (gen *codeGenerator) generateExpectArgsAre(methodName string, ftype *dst.FuncType, builderName, callName string) {
	paramNames := interfaceExtractParamNames(gen.fset, ftype)

	// Method signature
	gen.pf("// ExpectArgsAre waits for a %s call with exactly the specified argument values.\n", methodName)
	gen.pf("// Returns the call object for response injection. Fails the test if the call\n")
	gen.pf("// doesn't arrive within the timeout or if arguments don't match exactly.\n")
	gen.pf("// Uses == for comparable types and reflect.DeepEqual for others.\n")
	gen.pf("func (bldr *%s%s) ExpectArgsAre(", builderName, gen.formatTypeParamsUse())
	gen.writeMethodParams(ftype, paramNames)
	gen.pf(") *%s%s {\n", callName, gen.formatTypeParamsUse())

	// Validator function
	gen.pf("\tvalidator := func(callToCheck *%s%s) bool {\n", gen.callName, gen.formatTypeParamsUse())
	gen.pf("\t\tif callToCheck.Name() != %q {\n", methodName)
	gen.pf("\t\t\treturn false\n")
	gen.pf("\t	}\n")

	if hasParams(ftype) {
		gen.pf("\t\tmethodCall := callToCheck.As%s()\n", methodName)
		gen.writeExpectArgsAreChecks(ftype, paramNames)
	}

	gen.pf("\t\treturn true\n")
	gen.pf("\t}\n\n")

	// GetCall and return
	gen.pf("\tcall := bldr.imp.GetCall(bldr.timeout, validator)\n")
	gen.pf("\treturn call.As%s()\n", methodName)
	gen.pf("}\n\n")
}

// generateExpectArgsShould generates the matcher-based ExpectArgsShould method on the builder.
func (gen *codeGenerator) generateExpectArgsShould(
	methodName string, ftype *dst.FuncType, builderName, callName string,
) {
	paramNames := interfaceExtractParamNames(gen.fset, ftype)

	// Method signature - all params are 'any'
	gen.pf("// ExpectArgsShould waits for a %s call with arguments matching the given matchers.\n", methodName)
	gen.pf("// Use imptest.Any() to match any value, or imptest.Satisfies(fn) for custom matching.\n")
	gen.pf("// Returns the call object for response injection. Fails the test if the call\n")
	gen.pf("// doesn't arrive within the timeout or if any matcher fails.\n")
	gen.pf("func (bldr *%s%s) ExpectArgsShould(", builderName, gen.formatTypeParamsUse())
	gen.writeMethodParamsAsAny(ftype, paramNames)
	gen.pf(") *%s%s {\n", callName, gen.formatTypeParamsUse())

	// Validator function
	gen.pf("\tvalidator := func(callToCheck *%s%s) bool {\n", gen.callName, gen.formatTypeParamsUse())
	gen.pf("\t\tif callToCheck.Name() != %q {\n", methodName)
	gen.pf("\t\t\treturn false\n")
	gen.pf("\t	}\n")

	if hasParams(ftype) {
		gen.pf("\t\tmethodCall := callToCheck.As%s()\n", methodName)
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

// generateExpectCallIsStruct generates the struct for expecting specific method calls.
func (gen *codeGenerator) generateExpectCallIsStruct() {
	gen.templates.WriteExpectCallIsStruct(&gen.buf, gen.templateData())
}

// generateGetCurrentCallMethod generates the GetCurrentCall method that returns the current or next call.
func (gen *codeGenerator) generateGetCurrentCallMethod() {
	gen.templates.WriteGetCurrentCallMethod(&gen.buf, gen.templateData())
}

// generateHeader writes the package declaration and imports for the generated file.
func (gen *codeGenerator) generateHeader() {
	gen.templates.WriteHeader(&gen.buf, gen.templateData())
}

// generateInjectPanicMethod generates the InjectPanic method for simulating panics.
func (gen *codeGenerator) generateInjectPanicMethod(methodCallName string) {
	gen.templates.WriteInjectPanic(&gen.buf, gen.methodTemplateData(methodCallName))
}

// generateInjectResultMethod generates the InjectResult method for methods with a single return value.
func (gen *codeGenerator) generateInjectResultMethod(methodCallName string, ftype *dst.FuncType) {
	resultType := gen.typeWithQualifier(ftype.Results.List[0].Type)
	gen.pf("// InjectResult sets the return value for this method call and unblocks the caller.\n")
	gen.pf("// The mocked method will return the provided result value.\n")
	gen.pf(`func (c *%s%s) InjectResult(result %s) {
	c.done = true
	c.responseChan <- %sResponse%s{Type: "return"`,
		methodCallName, gen.formatTypeParamsUse(), resultType, methodCallName, gen.formatTypeParamsUse())

	if hasFieldNames(ftype.Results.List[0]) {
		gen.pf(", %s: result", ftype.Results.List[0].Names[0].Name)
	} else {
		gen.pf(", Result0: result")
	}

	gen.pf(`}
}
`)
}

// generateInjectResultsMethod generates the InjectResults method for methods with multiple return values.
func (gen *codeGenerator) generateInjectResultsMethod(methodCallName string, ftype *dst.FuncType) {
	gen.pf("// InjectResults sets the return values for this method call and unblocks the caller.\n")
	gen.pf("// The mocked method will return the provided result values in order.\n")
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

// generateInterfaceVerification generates a compile-time check that the mock implements the interface.
func (gen *codeGenerator) generateInterfaceVerification() {
	gen.templates.WriteInterfaceVerification(&gen.buf, gen.templateData())
}

// generateMainStruct generates the main implementation struct that handles test call tracking.
func (gen *codeGenerator) generateMainStruct() {
	gen.templates.WriteMainStruct(&gen.buf, gen.templateData())
}

// generateMethodBuilder generates the builder struct and all its methods for a single interface method.
func (gen *codeGenerator) generateMethodBuilder(methodName string, ftype *dst.FuncType) {
	builderName := gen.methodBuilderName(methodName)
	callName := gen.methodCallName(methodName)

	// Generate builder struct
	gen.pf("// %s%s provides a fluent API for setting expectations on %s calls.\n",
		builderName, gen.formatTypeParamsDecl(), methodName)
	gen.pf("// Use ExpectArgsAre for exact matching or ExpectArgsShould for matcher-based matching.\n")
	gen.pf("type %s%s struct {\n", builderName, gen.formatTypeParamsDecl())
	gen.pf("\timp     *%s%s\n", gen.impName, gen.formatTypeParamsUse())
	gen.pf("\ttimeout %s.Duration\n", "_time")
	gen.pf("}\n\n")

	// Generate ExpectCallIs.MethodName() -> returns builder
	gen.pf("// %s returns a builder for setting expectations on %s method calls.\n", methodName, methodName)
	gen.pf("func (e *%s%s) %s() *%s%s {\n",
		gen.expectCallIsName, gen.formatTypeParamsUse(), methodName, builderName, gen.formatTypeParamsUse())
	gen.pf("\treturn &%s%s{imp: e.imp, timeout: e.timeout}\n", builderName, gen.formatTypeParamsUse())
	gen.pf("}\n\n")

	// Only generate ExpectArgs methods if the method has parameters
	if hasParams(ftype) {
		// Generate ExpectArgsAre (type-safe)
		gen.generateExpectArgsAre(methodName, ftype, builderName, callName)

		// Generate ExpectArgsShould (matcher-based)
		gen.generateExpectArgsShould(methodName, ftype, builderName, callName)
	}

	// Generate shortcut InjectResult/InjectPanic/Resolve
	gen.generateBuilderShortcuts(methodName, ftype, builderName, callName)
}

// generateMethodBuilders generates builder structs and methods for each interface method.
func (gen *codeGenerator) generateMethodBuilders() {
	gen.forEachMethod(func(methodName string, ftype *dst.FuncType) {
		gen.generateMethodBuilder(methodName, ftype)
	})
}

// generateMethodCallStruct generates the call struct for a specific method, which tracks the method call parameters.
func (gen *codeGenerator) generateMethodCallStruct(methodName string, ftype *dst.FuncType) {
	callName := gen.methodCallName(methodName)

	// Generate callback request/response structs first (they need to exist before the call struct references them)
	funcParams := gen.extractFuncParams(ftype)

	callbackTypeNames := make(map[string]struct{ requestType, responseType string })
	for _, fp := range funcParams {
		requestType, responseType := gen.generateCallbackRequestResponseStructs(methodName, callName, fp)
		callbackTypeNames[fp.Name] = struct{ requestType, responseType string }{requestType, responseType}
	}

	gen.pf("// %s%s represents a captured call to the %s method.\n", callName, gen.formatTypeParamsDecl(), methodName)
	gen.pf("// Use InjectResult to set the return value, or InjectPanic to cause the method to panic.\n")
	gen.pf(`type %s%s struct {
	responseChan chan %sResponse%s
	done bool
`, callName, gen.formatTypeParamsDecl(), callName, gen.formatTypeParamsUse())

	if hasParams(ftype) {
		gen.generateCallStructParamFields(ftype)
	}

	// Add callback coordination channels with typed request channels
	if len(funcParams) > 0 {
		gen.pf("\t// Callback coordination channels\n")

		for _, fp := range funcParams {
			capitalizedName := strings.ToUpper(fp.Name[:1]) + fp.Name[1:]
			typeNames := callbackTypeNames[fp.Name]
			gen.pf("\tcallback%sChan chan %s\n", capitalizedName, typeNames.requestType)
		}
	}

	gen.pf("}\n\n")
}

// generateMethodResponseMethods generates the InjectResult, InjectResults, InjectPanic, and Resolve methods
// for a call struct.
func (gen *codeGenerator) generateMethodResponseMethods(methodName string, ftype *dst.FuncType) {
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

	// Generate callback invocation methods if there are func-typed parameters
	funcParams := gen.extractFuncParams(ftype)
	for _, fp := range funcParams {
		// Compute the request/response type names (same logic as in generateCallbackRequestResponseStructs)
		capitalizedName := strings.ToUpper(fp.Name[:1]) + fp.Name[1:]
		requestTypeName := fmt.Sprintf("%s%sRequest", callName, capitalizedName)
		responseTypeName := fmt.Sprintf("%s%sResponse", callName, capitalizedName)
		gen.generateCallbackInvocationMethod(methodName, callName, fp, requestTypeName, responseTypeName)
	}

	gen.pf("\n")
}

// generateMethodResponseStruct generates the response struct for a method, which holds return values or panic data.
func (gen *codeGenerator) generateMethodResponseStruct(methodName string, ftype *dst.FuncType) {
	callName := gen.methodCallName(methodName)
	gen.pf("// %sResponse%s holds the response configuration for the %s method.\n",
		callName, gen.formatTypeParamsDecl(), methodName)
	gen.pf("// Set Type to \"return\" for normal returns, \"panic\" to cause a panic, or \"resolve\" for void methods.\n")
	gen.pf(`type %sResponse%s struct {
	Type string // "return", "panic", or "resolve"
`, callName, gen.formatTypeParamsDecl())

	if hasResults(ftype) {
		gen.generateResponseStructResultFields(ftype)
	}

	gen.pf("	PanicValue any\n}\n\n")
}

// generateMethodStructs generates the call and response structs for each interface method.
func (gen *codeGenerator) generateMethodStructs() {
	gen.forEachMethod(func(methodName string, ftype *dst.FuncType) {
		gen.generateMethodCallStruct(methodName, ftype)
		gen.generateMethodResponseStruct(methodName, ftype)
		gen.generateMethodResponseMethods(methodName, ftype)
	})
}

// generateMockMethod generates a single mock method that creates a call, sends it to the imp, and handles the response.
func (gen *codeGenerator) generateMockMethod(methodName string, ftype *dst.FuncType) {
	callName := gen.methodCallName(methodName)
	paramNames := interfaceExtractParamNames(gen.fset, ftype)
	funcParams := gen.extractFuncParams(ftype)

	gen.writeMockMethodSignature(methodName, ftype, paramNames)
	gen.writeMockMethodCallCreation(callName, ftype, paramNames)
	gen.writeMockMethodEventDispatch(methodName)
	gen.writeMockMethodResponseHandling(callName, ftype, paramNames)

	// Only write return statement if there are no callbacks (otherwise it's in the select)
	if len(funcParams) == 0 {
		gen.writeReturnStatement(ftype)
	}

	gen.pf("}\n\n")
}

// generateMockMethods generates the mock methods that implement the interface on the mock struct.
func (gen *codeGenerator) generateMockMethods() {
	gen.forEachMethod(func(methodName string, ftype *dst.FuncType) {
		gen.generateMockMethod(methodName, ftype)
	})
}

// generateMockStruct generates the mock struct that wraps the implementation.
func (gen *codeGenerator) generateMockStruct() {
	gen.templates.WriteMockStruct(&gen.buf, gen.templateData())
}

// generateResolveMethod generates the Resolve method for methods with no return values.
func (gen *codeGenerator) generateResolveMethod(methodCallName string) {
	gen.templates.WriteResolve(&gen.buf, gen.methodTemplateData(methodCallName))
}

// generateResponseStructResultFields generates the result fields for a response struct.
func (gen *codeGenerator) generateResponseStructResultFields(ftype *dst.FuncType) {
	for _, r := range extractResults(gen.fset, ftype) {
		gen.pf("\t%s %s\n", r.Name, gen.typeWithQualifier(r.Field.Type))
	}
}

// generateTimedStruct generates the struct and method for timed call expectations.
func (gen *codeGenerator) generateTimedStruct() {
	gen.templates.WriteTimedStruct(&gen.buf, gen.templateData())
}

// getSourceImports returns combined import specs from all AST files.
// This collects imports from all files including source and generated files
// to ensure type resolution and import collection work correctly.
func (gen *codeGenerator) getSourceImports() []*dst.ImportSpec {
	var allImports []*dst.ImportSpec

	seen := make(map[string]bool)

	for _, file := range gen.astFiles {
		for _, imp := range file.Imports {
			path := imp.Path.Value
			if !seen[path] {
				seen[path] = true

				allImports = append(allImports, imp)
			}
		}
	}

	return allImports
}

// methodBuilderName returns the builder struct name for a method (e.g. "MyImpAddBuilder").
func (gen *codeGenerator) methodBuilderName(methodName string) string {
	return gen.impName + methodName + "Builder"
}

// methodCallName returns the call struct name for a method (e.g. "MyImpDoSomethingCall").
func (gen *codeGenerator) methodCallName(methodName string) string {
	return gen.impName + methodName + "Call"
}

// methodTemplateData returns template data for a specific method.
func (gen *codeGenerator) methodTemplateData(methodCallName string) methodTemplateData {
	return methodTemplateData{
		templateData:   gen.templateData(),
		MethodCallName: methodCallName,
	}
}

// renderField renders a single field with its name and type.
func (gen *codeGenerator) renderField(field *dst.Field, buf *bytes.Buffer) {
	// Names
	buf.WriteString(joinWith(field.Names, func(n *dst.Ident) string { return n.Name }, ", "))

	// Type
	if hasFieldNames(field) {
		buf.WriteString(" ")
	}

	buf.WriteString(gen.typeWithQualifier(field.Type))
}

// renderFieldList renders a *dst.FieldList as Go code for return types.
func (gen *codeGenerator) renderFieldList(fieldList *dst.FieldList) string {
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

// templateData returns common template data for this generator.
// The result is cached after the first call to avoid redundant struct construction.
func (gen *codeGenerator) templateData() templateData {
	if gen.cachedTemplateData != nil {
		return *gen.cachedTemplateData
	}

	data := templateData{
		baseTemplateData: baseTemplateData{
			PkgName:           gen.pkgName,
			ImpName:           gen.impName,
			PkgPath:           gen.pkgPath,
			Qualifier:         gen.qualifier,
			NeedsQualifier:    gen.needsQualifier,
			TypeParamsDecl:    gen.formatTypeParamsDecl(),
			TypeParamsUse:     gen.formatTypeParamsUse(),
			PkgTesting:        pkgTesting,
			PkgFmt:            pkgFmt,
			PkgImptest:        pkgImptest,
			PkgTime:           pkgTime,
			PkgReflect:        pkgReflect,
			NeedsFmt:          gen.needsFmt,
			NeedsReflect:      gen.needsReflect,
			NeedsImptest:      gen.needsImptest,
			AdditionalImports: gen.collectAdditionalImports(),
		},
		MockName:         gen.mockName,
		CallName:         gen.callName,
		ExpectCallIsName: gen.expectCallIsName,
		TimedName:        gen.timedName,
		InterfaceName:    gen.interfaceName,
		MethodNames:      gen.methodNames,
	}

	gen.cachedTemplateData = &data

	return data
}

// writeCallStructField writes a single field assignment for a call struct initialization.
func (gen *codeGenerator) writeCallStructField(
	param *dst.Field, paramType string, paramNames []string, paramNameIndex, unnamedIndex, totalParams int,
) (int, int) {
	return forEachParamField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams,
		func(fieldName, paramName string) {
			gen.pf("\t\t%s: %s,\n", fieldName, paramName)
		})
}

// writeCallStructFields writes the field assignments for initializing a call struct.
func (gen *codeGenerator) writeCallStructFields(ftype *dst.FuncType, paramNames []string) {
	visitParams(ftype, gen.typeWithQualifier, func(
		param *dst.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		return gen.writeCallStructField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams)
	})
}

// writeComparisonCheck writes either an equality or matcher-based comparison check.
// When useMatcher is true, uses imptest.MatchValue for flexible matching.
// When useMatcher is false, uses == or reflect.DeepEqual for equality checks.
func (gen *codeGenerator) writeComparisonCheck(fieldName, expectedName string, isComparable, useMatcher bool) {
	switch {
	case useMatcher:
		gen.pf("\tok, _ = %s.MatchValue(methodCall.%s, %s)\n", "_imptest", fieldName, expectedName)
		gen.pf("\t\tif !ok {\n")
	case isComparable:
		gen.pf("\t\tif methodCall.%s != %s {\n", fieldName, expectedName)
	default:
		gen.needsReflect = true
		gen.pf("\t\tif !%s.DeepEqual(methodCall.%s, %s) {\n", "_reflect", fieldName, expectedName)
	}

	gen.pf("\t\t\treturn false\n")
	gen.pf("\t\t}\n")
}

// writeExpectArgsAreChecks writes parameter equality checks for ExpectArgsAre.
func (gen *codeGenerator) writeExpectArgsAreChecks(ftype *dst.FuncType, paramNames []string) {
	gen.writeParamChecks(ftype, paramNames, false)
}

// writeExpectArgsShouldChecks writes matcher-based checks for ExpectArgsShould.
func (gen *codeGenerator) writeExpectArgsShouldChecks(ftype *dst.FuncType, paramNames []string) {
	gen.writeParamChecks(ftype, paramNames, true)
}

// writeInjectResultsArgs writes the argument list for InjectResults call.
func (gen *codeGenerator) writeInjectResultsArgs(ftype *dst.FuncType) {
	results := extractResults(gen.fset, ftype)
	for resultIndex := range results {
		if resultIndex > 0 {
			gen.pf(", ")
		}

		gen.pf("r%d", resultIndex)
	}
}

// writeInjectResultsParams writes the parameter list for InjectResults method and returns the result names.
func (gen *codeGenerator) writeInjectResultsParams(ftype *dst.FuncType) []string {
	results := extractResults(gen.fset, ftype)

	// Write parameters using shared formatter with proper type qualification
	gen.pf("%s", formatResultParameters(results, "r", 0, func(r fieldInfo) string {
		return gen.typeWithQualifier(r.Field.Type)
	}))

	// Build names array for return
	return generateResultVarNames(len(results), "r")
}

// writeInjectResultsResponseFields writes the response struct field assignments for InjectResults.
func (gen *codeGenerator) writeInjectResultsResponseFields(ftype *dst.FuncType, returnParamNames []string) {
	for resultIdx, result := range extractResults(gen.fset, ftype) {
		gen.pf(", %s: %s", result.Name, returnParamNames[resultIdx])
	}
}

// writeMethodParams writes the method parameters in the form "name type, name2 type2".
func (gen *codeGenerator) writeMethodParams(ftype *dst.FuncType, paramNames []string) {
	gen.writeMethodParamsWithFormatter(ftype, paramNames, func(t string) string { return t })
}

// writeMethodParamsAsAny writes method parameters with all types as 'any'.
func (gen *codeGenerator) writeMethodParamsAsAny(ftype *dst.FuncType, paramNames []string) {
	gen.writeMethodParamsWithFormatter(ftype, paramNames, func(_ string) string { return anyTypeString })
}

// writeMethodParamsWithFormatter writes method parameters using a custom type formatter.
// The typeFormatter function receives the qualified type string and returns the formatted type to use.
// This allows writing params as "name actualType" or "name any" with the same iteration logic.
func (gen *codeGenerator) writeMethodParamsWithFormatter(
	ftype *dst.FuncType,
	paramNames []string,
	typeFormatter func(qualifiedType string) string,
) {
	if !hasParams(ftype) {
		return
	}

	first := true
	paramNameIndex := 0

	visitParams(ftype, gen.typeWithQualifier, func(
		param *dst.Field, paramType string, _, _, _ int,
	) (int, int) {
		if hasFieldNames(param) {
			for _, name := range param.Names {
				if !first {
					gen.pf(", ")
				}

				first = false

				gen.pf("%s %s", name.Name, typeFormatter(paramType))
			}

			paramNameIndex += len(param.Names)
		} else {
			if !first {
				gen.pf(", ")
			}

			first = false

			gen.pf("%s %s", paramNames[paramNameIndex], typeFormatter(paramType))
			paramNameIndex++
		}

		return 0, 0 // Indices not used when using visitParams
	})
}

// writeMethodSignature writes the method name and parameters (e.g., "MethodName(a int, b string)").
func (gen *codeGenerator) writeMethodSignature(methodName string, ftype *dst.FuncType, paramNames []string) {
	gen.pf("%s(", methodName)
	gen.writeMethodParams(ftype, paramNames)
	gen.pf(")")
}

// writeMockMethodCallCreation writes the response channel and call struct creation.
func (gen *codeGenerator) writeMockMethodCallCreation(callName string, ftype *dst.FuncType, paramNames []string) {
	gen.pf("\tresponseChan := make(chan %sResponse%s, 1)\n", callName, gen.formatTypeParamsUse())

	// Create callback channels with typed requests
	funcParams := gen.extractFuncParams(ftype)
	for _, fp := range funcParams {
		capitalizedName := strings.ToUpper(fp.Name[:1]) + fp.Name[1:]
		requestTypeName := fmt.Sprintf("%s%sRequest", callName, capitalizedName)
		gen.pf("\tcallback%sChan := make(chan %s)\n", capitalizedName, requestTypeName)
	}

	gen.pf("\n\tcall := &%s%s{\n", callName, gen.formatTypeParamsUse())
	gen.pf("\t\tresponseChan: responseChan,\n")

	// Add callback channels to call struct
	for _, fp := range funcParams {
		capitalizedName := strings.ToUpper(fp.Name[:1]) + fp.Name[1:]
		gen.pf("\t\tcallback%sChan: callback%sChan,\n", capitalizedName, capitalizedName)
	}

	gen.writeCallStructFields(ftype, paramNames)
	gen.pf("\t}\n\n")
}

// writeMockMethodEventDispatch writes the call event creation and dispatch to the imp.
func (gen *codeGenerator) writeMockMethodEventDispatch(methodName string) {
	gen.pf("\tcallEvent := &%s%s{\n", gen.callName, gen.formatTypeParamsUse())
	gen.pf("\t\t%s: call,\n", lowerFirst(methodName))
	gen.pf("\t}\n\n")
	gen.pf("\tm.imp.CallChan <- callEvent\n\n")
}

// writeMockMethodResponseHandling writes the response reception and panic handling.
//
//nolint:cyclop,funlen,varnamelen,wsl // Callback handling requires select statement with multiple cases
func (gen *codeGenerator) writeMockMethodResponseHandling(callName string, ftype *dst.FuncType, _ []string) {
	funcParams := gen.extractFuncParams(ftype)

	if len(funcParams) == 0 {
		// No callbacks - simple response handling
		gen.pf("\tresp := <-responseChan\n\n")
		gen.pf("\tif resp.Type == \"panic\" {\n")
		gen.pf("\t\tpanic(resp.PanicValue)\n")
		gen.pf("\t}\n\n")

		return
	}

	// With callbacks - loop until final response
	gen.pf("\t// Handle callback invocations and final response\n")
	gen.pf("\tvar resp %sResponse%s\n", callName, gen.formatTypeParamsUse())
	gen.pf("\tfor {\n")
	gen.pf("\t\tselect {\n")

	// Generate case for each callback channel with type-safe invocation
	for _, fp := range funcParams {
		capitalizedName := strings.ToUpper(fp.Name[:1]) + fp.Name[1:]
		responseTypeName := fmt.Sprintf("%s%sResponse", callName, capitalizedName)

		gen.pf("\t\tcase cbReq := <-callback%sChan:\n", capitalizedName)
		gen.pf("\t\t\t// Invoke callback and capture panics\n")
		gen.pf("\t\t\tfunc() {\n")
		gen.pf("\t\t\t\tdefer func() {\n")
		gen.pf("\t\t\t\t\tif r := recover(); r != nil {\n")
		gen.pf("\t\t\t\t\t\tcbReq.ResultChan <- %s{Panicked: r}\n", responseTypeName)
		gen.pf("\t\t\t\t\t}\n")
		gen.pf("\t\t\t\t}()\n\n")

		// Invoke callback directly with typed request fields
		gen.pf("\t\t\t\t")

		if hasResults(fp.FuncType) {
			results := extractResults(nil, fp.FuncType)
			for i := range results {
				if i > 0 {
					gen.pf(", ")
				}

				gen.pf("result%d", i)
			}

			gen.pf(" := ")
		}

		gen.pf("%s(", fp.Name)

		params := extractParams(nil, fp.FuncType)
		for i, param := range params {
			if i > 0 {
				gen.pf(", ")
			}
			// Use the field name from the request struct
			fieldName := fmt.Sprintf("Arg%d", i)
			if param.Name != "" {
				fieldName = strings.ToUpper(param.Name[:1]) + param.Name[1:]
			}

			gen.pf("cbReq.%s", fieldName)
		}

		gen.pf(")\n")

		// Send typed response back
		gen.pf("\t\t\t\tcbReq.ResultChan <- %s{", responseTypeName)

		if hasResults(fp.FuncType) {
			results := extractResults(nil, fp.FuncType)
			for i := range results {
				if i > 0 {
					gen.pf(", ")
				}

				gen.pf("Result%d: result%d", i, i)
			}
		}

		gen.pf("}\n")
		gen.pf("\t\t\t}()\n")
	}

	// Generate case for final response
	gen.pf("\t\tcase resp = <-responseChan:\n")
	gen.pf("\t\t\t// Final response received\n")
	gen.pf("\t\t\tif resp.Type == \"panic\" {\n")
	gen.pf("\t\t\t\tpanic(resp.PanicValue)\n")
	gen.pf("\t\t\t}\n")
	gen.pf("\t\t\treturn")

	// Return early if there are results
	if hasResults(ftype) {
		gen.pf(" ")
		gen.writeReturnValues(ftype)
	}

	gen.pf("\n\t\t}\n")
	gen.pf("\t}\n")
}

// writeMockMethodSignature writes the mock method signature and opening brace.
func (gen *codeGenerator) writeMockMethodSignature(methodName string, ftype *dst.FuncType, paramNames []string) {
	gen.pf("// %s implements the interface method and records the call for testing.\n", methodName)
	gen.pf("// The method blocks until a response is injected via the test controller.\n")
	gen.pf("func (m *%s%s) ", gen.mockName, gen.formatTypeParamsUse())
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.pf("%s", gen.renderFieldList(ftype.Results))
	gen.pf(" {\n")
}

// writeNamedParamFields writes fields for named parameters.
func (gen *codeGenerator) writeNamedParamFields(param *dst.Field, paramType string, unnamedIndex, totalParams int) {
	structType := normalizeVariadicType(paramType)

	for i := range param.Names {
		fieldName := interfaceGetParamFieldName(param, i, unnamedIndex, structType, totalParams)
		gen.pf("\t%s %s\n", fieldName, structType)
	}
}

// writeParamChecks writes parameter comparison checks.
// When useMatcher is true, uses imptest.MatchValue for flexible matching.
// When useMatcher is false, uses == or reflect.DeepEqual for equality checks.
func (gen *codeGenerator) writeParamChecks(ftype *dst.FuncType, paramNames []string, useMatcher bool) {
	visitParams(ftype, gen.typeWithQualifier, func(
		param *dst.Field, paramType string, paramNameIndex, unnamedIndex, totalParams int,
	) (int, int) {
		isComparable := isComparableExpr(param.Type, gen.typesInfo)

		return forEachParamField(param, paramType, paramNames, paramNameIndex, unnamedIndex, totalParams,
			func(fieldName, paramName string) {
				gen.writeComparisonCheck(fieldName, paramName, isComparable, useMatcher)
			})
	})
}

// writeReturnStatement writes the return statement for a mock method.
func (gen *codeGenerator) writeReturnStatement(ftype *dst.FuncType) {
	if !hasResults(ftype) {
		gen.pf("\treturn\n")
		return
	}

	gen.pf("\treturn")
	gen.writeReturnValues(ftype)
	gen.pf("\n")
}

// writeReturnValues writes all return values from the response struct.
func (gen *codeGenerator) writeReturnValues(ftype *dst.FuncType) {
	for i, r := range extractResults(gen.fset, ftype) {
		if i > 0 {
			gen.pf(",")
		}

		gen.pf(" resp.%s", r.Name)
	}
}

// writeUnnamedParamField writes a field for an unnamed parameter.
func (gen *codeGenerator) writeUnnamedParamField(param *dst.Field, paramType string, unnamedIndex, totalParams int) {
	structType := normalizeVariadicType(paramType)

	fieldName := interfaceGetParamFieldName(param, 0, unnamedIndex, structType, totalParams)
	gen.pf("\t%s %s\n", fieldName, structType)
}

// forEachInterfaceMethod iterates over interface methods and calls the callback for each,
// expanding embedded interfaces.
func forEachInterfaceMethod(
	iface *dst.InterfaceType,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	callback func(methodName string, ftype *dst.FuncType),
) error {
	for _, field := range iface.Methods.List {
		err := interfaceProcessFieldMethods(field, astFiles, fset, pkgImportPath, pkgLoader, callback)
		if err != nil {
			return err
		}
	}

	return nil
}

// forEachParamField iterates over parameter fields, handling both named and unnamed parameters.
// It calls the action callback for each field with the computed field name and parameter name.
func forEachParamField(
	param *dst.Field,
	paramType string,
	paramNames []string,
	paramNameIndex, unnamedIndex, totalParams int,
	action func(fieldName, paramName string),
) (int, int) {
	if hasFieldNames(param) {
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

// Entry Point - Public

// generateImplementationCode generates the complete mock implementation code for an interface.
func generateImplementationCode(
	astFiles []*dst.File,
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

// Private Functions

// interfaceCollectMethodNames collects all method names from an interface, including embedded ones.
func interfaceCollectMethodNames(
	iface *dst.InterfaceType, astFiles []*dst.File, fset *token.FileSet, pkgImportPath string, pkgLoader PackageLoader,
) ([]string, error) {
	var methodNames []string

	err := forEachInterfaceMethod(
		iface, astFiles, fset, pkgImportPath, pkgLoader,
		func(methodName string, _ *dst.FuncType) {
			methodNames = append(methodNames, methodName)
		},
	)
	if err != nil {
		return nil, err
	}

	return methodNames, nil
}

// interfaceExpandEmbedded expands an embedded interface by loading its definition and recursively processing methods.
func interfaceExpandEmbedded(
	embeddedType dst.Expr,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	callback func(methodName string, ftype *dst.FuncType),
) error {
	var (
		embeddedInterfaceName string
		embeddedPkgPath       string
	)

	// Determine if it's a local interface or external

	switch typ := embeddedType.(type) {
	case *dst.Ident:
		// Local interface (e.g., "Reader")
		embeddedInterfaceName = typ.Name
		embeddedPkgPath = pkgImportPath
	case *dst.SelectorExpr:
		// External interface (e.g., "io.Reader")
		pkgIdent, ok := typ.X.(*dst.Ident)
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
		embeddedAstFiles []*dst.File
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

// interfaceExtractParamNames extracts or generates parameter names from a function type.
func interfaceExtractParamNames(fset *token.FileSet, ftype *dst.FuncType) []string {
	params := extractParams(fset, ftype)
	names := make([]string, len(params))

	for i, p := range params {
		names[i] = p.Name
	}

	return names
}

// interfaceGenerateParamName generates a field name for an unnamed parameter
// Uses common conventions: single string -> "S", single int -> "Input", multiple -> "A", "B", "C", etc.
//

func interfaceGenerateParamName(index int, paramType string, totalParams int) string {
	// Remove common prefixes/suffixes for comparison
	normalized := strings.TrimSpace(paramType)

	// Single parameter cases
	if totalParams == 1 {
		if normalized == "string" { //nolint:goconst,nolintlint // Type name comparison
			return "S"
		}

		if normalized == "int" { //nolint:goconst,nolintlint // Type name comparison
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

// interfaceGetParamFieldName returns the struct field name for a parameter.
// For named params, returns the name. For unnamed params, generates a name based on type/index.
func interfaceGetParamFieldName(
	param *dst.Field, nameIdx int, unnamedIdx int, paramType string, totalParams int,
) string {
	if hasFieldNames(param) {
		return param.Names[nameIdx].Name
	}

	return interfaceGenerateParamName(unnamedIdx, paramType, totalParams)
}

// interfaceProcessFieldMethods handles a single field in an interface's method list.
func interfaceProcessFieldMethods(
	field *dst.Field,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	callback func(methodName string, ftype *dst.FuncType),
) error {
	// Handle embedded interfaces (they have no names)
	if !hasFieldNames(field) {
		return interfaceExpandEmbedded(field.Type, astFiles, fset, pkgImportPath, pkgLoader, callback)
	}

	// Skip non-function types (shouldn't happen in a valid interface, but be safe)
	ftype, ok := field.Type.(*dst.FuncType)
	if !ok {
		return nil
	}

	// Process each method name with the same function type
	for _, methodName := range field.Names {
		callback(methodName.Name, ftype)
	}

	return nil
}

// newCodeGenerator initializes a codeGenerator with common properties and performs initial setup.
//
//nolint:funlen // Constructor with necessary initialization logic
func newCodeGenerator(
	astFiles []*dst.File,
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
	// Get package info for external interfaces OR when in a _test package (which needs to import the non-test package)
	if pkgImportPath != "." || strings.HasSuffix(info.pkgName, "_test") {
		pkgPath, qualifier, err = resolvePackageInfo(info, pkgLoader)
		if err != nil {
			return nil, fmt.Errorf("failed to get interface package info: %w", err)
		}
	}

	// Construct the full interface name for compile-time verification
	// When there's a package name conflict (e.g., local "time" package shadowing stdlib "time"),
	// we need to use the aliased stdlib package name in the type assertion.
	var interfaceName string

	if qualifier != "" {
		// Check if this is a stdlib package that needs aliasing due to a name conflict
		// A stdlib package has a simple import path (no slashes), and if the qualifier matches
		// the package path, it means there's a local package with the same name.
		qualifierToUse := qualifier
		if pkgPath != "" && !strings.Contains(pkgPath, "/") && pkgPath == qualifier {
			// This is a stdlib package with a name conflict - use the alias
			qualifierToUse = "_" + qualifier
		}

		interfaceName = qualifierToUse + "." + info.localInterfaceName
	} else {
		interfaceName = info.localInterfaceName
	}

	gen := &codeGenerator{
		baseGenerator: newBaseGenerator(

			fset, info.pkgName, impName, pkgPath, qualifier, ifaceWithDetails.typeParams, typesInfo,
		),

		mockName: impName + "Mock", callName: impName + "Call",

		expectCallIsName: impName + "ExpectCallIs", timedName: impName + "Timed",

		interfaceName: interfaceName,

		identifiedInterface: ifaceWithDetails.iface, astFiles: astFiles,

		pkgImportPath: pkgImportPath, pkgLoader: pkgLoader,
	}

	methodNames, err := interfaceCollectMethodNames(ifaceWithDetails.iface, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return nil, err
	}

	gen.methodNames = methodNames

	// Initialize template registry
	gen.templates, err = NewTemplateRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize template registry: %w", err)
	}

	return gen, nil
}

// resolvePackageInfo resolves the package path and qualifier for an interface.
// Handles special case of test packages needing to import the non-test version.
func resolvePackageInfo(info generatorInfo, pkgLoader PackageLoader) (pkgPath, qualifier string, err error) {
	pkgPath, qualifier, err = GetPackageInfo(
		info.interfaceName,
		pkgLoader,
		info.pkgName,
	)
	if err != nil {
		return "", "", err
	}

	// Special case: when in a test package (e.g., "imptest_test") and the interface
	// has no package qualifier (GetPackageInfo returned empty), the interface is from
	// the non-test version of this package. We need to import it with its full path.
	if qualifier == "" && strings.HasSuffix(info.pkgName, "_test") {
		basePkgPath, baseQualifier := resolveTestPackageImport(pkgLoader, info.pkgName)
		if basePkgPath != "" {
			return basePkgPath, baseQualifier, nil
		}
	}

	return pkgPath, qualifier, nil
}

// resolveTestPackageImport resolves the import path for the non-test version of a test package.
func resolveTestPackageImport(pkgLoader PackageLoader, pkgName string) (pkgPath, qualifier string) {
	// Strip _test suffix to get the base package name
	basePkgName := strings.TrimSuffix(pkgName, "_test")

	// Load the non-test package to get its import path
	basePkgFiles, baseFset, _, err := pkgLoader.Load(".")
	if err != nil || len(basePkgFiles) == 0 {
		return "", ""
	}

	// Get the import path from the package's own declaration
	path, err := getImportPathFromFiles(basePkgFiles, baseFset, "")
	if err != nil || path == "" {
		return "", ""
	}

	return path, basePkgName
}
