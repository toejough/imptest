// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"os"
	"strconv"
	"strings"

	"github.com/alexflint/go-arg"
)

// Run executes the impgen tool logic. It takes command-line arguments, an environment variable getter, a FileSystem
// interface for file operations, and a PackageLoader for package operations. It returns an error if any step fails. On
// success, it generates a Go source file implementing the specified interface, in the calling test package.
func Run(args []string, getEnv func(string) string, fileSys FileSystem, pkgLoader PackageLoader) error {
	info, err := getGeneratorCallInfo(args, getEnv)
	if err != nil {
		return err
	}

	pkgImportPath, err := getInterfacePackagePath(info.interfaceName, pkgLoader)
	if err != nil {
		return err
	}

	astFiles, fset, err := loadPackage(pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	iface, err := getMatchingInterfaceFromAST(astFiles, info.localInterfaceName, pkgImportPath)
	if err != nil {
		return err
	}

	code, err := generateImplementationCode(iface, info, fset)
	if err != nil {
		return err
	}

	err = writeGeneratedCodeToFile(code, info.impName, info.pkgName, fileSys)
	if err != nil {
		return err
	}

	return nil
}

func loadPackage(pkgImportPath string, pkgLoader PackageLoader) ([]*ast.File, *token.FileSet, error) {
	astFiles, fset, err := pkgLoader.Load(pkgImportPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load package %q: %w", pkgImportPath, err)
	}

	return astFiles, fset, nil
}

// FileSystem interface for mocking.
type FileSystem interface {
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// PackageLoader interface for loading external packages.
type PackageLoader interface {
	Load(importPath string) ([]*ast.File, *token.FileSet, error)
}

// generatorInfo holds information gathered for generation.
type generatorInfo struct {
	pkgName, interfaceName, localInterfaceName, impName string
}

// cliArgs defines the command-line arguments for the generator.
type cliArgs struct {
	Interface string `arg:"positional,required" help:"interface name to implement (e.g. MyInterface or pkg.MyInterface)"`
	Name      string `arg:"--name"              help:"name for the generated implementation (defaults to <Interface>Imp)"`
}

// getGeneratorCallInfo returns basic information about the current call to the generator.
func getGeneratorCallInfo(args []string, getEnv func(string) string) (generatorInfo, error) {
	pkgName := getEnv("GOPACKAGE")

	parsed, err := parseArgs(args)
	if err != nil {
		return generatorInfo{}, err
	}

	interfaceName := parsed.Interface
	localInterfaceName := getLocalInterfaceName(interfaceName)
	impName := parsed.Name

	// set impname if not provided
	if impName == "" {
		impName = localInterfaceName + "Imp" // default implementation name
	}

	return generatorInfo{
		pkgName:            pkgName,
		interfaceName:      interfaceName,
		localInterfaceName: localInterfaceName,
		impName:            impName,
	}, nil
}

// getLocalInterfaceName extracts the local interface name from a possibly qualified name
// (e.g., "MyInterface" from "pkg.MyInterface").
func getLocalInterfaceName(name string) string {
	if _, after, ok := strings.Cut(name, "."); ok {
		return after
	}

	return name
}

// parseArgs parses command-line arguments into cliArgs.
func parseArgs(args []string) (cliArgs, error) {
	var parsed cliArgs

	parser, err := arg.NewParser(arg.Config{}, &parsed)
	if err != nil {
		return cliArgs{}, fmt.Errorf("failed to create argument parser: %w", err)
	}

	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	err = parser.Parse(cmdArgs)
	if err != nil {
		return cliArgs{}, fmt.Errorf("failed to parse arguments: %w", err)
	}

	return parsed, nil
}

// getInterfacePackagePath determines the import path for the interface. Returns "." for local interfaces, or resolves
// the full import path for qualified names like "pkg.Interface".
func getInterfacePackagePath(qualifiedName string, pkgLoader PackageLoader) (string, error) {
	if isLocalInterface(qualifiedName) {
		return getLocalPackagePath(), nil
	}

	return getNonLocalPackagePath(qualifiedName, pkgLoader)
}

// isLocalInterface checks if the interface name is local (no package qualifier).
func isLocalInterface(qualifiedName string) bool {
	return !strings.Contains(qualifiedName, ".")
}

// getLocalPackagePath returns the path for local package interfaces.
func getLocalPackagePath() string {
	return "."
}

// getNonLocalPackagePath resolves the full import path for a qualified interface name.
func getNonLocalPackagePath(qualifiedName string, pkgLoader PackageLoader) (string, error) {
	targetPkgImport := extractPackageName(qualifiedName)

	astFiles, _, err := pkgLoader.Load(".")
	if err != nil {
		return "", fmt.Errorf("failed to load local package: %w", err)
	}

	importPath, err := findImportPath(astFiles, targetPkgImport)
	if err != nil {
		return "", err
	}

	return importPath, nil
}

// extractPackageName extracts the package name from a qualified interface name (e.g., "pkg" from "pkg.Interface").
func extractPackageName(qualifiedName string) string {
	before, _, _ := strings.Cut(qualifiedName, ".")
	return before
}

// findImportPath searches through AST files to find the full import path for a package name.
func findImportPath(astFiles []*ast.File, targetPkgImport string) (string, error) {
	for _, fileAst := range astFiles {
		importPath, err := searchFileImports(fileAst, targetPkgImport)
		if err != nil {
			return "", err
		}

		if importPath != "" {
			return importPath, nil
		}
	}

	return "", fmt.Errorf("%w: %q", errPackageNotFound, targetPkgImport)
}

// searchFileImports searches a single AST file's imports for a matching package name.
// Returns the full import path if found, empty string if not found, or an error if import paths are malformed.
func searchFileImports(fileAst *ast.File, targetPkgImport string) (string, error) {
	for _, imp := range fileAst.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return "", fmt.Errorf("failed to unquote import path %q: %w", imp.Path.Value, err)
		}

		if importPathMatchesPackageName(importPath, targetPkgImport) {
			return importPath, nil
		}
	}

	return "", nil
}

