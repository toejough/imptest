package run

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"strings"
	"unicode"
)

// callableGenerator holds state for generating callable wrapper code.
type callableGenerator struct {
	codeWriter

	pkgName   string
	impName   string
	funcDecl  *ast.FuncDecl
	pkgPath   string
	qualifier string
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

// hasReturns returns true if the function has return values.
func (g *callableGenerator) hasReturns() bool {
	return g.funcDecl.Type.Results != nil && len(g.funcDecl.Type.Results.List) > 0
}

// returnTypeName returns the appropriate type name for return channels and fields.
// Returns "{impName}Return" if the function has returns, otherwise "struct{}".
func (g *callableGenerator) returnTypeName() string {
	if g.hasReturns() {
		return g.impName + "Return"
	}

	return "struct{}"
}

// numReturns returns the total number of return values.
// This should only be called when hasReturns() is true.
func (g *callableGenerator) numReturns() int {
	return countFields(g.funcDecl.Type.Results)
}

// templateData returns the base template data for this generator.
func (g *callableGenerator) templateData() callableTemplateData {
	numReturns := 0
	if g.hasReturns() {
		numReturns = g.numReturns()
	}

	return callableTemplateData{
		PkgName:    g.pkgName,
		ImpName:    g.impName,
		PkgPath:    g.pkgPath,
		Qualifier:  g.qualifier,
		HasReturns: g.hasReturns(),
		ReturnType: g.returnTypeName(),
		NumReturns: numReturns,
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
		ResultComparisons:    g.resultComparisonsString("s.returned"),
		ResultComparisons2:   g.resultComparisonsString("ret"),
	}
}

// buildReturnFieldData builds return field data with types for templates.
func (g *callableGenerator) buildReturnFieldData(returnVars []string) []returnFieldData {
	if !g.hasReturns() {
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
	if !g.hasReturns() {
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
	g.ps(executeTemplate(callableHeaderTemplate, g.templateData()))
}

// generateReturnStruct generates the return value struct if function has returns.
// generateReturnStruct generates the return value struct if function has returns.
func (g *callableGenerator) generateReturnStruct() {
	g.ps(executeTemplate(callableReturnStructTemplate, g.extendedTemplateData()))
}

// generateMainStruct generates the main wrapper struct.
func (g *callableGenerator) generateMainStruct() {
	g.ps(executeTemplate(callableMainStructTemplate, g.extendedTemplateData()))
}

// generateConstructor generates the New{ImpName} constructor function.
func (g *callableGenerator) generateConstructor() {
	g.ps(executeTemplate(callableConstructorTemplate, g.extendedTemplateData()))
}

// generateStartMethod generates the Start method.
func (g *callableGenerator) generateStartMethod() {
	g.ps(executeTemplate(callableStartMethodTemplate, g.extendedTemplateData()))
}

// generateExpectReturnedValuesMethod generates the ExpectReturnedValues method.
func (g *callableGenerator) generateExpectReturnedValuesMethod() {
	g.ps(executeTemplate(callableExpectReturnedValuesTemplate, g.extendedTemplateData()))
}

// generateExpectPanicWithMethod generates the ExpectPanicWith method.
func (g *callableGenerator) generateExpectPanicWithMethod() {
	g.ps(executeTemplate(callableExpectPanicWithTemplate, g.templateData()))
}

// generateResponseStruct generates the response struct.
func (g *callableGenerator) generateResponseStruct() {
	g.ps(executeTemplate(callableResponseStructTemplate, g.templateData()))
}

// generateResponseMethods generates methods on the response struct.
func (g *callableGenerator) generateResponseMethods() {
	// Type method
	g.ps(executeTemplate(callableResponseTypeMethodTemplate, g.templateData()))

	// AsReturn method
	g.ps(executeTemplate(callableAsReturnMethodTemplate, g.extendedTemplateData()))
}

// generateGetResponseMethod generates the GetResponse method.
func (g *callableGenerator) generateGetResponseMethod() {
	g.ps(executeTemplate(callableGetResponseMethodTemplate, g.extendedTemplateData()))
}

// Helper methods

// returnVarNames generates return variable names for the function call.
func (g *callableGenerator) returnVarNames() []string {
	if !g.hasReturns() {
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
	params := g.funcDecl.Type.Params
	if params == nil || len(params.List) == 0 {
		return ""
	}

	var names []string

	for _, field := range params.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}

	return strings.Join(names, ", ")
}

// resultParamsString returns parameters for ExpectReturnedValues (v1 Type1, v2 Type2, ...).
func (g *callableGenerator) resultParamsString() string {
	if !g.hasReturns() {
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
	if !g.hasReturns() {
		return ""
	}

	var buf strings.Builder

	results := extractResults(g.fset, g.funcDecl.Type)

	for _, r := range results {
		resultName := fmt.Sprintf("v%d", r.Index+1)
		fmt.Fprintf(&buf, "\t\tif %s.Val%d != %s {\n", varName, r.Index, resultName)
		fmt.Fprintf(&buf, "\t\t\ts.t.Fatalf(\"expected return value %%d to be %%v, got %%v\", %d, %s, %s.Val%d)\n",
			r.Index, resultName, varName, r.Index)
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
			for j, name := range field.Names {
				if j > 0 {
					buf.WriteString(", ")
				}

				buf.WriteString(name.Name)
			}

			buf.WriteString(" ")
		}

		buf.WriteString(g.typeWithQualifier(field.Type))
	}
}

// writeResultTypesWithQualifiersTo writes function return types to a buffer.
func (g *callableGenerator) writeResultTypesWithQualifiersTo(buf *strings.Builder) {
	if !g.hasReturns() {
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

// typeWithQualifier returns a type expression as a string with package qualifier if needed.
func (g *callableGenerator) typeWithQualifier(expr ast.Expr) string {
	var buf strings.Builder

	switch typeExpr := expr.(type) {
	case *ast.Ident:
		if g.qualifier != "" && len(typeExpr.Name) > 0 &&
			typeExpr.Name[0] >= 'A' && typeExpr.Name[0] <= 'Z' {
			buf.WriteString(g.qualifier)
			buf.WriteString(".")
		}

		buf.WriteString(typeExpr.Name)
	case *ast.StarExpr:
		buf.WriteString("*")
		buf.WriteString(g.typeWithQualifier(typeExpr.X))
	case *ast.ArrayType:
		buf.WriteString("[")

		if typeExpr.Len != nil {
			printer.Fprint(&buf, g.fset, typeExpr.Len)
		}

		buf.WriteString("]")
		buf.WriteString(g.typeWithQualifier(typeExpr.Elt))
	default:
		printer.Fprint(&buf, g.fset, expr)
	}

	return buf.String()
}

// Functions

// generateCallableWrapperCode generates a type-safe wrapper for a callable function.
func generateCallableWrapperCode(
	astFiles []*ast.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
) (string, error) {
	funcDecl, err := findFunctionInAST(astFiles, info.localInterfaceName, pkgImportPath)
	if err != nil {
		return "", err
	}

	pkgPath, qualifier := getCallablePackageInfo(funcDecl, info.interfaceName)

	gen := &callableGenerator{
		codeWriter: codeWriter{fset: fset},
		pkgName:    info.pkgName,
		impName:    info.impName,
		funcDecl:   funcDecl,
		pkgPath:    pkgPath,
		qualifier:  qualifier,
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

// getCallablePackageInfo extracts package info for a callable function.
func getCallablePackageInfo(funcDecl *ast.FuncDecl, interfaceName string) (pkgPath, pkgName string) {
	if !strings.Contains(interfaceName, ".") {
		return "", ""
	}

	parts := strings.Split(interfaceName, ".")

	if !callableFuncUsesExportedTypes(funcDecl) {
		return "", ""
	}

	pkgName = parts[0]
	pkgPath = "github.com/toejough/imptest/UAT/" + pkgName

	return pkgPath, pkgName
}

// callableFuncUsesExportedTypes checks if a function uses exported types.
func callableFuncUsesExportedTypes(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			if callableHasExportedIdent(field.Type) {
				return true
			}
		}
	}

	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			if callableHasExportedIdent(field.Type) {
				return true
			}
		}
	}

	return false
}

// callableHasExportedIdent checks if an expression contains an exported identifier.
func callableHasExportedIdent(expr ast.Expr) bool {
	switch typeExpr := expr.(type) {
	case *ast.Ident:
		if unicode.IsUpper(rune(typeExpr.Name[0])) && !callableIsBuiltinType(typeExpr.Name) {
			return true
		}
	case *ast.StarExpr:
		return callableHasExportedIdent(typeExpr.X)
	case *ast.ArrayType:
		return callableHasExportedIdent(typeExpr.Elt)
	case *ast.MapType:
		return callableHasExportedIdent(typeExpr.Key) || callableHasExportedIdent(typeExpr.Value)
	case *ast.ChanType:
		return callableHasExportedIdent(typeExpr.Value)
	case *ast.SelectorExpr:
		return true
	}

	return false
}

// callableIsBuiltinType checks if a type name is a Go builtin.
func callableIsBuiltinType(name string) bool {
	builtins := map[string]bool{
		"bool": true, "byte": true, "complex64": true, "complex128": true,
		"error": true, "float32": true, "float64": true, "int": true,
		"int8": true, "int16": true, "int32": true, "int64": true,
		"rune": true, "string": true, "uint": true, "uint8": true,
		"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
		"any": true,
	}

	return builtins[name]
}
