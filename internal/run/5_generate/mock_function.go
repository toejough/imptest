package generate

import (
	"fmt"
	"go/format"
	"go/token"
	"strings"

	"github.com/dave/dst"

	detect "github.com/toejough/imptest/internal/run/3_detect"
)

// FunctionDependencyCode generates dependency mock code for a package-level function.
func FunctionDependencyCode(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	funcDecl *dst.FuncDecl,
) (string, error) {
	gen := newFunctionDependencyGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, funcDecl)
	return gen.generate()
}

// FunctionTypeDependencyCode generates dependency mock code for a function type.
// It creates a synthetic function declaration from the function type and delegates to FunctionDependencyCode.
func FunctionTypeDependencyCode(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	funcTypeDetails detect.FuncTypeWithDetails,
) (string, error) {
	// Create a synthetic function declaration from the function type
	// For a type like: type Handler func(w http.ResponseWriter, r *http.Request)
	// We create: func Handler(w http.ResponseWriter, r *http.Request) { ... }
	funcDecl := &dst.FuncDecl{
		Name: &dst.Ident{Name: funcTypeDetails.TypeName},
		Type: funcTypeDetails.FuncType,
	}

	// Attach type parameters if present (nil assignment is harmless)
	funcDecl.Type.TypeParams = funcTypeDetails.TypeParams

	return FunctionDependencyCode(astFiles, info, fset, pkgImportPath, pkgLoader, funcDecl)
}

type funcDepTemplateData struct {
	baseTemplateData //nolint:unused // Embedded struct accessed by templates

	MockName     string                // Constructor function name (e.g., "MockProcessOrder")
	MockTypeName string                // Struct type name (e.g., "ProcessOrderMock")
	FuncName     string                // Original function name (e.g., "ProcessOrder")
	FuncSig      string                // Function signature for Func() return type
	Method       depMethodTemplateData // Method template data
}

type functionDependencyGenerator struct {
	baseGenerator

	mockName     string // Constructor function name (e.g., "MockProcessOrder")
	mockTypeName string // Struct type name (e.g., "ProcessOrderMock")
	funcName     string // Original function name (e.g., "ProcessOrder")
	astFiles     []*dst.File
	funcDecl     *dst.FuncDecl
}

// buildFuncSig builds the function signature string for the Func() return type.
func (gen *functionDependencyGenerator) buildFuncSig() string {
	paramsStr, _ := gen.buildParamStrings(gen.funcDecl.Type)
	resultsStr, _ := gen.buildResultStrings(gen.funcDecl.Type)

	if resultsStr == "" {
		return fmt.Sprintf("func(%s)", paramsStr)
	}

	return fmt.Sprintf("func(%s) %s", paramsStr, resultsStr)
}

// buildMethodTemplateData builds template data for the function mock.
func (gen *functionDependencyGenerator) buildMethodTemplateData() depMethodTemplateData {
	ftype := gen.funcDecl.Type

	// Build parameter string and collect param names
	paramsStr, paramNames := gen.buildParamStrings(ftype)

	// Build results string and collect result types
	resultsStr, resultTypes := gen.buildResultStrings(ftype)

	// Check for variadic parameters and build argument strings
	variadicResult := buildVariadicArgs(ftype, paramNames)

	// Build result variables
	resultVars, returnList := buildResultVars(resultTypes)

	// Extract parameter fields for type-safe args
	paramFields := gen.buildParamFields(ftype)

	// Build typed return parameters for type-safe Return
	typedReturnParams, returnParamNames := buildTypedReturnParams(resultTypes)

	// Build method template data with base fields
	return depMethodTemplateData{
		baseTemplateData: baseTemplateData{
			PkgName:        gen.pkgName,
			PkgImptest:     "_imptest",
			PkgTime:        pkgTime,
			TypeParamsDecl: gen.formatTypeParamsDecl(),
			TypeParamsUse:  gen.formatTypeParamsUse(),
		},
		MethodName:        gen.funcName,
		InterfaceType:     "", // Not used for function mocks
		ImplName:          "",
		Params:            paramsStr,
		Results:           resultsStr,
		HasVariadic:       variadicResult.hasVariadic,
		NonVariadicArgs:   variadicResult.nonVariadicArgs,
		VariadicArg:       variadicResult.variadicArg,
		Args:              variadicResult.allArgs,
		ArgNames:          variadicResult.allArgs,
		HasResults:        len(resultTypes) > 0,
		ResultVars:        resultVars,
		ReturnList:        returnList,
		ReturnStatement:   "return " + returnList,
		ParamFields:       paramFields,
		HasParams:         len(paramFields) > 0,
		ArgsTypeName:      gen.mockTypeName + "Args",
		CallTypeName:      gen.mockTypeName + "Call",
		MethodTypeName:    gen.mockTypeName + "Method",
		TypedParams:       paramsStr,
		TypedReturnParams: typedReturnParams,
		ReturnParamNames:  returnParamNames,
	}
}