// importPathMatchesPackageName checks if the last segment of an import path matches the target package name.
func importPathMatchesPackageName(importPath, targetPkgImport string) bool {
	parts := strings.Split(importPath, "/")
	return len(parts) > 0 && parts[len(parts)-1] == targetPkgImport
}

// getMatchingInterfaceFromAST finds the interface by name in the ASTs.
func getMatchingInterfaceFromAST(
	astFiles []*ast.File, localInterfaceName, pkgImportPath string,
) (*ast.InterfaceType, error) {
	for _, fileAst := range astFiles {
		var found *ast.InterfaceType

		ast.Inspect(fileAst, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if ok {
				if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 && ts.Name.Name == localInterfaceName {
					found = iface
					return false
				}
			}

			return true
		})

		if found != nil {
			return found, nil
		}
	}

	return nil, fmt.Errorf("%w: named %q in package %q", errInterfaceNotFound, localInterfaceName, pkgImportPath)
}

var (
	errInterfaceNotFound = errors.New("interface not found")
	errPackageNotFound   = errors.New("package not found in imports")
)

// generateImplementationCode creates the Go code for the interface implementation.
func generateImplementationCode(
	identifiedInterface *ast.InterfaceType,
	info generatorInfo,
	fset *token.FileSet,
) (string, error) {
	impName := info.impName

	gen := &codeGenerator{
		fset:                fset,
		pkgName:             info.pkgName,
		impName:             impName,
		mockName:            impName + "Mock",
		callName:            impName + "Call",
		expectCallToName:    impName + "ExpectCallTo",
		timedName:           impName + "Timed",
		identifiedInterface: identifiedInterface,
	}

	gen.generateHeader()
	gen.generateMockStruct()
	gen.generateMainStruct()

	methodNames := gen.generateMethodStructs()
	gen.generateMockMethods()
	gen.generateCallStruct(methodNames)
	gen.generateExpectCallToStruct()
	gen.generateExpectCallToMethods()
	gen.generateTimedStruct()
	gen.generateGetCallMethod()
	gen.generateGetCurrentCallMethod()
	gen.generateConstructor()

	formatted, err := format.Source(gen.buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formatted), nil
}

