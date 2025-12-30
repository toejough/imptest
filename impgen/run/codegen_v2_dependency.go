//nolint:varnamelen,wsl_v5,perfsprint,prealloc,nestif,intrange,cyclop,funlen
package run

import (
	"fmt"
	"go/format"
	"go/token"
	"sort"
	"strings"

	"github.com/dave/dst"
)

// v2DependencyGenerator generates v2-style dependency mocks.
type v2DependencyGenerator struct {
	baseGenerator

	mockName            string // Constructor function name (e.g., "MockOps")
	mockTypeName        string // Struct type name (e.g., "OpsMock")
	interfaceName       string
	implName            string
	astFiles            []*dst.File
	pkgImportPath       string
	pkgLoader           PackageLoader
	methodNames         []string
	identifiedInterface *dst.InterfaceType
}

func (gen *v2DependencyGenerator) buildMethodTemplateData(
	methodName string,
	ftype *dst.FuncType,
	interfaceType string,
) v2DepMethodTemplateData {
	// Build parameter string and collect param names
	paramsStr, paramNames := gen.buildParamStrings(ftype)

	// Build results string and collect result types
	resultsStr, resultTypes := gen.buildResultStrings(ftype)

	// Check for variadic parameters and build argument strings
	variadicResult := gen.buildVariadicArgs(ftype, paramNames)

	// Build result variables
	resultVars, returnList := buildResultVars(resultTypes)

	// Extract parameter fields for type-safe args
	paramFields := gen.buildParamFields(ftype)

	// Build method template data with base fields
	return v2DepMethodTemplateData{
		baseTemplateData: baseTemplateData{
			PkgName:        gen.pkgName,
			PkgImptest:     "_imptest",
			PkgTime:        pkgTime,
			TypeParamsDecl: gen.formatTypeParamsDecl(),
			TypeParamsUse:  gen.formatTypeParamsUse(),
		},
		MethodName:      methodName,
		InterfaceType:   interfaceType,
		ImplName:        gen.implName,
		Params:          paramsStr,
		Results:         resultsStr,
		HasVariadic:     variadicResult.hasVariadic,
		NonVariadicArgs: variadicResult.nonVariadicArgs,
		VariadicArg:     variadicResult.variadicArg,
		Args:            variadicResult.allArgs,
		ArgNames:        variadicResult.allArgs,
		HasResults:      len(resultTypes) > 0,
		ResultVars:      resultVars,
		ReturnList:      returnList,
		ReturnStatement: fmt.Sprintf("return %s", returnList),
		ParamFields:     paramFields,
		HasParams:       len(paramFields) > 0,
		ArgsTypeName:    fmt.Sprintf("%s%sArgs", gen.mockTypeName, methodName),
		CallTypeName:    fmt.Sprintf("%s%sCall", gen.mockTypeName, methodName),
		MethodTypeName:  fmt.Sprintf("%s%sMethod", gen.mockTypeName, methodName),
		TypedParams:     paramsStr,
	}
}

