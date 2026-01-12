package generate

import (
	"fmt"
	"go/format"
	"go/token"
	"strings"

	"github.com/dave/dst"

	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

// TargetCode generates target wrapper code for a function.
func TargetCode(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	funcDecl *dst.FuncDecl,
) (string, error) {
	gen := newTargetGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, funcDecl, false)
	return gen.generate()
}

// TargetCodeFromFuncType generates target wrapper code for a function type.
// It creates a synthetic function declaration from the function type and delegates to TargetCode.
func TargetCodeFromFuncType(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	funcTypeDetails detect.FuncTypeWithDetails,
) (string, error) {
	funcType := funcTypeDetails.FuncType

	// Track if we need to add external package import for qualified types
	var externalPkgPath, externalQualifier string

	// For external function types (from outside the current module), qualify types from the source package.
	// E.g., http.HandlerFunc's ResponseWriter becomes http.ResponseWriter.
	// For local function types (within the module), the types are already qualified in the source
	// (e.g., fs.DirEntry), so we don't need to import the source package.
	if pkgImportPath != "." && !isLocalPackage(pkgImportPath) {
		qualifier := extractPkgNameFromPath(pkgImportPath)
		funcType = qualifyFuncType(funcType, qualifier)
		// Store the external package info so we can add the import later
		externalPkgPath = pkgImportPath
		externalQualifier = qualifier
	}

	// Create a synthetic function declaration from the function type
	// For a type like: type WalkFunc func(path string, info string) error
	// We create: func WalkFunc(path string, info string) error { ... }
	funcDecl := &dst.FuncDecl{
		Name: &dst.Ident{Name: funcTypeDetails.TypeName},
		Type: funcType,
	}

	// If the function type has type parameters, attach them to the FuncType
	if funcTypeDetails.TypeParams != nil {
		funcDecl.Type.TypeParams = funcTypeDetails.TypeParams
	}

	gen := newTargetGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, funcDecl, true)

	// Set the external package fields for import collection
	gen.externalPkgPath = externalPkgPath
	gen.externalQualifier = externalQualifier

	return gen.generate()
}

// targetGenerator generates target wrappers.
type targetGenerator struct {
	baseGenerator

	wrapName          string // Wrapper constructor name (e.g., "WrapAdd")
	wrapperType       string // Wrapper struct type (e.g., "WrapAddWrapper")
	callHandleType    string // Call handle struct type (e.g., "WrapAddCallHandle")
	returnsType       string // Returns struct type (e.g., "WrapAddReturns")
	funcDecl          *dst.FuncDecl
	astFiles          []*dst.File
	paramNames        []string
	resultTypes       []string
	hasResults        bool
	externalPkgPath   string // Import path for external function type package (e.g., "net/http")
	externalQualifier string // Qualifier for external function type package (e.g., "http")
}

// buildFunctionSignature builds the function signature string.
func (gen *targetGenerator) buildFunctionSignature() string {
	var sig strings.Builder

	sig.WriteString("func(")
	sig.WriteString(gen.formatFieldListTypes(gen.funcDecl.Type.Params))
	sig.WriteString(")")

	// Results
	if gen.funcDecl.Type.Results != nil && len(gen.funcDecl.Type.Results.List) > 0 {
		sig.WriteString(" ")

		resultTypes := gen.formatFieldListTypes(gen.funcDecl.Type.Results)

		if gen.hasMultipleResults() {
			sig.WriteString("(")
			sig.WriteString(resultTypes)
			sig.WriteString(")")
		} else {
			sig.WriteString(resultTypes)
		}
	}

	return sig.String()
}

// buildTargetTemplateData constructs the template data for target wrapper generation.
func (gen *targetGenerator) buildTargetTemplateData() targetTemplateData {
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

	// Build function signature string
	funcSig := gen.buildFunctionSignature()

	// Build params string for Start method
	var paramsStr strings.Builder
	gen.writeFunctionParamsToBuilder(&paramsStr, gen.funcDecl.Type.Params)

	// Build result data using shared helper
	resultData := (&ResultDataBuilder{
		ResultTypes: gen.resultTypes,
		VarPrefix:   "ret",
	}).Build()

	return targetTemplateData{
		baseTemplateData:  base,
		WrapName:          gen.wrapName,
		WrapperType:       gen.wrapperType,
		CallHandleType:    gen.callHandleType,
		ReturnsType:       gen.returnsType,
		FuncSig:           funcSig,
		Params:            paramsStr.String(),
		ParamNames:        strings.Join(gen.paramNames, ", "),
		HasResults:        resultData.HasResults,
		ResultVars:        resultData.Vars,
		ReturnAssignments: resultData.Assignments,
		WaitMethodName:    resultData.WaitMethodName,
		ExpectedParams:    resultData.ExpectedParams,
		MatcherParams:     resultData.MatcherParams,
		ResultChecks:      resultData.Checks,
		ResultFields:      resultData.Fields,
	}
}

