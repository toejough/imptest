//nolint:intrange
package generate

import (
	"fmt"
	"go/format"
	"go/token"
	"strings"

	"github.com/dave/dst"
	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

// GenerateInterfaceTargetCode generates target wrapper code for an interface or struct type.
//
//nolint:revive // stutter acceptable for exported API consistency
func GenerateInterfaceTargetCode(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	ifaceWithDetails detect.IfaceWithDetails,
	isStructType bool,
) (string, error) {
	gen, err := newInterfaceTargetGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, ifaceWithDetails)
	if err != nil {
		return "", err
	}

	return gen.generate(isStructType)
}

// interfaceTargetGenerator generates target wrappers for interfaces and struct types.
// Each interface method gets wrapped like a function with its own wrapper struct.
type interfaceTargetGenerator struct {
	baseGenerator

	wrapName            string // Wrapper constructor name (e.g., "WrapLogger")
	wrapperType         string // Main wrapper struct type (e.g., "WrapLoggerWrapper")
	interfaceName       string
	implName            string // Name for the real implementation field
	astFiles            []*dst.File
	pkgImportPath       string
	pkgLoader           detect.PackageLoader
	methodNames         []string
	identifiedInterface detect.IfaceWithDetails // full interface details including source imports
}

// buildInterfaceTargetTemplateData constructs the template data for interface target wrapper generation.
func (gen *interfaceTargetGenerator) buildInterfaceTargetTemplateData(isStructType bool) interfaceTargetTemplateData {
	// Determine if we need reflect (for ExpectReturnsEqual with DeepEqual)
	needsReflect := false
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.Iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			if ftype.Results != nil && len(ftype.Results.List) > 0 {
				needsReflect = true
			}
		},
	)

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
		NeedsFmt:          false, // Interface wrappers don't need fmt
		NeedsReflect:      needsReflect,
		NeedsImptest:      true, // Always needed for CallableController
		AdditionalImports: gen.collectAdditionalImports(),
	}

	// Construct the interface type with qualifier if needed
	interfaceType := gen.formatQualifiedInterfaceType()

	// Collect method wrappers for all interface methods
	var methodWrappers []methodWrapperData

	_ = forEachInterfaceMethod(
		gen.identifiedInterface.Iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			methodData := gen.buildMethodWrapperData(methodName, ftype)
			methodWrappers = append(methodWrappers, methodData)
		},
	)

	return interfaceTargetTemplateData{
		baseTemplateData: base,
		WrapName:         gen.wrapName,
		WrapperType:      gen.wrapperType,
		InterfaceName:    gen.interfaceName,
		InterfaceType:    interfaceType,
		ImplName:         gen.implName,
		MethodNames:      gen.methodNames,
		Methods:          methodWrappers,
		IsStructType:     isStructType,
	}
}

// buildMethodWrapperData builds wrapper data for a single interface method.
func (gen *interfaceTargetGenerator) buildMethodWrapperData(
	methodName string,
	ftype *dst.FuncType,
) methodWrapperData {
	// Build parameter string and collect param names
	paramsStr, paramNames := gen.buildParamStrings(ftype)
	resultsStr, resultTypes := gen.buildResultStrings(ftype)

	// Build parameter fields for call record
	paramFields := gen.buildParamFieldsFromNames(paramNames, ftype)
	paramFieldsStruct := gen.buildParamFieldsStruct(paramFields)

	// Build result data using shared helper
	resultData := (&ResultDataBuilder{
		ResultTypes: resultTypes,
		VarPrefix:   "r",
	}).Build()

	return methodWrapperData{
		MethodName:        methodName,
		WrapName:          fmt.Sprintf("wrap%s%s", gen.wrapperType, methodName),
		WrapperType:       fmt.Sprintf("%s%sWrapper", gen.wrapperType, methodName),
		CallHandleType:    fmt.Sprintf("%s%sCallHandle", gen.wrapperType, methodName),
		ReturnsType:       fmt.Sprintf("%s%sReturns", gen.wrapperType, methodName),
		Params:            paramsStr,
		ParamNames:        strings.Join(paramNames, ", "),
		ParamFields:       paramFields,
		ParamFieldsStruct: paramFieldsStruct,
		Results:           resultsStr,
		HasResults:        resultData.HasResults,
		ResultVars:        resultData.Vars,
		ReturnAssignments: resultData.Assignments,
		ResultReturnList:  resultData.ReturnList,
		ResultFields:      resultData.Fields,
		ResultChecks:      resultData.Checks,
		WaitMethodName:    resultData.WaitMethodName,
		ExpectedParams:    resultData.ExpectedParams,
		MatcherParams:     resultData.MatcherParams,
		PkgImptest:        pkgImptest,
		PkgReflect:        pkgReflect,
	}
}

