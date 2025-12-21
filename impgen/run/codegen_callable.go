package run

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	go_types "go/types"
	"strings"
	"text/template"
)

// Entry Point

// generateCallableWrapperCode generates a type-safe wrapper for a callable function.
func generateCallableWrapperCode(
	astFiles []*ast.File,
	info generatorInfo,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
) (string, error) {
	funcDecl, err := findFunctionInAST(astFiles, fset, info.localInterfaceName, pkgImportPath)
	if err != nil {
		return "", err
	}

	var pkgPath, qualifier string
	if pkgImportPath != "." {
		pkgPath, qualifier, err = callableGetPackageInfo(info.interfaceName, pkgLoader, info.pkgName)
		if err != nil {
			return "", fmt.Errorf("failed to get callable package info: %w", err)
		}
	}

	gen := &callableGenerator{
		baseGenerator: newBaseGenerator(
			fset,
			info.pkgName,
			info.impName,
			pkgPath,
			qualifier,
			funcDecl.Type.TypeParams,
			typesInfo,
		),
		funcDecl: funcDecl,
	}

	gen.checkIfQualifierNeeded()

	err = gen.checkIfValidForExternalUsage()
	if err != nil {
		return "", err
	}

	gen.generateHeader()
	gen.generateReturnStruct()
	gen.generateMainStruct()
	gen.generateConstructor()
	gen.generateStartMethod()
	gen.generateExpectReturnedValuesMethod()
	gen.generateExpectPanicWithMethod()
	gen.generateResponseStruct()
	gen.generateResponseMethods()
	gen.generateGetResponseMethod()

	formatted, err := format.Source(gen.bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// checkIfQualifierNeeded pre-scans function signature to determine if the package qualifier is needed.
func (g *callableGenerator) checkIfQualifierNeeded() {
	g.baseGenerator.checkIfQualifierNeeded(g.funcDecl.Type)
}

// checkIfValidForExternalUsage checks if the callable can be wrapped from an external package.
func (g *callableGenerator) checkIfValidForExternalUsage() error {
	return g.baseGenerator.checkIfValidForExternalUsage(g.funcDecl.Type)
}

// Types

// callableGenerator holds state for generating callable wrapper code.
type callableGenerator struct {
	baseGenerator

	funcDecl *ast.FuncDecl
}

// callableExtendedTemplateData extends callableTemplateData with dynamic signature info.
type callableExtendedTemplateData struct {
	callableTemplateData //nolint:unused // embedded fields accessed via templates

	CallableSignature  string
	CallableReturns    string
	ParamNames         string   // comma-separated parameter names for calling
	ReturnVars         string   // comma-separated return variable names (ret0, ret1, ...)
	ReturnVarsList     []string // slice of return variable names
	ReturnFields       []returnFieldData
	ResultParams       string // parameters for ExpectReturnedValues (v1 Type1, v2 Type2, ...)
	ResultComparisons  string // comparisons for ExpectReturnedValues
	ResultComparisons2 string // comparisons using "ret" variable
}

// returnFieldData holds data for a single return field.
type returnFieldData struct {
	Index int
	Name  string
	Type  string // Type name for struct field definitions
}

// returnTypeName returns the appropriate type name for return channels and fields.
// Returns "{impName}Return{TypeParams}" if the function has returns, otherwise "struct{}".
func (g *callableGenerator) returnTypeName() string {
	if hasResults(g.funcDecl.Type) {
		return g.impName + "Return" + g.formatTypeParamsUse()
	}

	return "struct{}"
}

// numReturns returns the total number of return values.
// This should only be called when hasResults(g.funcDecl.Type) is true.
func (g *callableGenerator) numReturns() int {
	return countFields(g.funcDecl.Type.Results)
}

// templateData returns the base template data for this generator.
func (g *callableGenerator) templateData() callableTemplateData {
	numReturns := 0
	if hasResults(g.funcDecl.Type) {
		numReturns = g.numReturns()
	}

	return callableTemplateData{
		PkgName:        g.pkgName,
		ImpName:        g.impName,
		PkgPath:        g.pkgPath,
		Qualifier:      g.qualifier,
		NeedsQualifier: g.needsQualifier,
		HasReturns:     hasResults(g.funcDecl.Type),
		ReturnType:     g.returnTypeName(),
		NumReturns:     numReturns,
		TypeParamsDecl: g.formatTypeParamsDecl(),
		TypeParamsUse:  g.formatTypeParamsUse(),
	}
}

// extendedTemplateData returns template data with dynamic signature info.
func (g *callableGenerator) extendedTemplateData() callableExtendedTemplateData {
	returnVars := g.returnVarNames()
	returnFields := g.buildReturnFieldData(returnVars)

	return callableExtendedTemplateData{
		callableTemplateData: g.templateData(),
		CallableSignature:    g.paramsString(),
		CallableReturns:      g.returnsString(),
		ParamNames:           g.paramNamesString(),
		ReturnVars:           strings.Join(returnVars, ", "),
		ReturnVarsList:       returnVars,
		ReturnFields:         returnFields,
		ResultParams:         g.resultParamsString(),
		ResultComparisons:    g.resultComparisonsString("s.Returned"),
		ResultComparisons2:   g.resultComparisonsString("ret"),
	}
}

// extendedTemplateDataForMatchers returns template data for matcher-based methods (ExpectReturnedValuesShould).
func (g *callableGenerator) extendedTemplateDataForMatchers() callableExtendedTemplateData {
	returnVars := g.returnVarNames()
	returnFields := g.buildReturnFieldData(returnVars)

	return callableExtendedTemplateData{
		callableTemplateData: g.templateData(),
		CallableSignature:    g.paramsString(),
		CallableReturns:      g.returnsString(),
		ParamNames:           g.paramNamesString(),
		ReturnVars:           strings.Join(returnVars, ", "),
		ReturnVarsList:       returnVars,
		ReturnFields:         returnFields,
		ResultParams:         g.resultParamsAsAnyString(),
		ResultComparisons:    g.resultComparisonsMatcherString("s.Returned"),
		ResultComparisons2:   g.resultComparisonsMatcherString("ret"),
	}
}

// execTemplate executes a template and writes the result to the buffer.
func (g *callableGenerator) execTemplate(tmpl *template.Template, data any) {
	g.ps(executeTemplate(tmpl, data))
}

// buildReturnFieldData builds return field data with types for templates.
func (g *callableGenerator) buildReturnFieldData(returnVars []string) []returnFieldData {
	if !hasResults(g.funcDecl.Type) {
		return nil
	}

	results := extractResults(g.fset, g.funcDecl.Type)
	fields := make([]returnFieldData, len(results))

	for resultIdx, result := range results {
		name := ""
		if resultIdx < len(returnVars) {
			name = returnVars[resultIdx]
		}

		fields[resultIdx] = returnFieldData{
			Index: result.Index,
			Name:  name,
			Type:  g.typeWithQualifier(result.Field.Type),
		}
	}

	return fields
}

// paramsString returns the parameter list as a string.
func (g *callableGenerator) paramsString() string {
	var buf strings.Builder
	g.writeParamsWithQualifiersTo(&buf)

	return buf.String()
}

// returnsString returns the return type list as a string.
func (g *callableGenerator) returnsString() string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(" (")
	g.writeResultTypesWithQualifiersTo(&buf)
	buf.WriteString(")")

	return buf.String()
}

// generateHeader writes the package declaration and imports.
func (g *callableGenerator) generateHeader() {
	g.execTemplate(callableHeaderTemplate, g.templateData())
}

// generateReturnStruct generates the return value struct if function has returns.
func (g *callableGenerator) generateReturnStruct() {
	g.execTemplate(callableReturnStructTemplate, g.extendedTemplateData())
}

// generateMainStruct generates the main wrapper struct.
func (g *callableGenerator) generateMainStruct() {
	g.execTemplate(callableMainStructTemplate, g.extendedTemplateData())
}

// generateConstructor generates the New{ImpName} constructor function.
func (g *callableGenerator) generateConstructor() {
	g.execTemplate(callableConstructorTemplate, g.extendedTemplateData())
}

// generateStartMethod generates the Start method.
func (g *callableGenerator) generateStartMethod() {
	g.execTemplate(callableStartMethodTemplate, g.extendedTemplateData())
}

// generateExpectReturnedValuesMethod generates both ExpectReturnedValuesAre and ExpectReturnedValuesShould methods.
func (g *callableGenerator) generateExpectReturnedValuesMethod() {
	// Generate type-safe version
	g.execTemplate(callableExpectReturnedValuesAreTemplate, g.extendedTemplateData())

	// Generate matcher-based version
	g.execTemplate(callableExpectReturnedValuesShouldTemplate, g.extendedTemplateDataForMatchers())
}

// generateExpectPanicWithMethod generates the ExpectPanicWith method.
func (g *callableGenerator) generateExpectPanicWithMethod() {
	g.execTemplate(callableExpectPanicWithTemplate, g.templateData())
}

// generateResponseStruct generates the response struct.
func (g *callableGenerator) generateResponseStruct() {
	g.execTemplate(callableResponseStructTemplate, g.templateData())
}

// generateResponseMethods generates methods on the response struct.
func (g *callableGenerator) generateResponseMethods() {
	// Type method
	g.execTemplate(callableResponseTypeMethodTemplate, g.templateData())

	// AsReturn method
	g.execTemplate(callableAsReturnMethodTemplate, g.extendedTemplateData())
}

// generateGetResponseMethod generates the GetResponse method.
func (g *callableGenerator) generateGetResponseMethod() {
	g.execTemplate(callableGetResponseMethodTemplate, g.extendedTemplateData())
}

// Helper methods

// returnVarNames generates return variable names for the function call.
func (g *callableGenerator) returnVarNames() []string {
	if !hasResults(g.funcDecl.Type) {
		return nil
	}

	numReturns := g.numReturns()
	vars := make([]string, numReturns)

	for i := range vars {
		vars[i] = fmt.Sprintf("ret%d", i)
	}

	return vars
}

// paramNamesString returns comma-separated parameter names for function calls.
func (g *callableGenerator) paramNamesString() string {
	params := extractParams(g.fset, g.funcDecl.Type)
	return paramNamesToString(params)
}

// resultParamsString returns parameters for ExpectReturnedValues (v1 Type1, v2 Type2, ...).
func (g *callableGenerator) resultParamsString() string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	var buf strings.Builder

	results := extractResults(g.fset, g.funcDecl.Type)

	for i, r := range results {
		if i > 0 {
			buf.WriteString(", ")
		}

		fmt.Fprintf(&buf, "v%d %s", r.Index+1, g.typeWithQualifier(r.Field.Type))
	}

	return buf.String()
}