// buildParamFields extracts parameter fields for type-safe args.
func (gen *v2DependencyGenerator) buildParamFields(ftype *dst.FuncType) []paramField {
	paramInfos := extractParams(gen.fset, ftype)
	var paramFields []paramField

	for _, pinfo := range paramInfos {
		// Use actual parameter name if present, otherwise generate A1, A2, A3 style names
		// to match the DependencyArgs pattern
		fieldName := pinfo.Name
		if strings.HasPrefix(pinfo.Name, "param") {
			// Unnamed parameter - convert "param0" -> "A1", "param1" -> "A2", etc.
			fieldName = fmt.Sprintf("A%d", pinfo.Index+1)
		} else {
			// Capitalize first letter for exported field
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

// buildMethodTemplateData builds template data for a single method.
// buildParamStrings builds the parameter string and collects parameter names.
func (gen *v2DependencyGenerator) buildParamStrings(
	ftype *dst.FuncType,
) (paramsStr string, paramNames []string) {
	var builder strings.Builder
	first := true

	if ftype.Params != nil {
		for _, field := range ftype.Params.List {
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
					paramNames = append(paramNames, name.Name)
				}
			} else {
				paramName := fmt.Sprintf("arg%d", len(paramNames)+1)
				if !first {
					builder.WriteString(", ")
				}
				first = false
				builder.WriteString(paramName)
				builder.WriteString(" ")
				builder.WriteString(fieldType)
				paramNames = append(paramNames, paramName)
			}
		}
	}

	return builder.String(), paramNames
}

// buildResultStrings builds the result string and collects result types.
func (gen *v2DependencyGenerator) buildResultStrings(
	ftype *dst.FuncType,
) (resultsStr string, resultTypes []string) {
	var builder strings.Builder

	if ftype.Results != nil && len(ftype.Results.List) > 0 {
		hasMultipleResults := len(ftype.Results.List) > 1 ||
			(len(ftype.Results.List) == 1 && len(ftype.Results.List[0].Names) > 1)

		if hasMultipleResults {
			builder.WriteString(" (")
		} else {
			builder.WriteString(" ")
		}

		first := true
		for _, field := range ftype.Results.List {
			fieldType := gen.typeWithQualifier(field.Type)

			count := len(field.Names)
			if count == 0 {
				count = 1
			}

			for i := 0; i < count; i++ {
				if !first {
					builder.WriteString(", ")
				}
				first = false
				builder.WriteString(fieldType)
				resultTypes = append(resultTypes, fieldType)
			}
		}

		if hasMultipleResults {
			builder.WriteString(")")
		}
	}

	return builder.String(), resultTypes
}

// buildVariadicArgs checks for variadic parameters and builds argument strings.
func (gen *v2DependencyGenerator) buildVariadicArgs(
	ftype *dst.FuncType,
	paramNames []string,
) variadicArgsResult {
	var hasVariadic bool
	var nonVariadicArgs, variadicArg, allArgs strings.Builder

	if ftype.Params != nil && len(ftype.Params.List) > 0 {
		lastField := ftype.Params.List[len(ftype.Params.List)-1]
		_, hasVariadic = lastField.Type.(*dst.Ellipsis)
	}

	if hasVariadic && len(paramNames) > 0 {
		// Build non-variadic args (all params except the last one)
		for i := 0; i < len(paramNames)-1; i++ {
			if i > 0 {
				nonVariadicArgs.WriteString(", ")
			}
			nonVariadicArgs.WriteString(paramNames[i])
		}
		variadicArg.WriteString(paramNames[len(paramNames)-1])

		// For variadic methods, allArgs is not used in the template
		// (the template uses NonVariadicArgs and VariadicArg to build args manually)
		// But we still populate it for consistency and potential future use
		for i, name := range paramNames {
			if i > 0 {
				allArgs.WriteString(", ")
			}
			allArgs.WriteString(name)
		}
	} else {
		// Build all args (non-variadic case)
		for i, name := range paramNames {
			if i > 0 {
				allArgs.WriteString(", ")
			}
			allArgs.WriteString(name)
		}
	}

	return variadicArgsResult{
		hasVariadic:     hasVariadic,
		nonVariadicArgs: nonVariadicArgs.String(),
		variadicArg:     variadicArg.String(),
		allArgs:         allArgs.String(),
	}
}

// checkIfQualifierNeeded determines if we need a package qualifier.
func (gen *v2DependencyGenerator) checkIfQualifierNeeded() {
	_ = forEachInterfaceMethod(
		gen.identifiedInterface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			gen.baseGenerator.checkIfQualifierNeeded(ftype)
		},
	)
}

// collectAdditionalImports collects all external type imports needed for interface method signatures.
//
//nolint:cyclop // Import collection requires iteration over interface methods and their parameters/results
func (gen *v2DependencyGenerator) collectAdditionalImports() []importInfo {
	if len(gen.astFiles) == 0 {
		return nil
	}

	// Get source imports from the first AST file
	var sourceImports []*dst.ImportSpec

	for _, file := range gen.astFiles {
		if len(file.Imports) > 0 {
			sourceImports = file.Imports
			break
		}
	}

	allImports := make(map[string]importInfo) // Deduplicate by path

	// Iterate over all interface methods to collect imports from their signatures
	_ = forEachInterfaceMethod(
		gen.identifiedInterface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			// Collect from parameters
			if ftype.Params != nil {
				for _, field := range ftype.Params.List {
					imports := collectExternalImports(field.Type, sourceImports)
					for _, imp := range imports {
						allImports[imp.Path] = imp
					}
				}
			}

			// Collect from return types
			if ftype.Results != nil {
				for _, field := range ftype.Results.List {
					imports := collectExternalImports(field.Type, sourceImports)
					for _, imp := range imports {
						allImports[imp.Path] = imp
					}
				}
			}
		},
	)

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

