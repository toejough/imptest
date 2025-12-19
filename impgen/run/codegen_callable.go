package run

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"text/template"
	"unicode"
)

// Entry Point

// generateCallableWrapperCode generates a type-safe wrapper for a callable function.
func generateCallableWrapperCode(
	astFiles []*ast.File,
	info generatorInfo,
	fset *token.FileSet,
	typesInfo *types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
) (string, error) {
	funcDecl, err := findFunctionInAST(astFiles, info.localInterfaceName, pkgImportPath)
	if err != nil {
		return "", err
	}

	pkgPath, qualifier, err := callableGetPackageInfo(funcDecl, info.interfaceName, pkgLoader)
	if err != nil {
		return "", fmt.Errorf("failed to get callable package info: %w", err)
	}

	gen := &callableGenerator{
		codeWriter: codeWriter{fset: fset},
		pkgName:    info.pkgName,
		impName:    info.impName,
		funcDecl:   funcDecl,
		pkgPath:    pkgPath,
		qualifier:  qualifier,
		typeParams: funcDecl.Type.TypeParams,
		typesInfo:  typesInfo,
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

// Types

// callableGenerator holds state for generating callable wrapper code.
type callableGenerator struct {
	codeWriter

	pkgName    string
	impName    string
	funcDecl   *ast.FuncDecl
	pkgPath    string
	qualifier  string
	typeParams *ast.FieldList // Type parameters for generic functions
	typesInfo  *types.Info    // Type information for comparability checks
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
		HasReturns:     hasResults(g.funcDecl.Type),
		ReturnType:     g.returnTypeName(),
		NumReturns:     numReturns,
		TypeParamsDecl: g.formatTypeParamsDecl(),
		TypeParamsUse:  g.formatTypeParamsUse(),
	}
}

// formatTypeParamsDecl formats type parameters for declaration (e.g., "[T any, U comparable]").
// Returns empty string if there are no type parameters.
func (g *callableGenerator) formatTypeParamsDecl() string {
	if g.typeParams == nil || len(g.typeParams.List) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("[")

	for i, field := range g.typeParams.List {
		if i > 0 {
			buf.WriteString(", ")
		}

		// Write parameter names
		for j, name := range field.Names {
			if j > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(name.Name)
		}

		// Write constraint
		if field.Type != nil {
			buf.WriteString(" ")
			buf.WriteString(exprToString(g.fset, field.Type))
		}
	}

	buf.WriteString("]")

	return buf.String()
}

// formatTypeParamsUse formats type parameters for instantiation (e.g., "[T, U]").
// Returns empty string if there are no type parameters.
func (g *callableGenerator) formatTypeParamsUse() string {
	if g.typeParams == nil || len(g.typeParams.List) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("[")

	first := true

	for _, field := range g.typeParams.List {
		for _, name := range field.Names {
			if !first {
				buf.WriteString(", ")
			}

			buf.WriteString(name.Name)

			first = false
		}
	}

	buf.WriteString("]")

	return buf.String()
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
		ResultComparisons:    g.resultComparisonsMatcherString("s.returned"),
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

		fmt.Fprintf(&buf, "\t\t\ts.t.Fatalf(\"expected return value %%d to be %%v, got %%v\", %d, %s, %s.Result%d)\n",
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
		fmt.Fprintf(&buf, "\t\t\ts.t.Fatalf(\"return value %%d: %%s\", %d, msg)\n", result.Index)
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

// typeWithQualifier returns a type expression as a string with package qualifier if needed.
func (g *callableGenerator) typeWithQualifier(expr ast.Expr) string {
	switch typeExpr := expr.(type) {
	case *ast.Ident:
		return g.typeWithQualifierIdent(typeExpr)
	case *ast.StarExpr:
		return g.typeWithQualifierStar(typeExpr)
	case *ast.ArrayType:
		return g.typeWithQualifierArray(typeExpr)
	case *ast.MapType:
		return g.typeWithQualifierMap(typeExpr)
	case *ast.ChanType:
		return g.typeWithQualifierChan(typeExpr)
	case *ast.FuncType:
		return g.typeWithQualifierFunc(typeExpr)
	case *ast.IndexExpr:
		return g.typeWithQualifierIndex(typeExpr)
	case *ast.IndexListExpr:
		return g.typeWithQualifierIndexList(typeExpr)
	default:
		var buf strings.Builder
		printer.Fprint(&buf, g.fset, expr)

		return buf.String()
	}
}

// typeWithQualifierArray handles array/slice types.
func (g *callableGenerator) typeWithQualifierArray(arrType *ast.ArrayType) string {
	var buf strings.Builder
	buf.WriteString("[")

	if arrType.Len != nil {
		printer.Fprint(&buf, g.fset, arrType.Len)
	}

	buf.WriteString("]")
	buf.WriteString(g.typeWithQualifier(arrType.Elt))

	return buf.String()
}

// typeWithQualifierChan handles channel types.
func (g *callableGenerator) typeWithQualifierChan(chanType *ast.ChanType) string {
	var buf strings.Builder

	switch chanType.Dir {
	case ast.SEND:
		buf.WriteString("chan<- ")
	case ast.RECV:
		buf.WriteString("<-chan ")
	default:
		buf.WriteString("chan ")
	}

	buf.WriteString(g.typeWithQualifier(chanType.Value))

	return buf.String()
}

// typeWithQualifierFunc handles function types.
func (g *callableGenerator) typeWithQualifierFunc(funcType *ast.FuncType) string {
	var buf strings.Builder
	buf.WriteString("func")

	// For function types, we need to recursively qualify parameter and result types
	// Using printer.Fprint would lose the qualification, so we format manually
	if funcType.Params != nil {
		buf.WriteString("(")

		for i, field := range funcType.Params.List {
			if i > 0 {
				buf.WriteString(", ")
			}

			// Write parameter type with qualification
			buf.WriteString(g.typeWithQualifier(field.Type))
		}

		buf.WriteString(")")
	}

	if funcType.Results != nil {
		if len(funcType.Results.List) > 1 {
			buf.WriteString(" (")
		} else {
			buf.WriteString(" ")
		}

		for i, field := range funcType.Results.List {
			if i > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(g.typeWithQualifier(field.Type))
		}

		if len(funcType.Results.List) > 1 {
			buf.WriteString(")")
		}
	}

	return buf.String()
}

// typeWithQualifierIdent handles simple identifier types.
func (g *callableGenerator) typeWithQualifierIdent(ident *ast.Ident) string {
	var buf strings.Builder

	// Don't qualify type parameters
	if !g.isTypeParameter(ident.Name) && g.qualifier != "" && len(ident.Name) > 0 && unicode.IsUpper(rune(ident.Name[0])) {
		buf.WriteString(g.qualifier)
		buf.WriteString(".")
	}

	buf.WriteString(ident.Name)

	return buf.String()
}

// isTypeParameter checks if a name is one of the function's type parameters.
func (g *callableGenerator) isTypeParameter(name string) bool {
	if g.typeParams == nil {
		return false
	}

	for _, field := range g.typeParams.List {
		for _, paramName := range field.Names {
			if paramName.Name == name {
				return true
			}
		}
	}

	return false
}

// typeWithQualifierIndex handles generic type instantiation with single type parameter.
func (g *callableGenerator) typeWithQualifierIndex(indexExpr *ast.IndexExpr) string {
	var buf strings.Builder

	// Generic type instantiation with single type parameter, e.g., Container[int]
	buf.WriteString(g.typeWithQualifier(indexExpr.X))
	buf.WriteString("[")
	buf.WriteString(g.typeWithQualifier(indexExpr.Index))
	buf.WriteString("]")

	return buf.String()
}

// typeWithQualifierIndexList handles generic type instantiation with multiple type parameters.
func (g *callableGenerator) typeWithQualifierIndexList(indexListExpr *ast.IndexListExpr) string {
	var buf strings.Builder

	// Generic type instantiation with multiple type parameters, e.g., Map[string, int]
	buf.WriteString(g.typeWithQualifier(indexListExpr.X))
	buf.WriteString("[")

	for i, index := range indexListExpr.Indices {
		if i > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(g.typeWithQualifier(index))
	}

	buf.WriteString("]")

	return buf.String()
}

// typeWithQualifierMap handles map types.
func (g *callableGenerator) typeWithQualifierMap(mapType *ast.MapType) string {
	var buf strings.Builder

	buf.WriteString("map[")
	buf.WriteString(g.typeWithQualifier(mapType.Key))
	buf.WriteString("]")
	buf.WriteString(g.typeWithQualifier(mapType.Value))

	return buf.String()
}

// typeWithQualifierStar handles pointer types.
func (g *callableGenerator) typeWithQualifierStar(t *ast.StarExpr) string {
	var buf strings.Builder

	buf.WriteString("*")
	buf.WriteString(g.typeWithQualifier(t.X))

	return buf.String()
}

// Functions

// callableGetPackageInfo extracts package info for a callable function.
func callableGetPackageInfo(
	funcDecl *ast.FuncDecl,
	interfaceName string,
	pkgLoader PackageLoader,
) (pkgPath, pkgName string, err error) {
	if !strings.Contains(interfaceName, ".") {
		return "", "", nil
	}

	if !callableFuncUsesExportedTypes(funcDecl) {
		return "", "", nil
	}

	pkgName = extractPackageName(interfaceName)

	astFiles, _, _, err := pkgLoader.Load(".")
	if err != nil {
		return "", "", fmt.Errorf("failed to load local package: %w", err)
	}

	pkgPath, err = findImportPath(astFiles, pkgName, pkgLoader)
	if err != nil {
		return "", "", err
	}

	return pkgPath, pkgName, nil
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
//
//nolint:cyclop // Simple type-switch dispatcher with no nested logic; complexity is inherent to AST node types
func callableHasExportedIdent(expr ast.Expr) bool {
	switch typeExpr := expr.(type) {
	case *ast.Ident:
		return callableHasExportedIdentInIdent(typeExpr)
	case *ast.StarExpr:
		return callableHasExportedIdentInStar(typeExpr)
	case *ast.ArrayType:
		return callableHasExportedIdentInArray(typeExpr)
	case *ast.MapType:
		return callableHasExportedIdentInMap(typeExpr)
	case *ast.ChanType:
		return callableHasExportedIdentInChan(typeExpr)
	case *ast.FuncType:
		return callableHasExportedIdentInFunc(typeExpr)
	case *ast.StructType:
		return callableHasExportedIdentInStruct(typeExpr)
	case *ast.SelectorExpr:
		return callableHasExportedIdentInSelector(typeExpr)
	case *ast.IndexExpr:
		return callableHasExportedIdentInIndex(typeExpr)
	case *ast.IndexListExpr:
		return callableHasExportedIdentInIndexList(typeExpr)
	}

	return false
}

// callableHasExportedIdentInArray checks if an array type contains exported identifiers.
func callableHasExportedIdentInArray(t *ast.ArrayType) bool {
	return callableHasExportedIdent(t.Elt)
}

// callableHasExportedIdentInChan checks if a channel type contains exported identifiers.
func callableHasExportedIdentInChan(t *ast.ChanType) bool {
	return callableHasExportedIdent(t.Value)
}

// callableHasExportedIdentInFunc checks if a function type contains exported identifiers.
func callableHasExportedIdentInFunc(funcType *ast.FuncType) bool {
	// Check parameters for exported types
	if funcType.Params != nil {
		for _, field := range funcType.Params.List {
			if callableHasExportedIdent(field.Type) {
				return true
			}
		}
	}

	// Check results for exported types
	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			if callableHasExportedIdent(field.Type) {
				return true
			}
		}
	}

	return false
}

// callableHasExportedIdentInIdent checks if an identifier is exported.
func callableHasExportedIdentInIdent(t *ast.Ident) bool {
	return len(t.Name) > 0 && unicode.IsUpper(rune(t.Name[0])) && !callableIsBuiltinType(t.Name)
}

// callableHasExportedIdentInIndex checks if a generic type instantiation contains exported identifiers.
func callableHasExportedIdentInIndex(t *ast.IndexExpr) bool {
	// Generic type instantiation with single type parameter, e.g., Container[int]
	return callableHasExportedIdent(t.X) || callableHasExportedIdent(t.Index)
}

// callableHasExportedIdentInIndexList checks if a multi-parameter generic type contains exported identifiers.
func callableHasExportedIdentInIndexList(indexList *ast.IndexListExpr) bool {
	// Generic type instantiation with multiple type parameters, e.g., Map[string, int]
	if callableHasExportedIdent(indexList.X) {
		return true
	}

	return slices.ContainsFunc(indexList.Indices, callableHasExportedIdent)
}

// callableHasExportedIdentInMap checks if a map type contains exported identifiers.
func callableHasExportedIdentInMap(t *ast.MapType) bool {
	return callableHasExportedIdent(t.Key) || callableHasExportedIdent(t.Value)
}

// callableHasExportedIdentInSelector checks if a selector expression is exported.
func callableHasExportedIdentInSelector(_ *ast.SelectorExpr) bool {
	return true
}

// callableHasExportedIdentInStar checks if a pointer type contains exported identifiers.
func callableHasExportedIdentInStar(t *ast.StarExpr) bool {
	return callableHasExportedIdent(t.X)
}

// callableHasExportedIdentInStruct checks if a struct type contains exported identifiers.
func callableHasExportedIdentInStruct(t *ast.StructType) bool {
	// Check struct fields for exported types
	if t.Fields != nil {
		for _, field := range t.Fields.List {
			if callableHasExportedIdent(field.Type) {
				return true
			}
		}
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