// resultComparisonsString returns comparison code for return values.
func (g *callableGenerator) resultComparisonsString(varName string) string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	var buf strings.Builder

	results := extractResults(g.fset, g.funcDecl.Type)

	for _, result := range results {
		resultName := fmt.Sprintf("v%d", result.Index+1)
		isComparable := isComparableExpr(result.Field.Type, g.typesInfo)

		if isComparable {
			fmt.Fprintf(&buf, "\t\tif %s.Result%d != %s {\n", varName, result.Index, resultName)
		} else {
			fmt.Fprintf(&buf, "\t\tif !reflect.DeepEqual(%s.Result%d, %s) {\n", varName, result.Index, resultName)
		}

		fmt.Fprintf(&buf, "\t\t\ts.T.Fatalf(\"expected return value %d to be %%v, got %%v\", %s, %s.Result%d)\n",
			result.Index, resultName, varName, result.Index)
		buf.WriteString("\t\t}\n")
	}

	return buf.String()
}

// resultParamsAsAnyString returns parameters for ExpectReturnedValuesShould (v1 any, v2 any, ...).
func (g *callableGenerator) resultParamsAsAnyString() string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	var buf strings.Builder

	results := extractResults(g.fset, g.funcDecl.Type)

	for i, r := range results {
		if i > 0 {
			buf.WriteString(", ")
		}

		fmt.Fprintf(&buf, "v%d any", r.Index+1)
	}

	return buf.String()
}

