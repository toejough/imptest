package run

import (
	"fmt"
	"go/format"
	"go/token"
	go_types "go/types"
	"strings"

	"github.com/dave/dst"
)

// callableExtendedTemplateData extends callableTemplateData with dynamic signature info.
type callableExtendedTemplateData struct {
	callableTemplateData //nolint:unused // Embedded fields accessed via promotion

	CallableSignature string
	CallableReturns   string
	ParamNames        string   // comma-separated parameter names for calling
	ReturnVars        string   // comma-separated return variable names (ret0, ret1, ...)
	ReturnVarsList    []string // slice of return variable names
	ReturnFields      []returnFieldData
	ResultParams      string // parameters for ExpectReturnedValues (v1 Type1, v2 Type2, ...)
	ResultParamsAny   string // parameters for ExpectReturnedValuesShould (v1 any, v2 any, ...)
	ResultComparisons string // comparisons for ExpectReturnedValues
	ResultMatchers    string // matcher-based comparisons for ExpectReturnedValuesShould
}

// Types

// callableGenerator holds state for generating callable wrapper code.
type callableGenerator struct {
	baseGenerator

	templates                  *TemplateRegistry
	funcDecl                   *dst.FuncDecl
	astFiles                   []*dst.File                   // Source AST files for import resolution
	cachedTemplateData         *callableTemplateData         // Cache to avoid redundant templateData() calls
	cachedExtendedTemplateData *callableExtendedTemplateData // Cache to avoid redundant extendedTemplateData() calls
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

// checkIfReflectNeeded scans return types and sets needsReflect if any are non-comparable.
// This must be called before generating templates to ensure the reflect import is included.
func (g *callableGenerator) checkIfReflectNeeded(funcType *dst.FuncType) {
	if !hasResults(funcType) {
		return
	}

	results := extractResults(g.fset, funcType)
	for _, result := range results {
		if !isComparableExpr(result.Field.Type, g.typesInfo) {
			g.needsReflect = true
			return
		}
	}
}

// collectAdditionalImports collects all external type imports needed for the callable function signature.
func (g *callableGenerator) collectAdditionalImports() []importInfo {
	return collectImportsFromFuncDecl(g.funcDecl, g.astFiles)
}

// extendedTemplateData returns template data with dynamic signature info.
// The result is cached after the first call to avoid redundant struct construction.
func (g *callableGenerator) extendedTemplateData() callableExtendedTemplateData {
	if g.cachedExtendedTemplateData != nil {
		return *g.cachedExtendedTemplateData
	}

	returnVars := g.returnVarNames()
	returnFields := g.buildReturnFieldData(returnVars)

	data := callableExtendedTemplateData{
		callableTemplateData: g.templateData(),
		CallableSignature:    g.paramsString(),
		CallableReturns:      g.returnsString(),
		ParamNames:           g.paramNamesString(),
		ReturnVars:           strings.Join(returnVars, ", "),
		ReturnVarsList:       returnVars,
		ReturnFields:         returnFields,
		ResultParams:         g.resultParamsString(),
		ResultParamsAny:      g.resultParamsAnyString(),
		ResultComparisons:    g.resultComparisonsString("s.Returned"),
		ResultMatchers:       g.resultMatchersString("s.Returned"),
	}

	g.cachedExtendedTemplateData = &data

	return data
}

// generateCallableTemplates executes all templates to generate the callable wrapper code.
func (g *callableGenerator) generateCallableTemplates() {
	// Generate header
	baseData := g.templateData()
	g.templates.WriteCallableHeader(&g.buf, baseData.baseTemplateData)

	// Generate structs and methods
	extData := g.extendedTemplateData()
	g.templates.WriteCallableReturnStruct(&g.buf, extData)
	g.templates.WriteCallableMainStruct(&g.buf, extData)
	g.templates.WriteCallableConstructor(&g.buf, extData)
	g.templates.WriteCallableStartMethod(&g.buf, extData)

	// Generate ExpectReturnedValues methods
	g.templates.WriteCallableExpectReturnedValuesAre(&g.buf, extData)
	g.templates.WriteCallableExpectReturnedValuesShould(&g.buf, extData)

	// Generate panic and response methods
	g.templates.WriteCallableExpectPanicWith(&g.buf, baseData)
	g.templates.WriteCallableResponseStruct(&g.buf, baseData)
	g.templates.WriteCallableResponseTypeMethod(&g.buf, baseData)
}

// numReturns returns the total number of return values.
// This should only be called when hasResults(g.funcDecl.Type) is true.
func (g *callableGenerator) numReturns() int {
	return countFields(g.funcDecl.Type.Results)
}

// paramNamesString returns comma-separated parameter names for function calls.
func (g *callableGenerator) paramNamesString() string {
	params := extractParams(g.fset, g.funcDecl.Type)
	return paramNamesToString(params)
}

// paramsString returns the parameter list as a string.
func (g *callableGenerator) paramsString() string {
	var buf strings.Builder
	g.writeParamsWithQualifiersTo(&buf)

	return buf.String()
}

// resultComparisonsString returns comparison code for return values.
func (g *callableGenerator) resultComparisonsString(varName string) string {
	return g.writeResultChecks(varName, false)
}

// resultMatchersString returns matcher-based comparison code for return values.
func (g *callableGenerator) resultMatchersString(varName string) string {
	return g.writeResultChecks(varName, true)
}

// resultParamsAnyString returns parameters for ExpectReturnedValuesShould (v1 any, v2 any, ...).
func (g *callableGenerator) resultParamsAnyString() string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	results := extractResults(g.fset, g.funcDecl.Type)

	return formatResultParameters(results, "v", 1, func(fieldInfo) string {
		return anyTypeString
	})
}

