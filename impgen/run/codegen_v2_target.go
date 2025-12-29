package run

import (
	"fmt"
	"go/format"
	"go/token"
	go_types "go/types"
	"strings"

	"github.com/dave/dst"
)

// v2TargetGenerator generates v2-style target wrappers.
type v2TargetGenerator struct {
	baseGenerator

	wrapName       string   // Wrapper constructor name (e.g., "WrapAdd")
	wrapperType    string   // Wrapper struct type (e.g., "WrapAddWrapper")
	returnsType    string   // Returns struct type (e.g., "WrapAddReturns")
	funcDecl       *dst.FuncDecl
	astFiles       []*dst.File
	pkgImportPath  string
	pkgLoader      PackageLoader
	paramNames     []string
	resultTypes    []string
	hasResults     bool
}

// newV2TargetGenerator creates a new v2 target wrapper generator.
func newV2TargetGenerator(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
	funcDecl *dst.FuncDecl,
) (*v2TargetGenerator, error) {
	var (
		pkgPath, qualifier string
		err                error
	)

	// Get package info for external functions OR when in a _test package
	if pkgImportPath != "." || strings.HasSuffix(info.pkgName, "_test") {
		pkgPath, qualifier, err = resolvePackageInfo(info, pkgLoader)
		if err != nil {
			return nil, fmt.Errorf("failed to get function package info: %w", err)
		}
	}

	// Wrapper type naming: WrapAdd -> WrapAddWrapper
	wrapperType := info.impName + "Wrapper"
	returnsType := info.impName + "Returns"

	gen := &v2TargetGenerator{
		baseGenerator: newBaseGenerator(
			fset, info.pkgName, info.impName, pkgPath, qualifier, funcDecl.Type.TypeParams, typesInfo,
		),
		wrapName:      info.impName,
		wrapperType:   wrapperType,
		returnsType:   returnsType,
		funcDecl:      funcDecl,
		astFiles:      astFiles,
		pkgImportPath: pkgImportPath,
		pkgLoader:     pkgLoader,
	}

	// Extract parameter names and result types
	gen.paramNames = extractParamNames(funcDecl.Type)
	gen.hasResults = funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0
	if gen.hasResults {
		gen.resultTypes = gen.extractResultTypes(funcDecl.Type.Results)
	}

	return gen, nil
}

