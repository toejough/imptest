package run

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

	// Build method template data
	methodData := v2DepMethodTemplateData{
		MethodName:      methodName,
		InterfaceType:   interfaceType,
		ImplName:        gen.implName,
		Params:          paramsStr.String(),
		Results:         resultsStr.String(),
		HasVariadic:     hasVariadic,
		NonVariadicArgs: nonVariadicArgs.String(),
		VariadicArg:     variadicArg.String(),
		Args:            allArgs.String(),
		HasResults:      len(resultTypes) > 0,
		ResultVars:      resultVars,
		ReturnList:      returnList.String(),
	}

	templates.WriteV2DepImplMethod(&gen.buf, methodData)
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

// generateHeader writes the package declaration and imports.
func (gen *v2DependencyGenerator) generateHeader() {
	gen.pf("// Code generated by impgen. DO NOT EDIT.\n\n")
	gen.pf("package %s\n\n", gen.pkgName)

	gen.pf("import (\n")
	gen.pf("\t\"github.com/toejough/imptest/imptest\"\n")

	if gen.needsQualifier && gen.pkgPath != "" {
		gen.pf("\t%s \"%s\"\n", gen.qualifier, gen.pkgPath)
	}

	gen.pf(")\n\n")
}

// generateMockStruct writes the mock struct with DependencyMethod fields.
func (gen *v2DependencyGenerator) generateMockStruct() {
	gen.pf("// %s is the mock implementation returned by %s.\n", gen.mockTypeName, gen.mockName)
	gen.pf("type %s struct {\n", gen.mockTypeName)
	gen.pf("\timp *imptest.Imp\n")

	for _, methodName := range gen.methodNames {
		gen.pf("\t%s *imptest.DependencyMethod\n", methodName)
	}

	gen.pf("}\n\n")
}

// generateConstructor writes the constructor function.
func (gen *v2DependencyGenerator) generateConstructor() {
	// Remove "Mock" prefix to get base name
	baseName := strings.TrimPrefix(gen.mockName, "Mock")

	gen.pf("// %s creates a new mock for the %s interface.\n", gen.mockName, baseName)
	gen.pf("func %s(testReporter imptest.TestReporter) *%s {\n", gen.mockName, gen.mockTypeName)
	gen.pf("\timp, ok := testReporter.(*imptest.Imp)\n")
	gen.pf("\tif !ok {\n")
	gen.pf("\t\timp = imptest.NewImp(testReporter)\n")
	gen.pf("\t}\n\n")
	gen.pf("\treturn &%s{\n", gen.mockTypeName)
	gen.pf("\t\timp: imp,\n")

	for _, methodName := range gen.methodNames {
		gen.pf("\t\t%s: imptest.NewDependencyMethod(imp, %q),\n", methodName, methodName)
	}

	gen.pf("\t}\n")
	gen.pf("}\n\n")
}

// generateInterfaceMethod writes the Interface() method.
func (gen *v2DependencyGenerator) generateInterfaceMethod() {
	// Construct the interface type with qualifier if needed
	interfaceType := gen.interfaceName
	if gen.qualifier != "" && gen.needsQualifier {
		// Check if this is a stdlib package that needs aliasing due to a name conflict
		qualifierToUse := gen.qualifier
		if gen.pkgPath != "" && !strings.Contains(gen.pkgPath, "/") && gen.pkgPath == gen.qualifier {
			// This is a stdlib package with a name conflict - use the alias
			qualifierToUse = "_" + gen.qualifier
		}
		interfaceType = qualifierToUse + "." + gen.interfaceName
	}

	gen.pf("// Interface returns the mock as a %s interface implementation.\n", interfaceType)
	gen.pf("func (m *%s) Interface() %s {\n", gen.mockTypeName, interfaceType)
	gen.pf("\treturn &%s{mock: m}\n", gen.implName)
	gen.pf("}\n\n")
}

// generateImplStruct writes the implementation struct.
func (gen *v2DependencyGenerator) generateImplStruct() {
	gen.pf("// %s implements the %s interface by forwarding to the mock.\n", gen.implName, gen.interfaceName)
	gen.pf("type %s struct {\n", gen.implName)
	gen.pf("\tmock *%s\n", gen.mockTypeName)
	gen.pf("}\n\n")
}

// generateImplMethods writes the interface implementation methods.
func (gen *v2DependencyGenerator) generateImplMethods() {
	_ = forEachInterfaceMethod(
		gen.identifiedInterface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			gen.generateImplMethod(methodName, ftype)
		},
	)
}