// writeGeneratedCodeToFile writes the generated code to <impName>.go.
func writeGeneratedCodeToFile(code string, impName string, pkgName string, fileSys FileSystem) error {
	const generatedFilePermissions = 0o600

	filename := impName
	// If we're in a test package, append _test to the filename
	if strings.HasSuffix(pkgName, "_test") && !strings.HasSuffix(impName, "_test") {
		filename = strings.TrimSuffix(impName, ".go") + "_test.go"
	} else if !strings.HasSuffix(filename, ".go") {
		filename += ".go"
	}

	err := fileSys.WriteFile(filename, []byte(code), generatedFilePermissions)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	fmt.Printf("%s written successfully.\n", filename)

	return nil
}

// codeGenerator holds state for code generation.
type codeGenerator struct {
	buf                 bytes.Buffer
	fset                *token.FileSet
	pkgName             string
	impName             string
	mockName            string
	callName            string
	expectCallToName    string
	timedName           string
	identifiedInterface *ast.InterfaceType
}

// p writes a formatted string to the buffer (short for "print").
func (gen *codeGenerator) pf(format string, args ...any) {
	fmt.Fprintf(&gen.buf, format, args...)
}

func (gen *codeGenerator) generateHeader() {
	gen.pf(`package %s

// Code generated by impgen. DO NOT EDIT.

import "sync"
import "testing"
import "time"

`, gen.pkgName)
}

func (gen *codeGenerator) generateMockStruct() {
	gen.pf("type %s struct {\n", gen.mockName)
	gen.pf("\timp *%s\n", gen.impName)
	gen.pf("}\n\n")
}

func (gen *codeGenerator) generateMainStruct() {
	gen.pf("type %s struct {\n", gen.impName)
	gen.pf("\tt *testing.T\n")
	gen.pf("\tMock *%s\n", gen.mockName)
	gen.pf("\tcallChan chan *%s\n", gen.callName)
	gen.pf("\tExpectCallTo *%s\n", gen.expectCallToName)
	gen.pf("\tcurrentCall *%s\n", gen.callName)
	gen.pf("\tcallQueue []*%s\n", gen.callName)
	gen.pf("\tqueueLock sync.Mutex\n")
	gen.pf("}\n\n")
}

// methodCallName returns the call struct name for a method (e.g. "MyImpDoSomethingCall").
func (gen *codeGenerator) methodCallName(methodName string) string {
	return gen.impName + methodName + "Call"
}

// writeMethodSignature writes the method name and parameters (e.g., "MethodName(a int, b string)").
func (gen *codeGenerator) writeMethodSignature(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.pf("%s(", methodName)
	gen.writeMethodParams(ftype, paramNames)
	gen.pf(")")
}

// forEachMethod iterates over interface methods and calls the callback for each.
func (gen *codeGenerator) forEachMethod(callback func(methodName string, ftype *ast.FuncType)) {
	for _, field := range gen.identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			callback(methodName.Name, ftype)
		}
	}
}

func (gen *codeGenerator) generateMethodStructs() []string {
	var methodNames []string

	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		methodNames = append(methodNames, methodName)
		gen.generateMethodCallStruct(methodName, ftype)
		gen.generateMethodResponseStruct(methodName, ftype)
		gen.generateMethodResponseMethods(methodName, ftype)
	})

	return methodNames
}

func (gen *codeGenerator) generateMethodCallStruct(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	gen.pf("type %s struct {\n", callName)
	gen.pf("\tresponseChan chan %sResponse\n", callName)
	gen.pf("\tdone bool\n")

	if hasParams(ftype) {
		totalParams := countFields(ftype.Params)
		unnamedIndex := 0

		for _, param := range ftype.Params.List {
			paramType := exprToString(gen.fset, param.Type)
			if len(param.Names) > 0 {
				for i := range param.Names {
					fieldName := getParamFieldName(param, i, unnamedIndex, paramType, totalParams)
					gen.pf("\t%s %s\n", fieldName, paramType)
				}
			} else {
				fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
				gen.pf("\t%s %s\n", fieldName, paramType)

				unnamedIndex++
			}
		}
	}

	gen.pf("}\n\n")
}