// resultParamsString returns parameters for ExpectReturnedValues (v1 Type1, v2 Type2, ...).
func (g *callableGenerator) resultParamsString() string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	results := extractResults(g.fset, g.funcDecl.Type)

	return formatResultParameters(results, "v", 1, func(r fieldInfo) string {
		return g.typeWithQualifier(r.Field.Type)
	})
}

// returnTypeName returns the appropriate type name for return channels and fields.
// Returns "{impName}Return{TypeParams}" if the function has returns, otherwise "struct{}".
func (g *callableGenerator) returnTypeName() string {
	if hasResults(g.funcDecl.Type) {
		return g.impName + "Return" + g.formatTypeParamsUse()
	}

	return "struct{}"
}

// returnVarNames generates return variable names for the function call.
func (g *callableGenerator) returnVarNames() []string {
	if !hasResults(g.funcDecl.Type) {
		return nil
	}

	return generateResultVarNames(g.numReturns(), "ret")
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

// templateData returns the base template data for this generator.
// The result is cached after the first call to avoid redundant struct construction.
func (g *callableGenerator) templateData() callableTemplateData {
	if g.cachedTemplateData != nil {
		return *g.cachedTemplateData
	}

	numReturns := 0
	if hasResults(g.funcDecl.Type) {
		numReturns = g.numReturns()
	}

	data := callableTemplateData{
		baseTemplateData: baseTemplateData{
			PkgName:           g.pkgName,
			ImpName:           g.impName,
			PkgPath:           g.pkgPath,
			Qualifier:         g.qualifier,
			NeedsQualifier:    g.needsQualifier,
			TypeParamsDecl:    g.formatTypeParamsDecl(),
			TypeParamsUse:     g.formatTypeParamsUse(),
			PkgTesting:        pkgTesting,
			PkgFmt:            pkgFmt,
			PkgImptest:        pkgImptest,
			PkgTime:           pkgTime,
			PkgReflect:        pkgReflect,
			NeedsFmt:          g.needsFmt,
			NeedsReflect:      g.needsReflect,
			NeedsImptest:      g.needsImptest,
			AdditionalImports: g.collectAdditionalImports(),
		},
		HasReturns: hasResults(g.funcDecl.Type),
		ReturnType: g.returnTypeName(),
		NumReturns: numReturns,
	}

	g.cachedTemplateData = &data

	return data
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

		if hasFieldNames(field) {
			buf.WriteString(joinWith(field.Names, func(n *dst.Ident) string { return n.Name }, ", "))
			buf.WriteString(" ")
		}

		buf.WriteString(g.typeWithQualifier(field.Type))
	}
}