// generate produces the v2 dependency mock code using templates.
func (gen *v2DependencyGenerator) generate() (string, error) {
	// Pre-scan to determine what imports are needed
	gen.checkIfQualifierNeeded()

	// If we have an interface from an external package, we need the qualifier
	if gen.interfaceName != "" && gen.qualifier != "" && gen.pkgPath != "" {
		gen.needsQualifier = true
	}

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
func (gen *v2DependencyGenerator) generateWithTemplates(templates *TemplateRegistry) {
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

	// Construct the interface type with qualifier if needed
	interfaceType := gen.interfaceName
	if gen.qualifier != "" && gen.needsQualifier {
		qualifierToUse := gen.qualifier
		// Check if this is a stdlib package that needs aliasing due to a name conflict
		if gen.pkgPath != "" && !strings.Contains(gen.pkgPath, "/") && gen.pkgPath == gen.qualifier {
			qualifierToUse = "_" + gen.qualifier
		}
		interfaceType = qualifierToUse + "." + gen.interfaceName
	}
	// Add type parameters to interface type if present
	if gen.formatTypeParamsUse() != "" {
		interfaceType += gen.formatTypeParamsUse()
	}

	// Collect method data for all methods first (needed for typed wrappers)
	var methods []v2DepMethodTemplateData
	_ = forEachInterfaceMethod(
		gen.identifiedInterface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			methodData := gen.buildMethodTemplateData(methodName, ftype, interfaceType)
			methods = append(methods, methodData)
		},
	)

	// Build v2 dependency template data
	baseName := strings.TrimPrefix(gen.mockName, "Mock")
	data := v2DepTemplateData{
		baseTemplateData: base,
		MockName:         gen.mockName,
		MockTypeName:     gen.mockTypeName,
		BaseName:         baseName,
		InterfaceName:    gen.interfaceName,
		InterfaceType:    interfaceType,
		ImplName:         gen.implName,
		MethodNames:      gen.methodNames,
		Methods:          methods,
	}

	// Generate each section using templates
	templates.WriteV2DepHeader(&gen.buf, data)
	templates.WriteV2DepMockStruct(&gen.buf, data)
	templates.WriteV2DepInterfaceMethod(&gen.buf, data)
	templates.WriteV2DepConstructor(&gen.buf, data)
	templates.WriteV2DepImplStruct(&gen.buf, data)

	// Generate implementation methods and type-safe wrappers for each interface method
	for _, methodData := range methods {
		templates.WriteV2DepImplMethod(&gen.buf, methodData)
		templates.WriteV2DepArgsStruct(&gen.buf, methodData)
		templates.WriteV2DepCallWrapper(&gen.buf, methodData)
		templates.WriteV2DepMethodWrapper(&gen.buf, methodData)
	}
}

// variadicArgsResult holds the result of buildVariadicArgs.
type variadicArgsResult struct {
	hasVariadic     bool
	nonVariadicArgs string
	variadicArg     string
	allArgs         string
}

// buildResultVars builds result variables and return list from result types.
func buildResultVars(resultTypes []string) (resultVars []resultVar, returnList string) {
	var returnListBuilder strings.Builder

	for i, resultType := range resultTypes {
		resultVars = append(resultVars, resultVar{
			Name:  fmt.Sprintf("result%d", i+1),
			Type:  resultType,
			Index: i,
		})

		if i > 0 {
			returnListBuilder.WriteString(", ")
		}
		returnListBuilder.WriteString(fmt.Sprintf("result%d", i+1))
	}

	return resultVars, returnListBuilder.String()
}

// generateV2DependencyCode generates v2-style dependency mock code for an interface.
func generateV2DependencyCode(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (string, error) {
	gen, err := newV2DependencyGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, ifaceWithDetails)
	if err != nil {
		return "", err
	}

	return gen.generate()
}

// newV2DependencyGenerator creates a new v2 dependency mock generator.
func newV2DependencyGenerator(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (*v2DependencyGenerator, error) {
	var (
		pkgPath, qualifier string
		err                error
	)

	// Get package info for external interfaces OR when in a _test package
	if pkgImportPath != "." || strings.HasSuffix(info.pkgName, "_test") {
		pkgPath, qualifier, err = resolvePackageInfo(info, pkgLoader)
		if err != nil {
			return nil, fmt.Errorf("failed to get interface package info: %w", err)
		}
	}

	// Convert MockXxx -> XxxMock for the struct type name
	// This avoids naming conflict between the constructor function and the struct type
	// Note: When using --name with a value ending in "Mock" (e.g., CustomOpsMock),
	//       you'll get MockMock in the struct name to avoid conflicts.
	//       Recommend using --name without the Mock suffix (e.g., --name CustomOps)
	mockTypeName := strings.TrimPrefix(info.impName, "Mock") + "Mock"

	gen := &v2DependencyGenerator{
		baseGenerator: newBaseGenerator(
			fset, info.pkgName, info.impName, pkgPath, qualifier, ifaceWithDetails.typeParams,
		),
		mockName:            info.impName,
		mockTypeName:        mockTypeName,
		interfaceName:       info.localInterfaceName,
		implName:            strings.ToLower(string(info.impName[0])) + info.impName[1:] + "Impl",
		astFiles:            astFiles,
		pkgImportPath:       pkgImportPath,
		pkgLoader:           pkgLoader,
		identifiedInterface: ifaceWithDetails.iface,
	}

	// Collect method names
	methodNames, err := interfaceCollectMethodNames(ifaceWithDetails.iface, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return nil, err
	}
	gen.methodNames = methodNames

	return gen, nil
}