// collectAdditionalImports collects all external type imports needed for function signatures.
func (gen *targetGenerator) collectAdditionalImports() []importInfo {
	imports := collectImportsFromFuncDecl(gen.funcDecl, gen.astFiles)

	// For function type wrappers (where pkgPath is empty):
	if gen.pkgPath == "" {
		// If we qualified types from an external package (e.g., http.HandlerFunc),
		// add the external package import
		if gen.externalPkgPath != "" {
			imports = append(imports, importInfo{
				Alias: gen.externalQualifier,
				Path:  gen.externalPkgPath,
			})
		}
		// Return imports as-is for function type wrappers
		return imports
	}

	// For regular function wrappers, filter out the source package import from additional imports.
	// The source package import (if needed) is added separately via NeedsQualifier.
	filtered := make([]importInfo, 0, len(imports))
	for _, imp := range imports {
		if imp.Path != gen.pkgPath {
			filtered = append(filtered, imp)
		}
	}

	return filtered
}

// extractResultTypes extracts result types from a field list.
func (gen *targetGenerator) extractResultTypes(results *dst.FieldList) []string {
	var types []string

	for _, field := range results.List {
		fieldType := gen.typeWithQualifier(field.Type)

		count := len(field.Names)
		if count == 0 {
			count = 1
		}

		for range count {
			types = append(types, fieldType)
		}
	}

	return types
}

// formatFieldListTypes formats a field list into comma-separated type strings.
func (gen *targetGenerator) formatFieldListTypes(fields *dst.FieldList) string {
	if fields == nil {
		return ""
	}

	var types []string

	for _, field := range fields.List {
		fieldType := gen.typeWithQualifier(field.Type)
		count := len(field.Names)

		if count == 0 {
			count = 1
		}

		for range count {
			types = append(types, fieldType)
		}
	}

	return strings.Join(types, ", ")
}