// writeResultChecks generates comparison code for return values.
// When useMatcher is true, uses imptest.MatchValue for flexible matching.
// When useMatcher is false, uses == or reflect.DeepEqual for equality checks.
func (g *callableGenerator) writeResultChecks(varName string, useMatcher bool) string {
	if !hasResults(g.funcDecl.Type) {
		return ""
	}

	var buf strings.Builder

	results := extractResults(g.fset, g.funcDecl.Type)

	for _, result := range results {
		expectedName := fmt.Sprintf("v%d", result.Index+1)
		actualExpr := fmt.Sprintf("%s.Result%d", varName, result.Index)

		if useMatcher {
			fmt.Fprintf(&buf, "\t\tok, msg = %s.MatchValue(%s, %s)\n", pkgImptest, actualExpr, expectedName)
			fmt.Fprintf(&buf, "\t\tif !ok {\n")
			fmt.Fprintf(&buf, "\t\t\ts.T.Fatalf(\"return value %d: %%s\", msg)\n", result.Index)
		} else {
			isComparable := isComparableExpr(result.Field.Type, g.typesInfo)

			if isComparable {
				fmt.Fprintf(&buf, "\t\tif %s != %s {\n", actualExpr, expectedName)
			} else {
				g.needsReflect = true

				fmt.Fprintf(&buf, "\t\tif !%s.DeepEqual(%s, %s) {\n", pkgReflect, actualExpr, expectedName)
			}

			fmt.Fprintf(&buf, "\t\t\ts.T.Fatalf(\"expected return value %d to be %%v, got %%v\", %s, %s)\n",
				result.Index, expectedName, actualExpr)
		}

		buf.WriteString("\t\t}\n")
	}

	return buf.String()
}

// writeResultTypesWithQualifiersTo writes function return types to a buffer.
func (g *callableGenerator) writeResultTypesWithQualifiersTo(buf *strings.Builder) {
	if !hasResults(g.funcDecl.Type) {
		return
	}

	results := extractResults(g.fset, g.funcDecl.Type)

	buf.WriteString(joinWith(results, func(r fieldInfo) string {
		return g.typeWithQualifier(r.Field.Type)
	}, ", "))
}

// Functions

// returnFieldData holds data for a single return field.
type returnFieldData struct {
	Index int
	Name  string
	Type  string // Type name for struct field definitions
}

// Entry Point

// generateCallableWrapperCode generates a type-safe wrapper for a callable function.
//
//nolint:cyclop,funlen,nestif // Code generation requires conditional logic for package resolution
func generateCallableWrapperCode(
	astFiles []*dst.File,
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
		// For callables, GetPackageInfo won't work since function names don't have package qualifiers
		// Instead, if we're in a test package, we need to import the source package
		if strings.HasSuffix(info.pkgName, "_test") {
			// We're in a test package (e.g., visitor_test) and the callable is from the source package (e.g., visitor)
			// We need to import the source package
			pkgPath = pkgImportPath

			// Get the actual package name by loading the package
			srcFiles, _, _, err := pkgLoader.Load(pkgImportPath)
			if err != nil {
				return "", fmt.Errorf("failed to load source package: %w", err)
			}

			// Extract package name from the first file
			if len(srcFiles) > 0 && srcFiles[0].Name != nil {
				qualifier = srcFiles[0].Name.Name
			} else {
				// Fallback: extract from import path (last component)
				parts := strings.Split(pkgImportPath, "/")
				qualifier = parts[len(parts)-1]
			}
		} else {
			pkgPath, qualifier, err = GetPackageInfo(info.interfaceName, pkgLoader, info.pkgName)
			if err != nil {
				return "", fmt.Errorf("failed to get callable package info: %w", err)
			}
		}
	} else if strings.HasSuffix(info.pkgName, "_test") {
		// Special case: when pkgImportPath is "." and we're in a test package,
		// the callable is from the non-test version of this package
		pkgPath, qualifier = resolveTestPackageImport(pkgLoader, info.pkgName)
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
		astFiles: astFiles,
	}

	gen.checkIfQualifierNeeded(gen.funcDecl.Type)
	gen.checkIfReflectNeeded(gen.funcDecl.Type)

	err = gen.checkIfValidForExternalUsage(gen.funcDecl.Type)
	if err != nil {
		return "", err
	}

	// Initialize template registry
	gen.templates, err = NewTemplateRegistry()
	if err != nil {
		return "", fmt.Errorf("failed to initialize template registry: %w", err)
	}

	gen.generateCallableTemplates()

	formatted, err := format.Source(gen.bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}