func (gen *codeGenerator) generateMethodResponseStruct(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	gen.pf("type %sResponse struct {\n", callName)
	gen.pf("\tType string // \"return\", \"panic\", or \"resolve\"\n")

	if hasResults(ftype) {
		returnIndex := 0

		for _, result := range ftype.Results.List {
			resultType := exprToString(gen.fset, result.Type)
			if len(result.Names) > 0 {
				for _, name := range result.Names {
					gen.pf("\t%s %s\n", name.Name, resultType)

					returnIndex++
				}
			} else {
				gen.pf("\tResult%d %s\n", returnIndex, resultType)
				returnIndex++
			}
		}
	}

	gen.pf("\tPanicValue interface{}\n")
	gen.pf("}\n\n")
}

func (gen *codeGenerator) generateMethodResponseMethods(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)

	if hasResults(ftype) {
		totalReturns := countFields(ftype.Results)

		if totalReturns == 1 {
			gen.generateInjectResultMethod(callName, ftype)
		} else {
			gen.generateInjectResultsMethod(callName, ftype)
		}

		gen.generateInjectPanicMethod(callName)
	} else {
		gen.generateResolveMethod(callName)
		gen.generateInjectPanicMethod(callName)
	}

	gen.pf("\n")
}

func (gen *codeGenerator) generateInjectResultMethod(methodCallName string, ftype *ast.FuncType) {
	resultType := exprToString(gen.fset, ftype.Results.List[0].Type)
	gen.pf("func (c *%s) InjectResult(result %s) {\n", methodCallName, resultType)
	gen.pf("\tc.done = true\n")
	gen.pf("\tc.responseChan <- %sResponse{Type: \"return\"", methodCallName)

	if len(ftype.Results.List[0].Names) > 0 {
		gen.pf(", %s: result", ftype.Results.List[0].Names[0].Name)
	} else {
		gen.pf(", Result0: result")
	}

	gen.pf("}\n")
	gen.pf("}\n")
}

func (gen *codeGenerator) generateInjectResultsMethod(methodCallName string, ftype *ast.FuncType) {
	gen.pf("func (c *%s) InjectResults(", methodCallName)

	returnIndex := 0

	returnParamNames := make([]string, 0)

	for _, result := range ftype.Results.List {
		resultType := exprToString(gen.fset, result.Type)
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				if returnIndex > 0 {
					gen.pf(", ")
				}

				gen.pf("%s %s", name.Name, resultType)
				returnParamNames = append(returnParamNames, name.Name)
				returnIndex++
			}
		} else {
			if returnIndex > 0 {
				gen.pf(", ")
			}

			paramName := fmt.Sprintf("result%d", returnIndex)
			gen.pf("%s %s", paramName, resultType)
			returnParamNames = append(returnParamNames, paramName)
			returnIndex++
		}
	}

	gen.pf(") {\n")
	gen.pf("\tc.done = true\n")
	gen.pf("\tresp := %sResponse{Type: \"return\"", methodCallName)

	returnIndex = 0

	for _, result := range ftype.Results.List {
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				gen.pf(", %s: %s", name.Name, returnParamNames[returnIndex])
				returnIndex++
			}
		} else {
			gen.pf(", Result%d: %s", returnIndex, returnParamNames[returnIndex])
			returnIndex++
		}
	}

	gen.pf("}\n")
	gen.pf("\tc.responseChan <- resp\n")
	gen.pf("}\n")
}

func (gen *codeGenerator) generateInjectPanicMethod(methodCallName string) {
	gen.pf("func (c *%s) InjectPanic(msg interface{}) {\n", methodCallName)
	gen.pf("\tc.done = true\n")
	gen.pf("\tc.responseChan <- %sResponse{Type: \"panic\", PanicValue: msg}\n", methodCallName)
	gen.pf("}\n")
}

func (gen *codeGenerator) generateResolveMethod(methodCallName string) {
	gen.pf("func (c *%s) Resolve() {\n", methodCallName)
	gen.pf("\tc.done = true\n")
	gen.pf("\tc.responseChan <- %sResponse{Type: \"resolve\"}\n", methodCallName)
	gen.pf("}\n")
}