// buildParamFields extracts parameter fields for type-safe args.
func (gen *functionDependencyGenerator) buildParamFields(ftype *dst.FuncType) []paramField {
	paramInfos := extractParams(gen.fset, ftype)
	paramFields := make([]paramField, 0, len(paramInfos))

	for _, pinfo := range paramInfos {
		fieldName := pinfo.Name
		if strings.HasPrefix(pinfo.Name, "param") {
			fieldName = fmt.Sprintf("A%d", pinfo.Index+1)
		} else {
			fieldName = strings.ToUpper(string(fieldName[0])) + fieldName[1:]
		}

		paramFields = append(paramFields, paramField{
			Name:  fieldName,
			Type:  normalizeVariadicType(gen.typeWithQualifier(pinfo.Field.Type)),
			Index: pinfo.Index,
		})
	}

	return paramFields
}

// checkIfQualifierNeeded determines if we need a package qualifier.
func (gen *functionDependencyGenerator) checkIfQualifierNeeded() {
	gen.baseGenerator.checkIfQualifierNeeded(gen.funcDecl.Type)
}

// collectAdditionalImports collects imports needed for function parameter/return types.
func (gen *functionDependencyGenerator) collectAdditionalImports() []importInfo {
	var imports []importInfo

	seenPaths := make(map[string]bool)

	// Get imports from the file containing the function
	var sourceImports []*dst.ImportSpec

	for _, file := range gen.astFiles {
		if len(file.Imports) > 0 {
			sourceImports = append(sourceImports, file.Imports...)
		}
	}

	// Collect external types from parameters
	if gen.funcDecl.Type.Params != nil {
		for _, field := range gen.funcDecl.Type.Params.List {
			imports = append(
				imports,
				gen.collectImportsFromExpr(field.Type, sourceImports, seenPaths)...)
		}
	}

	// Collect external types from results
	if gen.funcDecl.Type.Results != nil {
		for _, field := range gen.funcDecl.Type.Results.List {
			imports = append(
				imports,
				gen.collectImportsFromExpr(field.Type, sourceImports, seenPaths)...)
		}
	}

	return imports
}

// collectImportFromSelector extracts import info from a selector expression (e.g., http.Request).
func (gen *functionDependencyGenerator) collectImportFromSelector(
	sel *dst.SelectorExpr,
	sourceImports []*dst.ImportSpec,
	seenPaths map[string]bool,
) []importInfo {
	ident, ok := sel.X.(*dst.Ident)
	if !ok {
		return nil
	}

	pkgName := ident.Name

	for _, imp := range sourceImports {
		path := strings.Trim(imp.Path.Value, `"`)

		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}

		// Match by alias or by path suffix
		if (alias != "" && alias == pkgName) || strings.HasSuffix(path, "/"+pkgName) ||
			path == pkgName {
			if !seenPaths[path] {
				seenPaths[path] = true
				return []importInfo{{Alias: pkgName, Path: path}}
			}

			break
		}
	}

	return nil
}