// buildParamFieldsFromNames creates paramField slice from parameter names and types.
func (gen *interfaceTargetGenerator) buildParamFieldsFromNames(
	paramNames []string,
	ftype *dst.FuncType,
) []paramField {
	if ftype.Params == nil || len(paramNames) == 0 {
		return nil
	}

	// Preallocating with exact capacity needed
	paramFields := make([]paramField, 0, len(paramNames))
	paramIndex := 0

	for _, field := range ftype.Params.List {
		fieldType := gen.typeWithQualifier(field.Type)

		count := len(field.Names)
		if count == 0 {
			count = 1
		}

		for i := 0; i < count; i++ {
			if paramIndex < len(paramNames) {
				name := paramNames[paramIndex]
				// Capitalize first letter for exported field
				fieldName := strings.ToUpper(string(name[0])) + name[1:]
				paramFields = append(paramFields, paramField{
					Name:  fieldName,
					Type:  fieldType,
					Index: paramIndex,
				})
				paramIndex++
			}
		}
	}

	return paramFields
}

// buildParamFieldsStruct builds struct field definition string for inline parameter struct.
func (gen *interfaceTargetGenerator) buildParamFieldsStruct(fields []paramField) string {
	if len(fields) == 0 {
		return ""
	}

	var result strings.Builder

	for i, field := range fields {
		if i > 0 {
			result.WriteString("; ")
		}

		result.WriteString(fmt.Sprintf("%s %s", field.Name, field.Type))
	}

	return result.String()
}

// methodWrapperData is now defined in templates.go for consistency with other generators

// checkIfQualifierNeeded determines if we need a package qualifier.
func (gen *interfaceTargetGenerator) checkIfQualifierNeeded() {
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.Iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			gen.baseGenerator.checkIfQualifierNeeded(ftype)
		},
	)
}

// collectAdditionalImports collects imports needed for interface method signatures.
func (gen *interfaceTargetGenerator) collectAdditionalImports() []importInfo {
	// Use the source imports from the interface's file (tracked during parsing)
	sourceImports := gen.identifiedInterface.SourceImports

	// Fallback: if interface's file has no imports, collect from all files
	if len(sourceImports) == 0 {
		for _, file := range gen.astFiles {
			if len(file.Imports) > 0 {
				sourceImports = append(sourceImports, file.Imports...)
			}
		}
	}

	return gen.collectAdditionalImportsFromInterface(
		gen.identifiedInterface.Iface,
		gen.astFiles,
		gen.pkgImportPath,
		gen.pkgLoader,
		sourceImports,
	)
}

// formatQualifiedInterfaceType constructs the interface type string with qualifier if needed.
func (gen *interfaceTargetGenerator) formatQualifiedInterfaceType() string {
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

	return interfaceType
}

// generate produces the interface target wrapper code using templates.
func (gen *interfaceTargetGenerator) generate(isStructType bool) (string, error) {
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

	// Generate using templates - pass isStructType from parameter
	gen.generateWithTemplates(templates, isStructType)

	formatted, err := format.Source(gen.bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// generateWithTemplates generates code using templates instead of direct code generation.
func (gen *interfaceTargetGenerator) generateWithTemplates(templates *TemplateRegistry, isStructType bool) {
	data := gen.buildInterfaceTargetTemplateData(isStructType)

	// Generate each section using templates
	templates.WriteInterfaceTargetHeader(&gen.buf, data)
	templates.WriteInterfaceTargetWrapperStruct(&gen.buf, data)
	templates.WriteInterfaceTargetConstructor(&gen.buf, data)

	// Generate method wrappers for each interface method
	for _, methodData := range data.Methods {
		templates.WriteInterfaceTargetMethodWrapperFunc(&gen.buf, methodData)
		templates.WriteInterfaceTargetMethodWrapperStruct(&gen.buf, methodData)
		templates.WriteInterfaceTargetMethodCallHandleStruct(&gen.buf, methodData)
		templates.WriteInterfaceTargetMethodStart(&gen.buf, methodData)
		templates.WriteInterfaceTargetMethodReturns(&gen.buf, methodData)
		templates.WriteInterfaceTargetMethodExpectReturns(&gen.buf, methodData)
		templates.WriteInterfaceTargetMethodExpectCompletes(&gen.buf, methodData)
		templates.WriteInterfaceTargetMethodExpectPanic(&gen.buf, methodData)
	}
}

// newInterfaceTargetGenerator creates a new interface or struct target wrapper generator.
func newInterfaceTargetGenerator(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	ifaceWithDetails detect.IfaceWithDetails,
) (*interfaceTargetGenerator, error) {
	pkgPath, qualifier, err := resolveInterfaceGeneratorPackage(info, pkgImportPath, pkgLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface package info: %w", err)
	}

	// Wrapper type naming: WrapLogger -> WrapLoggerWrapper
	wrapperType := info.ImpName + "Wrapper"

	gen := &interfaceTargetGenerator{
		baseGenerator: newBaseGenerator(
			fset, info.PkgName, info.ImpName, pkgPath, qualifier, ifaceWithDetails.TypeParams,
		),
		wrapName:            info.ImpName,
		wrapperType:         wrapperType,
		interfaceName:       info.LocalInterfaceName,
		implName:            "impl",
		astFiles:            astFiles,
		pkgImportPath:       pkgImportPath,
		pkgLoader:           pkgLoader,
		identifiedInterface: ifaceWithDetails,
	}

	// Collect method names
	methodNames, err := interfaceCollectMethodNames(ifaceWithDetails.Iface, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return nil, err
	}

	gen.methodNames = methodNames

	return gen, nil
}
