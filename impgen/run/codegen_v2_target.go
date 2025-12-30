//nolint:varnamelen,wsl_v5,staticcheck,cyclop,gocognit,intrange,funlen
package run

import (
	"fmt"
	"go/format"
	"go/token"
	"strings"

	"github.com/dave/dst"
)

// v2TargetGenerator generates v2-style target wrappers.
type v2TargetGenerator struct {
	baseGenerator

	wrapName    string // Wrapper constructor name (e.g., "WrapAdd")
	wrapperType string // Wrapper struct type (e.g., "WrapAddWrapper")
	returnsType string // Returns struct type (e.g., "WrapAddReturns")
	funcDecl    *dst.FuncDecl
	astFiles    []*dst.File
	paramNames  []string
	resultTypes []string
	hasResults  bool
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

// collectAdditionalImports collects all external type imports needed for function signatures.
func (gen *v2TargetGenerator) collectAdditionalImports() []importInfo {
	return collectImportsFromFuncDecl(gen.funcDecl, gen.astFiles)
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

	// Build comma-separated param names
	var paramNamesStr strings.Builder
	for i, name := range gen.paramNames {
		if i > 0 {
			paramNamesStr.WriteString(", ")
		}
		paramNamesStr.WriteString(name)
	}

	// Build result vars and return assignments (using zero-indexed Result0, Result1 to match v1 pattern)
	var resultVarsStr, returnAssignmentsStr strings.Builder
	if gen.hasResults {
		for i := range gen.resultTypes {
			if i > 0 {
				resultVarsStr.WriteString(", ")
				returnAssignmentsStr.WriteString(", ")
			}
			fmt.Fprintf(&resultVarsStr, "ret%d", i)
			fmt.Fprintf(&returnAssignmentsStr, "Result%d: ret%d", i, i)
		}
	}

	// Determine wait method name
	waitMethodName := "WaitForCompletion"
	if gen.hasResults {
		waitMethodName = "WaitForResponse"
	}

	// Build expected params for ExpectReturnsEqual and matcher params for ExpectReturnsMatch
	var expectedParamsStr, matcherParamsStr strings.Builder
	var resultChecks []resultCheck
	if gen.hasResults {
		for i, resultType := range gen.resultTypes {
			if i > 0 {
				expectedParamsStr.WriteString(", ")
				matcherParamsStr.WriteString(", ")
			}
			fmt.Fprintf(&expectedParamsStr, "v%d %s", i, resultType)
			fmt.Fprintf(&matcherParamsStr, "v%d any", i)
			resultChecks = append(resultChecks, resultCheck{
				Field:    fmt.Sprintf("Result%d", i),
				Expected: fmt.Sprintf("v%d", i),
				Index:    i,
			})
		}
	}

	// Build result fields for Returns struct
	var resultFields []resultField
	if gen.hasResults {
		for i, resultType := range gen.resultTypes {
			resultFields = append(resultFields, resultField{
				Name: fmt.Sprintf("Result%d", i),
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
		MatcherParams:     matcherParamsStr.String(),
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
	argCounter := 1
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
			builder.WriteString(fmt.Sprintf("arg%d ", argCounter))
			builder.WriteString(fieldType)
			argCounter++
		}
	}
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
	pkgImportPath string,
	pkgLoader PackageLoader,
	funcDecl *dst.FuncDecl,
) (string, error) {
	gen, err := newV2TargetGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, funcDecl)
	if err != nil {
		return "", err
	}

	return gen.generate()
}

// generateV2TargetCodeFromFuncType generates v2-style target wrapper code for a function type.
// It creates a synthetic function declaration from the function type and delegates to generateV2TargetCode.
func generateV2TargetCodeFromFuncType(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	funcTypeDetails funcTypeWithDetails,
) (string, error) {
	// Create a synthetic function declaration from the function type
	// For a type like: type WalkFunc func(path string, info string) error
	// We create: func WalkFunc(path string, info string) error { ... }
	funcDecl := &dst.FuncDecl{
		Name: &dst.Ident{Name: funcTypeDetails.typeName},
		Type: funcTypeDetails.funcType,
	}

	// If the function type has type parameters, attach them to the FuncType
	if funcTypeDetails.typeParams != nil {
		funcDecl.Type.TypeParams = funcTypeDetails.typeParams
	}

	return generateV2TargetCode(astFiles, info, fset, pkgImportPath, pkgLoader, funcDecl)
}

// newV2TargetGenerator creates a new v2 target wrapper generator.
func newV2TargetGenerator(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
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
			fset, info.pkgName, info.impName, pkgPath, qualifier, funcDecl.Type.TypeParams,
		),
		wrapName:    info.impName,
		wrapperType: wrapperType,
		returnsType: returnsType,
		funcDecl:    funcDecl,
		astFiles:    astFiles,
	}

	// Extract parameter names and result types
	gen.paramNames = extractParamNames(funcDecl.Type)
	gen.hasResults = funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0
	if gen.hasResults {
		gen.resultTypes = gen.extractResultTypes(funcDecl.Type.Results)
		// V2 target wrappers need reflect for DeepEqual in ExpectReturnsEqual (only when there are results)
		gen.needsReflect = true
	}

	return gen, nil
}