// generate produces the target wrapper code using templates.
func (gen *targetGenerator) generate() (string, error) {
	// Pre-scan to determine what imports are needed
	gen.checkIfQualifierNeeded(gen.funcDecl.Type)

	// Initialize template registry (can't fail with valid hardcoded templates)
	templates, _ := NewTemplateRegistry()

	// Generate using templates
	gen.generateWithTemplates(templates)

	formatted, err := format.Source(gen.bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// generateWithTemplates generates code using templates instead of direct code generation.
func (gen *targetGenerator) generateWithTemplates(templates *TemplateRegistry) {
	data := gen.buildTargetTemplateData()

	// Generate each section using templates
	templates.WriteTargetHeader(&gen.buf, data)
	templates.WriteTargetReturnsStruct(&gen.buf, data)
	templates.WriteTargetConstructor(&gen.buf, data)
	templates.WriteTargetWrapperStruct(&gen.buf, data)
	templates.WriteTargetCallHandleStruct(&gen.buf, data)
	templates.WriteTargetStartMethod(&gen.buf, data)
	templates.WriteTargetWaitMethod(&gen.buf, data)

	// Generate expect methods based on whether function has results
	if gen.hasResults {
		templates.WriteTargetExpectReturns(&gen.buf, data)
	} else {
		templates.WriteTargetExpectCompletes(&gen.buf, data)
	}

	templates.WriteTargetExpectPanic(&gen.buf, data)
}

// hasMultipleResults checks if the function has multiple return values.
func (gen *targetGenerator) hasMultipleResults() bool {
	results := gen.funcDecl.Type.Results
	if results == nil || len(results.List) == 0 {
		return false
	}
	// Multiple fields, or single field with multiple names (e.g., (a, b int))
	return len(results.List) > 1 || len(results.List[0].Names) > 1
}

// writeFunctionParamsToBuilder writes function parameters to a string builder.
// Blank identifiers are replaced with synthetic names like "arg1".
func (gen *targetGenerator) writeFunctionParamsToBuilder(
	builder *strings.Builder,
	params *dst.FieldList,
) {
	if params == nil {
		return
	}

	first := true
	argCounter := 1

	for _, field := range params.List {
		fieldType := gen.typeWithQualifier(field.Type)
		names := getFieldParamNames(field, argCounter)

		for _, name := range names {
			if !first {
				builder.WriteString(", ")
			}

			first = false

			builder.WriteString(name)
			builder.WriteString(" ")
			builder.WriteString(fieldType)

			argCounter++
		}
	}
}

// extractParamNames extracts parameter names from a function type.
// For blank identifiers or unnamed parameters, generates synthetic names like "arg1".
func extractParamNames(funcType *dst.FuncType) []string {
	var names []string

	if funcType.Params == nil {
		return names
	}

	argCounter := 1

	for _, field := range funcType.Params.List {
		if len(field.Names) > 0 {
			for _, name := range field.Names {
				// Blank identifier can't be used as a value, generate synthetic name
				if name.Name == "_" {
					names = append(names, fmt.Sprintf("arg%d", argCounter))
				} else {
					names = append(names, name.Name)
				}

				argCounter++
			}
		} else {
			names = append(names, fmt.Sprintf("arg%d", argCounter))
			argCounter++
		}
	}

	return names
}

// getFieldParamNames returns parameter names for a field, generating synthetic names for blank identifiers.
func getFieldParamNames(field *dst.Field, startIndex int) []string {
	if len(field.Names) == 0 {
		return []string{fmt.Sprintf("arg%d", startIndex)}
	}

	names := make([]string, len(field.Names))

	for i, name := range field.Names {
		if name.Name == "_" {
			names[i] = fmt.Sprintf("arg%d", startIndex+i)
		} else {
			names[i] = name.Name
		}
	}

	return names
}

// newTargetGenerator creates a new target wrapper generator.
// isFunctionType should be true when wrapping a function type (not a function declaration).
func newTargetGenerator(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	funcDecl *dst.FuncDecl,
	isFunctionType bool,
) *targetGenerator {
	var pkgPath, qualifier string

	// For function type wrappers, ALWAYS leave pkgPath empty.
	// Function types use the underlying signature types directly (e.g., func(http.ResponseWriter, *http.Request)),
	// not the function type name itself (e.g., http.HandlerFunc).
	//
	// The AST transformation in generateTargetCodeFromFuncType handles qualifying types from
	// external packages, and collectAdditionalImports picks up those qualified type imports automatically.
	//
	// For LOCAL function types (e.g., visitor.WalkFunc), the types are already qualified (e.g., fs.DirEntry),
	// so no source package import is needed.
	//
	// For EXTERNAL function types (e.g., http.HandlerFunc), the AST transformation adds qualifiers
	// (ResponseWriter → http.ResponseWriter), and collectImportsFromFuncDecl adds the http import.
	if !isFunctionType {
		// For non-function-type cases (regular functions), use the original logic
		// Note: resolvePackageInfo converts all ErrNotPackageReference errors to empty returns,
		// and GetPackageInfo only returns nil or ErrNotPackageReference, so this cannot fail.
		if pkgImportPath != "." || strings.HasSuffix(info.PkgName, "_test") {
			pkgPath, qualifier, _ = resolvePackageInfo(info, pkgLoader)
		}
	}
	// For function types, pkgPath and qualifier remain empty (initialized above)

	// Wrapper type naming: WrapAdd -> WrapAddWrapper, WrapAddCallHandle
	wrapperType := info.ImpName + "Wrapper"
	callHandleType := info.ImpName + "CallHandle"
	returnsType := info.ImpName + "Returns"

	gen := &targetGenerator{
		baseGenerator: newBaseGenerator(
			fset, info.PkgName, info.ImpName, pkgPath, qualifier, funcDecl.Type.TypeParams,
		),
		wrapName:       info.ImpName,
		wrapperType:    wrapperType,
		callHandleType: callHandleType,
		returnsType:    returnsType,
		funcDecl:       funcDecl,
		astFiles:       astFiles,
	}

	// Extract parameter names and result types
	gen.paramNames = extractParamNames(funcDecl.Type)

	gen.hasResults = funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0
	if gen.hasResults {
		gen.resultTypes = gen.extractResultTypes(funcDecl.Type.Results)
		// Target wrappers need reflect for DeepEqual in ExpectReturnsEqual (only when there are results)
		gen.needsReflect = true
	}

	return gen
}