// generateImplMethod writes a single interface method implementation.
func (gen *v2DependencyGenerator) generateImplMethod(methodName string, ftype *dst.FuncType) {
	// Construct the interface type with qualifier if needed
	interfaceType := gen.interfaceName
	if gen.qualifier != "" && gen.needsQualifier {
		// Check if this is a stdlib package that needs aliasing due to a name conflict
		qualifierToUse := gen.qualifier
		if gen.pkgPath != "" && !strings.Contains(gen.pkgPath, "/") && gen.pkgPath == gen.qualifier {
			// This is a stdlib package with a name conflict - use the alias
			qualifierToUse = "_" + gen.qualifier
		}
		interfaceType = qualifierToUse + "." + gen.interfaceName
	}

	// Method signature
	gen.pf("// %s implements %s.%s by sending a call to the Controller and blocking on response.\n",
		methodName, interfaceType, methodName)
	gen.pf("func (impl *%s) %s(", gen.implName, methodName)

	// Parameters
	var paramNames []string
	if ftype.Params != nil && len(ftype.Params.List) > 0 {
		paramNames = gen.writeFunctionParams(ftype.Params)
	}

	gen.pf(")")

	// Results
	var resultTypes []string
	if ftype.Results != nil && len(ftype.Results.List) > 0 {
		gen.pf(" ")
		hasMultipleResults := len(ftype.Results.List) > 1 ||
			(len(ftype.Results.List) == 1 && len(ftype.Results.List[0].Names) > 1)

		if hasMultipleResults {
			gen.pf("(")
		}

		resultTypes = gen.writeFunctionResults(ftype.Results)

		if hasMultipleResults {
			gen.pf(")")
		}
	}

	gen.pf(" {\n")

	// Method body
	gen.pf("\tresponseChan := make(chan imptest.GenericResponse, 1)\n\n")

	// Check if last parameter is variadic
	var hasVariadic bool
	if ftype.Params != nil && len(ftype.Params.List) > 0 {
		lastField := ftype.Params.List[len(ftype.Params.List)-1]
		_, hasVariadic = lastField.Type.(*dst.Ellipsis)
	}

	// Build args - unpack variadic if present
	if hasVariadic && len(paramNames) > 0 {
		gen.pf("\targs := []any{")
		// Add all non-variadic params
		for i := 0; i < len(paramNames)-1; i++ {
			if i > 0 {
				gen.pf(", ")
			}
			gen.pf("%s", paramNames[i])
		}
		gen.pf("}\n")
		// Unpack variadic param
		variadicParam := paramNames[len(paramNames)-1]
		gen.pf("\tfor _, v := range %s {\n", variadicParam)
		gen.pf("\t\targs = append(args, v)\n")
		gen.pf("\t}\n\n")

		gen.pf("\tcall := &imptest.GenericCall{\n")
		gen.pf("\t\tMethodName:   %q,\n", methodName)
		gen.pf("\t\tArgs:         args,\n")
	} else {
		gen.pf("\tcall := &imptest.GenericCall{\n")
		gen.pf("\t\tMethodName:   %q,\n", methodName)
		gen.pf("\t\tArgs:         []any{")

		// Build args list
		for i, paramName := range paramNames {
			if i > 0 {
				gen.pf(", ")
			}
			gen.pf("%s", paramName)
		}

		gen.pf("},\n")
	}

	gen.pf("\t\tResponseChan: responseChan,\n")
	gen.pf("\t}\n\n")

	// Send call to Controller
	gen.pf("\t// Send call to Controller\n")
	gen.pf("\timpl.mock.imp.CallChan <- call\n\n")
	gen.pf("\t// Block waiting for test to inject response\n")
	gen.pf("\tresp := <-responseChan\n\n")
	gen.pf("\tif resp.Type == \"panic\" {\n")
	gen.pf("\t\tpanic(resp.PanicValue)\n")
	gen.pf("\t}\n\n")

	// Extract return values
	if len(resultTypes) > 0 {
		gen.generateReturnExtraction(resultTypes)
	}

	gen.pf("}\n\n")
}

// writeFunctionParams writes function parameters and returns their names.
func (gen *v2DependencyGenerator) writeFunctionParams(params *dst.FieldList) []string {
	var names []string
	first := true

	for _, field := range params.List {
		fieldType := gen.typeWithQualifier(field.Type)

		if len(field.Names) > 0 {
			// Named parameters
			for _, name := range field.Names {
				if !first {
					gen.pf(", ")
				}
				first = false
				gen.pf("%s %s", name.Name, fieldType)
				names = append(names, name.Name)
			}
		} else {
			// Unnamed parameter - generate a name
			paramName := fmt.Sprintf("arg%d", len(names)+1)
			if !first {
				gen.pf(", ")
			}
			first = false
			gen.pf("%s %s", paramName, fieldType)
			names = append(names, paramName)
		}
	}

	return names
}

// writeFunctionResults writes function results and returns their types.
func (gen *v2DependencyGenerator) writeFunctionResults(results *dst.FieldList) []string {
	var types []string
	first := true

	for _, field := range results.List {
		fieldType := gen.typeWithQualifier(field.Type)

		count := len(field.Names)
		if count == 0 {
			count = 1
		}

		for i := 0; i < count; i++ {
			if !first {
				gen.pf(", ")
			}
			first = false
			gen.pf("%s", fieldType)
			types = append(types, fieldType)
		}
	}

	return types
}

// generateReturnExtraction writes code to extract return values from the response.
func (gen *v2DependencyGenerator) generateReturnExtraction(resultTypes []string) {
	// Declare result variables
	for i, resultType := range resultTypes {
		varName := fmt.Sprintf("result%d", i+1)
		gen.pf("\tvar %s %s\n", varName, resultType)
	}

	gen.pf("\n")

	// Extract values from response
	for i, resultType := range resultTypes {
		varName := fmt.Sprintf("result%d", i+1)
		gen.pf("\tif len(resp.ReturnValues) > %d {\n", i)
		gen.pf("\t\tif value, ok := resp.ReturnValues[%d].(%s); ok {\n", i, resultType)
		gen.pf("\t\t\t%s = value\n", varName)
		gen.pf("\t\t}\n")
		gen.pf("\t}\n\n")
	}

	// Return statement
	gen.pf("\treturn ")
	for i := range resultTypes {
		if i > 0 {
			gen.pf(", ")
		}
		gen.pf("result%d", i+1)
	}
	gen.pf("\n")
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