// collectImportsFromExpr collects imports from a type expression.
func (gen *functionDependencyGenerator) collectImportsFromExpr(
	expr dst.Expr,
	sourceImports []*dst.ImportSpec,
	seenPaths map[string]bool,
) []importInfo {
	var imports []importInfo

	switch typedExpr := expr.(type) {
	case *dst.SelectorExpr:
		imports = gen.collectImportFromSelector(typedExpr, sourceImports, seenPaths)

	case *dst.StarExpr:
		imports = gen.collectImportsFromExpr(typedExpr.X, sourceImports, seenPaths)

	case *dst.ArrayType:
		imports = gen.collectImportsFromExpr(typedExpr.Elt, sourceImports, seenPaths)

	case *dst.MapType:
		imports = gen.collectImportsFromExpr(typedExpr.Key, sourceImports, seenPaths)
		imports = append(imports, gen.collectImportsFromExpr(typedExpr.Value, sourceImports, seenPaths)...)

	case *dst.FuncType:
		if typedExpr.Params != nil {
			for _, field := range typedExpr.Params.List {
				imports = append(imports, gen.collectImportsFromExpr(field.Type, sourceImports, seenPaths)...)
			}
		}

		if typedExpr.Results != nil {
			for _, field := range typedExpr.Results.List {
				imports = append(imports, gen.collectImportsFromExpr(field.Type, sourceImports, seenPaths)...)
			}
		}
	}

	return imports
}

// generate produces the function dependency mock code.
func (gen *functionDependencyGenerator) generate() (string, error) {
	// Pre-scan to determine what imports are needed
	gen.checkIfQualifierNeeded()

	// Get the global template registry (initialized at package load time)
	templates := NewTemplateRegistry()

	// Generate using templates
	gen.generateWithTemplates(templates)

	formatted, err := format.Source(gen.bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// generateWithTemplates generates code using templates.
func (gen *functionDependencyGenerator) generateWithTemplates(templates *TemplateRegistry) {
	methodData := gen.buildMethodTemplateData()

	// Build base template data
	base := baseTemplateData{
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
	}

	// Build function dependency template data
	data := funcDepTemplateData{
		baseTemplateData: base,
		MockName:         gen.mockName,
		MockTypeName:     gen.mockTypeName,
		FuncName:         gen.funcName,
		FuncSig:          gen.buildFuncSig(),
		Method:           methodData,
	}

	// Write header
	templates.WriteDepHeader(&gen.buf, data)

	// Write type-safe wrappers first (Args, Call, Method types) - needed before Handle struct
	templates.WriteDepArgsStruct(&gen.buf, methodData)
	templates.WriteDepCallWrapper(&gen.buf, methodData)
	templates.WriteFuncDepMethodWrapper(&gen.buf, data)

	// Write function mock handle struct
	templates.WriteFuncDepMockStruct(&gen.buf, data)

	// Write constructor (creates Handle with inlined Mock function)
	templates.WriteFuncDepConstructor(&gen.buf, data)
}

// newFunctionDependencyGenerator creates a new function dependency mock generator.
func newFunctionDependencyGenerator(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	funcDecl *dst.FuncDecl,
) *functionDependencyGenerator {
	pkgPath, qualifier := resolveFunctionPackageInfo(info, pkgImportPath, pkgLoader)

	// Convert MockXxx -> XxxMock for the struct type name
	mockTypeName := strings.TrimPrefix(info.ImpName, "Mock") + "Mock"

	return &functionDependencyGenerator{
		baseGenerator: newBaseGenerator(fset, info.PkgName, info.ImpName, pkgPath, qualifier, nil),
		mockName:      info.ImpName,
		mockTypeName:  mockTypeName,
		funcName:      info.LocalInterfaceName, // This is actually the function name
		astFiles:      astFiles,
		funcDecl:      funcDecl,
	}
}

// resolveFunctionPackageInfo determines the package path and qualifier for a function.
// Note: resolvePackageInfo converts all ErrNotPackageReference errors to empty returns,
// and GetPackageInfo only returns nil or ErrNotPackageReference, so this cannot fail.
func resolveFunctionPackageInfo(
	info GeneratorInfo, pkgImportPath string, pkgLoader detect.PackageLoader,
) (string, string) {
	// External package
	if pkgImportPath != "." {
		if strings.Contains(info.InterfaceName, ".") {
			pkgPath, qualifier, _ := resolvePackageInfo(info, pkgLoader)
			return pkgPath, qualifier
		}

		// Local reference to external package
		pkgPath := pkgImportPath
		parts := strings.Split(pkgImportPath, "/")

		return pkgPath, parts[len(parts)-1]
	}

	// Test package needs to import the main package
	if strings.HasSuffix(info.PkgName, "_test") {
		pkgPath, qualifier, _ := resolvePackageInfo(info, pkgLoader)
		return pkgPath, qualifier
	}

	return "", ""
}
