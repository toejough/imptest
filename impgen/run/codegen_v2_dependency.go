//nolint:varnamelen,wsl_v5,perfsprint,prealloc,nestif,gocognit,intrange,cyclop,funlen
package run // Phase 1 infrastructure - will refine in Phase 2

import (
	"fmt"
	"go/format"
	"go/token"
	go_types "go/types"
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

// checkIfQualifierNeeded determines if we need a package qualifier.
func (gen *v2DependencyGenerator) checkIfQualifierNeeded() {
	_ = forEachInterfaceMethod(
		gen.identifiedInterface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			gen.baseGenerator.checkIfQualifierNeeded(ftype)
		},
	)
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

// generateImplMethodWithTemplate generates a single impl method using templates.
func (gen *v2DependencyGenerator) generateImplMethodWithTemplate(
	templates *TemplateRegistry,
	methodName string,
	ftype *dst.FuncType,
	interfaceType string,
) {
	// Build parameter string and collect param names
	var paramsStr strings.Builder
	var paramNames []string
	first := true

	if ftype.Params != nil {
		for _, field := range ftype.Params.List {
			fieldType := gen.typeWithQualifier(field.Type)

			if len(field.Names) > 0 {
				for _, name := range field.Names {
					if !first {
						paramsStr.WriteString(", ")
					}
					first = false
					paramsStr.WriteString(name.Name)
					paramsStr.WriteString(" ")
					paramsStr.WriteString(fieldType)
					paramNames = append(paramNames, name.Name)
				}
			} else {
				paramName := fmt.Sprintf("arg%d", len(paramNames)+1)
				if !first {
					paramsStr.WriteString(", ")
				}
				first = false
				paramsStr.WriteString(paramName)
				paramsStr.WriteString(" ")
				paramsStr.WriteString(fieldType)
				paramNames = append(paramNames, paramName)
			}
		}
	}

	// Build results string and collect result types
	var resultsStr strings.Builder
	var resultTypes []string

	if ftype.Results != nil && len(ftype.Results.List) > 0 {
		hasMultipleResults := len(ftype.Results.List) > 1 ||
			(len(ftype.Results.List) == 1 && len(ftype.Results.List[0].Names) > 1)

		if hasMultipleResults {
			resultsStr.WriteString(" (")
		} else {
			resultsStr.WriteString(" ")
		}

		first = true
		for _, field := range ftype.Results.List {
			fieldType := gen.typeWithQualifier(field.Type)

			count := len(field.Names)
			if count == 0 {
				count = 1
			}

			for i := 0; i < count; i++ {
				if !first {
					resultsStr.WriteString(", ")
				}
				first = false
				resultsStr.WriteString(fieldType)
				resultTypes = append(resultTypes, fieldType)
			}
		}

		if hasMultipleResults {
			resultsStr.WriteString(")")
		}
	}

	// Check for variadic parameters
	var hasVariadic bool
	var nonVariadicArgs, variadicArg, allArgs strings.Builder

	if ftype.Params != nil && len(ftype.Params.List) > 0 {
		lastField := ftype.Params.List[len(ftype.Params.List)-1]
		_, hasVariadic = lastField.Type.(*dst.Ellipsis)
	}

	if hasVariadic && len(paramNames) > 0 {
		// Build non-variadic args
		for i := 0; i < len(paramNames)-1; i++ {
			if i > 0 {
				nonVariadicArgs.WriteString(", ")
			}
			nonVariadicArgs.WriteString(paramNames[i])
		}
		variadicArg.WriteString(paramNames[len(paramNames)-1])
	} else {
		// Build all args
		for i, name := range paramNames {
			if i > 0 {
				allArgs.WriteString(", ")
			}
			allArgs.WriteString(name)
		}
	}

	// Build result variables
	var resultVars []resultVar
	var returnList strings.Builder

	for i, resultType := range resultTypes {
		resultVars = append(resultVars, resultVar{
			Name:  fmt.Sprintf("result%d", i+1),
			Type:  resultType,
			Index: i,
		})

		if i > 0 {
			returnList.WriteString(", ")
		}
		returnList.WriteString(fmt.Sprintf("result%d", i+1))
	}

	// Build method template data with base fields
	methodData := v2DepMethodTemplateData{
		baseTemplateData: baseTemplateData{
			PkgName:    gen.pkgName,
			PkgImptest: "_imptest",
		},
		MethodName:      methodName,
		InterfaceType:   interfaceType,
		ImplName:        gen.implName,
		Params:          paramsStr.String(),
		Results:         resultsStr.String(),
		HasVariadic:     hasVariadic,
		NonVariadicArgs: nonVariadicArgs.String(),
		VariadicArg:     variadicArg.String(),
		Args:            allArgs.String(),
		ArgNames:        allArgs.String(),
		HasResults:      len(resultTypes) > 0,
		ResultVars:      resultVars,
		ReturnList:      returnList.String(),
		ReturnStatement: fmt.Sprintf("return %s", returnList.String()),
	}

	templates.WriteV2DepImplMethod(&gen.buf, methodData)
}

// generateWithTemplates generates code using templates instead of direct code generation.
func (gen *v2DependencyGenerator) generateWithTemplates(templates *TemplateRegistry) {
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
	}

	// Generate each section using templates
	templates.WriteV2DepHeader(&gen.buf, data)
	templates.WriteV2DepMockStruct(&gen.buf, data)
	templates.WriteV2DepInterfaceMethod(&gen.buf, data)
	templates.WriteV2DepConstructor(&gen.buf, data)
	templates.WriteV2DepImplStruct(&gen.buf, data)

	// Generate implementation methods for each interface method
	_ = forEachInterfaceMethod(
		gen.identifiedInterface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			gen.generateImplMethodWithTemplate(templates, methodName, ftype, interfaceType)
		},
	)
}

// generateV2DependencyCode generates v2-style dependency mock code for an interface.
func generateV2DependencyCode(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (string, error) {
	gen, err := newV2DependencyGenerator(astFiles, info, fset, typesInfo, pkgImportPath, pkgLoader, ifaceWithDetails)
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
	typesInfo *go_types.Info,
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
			fset, info.pkgName, info.impName, pkgPath, qualifier, ifaceWithDetails.typeParams, typesInfo,
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