// generate produces the v2 target wrapper code using templates.
func (gen *v2TargetGenerator) generate() (string, error) {
	// Pre-scan to determine what imports are needed
	gen.checkIfQualifierNeeded(gen.funcDecl.Type)

	// Initialize template registry
	templates, err := NewTemplateRegistry()
	if err != nil {
		return "", fmt.Errorf("failed to initialize template registry: %w", err)
	}

	// Generate using templates
	gen.generateWithTemplates(templates)

	formatted, err := format.Source(gen.bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// generateWithTemplates generates code using templates instead of direct code generation.
func (gen *v2TargetGenerator) generateWithTemplates(templates *TemplateRegistry) {
	// Build base template data
	base := baseTemplateData{
		PkgName:        gen.pkgName,
		ImpName:        gen.impName,
		PkgPath:        gen.pkgPath,
		Qualifier:      gen.qualifier,
		NeedsQualifier: gen.needsQualifier,
		TypeParamsDecl: gen.formatTypeParamsDecl(),
		TypeParamsUse:  gen.formatTypeParamsUse(),
		PkgTesting:     pkgTesting,
		PkgFmt:         pkgFmt,
		PkgImptest:     pkgImptest,
		PkgTime:        pkgTime,
		PkgReflect:     pkgReflect,
		NeedsFmt:       gen.needsFmt,
		NeedsReflect:   gen.needsReflect,
		NeedsImptest:   gen.needsImptest,
	}

	// Build function signature string
	funcSig := gen.buildFunctionSignature()

	// Build params string for Start method
	var paramsStr strings.Builder
	gen.writeFunctionParamsToBuilder(&paramsStr, gen.funcDecl.Type.Params)

	// Build comma-separated param names
	var paramNamesStr strings.Builder
	for i, name := range gen.paramNames {
		if i > 0 {
			paramNamesStr.WriteString(", ")
		}
		paramNamesStr.WriteString(name)
	}

	// Build result vars and return assignments
	var resultVarsStr, returnAssignmentsStr strings.Builder
	if gen.hasResults {
		for i := range gen.resultTypes {
			if i > 0 {
				resultVarsStr.WriteString(", ")
				returnAssignmentsStr.WriteString(", ")
			}
			fmt.Fprintf(&resultVarsStr, "r%d", i+1)
			fmt.Fprintf(&returnAssignmentsStr, "R%d: r%d", i+1, i+1)
		}
	}

	// Determine wait method name
	waitMethodName := "WaitForCompletion"
	if gen.hasResults {
		waitMethodName = "WaitForResponse"
	}

	// Build expected params and result checks for ExpectReturnsEqual
	var expectedParamsStr strings.Builder
	var resultChecks []resultCheck
	if gen.hasResults {
		for i, resultType := range gen.resultTypes {
			if i > 0 {
				expectedParamsStr.WriteString(", ")
			}
			fmt.Fprintf(&expectedParamsStr, "expected%d %s", i+1, resultType)
			resultChecks = append(resultChecks, resultCheck{
				Field:    fmt.Sprintf("R%d", i+1),
				Expected: fmt.Sprintf("expected%d", i+1),
				Index:    i + 1,
			})
		}
	}

	// Build result fields for Returns struct
	var resultFields []resultField
	if gen.hasResults {
		for i, resultType := range gen.resultTypes {
			resultFields = append(resultFields, resultField{
				Name: fmt.Sprintf("R%d", i+1),
				Type: resultType,
			})
		}
	}

	// Build v2 target template data
	data := v2TargetTemplateData{
		baseTemplateData:  base,
		WrapName:          gen.wrapName,
		WrapperType:       gen.wrapperType,
		ReturnsType:       gen.returnsType,
		FuncSig:           funcSig,
		Params:            paramsStr.String(),
		ParamNames:        paramNamesStr.String(),
		HasResults:        gen.hasResults,
		ResultVars:        resultVarsStr.String(),
		ReturnAssignments: returnAssignmentsStr.String(),
		WaitMethodName:    waitMethodName,
		ExpectedParams:    expectedParamsStr.String(),
		ResultChecks:      resultChecks,
		ResultFields:      resultFields,
	}

	// Generate each section using templates
	templates.WriteV2TargetHeader(&gen.buf, data)
	templates.WriteV2TargetConstructor(&gen.buf, data)
	templates.WriteV2TargetWrapperStruct(&gen.buf, data)
	templates.WriteV2TargetStartMethod(&gen.buf, data)
	templates.WriteV2TargetWaitMethod(&gen.buf, data)

	// Generate expect methods based on whether function has results
	if gen.hasResults {
		templates.WriteV2TargetExpectReturns(&gen.buf, data)
	} else {
		templates.WriteV2TargetExpectCompletes(&gen.buf, data)
	}
	templates.WriteV2TargetExpectPanic(&gen.buf, data)

	// Generate returns struct
	templates.WriteV2TargetReturnsStruct(&gen.buf, data)
}

// writeFunctionParamsToBuilder writes function parameters to a string builder.
func (gen *v2TargetGenerator) writeFunctionParamsToBuilder(builder *strings.Builder, params *dst.FieldList) {
	if params == nil {
		return
	}

	first := true
	for _, field := range params.List {
		fieldType := gen.typeWithQualifier(field.Type)

		if len(field.Names) > 0 {
			for _, name := range field.Names {
				if !first {
					builder.WriteString(", ")
				}
				first = false
				builder.WriteString(name.Name)
				builder.WriteString(" ")
				builder.WriteString(fieldType)
			}
		} else {
			if !first {
				builder.WriteString(", ")
			}
			first = false
			builder.WriteString(fmt.Sprintf("arg%d ", len(gen.paramNames)+1))
			builder.WriteString(fieldType)
		}
	}
}

// generateHeader writes the package declaration and imports.
func (gen *v2TargetGenerator) generateHeader() {
	gen.pf("// Code generated by impgen. DO NOT EDIT.\n\n")
	gen.pf("package %s\n\n", gen.pkgName)

	gen.pf("import (\n")
	gen.pf("\t\"github.com/toejough/imptest/imptest\"\n")

	if gen.needsQualifier && gen.pkgPath != "" {
		gen.pf("\t%s \"%s\"\n", gen.qualifier, gen.pkgPath)
	}

	gen.pf(")\n\n")
}

// generateConstructor writes the wrapper constructor function.
func (gen *v2TargetGenerator) generateConstructor() {
	// Build function type signature
	funcSig := gen.buildFunctionSignature()

	gen.pf("// %s wraps the function for testing.\n", gen.wrapName)
	gen.pf("func %s(testReporter imptest.TestReporter, fn %s) *%s {\n", gen.wrapName, funcSig, gen.wrapperType)
	gen.pf("\timp, ok := testReporter.(*imptest.Imp)\n")
	gen.pf("\tif !ok {\n")
	gen.pf("\t\timp = imptest.NewImp(testReporter)\n")
	gen.pf("\t}\n\n")
	gen.pf("\treturn &%s{\n", gen.wrapperType)
	gen.pf("\t\timp:        imp,\n")
	gen.pf("\t\tfn:         fn,\n")
	gen.pf("\t\treturnChan: make(chan %s, 1),\n", gen.returnsType)
	gen.pf("\t\tpanicChan:  make(chan any, 1),\n")
	gen.pf("\t}\n")
	gen.pf("}\n\n")
}

// generateWrapperStruct writes the wrapper struct.
func (gen *v2TargetGenerator) generateWrapperStruct() {
	funcSig := gen.buildFunctionSignature()

	gen.pf("// %s provides a fluent API for calling and verifying the function.\n", gen.wrapperType)
	gen.pf("type %s struct {\n", gen.wrapperType)
	gen.pf("\timp        *imptest.Imp\n")
	gen.pf("\tfn         %s\n", funcSig)
	gen.pf("\treturnChan chan %s\n", gen.returnsType)
	gen.pf("\tpanicChan  chan any\n")
	gen.pf("\treturned   *%s\n", gen.returnsType)
	gen.pf("\tpanicked   any\n")
	gen.pf("}\n\n")
}

// generateStartMethod writes the Start method.
func (gen *v2TargetGenerator) generateStartMethod() {
	gen.pf("// Start begins execution of the function in a goroutine.\n")
	gen.pf("func (w *%s) Start(", gen.wrapperType)

	// Write parameters
	gen.writeFunctionParams(gen.funcDecl.Type.Params)

	gen.pf(") *%s {\n", gen.wrapperType)
	gen.pf("\tgo func() {\n")
	gen.pf("\t\tdefer func() {\n")
	gen.pf("\t\t\tif r := recover(); r != nil {\n")
	gen.pf("\t\t\t\tw.panicChan <- r\n")
	gen.pf("\t\t\t}\n")
	gen.pf("\t\t}()\n\n")

	// Call function and capture results
	if gen.hasResults {
		gen.pf("\t\t")
		for i := range gen.resultTypes {
			if i > 0 {
				gen.pf(", ")
			}
			gen.pf("r%d", i+1)
		}
		gen.pf(" := w.fn(")
	} else {
		gen.pf("\t\tw.fn(")
	}

	// Pass parameters
	for i, paramName := range gen.paramNames {
		if i > 0 {
			gen.pf(", ")
		}
		gen.pf("%s", paramName)
	}
	gen.pf(")\n")

	// Send results to channel
	gen.pf("\t\tw.returnChan <- %s{", gen.returnsType)
	if gen.hasResults {
		for i := range gen.resultTypes {
			if i > 0 {
				gen.pf(", ")
			}
			gen.pf("R%d: r%d", i+1, i+1)
		}
	}
	gen.pf("}\n")

	gen.pf("\t}()\n\n")
	gen.pf("\treturn w\n")
	gen.pf("}\n\n")
}

// generateWaitMethod writes the wait method.
func (gen *v2TargetGenerator) generateWaitMethod() {
	methodName := "WaitForCompletion"
	if gen.hasResults {
		methodName = "WaitForResponse"
	}

	gen.pf("// %s blocks until the function completes (return or panic).\n", methodName)
	gen.pf("func (w *%s) %s() {\n", gen.wrapperType, methodName)
	gen.pf("\tif w.returned != nil || w.panicked != nil {\n")
	gen.pf("\t\treturn\n")
	gen.pf("\t}\n\n")
	gen.pf("\tselect {\n")
	gen.pf("\tcase ret := <-w.returnChan:\n")
	gen.pf("\t\tw.returned = &ret\n")
	gen.pf("\tcase p := <-w.panicChan:\n")
	gen.pf("\t\tw.panicked = p\n")
	gen.pf("\t}\n")
	gen.pf("}\n\n")
}

// generateExpectMethods writes the expectation methods.
func (gen *v2TargetGenerator) generateExpectMethods() {
	waitMethod := "WaitForCompletion"
	if gen.hasResults {
		waitMethod = "WaitForResponse"
		gen.generateExpectReturnsEqual()
	} else {
		gen.generateExpectCompletes()
	}

	gen.generateExpectPanicEquals(waitMethod)
}

// generateExpectReturnsEqual writes the ExpectReturnsEqual method.
func (gen *v2TargetGenerator) generateExpectReturnsEqual() {
	gen.pf("// ExpectReturnsEqual verifies the function returned exact values.\n")
	gen.pf("func (w *%s) ExpectReturnsEqual(", gen.wrapperType)

	// Parameters for expected values
	for i, resultType := range gen.resultTypes {
		if i > 0 {
			gen.pf(", ")
		}
		gen.pf("expected%d %s", i+1, resultType)
	}

	gen.pf(") {\n")
	gen.pf("\tw.imp.Helper()\n")
	gen.pf("\tw.WaitForResponse()\n\n")
	gen.pf("\tif w.panicked != nil {\n")
	gen.pf("\t\tw.imp.Fatalf(\"expected function to return, but it panicked with: %%v\", w.panicked)\n")
	gen.pf("\t\treturn\n")
	gen.pf("\t}\n\n")

	// Compare each return value
	for i := range gen.resultTypes {
		gen.pf("\tif w.returned.R%d != expected%d {\n", i+1, i+1)
		gen.pf("\t\tw.imp.Fatalf(\"return value %d: expected %%v, got %%v\", expected%d, w.returned.R%d)\n", i+1, i+1, i+1)
		gen.pf("\t}\n")
	}

	gen.pf("}\n\n")
}

// generateExpectCompletes writes the ExpectCompletes method for void functions.
func (gen *v2TargetGenerator) generateExpectCompletes() {
	gen.pf("// ExpectCompletes verifies the function completed without panicking.\n")
	gen.pf("func (w *%s) ExpectCompletes() {\n", gen.wrapperType)
	gen.pf("\tw.imp.Helper()\n")
	gen.pf("\tw.WaitForCompletion()\n\n")
	gen.pf("\tif w.panicked != nil {\n")
	gen.pf("\t\tw.imp.Fatalf(\"expected function to complete, but it panicked with: %%v\", w.panicked)\n")
	gen.pf("\t}\n")
	gen.pf("}\n\n")
}

// generateExpectPanicEquals writes the ExpectPanicEquals method.
func (gen *v2TargetGenerator) generateExpectPanicEquals(waitMethod string) {
	gen.pf("// ExpectPanicEquals verifies the function panicked with the exact value.\n")
	gen.pf("func (w *%s) ExpectPanicEquals(expected any) {\n", gen.wrapperType)
	gen.pf("\tw.imp.Helper()\n")
	gen.pf("\tw.%s()\n\n", waitMethod)
	gen.pf("\tif w.panicked == nil {\n")
	gen.pf("\t\tw.imp.Fatalf(\"expected panic with %%v, but function completed normally\", expected)\n")
	gen.pf("\t\treturn\n")
	gen.pf("\t}\n\n")
	gen.pf("\tif w.panicked != expected {\n")
	gen.pf("\t\tw.imp.Fatalf(\"expected panic with %%v, got %%v\", expected, w.panicked)\n")
	gen.pf("\t}\n")
	gen.pf("}\n\n")
}

// generateReturnsStruct writes the returns struct.
func (gen *v2TargetGenerator) generateReturnsStruct() {
	gen.pf("// %s provides type-safe access to return values", gen.returnsType)
	if !gen.hasResults {
		gen.pf(" (empty for void functions)")
	}
	gen.pf(".\n")
	gen.pf("type %s struct {\n", gen.returnsType)

	if gen.hasResults {
		for i, resultType := range gen.resultTypes {
			gen.pf("\tR%d %s\n", i+1, resultType)
		}
	}

	gen.pf("}\n")
}

// buildFunctionSignature builds the function signature string.
func (gen *v2TargetGenerator) buildFunctionSignature() string {
	var sig strings.Builder

	sig.WriteString("func(")

	// Parameters
	if gen.funcDecl.Type.Params != nil {
		first := true
		for _, field := range gen.funcDecl.Type.Params.List {
			fieldType := gen.typeWithQualifier(field.Type)

			count := len(field.Names)
			if count == 0 {
				count = 1
			}

			for i := 0; i < count; i++ {
				if !first {
					sig.WriteString(", ")
				}
				first = false
				sig.WriteString(fieldType)
			}
		}
	}

	sig.WriteString(")")

	// Results
	if gen.funcDecl.Type.Results != nil && len(gen.funcDecl.Type.Results.List) > 0 {
		sig.WriteString(" ")

		hasMultipleResults := len(gen.funcDecl.Type.Results.List) > 1 ||
			(len(gen.funcDecl.Type.Results.List) == 1 && len(gen.funcDecl.Type.Results.List[0].Names) > 1)

		if hasMultipleResults {
			sig.WriteString("(")
		}

		first := true
		for _, field := range gen.funcDecl.Type.Results.List {
			fieldType := gen.typeWithQualifier(field.Type)

			count := len(field.Names)
			if count == 0 {
				count = 1
			}

			for i := 0; i < count; i++ {
				if !first {
					sig.WriteString(", ")
				}
				first = false
				sig.WriteString(fieldType)
			}
		}

		if hasMultipleResults {
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// writeFunctionParams writes function parameters.
func (gen *v2TargetGenerator) writeFunctionParams(params *dst.FieldList) {
	if params == nil {
		return
	}

	first := true
	for _, field := range params.List {
		fieldType := gen.typeWithQualifier(field.Type)

		if len(field.Names) > 0 {
			for _, name := range field.Names {
				if !first {
					gen.pf(", ")
				}
				first = false
				gen.pf("%s %s", name.Name, fieldType)
			}
		} else {
			if !first {
				gen.pf(", ")
			}
			first = false
			gen.pf("arg%d %s", len(gen.paramNames)+1, fieldType)
		}
	}
}

// extractResultTypes extracts result types from a field list.
func (gen *v2TargetGenerator) extractResultTypes(results *dst.FieldList) []string {
	var types []string

	for _, field := range results.List {
		fieldType := gen.typeWithQualifier(field.Type)

		count := len(field.Names)
		if count == 0 {
			count = 1
		}

		for i := 0; i < count; i++ {
			types = append(types, fieldType)
		}
	}

	return types
}

// extractParamNames extracts parameter names from a function type.
func extractParamNames(funcType *dst.FuncType) []string {
	var names []string

	if funcType.Params == nil {
		return names
	}

	argCounter := 1
	for _, field := range funcType.Params.List {
		if len(field.Names) > 0 {
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
		} else {
			names = append(names, fmt.Sprintf("arg%d", argCounter))
			argCounter++
		}
	}

	return names
}

// generateV2TargetCode generates v2-style target wrapper code for a function.
func generateV2TargetCode(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
	funcDecl *dst.FuncDecl,
) (string, error) {
	gen, err := newV2TargetGenerator(astFiles, info, fset, typesInfo, pkgImportPath, pkgLoader, funcDecl)
	if err != nil {
		return "", err
	}

	return gen.generate()
}
