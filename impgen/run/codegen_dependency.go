//nolint:varnamelen,wsl_v5,perfsprint,prealloc,nestif,funlen
package run

import (
	"fmt"
	"go/format"
	"go/token"
	"strings"

	"github.com/dave/dst"
)

// dependencyGenerator generates dependency mocks.
type dependencyGenerator struct {
	baseGenerator

	mockName            string // Constructor function name (e.g., "MockOps")
	mockTypeName        string // Struct type name (e.g., "OpsMock")
	interfaceName       string
	implName            string
	astFiles            []*dst.File
	pkgImportPath       string
	pkgLoader           PackageLoader
	methodNames         []string
	identifiedInterface ifaceWithDetails // full interface details including source imports
}

func (gen *dependencyGenerator) buildMethodTemplateData(
	methodName string,
	ftype *dst.FuncType,
	interfaceType string,
) depMethodTemplateData {
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

	// Build typed return parameters for type-safe InjectReturnValues
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
		MethodName:        methodName,
		InterfaceType:     interfaceType,
		ImplName:          gen.implName,
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
		ReturnStatement:   fmt.Sprintf("return %s", returnList),
		ParamFields:       paramFields,
		HasParams:         len(paramFields) > 0,
		ArgsTypeName:      fmt.Sprintf("%s%sArgs", gen.mockTypeName, methodName),
		CallTypeName:      fmt.Sprintf("%s%sCall", gen.mockTypeName, methodName),
		MethodTypeName:    fmt.Sprintf("%s%sMethod", gen.mockTypeName, methodName),
		TypedParams:       paramsStr,
		TypedReturnParams: typedReturnParams,
		ReturnParamNames:  returnParamNames,
	}
}

// buildParamFields extracts parameter fields for type-safe args.
func (gen *dependencyGenerator) buildParamFields(ftype *dst.FuncType) []paramField {
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

// checkIfQualifierNeeded determines if we need a package qualifier.
func (gen *dependencyGenerator) checkIfQualifierNeeded() {
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			gen.baseGenerator.checkIfQualifierNeeded(ftype)
		},
	)
}

// collectAdditionalImports collects imports needed for interface method signatures.
func (gen *dependencyGenerator) collectAdditionalImports() []importInfo {
	// Use the source imports from the interface's file (tracked during parsing)
	sourceImports := gen.identifiedInterface.sourceImports

	// Fallback: if interface's file has no imports, collect from all files
	if len(sourceImports) == 0 {
		for _, file := range gen.astFiles {
			if len(file.Imports) > 0 {
				sourceImports = append(sourceImports, file.Imports...)
			}
		}
	}

	return gen.collectAdditionalImportsFromInterface(
		gen.identifiedInterface.iface,
		gen.astFiles,
		gen.pkgImportPath,
		gen.pkgLoader,
		sourceImports,
	)
}

// generate produces the dependency mock code using templates.
func (gen *dependencyGenerator) generate() (string, error) {
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
func (gen *dependencyGenerator) generateWithTemplates(templates *TemplateRegistry) {
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
	var methods []depMethodTemplateData
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			methodData := gen.buildMethodTemplateData(methodName, ftype, interfaceType)
			methods = append(methods, methodData)
		},
	)

	// Build dependency template data
	baseName := strings.TrimPrefix(gen.mockName, "Mock")
	data := depTemplateData{
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
	templates.WriteDepHeader(&gen.buf, data)
	templates.WriteDepMockStruct(&gen.buf, data)
	templates.WriteDepInterfaceMethod(&gen.buf, data)
	templates.WriteDepConstructor(&gen.buf, data)
	templates.WriteDepImplStruct(&gen.buf, data)

	// Generate implementation methods and type-safe wrappers for each interface method
	for _, methodData := range methods {
		templates.WriteDepImplMethod(&gen.buf, methodData)
		templates.WriteDepArgsStruct(&gen.buf, methodData)
		templates.WriteDepCallWrapper(&gen.buf, methodData)
		templates.WriteDepMethodWrapper(&gen.buf, methodData)
	}
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

// buildTypedReturnParams builds typed return parameters for InjectReturnValues method.
// Returns (typedParams, paramNames) like ("result0 int, result1 error", "result0, result1").
func buildTypedReturnParams(resultTypes []string) (typedParams, paramNames string) {
	if len(resultTypes) == 0 {
		return "", ""
	}

	var typedBuilder, namesBuilder strings.Builder

	for i, resultType := range resultTypes {
		paramName := fmt.Sprintf("result%d", i)

		if i > 0 {
			typedBuilder.WriteString(", ")
			namesBuilder.WriteString(", ")
		}

		typedBuilder.WriteString(paramName)
		typedBuilder.WriteString(" ")
		typedBuilder.WriteString(resultType)

		namesBuilder.WriteString(paramName)
	}

	return typedBuilder.String(), namesBuilder.String()
}

// generateDependencyCode generates dependency mock code for an interface.
func generateDependencyCode(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (string, error) {
	gen, err := newDependencyGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, ifaceWithDetails)
	if err != nil {
		return "", err
	}

	return gen.generate()
}

// newDependencyGenerator creates a new dependency mock generator.
func newDependencyGenerator(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (*dependencyGenerator, error) {
	var (
		pkgPath, qualifier string
		err                error
	)

	// Get package info for external interfaces OR when in a _test package
	if pkgImportPath != "." {
		// Symbol found in external package (e.g., via dot import or qualified name)
		// For qualified names (e.g., "basic.Ops"), resolve package info normally
		// For unqualified names from dot imports (e.g., "Storage"), use pkgImportPath directly
		if strings.Contains(info.interfaceName, ".") {
			// Qualified name - use normal resolution
			pkgPath, qualifier, err = resolvePackageInfo(info, pkgLoader)
			if err != nil {
				return nil, fmt.Errorf("failed to get interface package info: %w", err)
			}
		} else {
			// Unqualified name - must be from dot import, use pkgImportPath directly
			pkgPath = pkgImportPath
			parts := strings.Split(pkgImportPath, "/")
			qualifier = parts[len(parts)-1]
		}
	} else if strings.HasSuffix(info.pkgName, "_test") {
		// In test package, interface is from non-test version of same package
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

	gen := &dependencyGenerator{
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
		identifiedInterface: ifaceWithDetails,
	}

	// Collect method names
	methodNames, err := interfaceCollectMethodNames(ifaceWithDetails.iface, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return nil, err
	}
	gen.methodNames = methodNames

	return gen, nil
}