// resultComparisonsMatcherString returns comparison code for return values using matchers.
func (g *callableGenerator) resultComparisonsMatcherString(varName string) string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	var buf strings.Builder

	results := extractResults(g.fset, g.funcDecl.Type)

	// Declare variables once
	if len(results) > 0 {
		buf.WriteString("\t\tvar ok bool\n")
		buf.WriteString("\t\tvar msg string\n")
	}

	for _, result := range results {
		resultName := fmt.Sprintf("v%d", result.Index+1)

		fmt.Fprintf(&buf, "\t\tok, msg = imptest.MatchValue(%s.Result%d, %s)\n", varName, result.Index, resultName)
		buf.WriteString("\t\tif !ok {\n")
		fmt.Fprintf(&buf, "\t\t\ts.T.Fatalf(\"return value %d: %%s\", msg)\n", result.Index)
		buf.WriteString("\t\t}\n")
	}

	return buf.String()
}

// writeParamsWithQualifiersTo writes function parameters with package qualifiers to a buffer.
func (g *callableGenerator) writeParamsWithQualifiersTo(buf *strings.Builder) {
	params := g.funcDecl.Type.Params
	if params == nil || len(params.List) == 0 {
		return
	}

	for i, field := range params.List {
		if i > 0 {
			buf.WriteString(", ")
		}

		if len(field.Names) > 0 {
			writeFieldNamesTo(buf, field.Names)
			buf.WriteString(" ")
		}

		buf.WriteString(g.typeWithQualifier(field.Type))
	}
}

// writeFieldNamesTo writes parameter field names to a buffer, separated by commas.
func writeFieldNamesTo(buf *strings.Builder, names []*ast.Ident) {
	for j, name := range names {
		if j > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(name.Name)
	}
}

// writeResultTypesWithQualifiersTo writes function return types to a buffer.
func (g *callableGenerator) writeResultTypesWithQualifiersTo(buf *strings.Builder) {
	if !hasResults(g.funcDecl.Type) {
		return
	}

	results := extractResults(g.fset, g.funcDecl.Type)

	for i, r := range results {
		if i > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(g.typeWithQualifier(r.Field.Type))
	}
}

// Functions

// callableGetPackageInfo extracts package info for a callable function.
func callableGetPackageInfo(
	interfaceName string,
	pkgLoader PackageLoader,
	currentPkgName string,
) (pkgPath, pkgName string, err error) {
	return GetPackageInfo(interfaceName, pkgLoader, currentPkgName)
}
