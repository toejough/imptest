//nolint:varnamelen,wsl_v5,perfsprint,prealloc,nestif,intrange,cyclop,funlen
package run

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"sort"
	"strings"

	"github.com/dave/dst"
)

// v2InterfaceTargetGenerator generates v2-style target wrappers for interfaces.
// Each interface method gets wrapped like a function with its own wrapper struct.
type v2InterfaceTargetGenerator struct {
	baseGenerator

	wrapName            string // Wrapper constructor name (e.g., "WrapLogger")
	wrapperType         string // Main wrapper struct type (e.g., "WrapLoggerWrapper")
	interfaceName       string
	implName            string // Name for the real implementation field
	astFiles            []*dst.File
	pkgImportPath       string
	pkgLoader           PackageLoader
	methodNames         []string
	identifiedInterface ifaceWithDetails // full interface details including source imports
}

// methodWrapperData holds data for a single method wrapper.
type methodWrapperData struct {
	MethodName        string
	WrapName          string // Internal wrapper constructor (e.g., "wrapWrapLoggerWrapperLog")
	WrapperType       string // Wrapper type (e.g., "WrapLoggerWrapperLogWrapper")
	ReturnsType       string // Returns type (e.g., "WrapLoggerWrapperLogReturns")
	Params            string
	ParamNames        string
	HasResults        bool
	ResultVars        string
	ReturnAssignments string
	WaitMethodName    string
	ExpectedParams    string
	MatcherParams     string
	ResultChecks      []resultCheck
	ResultFields      []resultField
}

// checkIfQualifierNeeded determines if we need a package qualifier.
func (gen *v2InterfaceTargetGenerator) checkIfQualifierNeeded() {
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(_ string, ftype *dst.FuncType) {
			gen.baseGenerator.checkIfQualifierNeeded(ftype)
		},
	)
}