func (gen *codeGenerator) generateMockMethods() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateMockMethod(methodName, ftype)
	})
}

func (gen *codeGenerator) generateMockMethod(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	paramNames := extractParamNames(ftype)

	gen.pf("func (m *%s) ", gen.mockName)
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.pf("%s", renderFieldList(gen.fset, ftype.Results))
	gen.pf(" {\n")

	gen.pf("\tresponseChan := make(chan %sResponse, 1)\n", callName)
	gen.pf("\n")

	gen.pf("\tcall := &%s{\n", callName)
	gen.pf("\t\tresponseChan: responseChan,\n")
	gen.writeCallStructFields(ftype, paramNames)
	gen.pf("\t}\n\n")

	gen.pf("\tcallEvent := &%s{\n", gen.callName)
	gen.pf("\t\t%s: call,\n", methodName)
	gen.pf("\t}\n\n")

	gen.pf("\tm.imp.callChan <- callEvent\n\n")
	gen.pf("\tresp := <-responseChan\n\n")

	gen.pf("\tif resp.Type == \"panic\" {\n")
	gen.pf("\t\tpanic(resp.PanicValue)\n")
	gen.pf("\t}\n\n")

	gen.writeReturnStatement(ftype)
	gen.pf("}\n\n")
}

func (gen *codeGenerator) writeMethodParams(ftype *ast.FuncType, paramNames []string) {
	if !hasParams(ftype) {
		return
	}

	paramNameIndex := 0

	for i, param := range ftype.Params.List {
		if i > 0 {
			gen.pf(", ")
		}

		paramType := exprToString(gen.fset, param.Type)
		if len(param.Names) > 0 {
			for j, name := range param.Names {
				if j > 0 {
					gen.pf(", ")
				}

				gen.pf("%s %s", name.Name, paramType)

				paramNameIndex++
			}
		} else {
			gen.pf("%s %s", paramNames[paramNameIndex], paramType)
			paramNameIndex++
		}
	}
}

func (gen *codeGenerator) writeCallStructFields(ftype *ast.FuncType, paramNames []string) {
	if !hasParams(ftype) {
		return
	}

	totalParams := countFields(ftype.Params)
	paramNameIndex := 0
	unnamedIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(gen.fset, param.Type)
		if len(param.Names) > 0 {
			for i, name := range param.Names {
				fieldName := getParamFieldName(param, i, unnamedIndex, paramType, totalParams)
				gen.pf("\t\t%s: %s,\n", fieldName, name.Name)

				paramNameIndex++
			}
		} else {
			fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
			gen.pf("\t\t%s: %s,\n", fieldName, paramNames[paramNameIndex])
			paramNameIndex++
			unnamedIndex++
		}
	}
}

func (gen *codeGenerator) writeReturnStatement(ftype *ast.FuncType) {
	if !hasResults(ftype) {
		gen.pf("\treturn\n")
		return
	}

	gen.pf("\treturn")

	returnIndex := 0

	for _, result := range ftype.Results.List {
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				if returnIndex > 0 {
					gen.pf(", ")
				}

				gen.pf(" resp.%s", name.Name)

				returnIndex++
			}
		} else {
			if returnIndex > 0 {
				gen.pf(", ")
			}

			gen.pf(" resp.Result%d", returnIndex)
			returnIndex++
		}
	}

	gen.pf("\n")
}

func (gen *codeGenerator) generateCallStruct(methodNames []string) {
	gen.pf("type %s struct {\n", gen.callName)

	for _, methodName := range methodNames {
		gen.pf("\t%s *%s\n", methodName, gen.methodCallName(methodName))
	}

	gen.pf("}\n\n")

	gen.generateCallNameMethod(methodNames)
	gen.generateCallDoneMethod(methodNames)
	gen.generateCallAsMethod(methodNames)
}

func (gen *codeGenerator) generateCallNameMethod(methodNames []string) {
	gen.pf("func (c *%s) Name() string {\n", gen.callName)

	for _, methodName := range methodNames {
		gen.pf("\tif c.%s != nil {\n", methodName)
		gen.pf("\t\treturn %q\n", methodName)
		gen.pf("\t}\n")
	}

	gen.pf("\treturn \"\"\n")
	gen.pf("}\n\n")
}