// collectAdditionalImports collects all external type imports needed for interface method signatures.
func (gen *v2InterfaceTargetGenerator) collectAdditionalImports() []importInfo {
	if len(gen.astFiles) == 0 {
		return nil
	}

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

	allImports := make(map[string]importInfo) // Deduplicate by path

	// Iterate over all interface methods to collect imports from their signatures
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
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

// buildMethodWrapperData builds wrapper data for a single interface method.
func (gen *v2InterfaceTargetGenerator) buildMethodWrapperData(
	methodName string,
	ftype *dst.FuncType,
) methodWrapperData {
	// Build parameter string and collect param names
	paramsStr, paramNames := gen.buildParamStrings(ftype)

	// Build results string and collect result types
	_, resultTypes := gen.buildResultStrings(ftype)

	// Build param names comma-separated
	var paramNamesStr strings.Builder
	for i, name := range paramNames {
		if i > 0 {
			paramNamesStr.WriteString(", ")
		}
		paramNamesStr.WriteString(name)
	}

	// Build result vars and return assignments
	var resultVarsStr, returnAssignmentsStr strings.Builder
	hasResults := len(resultTypes) > 0
	if hasResults {
		for i := range resultTypes {
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
	if hasResults {
		waitMethodName = "WaitForResponse"
	}

	// Build result fields
	resultFields := make([]resultField, 0, len(resultTypes))
	if hasResults {
		for i, resultType := range resultTypes {
			resultFields = append(resultFields, resultField{
				Name: fmt.Sprintf("Result%d", i),
				Type: resultType,
			})
		}
	}

	// Build expected params and matcher params for result verification
	var expectedParamsStr, matcherParamsStr strings.Builder
	var resultChecks []resultCheck
	if hasResults {
		for i, resultType := range resultTypes {
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

	return methodWrapperData{
		MethodName:        methodName,
		WrapName:          fmt.Sprintf("wrap%s%s", gen.wrapperType, methodName),
		WrapperType:       fmt.Sprintf("%s%sWrapper", gen.wrapperType, methodName),
		ReturnsType:       fmt.Sprintf("%s%sReturns", gen.wrapperType, methodName),
		Params:            paramsStr,
		ParamNames:        paramNamesStr.String(),
		HasResults:        hasResults,
		ResultVars:        resultVarsStr.String(),
		ReturnAssignments: returnAssignmentsStr.String(),
		WaitMethodName:    waitMethodName,
		ExpectedParams:    expectedParamsStr.String(),
		MatcherParams:     matcherParamsStr.String(),
		ResultChecks:      resultChecks,
		ResultFields:      resultFields,
	}
}

// buildParamStrings builds the parameter string and collects parameter names.
func (gen *v2InterfaceTargetGenerator) buildParamStrings(
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
func (gen *v2InterfaceTargetGenerator) buildResultStrings(
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

// generate produces the v2 interface target wrapper code.
func (gen *v2InterfaceTargetGenerator) generate() (string, error) {
	// Pre-scan to determine what imports are needed
	gen.checkIfQualifierNeeded()

	// If we have an interface from an external package, we need the qualifier
	if gen.interfaceName != "" && gen.qualifier != "" && gen.pkgPath != "" {
		gen.needsQualifier = true
	}

	// Collect method wrappers for all interface methods
	var methodWrappers []methodWrapperData
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			methodData := gen.buildMethodWrapperData(methodName, ftype)
			methodWrappers = append(methodWrappers, methodData)
		},
	)

	// Build the interface type string
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

	// Generate code manually (without templates for now)
	var buf bytes.Buffer

	// Write header
	gen.writeHeader(&buf)

	// Write main wrapper struct
	gen.writeWrapperStruct(&buf, methodWrappers, interfaceType)

	// Write constructor
	gen.writeConstructor(&buf, methodWrappers, interfaceType)

	// Write interceptor
	gen.writeInterceptor(&buf, methodWrappers, interfaceType)

	// Write Interface() method
	gen.writeInterfaceMethod(&buf, interfaceType)

	// Write per-method wrappers
	for _, methodData := range methodWrappers {
		gen.writeMethodWrapper(&buf, methodData)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// generateV2InterfaceTargetCode generates v2-style target wrapper code for an interface.
func generateV2InterfaceTargetCode(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (string, error) {
	gen, err := newV2InterfaceTargetGenerator(astFiles, info, fset, pkgImportPath, pkgLoader, ifaceWithDetails)
	if err != nil {
		return "", err
	}

	return gen.generate()
}

// newV2InterfaceTargetGenerator creates a new v2 interface target wrapper generator.
func newV2InterfaceTargetGenerator(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	ifaceWithDetails ifaceWithDetails,
) (*v2InterfaceTargetGenerator, error) {
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

	// Wrapper type naming: WrapLogger -> WrapLoggerWrapper
	wrapperType := info.impName + "Wrapper"

	gen := &v2InterfaceTargetGenerator{
		baseGenerator: newBaseGenerator(
			fset, info.pkgName, info.impName, pkgPath, qualifier, ifaceWithDetails.typeParams,
		),
		wrapName:            info.impName,
		wrapperType:         wrapperType,
		interfaceName:       info.localInterfaceName,
		implName:            "impl",
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

// writeHeader writes the package declaration and imports.
func (gen *v2InterfaceTargetGenerator) writeHeader(buf *bytes.Buffer) {
	buf.WriteString(fmt.Sprintf("package %s\n\n", gen.pkgName))

	// Write imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"testing\"\n")

	// Add external package import if needed
	if gen.needsQualifier && gen.pkgPath != "" {
		alias := gen.qualifier
		if !strings.Contains(gen.pkgPath, "/") && gen.pkgPath == gen.qualifier {
			alias = "_" + gen.qualifier
		}
		buf.WriteString(fmt.Sprintf("\t%s \"%s\"\n", alias, gen.pkgPath))
	}

	// Add additional imports
	additionalImports := gen.collectAdditionalImports()
	for _, imp := range additionalImports {
		if imp.Alias != "" {
			buf.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
		} else {
			buf.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
		}
	}

	buf.WriteString(")\n\n")
}

// writeWrapperStruct writes the main wrapper struct that holds all method wrappers.
func (gen *v2InterfaceTargetGenerator) writeWrapperStruct(
	buf *bytes.Buffer,
	methodWrappers []methodWrapperData,
	interfaceType string,
) {
	buf.WriteString(fmt.Sprintf("// %s wraps an implementation of %s to intercept method calls.\n", gen.wrapperType, interfaceType))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", gen.wrapperType))
	buf.WriteString(fmt.Sprintf("\t%s %s\n", gen.implName, interfaceType))
	buf.WriteString("\tinterceptor *interceptor\n")
	for _, method := range methodWrappers {
		buf.WriteString(fmt.Sprintf("\t%s *%s\n", method.MethodName, method.WrapperType))
	}
	buf.WriteString("}\n\n")
}

// writeConstructor writes the constructor function.
func (gen *v2InterfaceTargetGenerator) writeConstructor(
	buf *bytes.Buffer,
	methodWrappers []methodWrapperData,
	interfaceType string,
) {
	buf.WriteString(fmt.Sprintf("// %s creates a new wrapper for the given %s implementation.\n", gen.wrapName, interfaceType))
	buf.WriteString(fmt.Sprintf("func %s(t *testing.T, %s %s) *%s {\n", gen.wrapName, gen.implName, interfaceType, gen.wrapperType))
	buf.WriteString(fmt.Sprintf("\tw := &%s{\n", gen.wrapperType))
	buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n", gen.implName, gen.implName))
	buf.WriteString("\t}\n")
	buf.WriteString("\tw.interceptor = &interceptor{wrapper: w, t: t, impl: impl}\n")

	// Initialize each method wrapper - wrap the REAL implementation's methods, not interceptor
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			paramsStr, paramNames := gen.buildParamStrings(ftype)
			_, resultTypes := gen.buildResultStrings(ftype)

			// Build param names comma-separated
			var paramNamesStr strings.Builder
			for i, name := range paramNames {
				if i > 0 {
					paramNamesStr.WriteString(", ")
				}
				paramNamesStr.WriteString(name)
			}

			// Find the method wrapper data for this method
			var methodData methodWrapperData
			for _, mw := range methodWrappers {
				if mw.MethodName == methodName {
					methodData = mw
					break
				}
			}

			// Create a closure that calls the real implementation and returns the result
			buf.WriteString(fmt.Sprintf("\tw.%s = %s(t, func(%s) %s {\n",
				methodName, methodData.WrapName, paramsStr,
				methodData.ReturnsType))

			// Call the real implementation
			if len(resultTypes) > 0 {
				// Build result variable list for receiving return values
				var resultVars strings.Builder
				for i := range resultTypes {
					if i > 0 {
						resultVars.WriteString(", ")
					}
					fmt.Fprintf(&resultVars, "r%d", i)
				}

				buf.WriteString(fmt.Sprintf("\t\t%s := w.%s.%s(%s)\n", resultVars.String(), gen.implName, methodName, paramNamesStr.String()))
				buf.WriteString(fmt.Sprintf("\t\treturn %s{", methodData.ReturnsType))
				for i := range resultTypes {
					if i > 0 {
						buf.WriteString(", ")
					}
					buf.WriteString(fmt.Sprintf("Result%d: r%d", i, i))
				}
				buf.WriteString("}\n")
			} else {
				buf.WriteString(fmt.Sprintf("\t\tw.%s.%s(%s)\n", gen.implName, methodName, paramNamesStr.String()))
				buf.WriteString(fmt.Sprintf("\t\treturn %s{}\n", methodData.ReturnsType))
			}

			buf.WriteString("\t})\n")
		},
	)

	buf.WriteString("\treturn w\n")
	buf.WriteString("}\n\n")
}

// writeInterceptor writes the interceptor struct and its methods.
func (gen *v2InterfaceTargetGenerator) writeInterceptor(
	buf *bytes.Buffer,
	methodWrappers []methodWrapperData,
	interfaceType string,
) {
	buf.WriteString(fmt.Sprintf("// interceptor implements %s and routes calls through method wrappers.\n", interfaceType))
	buf.WriteString("type interceptor struct {\n")
	buf.WriteString(fmt.Sprintf("\twrapper *%s\n", gen.wrapperType))
	buf.WriteString(fmt.Sprintf("\timpl %s\n", interfaceType))
	buf.WriteString("\tt *testing.T\n")
	buf.WriteString("}\n\n")

	// Write each interceptor method
	_ = forEachInterfaceMethod(
		gen.identifiedInterface.iface, gen.astFiles, gen.fset, gen.pkgImportPath, gen.pkgLoader,
		func(methodName string, ftype *dst.FuncType) {
			gen.writeInterceptorMethod(buf, methodName, ftype)
		},
	)
}

// writeInterceptorMethod writes a single interceptor method.
func (gen *v2InterfaceTargetGenerator) writeInterceptorMethod(
	buf *bytes.Buffer,
	methodName string,
	ftype *dst.FuncType,
) {
	paramsStr, paramNames := gen.buildParamStrings(ftype)
	resultsStr, _ := gen.buildResultStrings(ftype)

	// Build param names comma-separated
	var paramNamesStr strings.Builder
	for i, name := range paramNames {
		if i > 0 {
			paramNamesStr.WriteString(", ")
		}
		paramNamesStr.WriteString(name)
	}

	// Write method signature
	buf.WriteString(fmt.Sprintf("func (i *interceptor) %s(%s)%s {\n", methodName, paramsStr, resultsStr))

	// Call real implementation through wrapper and return
	if resultsStr != "" {
		buf.WriteString(fmt.Sprintf("\treturn i.wrapper.%s.Start(%s).%s()\n", methodName, paramNamesStr.String(), "WaitForResponse"))
	} else {
		buf.WriteString(fmt.Sprintf("\ti.wrapper.%s.Start(%s).%s()\n", methodName, paramNamesStr.String(), "WaitForCompletion"))
	}

	buf.WriteString("}\n\n")
}

// writeInterfaceMethod writes the Interface() method that returns the interceptor.
func (gen *v2InterfaceTargetGenerator) writeInterfaceMethod(buf *bytes.Buffer, interfaceType string) {
	buf.WriteString(fmt.Sprintf("// Interface returns the wrapped %s implementation.\n", interfaceType))
	buf.WriteString(fmt.Sprintf("func (w *%s) Interface() %s {\n", gen.wrapperType, interfaceType))
	buf.WriteString("\treturn w.interceptor\n")
	buf.WriteString("}\n\n")
}

// writeMethodWrapper writes a complete wrapper for a single method (borrowed from v2 target pattern).
func (gen *v2InterfaceTargetGenerator) writeMethodWrapper(buf *bytes.Buffer, data methodWrapperData) {
	// Write wrapper function
	buf.WriteString(fmt.Sprintf("func %s(t *testing.T, fn func(%s)%s) *%s {\n",
		data.WrapName, data.Params,
		func() string {
			if data.HasResults {
				return " " + data.ReturnsType
			}
			return ""
		}(),
		data.WrapperType))
	buf.WriteString(fmt.Sprintf("\treturn &%s{t: t, fn: fn}\n", data.WrapperType))
	buf.WriteString("}\n\n")

	// Write wrapper struct
	buf.WriteString(fmt.Sprintf("type %s struct {\n", data.WrapperType))
	buf.WriteString("\tt *testing.T\n")
	buf.WriteString(fmt.Sprintf("\tfn func(%s)%s\n", data.Params,
		func() string {
			if data.HasResults {
				return " " + data.ReturnsType
			}
			return ""
		}()))
	buf.WriteString(fmt.Sprintf("\tcalls []%sCallRecord\n", data.WrapperType))
	buf.WriteString("}\n\n")

	// Write callRecord helper (unique name per method to avoid conflicts)
	buf.WriteString(fmt.Sprintf("type %sCallRecord struct {\n", data.WrapperType))
	if data.Params != "" {
		buf.WriteString("\tParams struct {\n")
		// Parse params to extract field names and types
		gen.writeParamFields(buf, data.Params)
		buf.WriteString("\t}\n")
	}
	if data.HasResults {
		buf.WriteString(fmt.Sprintf("\tReturns %s\n", data.ReturnsType))
	}
	buf.WriteString("}\n\n")

	// Write Start method
	buf.WriteString(fmt.Sprintf("func (w *%s) Start(%s) *%s {\n", data.WrapperType, data.Params, data.ReturnsType))
	// Call the wrapper function which returns the Returns struct
	buf.WriteString(fmt.Sprintf("\treturns := w.fn(%s)\n", data.ParamNames))
	buf.WriteString(fmt.Sprintf("\tw.calls = append(w.calls, %sCallRecord{", data.WrapperType))
	if data.Params != "" {
		buf.WriteString(fmt.Sprintf("Params: struct{%s}{%s}, ", gen.extractParamTypesForStruct(data.Params), data.ParamNames))
	}
	buf.WriteString("Returns: returns})\n")
	buf.WriteString("\treturn &returns\n")
	buf.WriteString("}\n\n")

	// Write Returns struct
	buf.WriteString(fmt.Sprintf("type %s struct {\n", data.ReturnsType))
	for _, field := range data.ResultFields {
		buf.WriteString(fmt.Sprintf("\t%s %s\n", field.Name, field.Type))
	}
	buf.WriteString("}\n\n")

	// Write WaitForResponse/WaitForCompletion methods
	if data.HasResults {
		buf.WriteString(fmt.Sprintf("func (r *%s) WaitForResponse() (%s) {\n", data.ReturnsType, gen.buildResultReturnList(data.ResultFields)))
		buf.WriteString("\treturn ")
		for i, field := range data.ResultFields {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("r.%s", field.Name))
		}
		buf.WriteString("\n}\n\n")
	} else {
		buf.WriteString(fmt.Sprintf("func (r *%s) WaitForCompletion() {}\n\n", data.ReturnsType))
	}

	// Write GetCalls method
	buf.WriteString(fmt.Sprintf("func (w *%s) GetCalls() []%sCallRecord {\n", data.WrapperType, data.WrapperType))
	buf.WriteString("\treturn w.calls\n")
	buf.WriteString("}\n\n")
}

// writeParamFields writes parameter fields for the callRecord struct.
func (gen *v2InterfaceTargetGenerator) writeParamFields(buf *bytes.Buffer, params string) {
	// Parse "name type, name2 type2" format
	parts := strings.Split(params, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			name := fields[0]
			typ := strings.Join(fields[1:], " ")
			// Capitalize first letter
			fieldName := strings.ToUpper(name[:1]) + name[1:]
			buf.WriteString(fmt.Sprintf("\t\t%s %s\n", fieldName, typ))
		}
	}
}

// extractParamTypesForStruct extracts parameter types from "name type, name2 type2" format
// and returns them in struct field format "FieldName type; FieldName2 type2".
func (gen *v2InterfaceTargetGenerator) extractParamTypesForStruct(params string) string {
	var result strings.Builder
	parts := strings.Split(params, ",")
	for i, part := range parts {
		if i > 0 {
			result.WriteString("; ")
		}
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			name := fields[0]
			typ := strings.Join(fields[1:], " ")
			// Capitalize first letter
			fieldName := strings.ToUpper(name[:1]) + name[1:]
			result.WriteString(fmt.Sprintf("%s %s", fieldName, typ))
		}
	}

	return result.String()
}

// buildResultReturnList builds the return type list from result fields.
func (gen *v2InterfaceTargetGenerator) buildResultReturnList(fields []resultField) string {
	if len(fields) == 0 {
		return ""
	}
	if len(fields) == 1 {
		return fields[0].Type
	}

	var result strings.Builder
	result.WriteString("(")
	for i, field := range fields {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(field.Type)
	}
	result.WriteString(")")

	return result.String()
}