func (gen *codeGenerator) generateCallDoneMethod(methodNames []string) {
	gen.pf("func (c *%s) Done() bool {\n", gen.callName)

	for _, methodName := range methodNames {
		gen.pf("\tif c.%s != nil {\n", methodName)
		gen.pf("\t\treturn c.%s.done\n", methodName)
		gen.pf("\t}\n")
	}

	gen.pf("\treturn false\n")
	gen.pf("}\n\n")
}

func (gen *codeGenerator) generateCallAsMethod(methodNames []string) {
	for _, methodName := range methodNames {
		gen.pf("func (c *%s) As%s() *%s { return c.%s }\n\n",
			gen.callName, methodName, gen.methodCallName(methodName), methodName)
	}
}

func (gen *codeGenerator) generateExpectCallToStruct() {
	gen.pf("type %s struct {\n", gen.expectCallToName)
	gen.pf("\timp *%s\n", gen.impName)
	gen.pf("\ttimeout time.Duration\n")
	gen.pf("}\n\n")
}

func (gen *codeGenerator) generateExpectCallToMethods() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateExpectCallToMethod(methodName, ftype)
	})
}

func (gen *codeGenerator) generateExpectCallToMethod(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	paramNames := extractParamNames(ftype)

	gen.pf("func (e *%s) ", gen.expectCallToName)
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.pf(" *%s {\n", callName)

	gen.generateValidatorFunction(methodName, ftype, paramNames)

	gen.pf("\tcall := e.imp.GetCall(e.timeout, validator)\n")
	gen.pf("\treturn call.As%s()\n", methodName)
	gen.pf("}\n\n")
}

func (gen *codeGenerator) generateValidatorFunction(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.pf("\tvalidator := func(c *%s) bool {\n", gen.callName)
	gen.pf("\t\tif c.Name() != %q {\n", methodName)
	gen.pf("\t\t\treturn false\n")
	gen.pf("\t\t}\n")

	if hasParams(ftype) {
		gen.pf("\t\tmethodCall := c.As%s()\n", methodName)
		gen.writeValidatorChecks(ftype, paramNames)
	}

	gen.pf("\t\treturn true\n")
	gen.pf("\t}\n\n")
}

func (gen *codeGenerator) writeValidatorChecks(ftype *ast.FuncType, paramNames []string) {
	totalParams := countFields(ftype.Params)
	paramNameIndex := 0
	unnamedIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(gen.fset, param.Type)
		if len(param.Names) > 0 {
			for i, name := range param.Names {
				fieldName := getParamFieldName(param, i, unnamedIndex, paramType, totalParams)
				gen.pf("\t\tif methodCall.%s != %s {\n", fieldName, name.Name)
				gen.pf("\t\t\treturn false\n")
				gen.pf("\t\t}\n")

				paramNameIndex++
			}
		} else {
			fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
			gen.pf("\t\tif methodCall.%s != %s {\n", fieldName, paramNames[paramNameIndex])
			gen.pf("\t\t\treturn false\n")
			gen.pf("\t\t}\n")

			paramNameIndex++
			unnamedIndex++
		}
	}
}

func (gen *codeGenerator) generateTimedStruct() {
	gen.pf(`type %s struct {
	ExpectCallTo *%s
}

func (i *%s) Within(d time.Duration) *%s {
	return &%s{
		ExpectCallTo: &%s{imp: i, timeout: d},
	}
}

`, gen.timedName, gen.expectCallToName, gen.impName, gen.timedName, gen.timedName, gen.expectCallToName)
}

func (gen *codeGenerator) generateGetCallMethod() {
	gen.pf(`func (i *%s) GetCall(d time.Duration, validator func(*%s) bool) *%s {
	i.queueLock.Lock()
	defer i.queueLock.Unlock()

	for index, call := range i.callQueue {
		if validator(call) {
			// Remove from queue
			i.callQueue = append(i.callQueue[:index], i.callQueue[index+1:]...)
			return call
		}
	}

	var timeoutChan <-chan time.Time
	if d > 0 {
		timeoutChan = time.After(d)
	}

	for {
		select {
		case call := <-i.callChan:
			if validator(call) {
				return call
			}
			// Queue it
			i.callQueue = append(i.callQueue, call)
		case <-timeoutChan:
			i.t.Fatalf("timeout waiting for call matching validator")
			return nil
		}
	}
}

`, gen.impName, gen.callName, gen.callName)
}

func (gen *codeGenerator) generateGetCurrentCallMethod() {
	gen.pf(`func (i *%s) GetCurrentCall() *%s {
	if i.currentCall != nil && !i.currentCall.Done() {
		return i.currentCall
	}
	i.currentCall = i.GetCall(0, func(c *%s) bool { return true })
	return i.currentCall
}

`, gen.impName, gen.callName, gen.callName)
}

func (gen *codeGenerator) generateConstructor() {
	gen.pf(`func New%s(t *testing.T) *%s {
	imp := &%s{
		t: t,
		callChan: make(chan *%s, 1),
	}
	imp.Mock = &%s{imp: imp}
	imp.ExpectCallTo = &%s{imp: imp}
	return imp
}

`, gen.impName, gen.impName, gen.impName, gen.callName, gen.mockName, gen.expectCallToName)
}

// Helper functions

func hasParams(ftype *ast.FuncType) bool {
	return ftype.Params != nil && len(ftype.Params.List) > 0
}

func hasResults(ftype *ast.FuncType) bool {
	return ftype.Results != nil && len(ftype.Results.List) > 0
}

// getParamFieldName returns the struct field name for a parameter.
// For named params, returns the name. For unnamed params, generates a name based on type/index.
func getParamFieldName(param *ast.Field, nameIdx int, unnamedIdx int, paramType string, totalParams int) string {
	if len(param.Names) > 0 {
		return param.Names[nameIdx].Name
	}

	return generateParamName(unnamedIdx, paramType, totalParams)
}

func countFields(fields *ast.FieldList) int {
	total := 0

	for _, field := range fields.List {
		if len(field.Names) > 0 {
			total += len(field.Names)
		} else {
			total++
		}
	}

	return total
}

func extractParamNames(ftype *ast.FuncType) []string {
	paramNames := make([]string, 0)
	if !hasParams(ftype) {
		return paramNames
	}

	paramIndex := 0

	for _, param := range ftype.Params.List {
		if len(param.Names) > 0 {
			for _, name := range param.Names {
				paramNames = append(paramNames, name.Name)
			}
		} else {
			paramName := fmt.Sprintf("param%d", paramIndex)
			paramNames = append(paramNames, paramName)
			paramIndex++
		}
	}

	return paramNames
}

// renderFieldList renders a *ast.FieldList as Go code for return types.
func renderFieldList(fset *token.FileSet, fieldList *ast.FieldList) string {
	if fieldList == nil || len(fieldList.List) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("(")

	for i, field := range fieldList.List {
		if i > 0 {
			buf.WriteString(", ")
		}
		// Names
		for j, name := range field.Names {
			if j > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(name.Name)
		}
		// Type
		if len(field.Names) > 0 {
			buf.WriteString(" ")
		}

		buf.WriteString(exprToString(fset, field.Type))
	}

	buf.WriteString(")")

	return buf.String()
}

// exprToString renders an ast.Expr to Go code.
func exprToString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, expr)

	return buf.String()
}

// generateParamName generates a field name for an unnamed parameter
// Uses common conventions: single string -> "S", single int -> "Input", multiple -> "A", "B", "C", etc.
func generateParamName(index int, paramType string, totalParams int) string {
	// Remove common prefixes/suffixes for comparison
	normalized := strings.TrimSpace(paramType)

	// Single parameter cases
	if totalParams == 1 {
		if normalized == "string" {
			return "S"
		}

		if normalized == "int" {
			return "I"
		}
	}

	// Multiple parameters - use A, B, C, etc.
	names := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	if index < len(names) {
		return names[index]
	}

	// Fallback
	return fmt.Sprintf("Arg%d", index)
}
