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

// Run executes the impgen tool logic. It takes command-line arguments, an environment variable getter, and a FileSystem
// interface for file operations. It returns an error if any step fails. On success, it generates a Go source file
// implementing the specified interface, in the calling test package.
func Run(args []string, getEnv func(string) string, fileSys FileSystem, pkgLoader PackageLoader) error {
	info, err := getGeneratorInfo(args, getEnv)
	if err != nil {
		return err
	}

	pkgImportPath, matchName, err := getPackageAndMatchName(info, pkgLoader)
	if err != nil {
		return err
	}

	astFiles, fset, err := pkgLoader.Load(pkgImportPath)
	if err != nil {
		return fmt.Errorf("failed to load package %q: %w", pkgImportPath, err)
	}

	iface := getMatchingInterfaceFromAST(astFiles, matchName)
	if iface == nil {
		return fmt.Errorf("%w: named %q in package %q", errInterfaceNotFound, matchName, pkgImportPath)
	}

	code, err := generateImplementationCode(iface, info, fset)
	if err != nil {
		return err
	}

	return writeGeneratedCodeToFile(code, info.impName, info.pkgName, fileSys)
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
	pkgName, matchName, impName string
}

// cliArgs defines the command-line arguments for the generator.
type cliArgs struct {
	Interface string `arg:"positional,required" help:"interface name to implement (e.g. MyInterface or pkg.MyInterface)"`
	Name      string `arg:"--name"              help:"name for the generated implementation (defaults to <Interface>Imp)"`
}

// getGeneratorInfo gathers basic information about the generator call.
func getGeneratorInfo(args []string, getEnv func(string) string) (generatorInfo, error) {
	pkgName := getEnv("GOPACKAGE")

	parsed, err := parseArgs(args)
	if err != nil {
		return generatorInfo{}, err
	}

	matchName := parsed.Interface
	impName := parsed.Name

	// set impname if not provided
	if impName == "" {
		impName = matchName + "Imp" // default implementation name
	}

	return generatorInfo{pkgName: pkgName, matchName: matchName, impName: impName}, nil
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

// getPackageAndMatchName determines the import path and interface name to match.
func getPackageAndMatchName(info generatorInfo, pkgLoader PackageLoader) (string, string, error) {
	matchName := info.matchName
	// Check if matchName contains a dot, e.g. "run.ExampleInt"
	if dot := strings.Index(matchName, "."); dot != -1 {
		targetPkgImport := matchName[:dot]
		matchName = matchName[dot+1:]
		// Resolve the full import path for the target package
		astFiles, _, err := pkgLoader.Load(".")
		if err != nil {
			return "", "", fmt.Errorf("failed to load local package: %w", err)
		}

		for _, fileAst := range astFiles {
			for _, imp := range fileAst.Imports {
				importPath, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					continue
				}
				// Check if the last segment matches the targetPkgImport
				parts := strings.Split(importPath, "/")
				if len(parts) > 0 && parts[len(parts)-1] == targetPkgImport {
					return importPath, matchName, nil
				}
			}
		}

		return "", "", fmt.Errorf("%w: %q", errPackageNotFound, targetPkgImport)
	}

	return ".", matchName, nil
}

// getMatchingInterfaceFromAST finds the interface by name in the ASTs.
func getMatchingInterfaceFromAST(astFiles []*ast.File, matchName string) *ast.InterfaceType {
	for _, fileAst := range astFiles {
		var found *ast.InterfaceType

		ast.Inspect(fileAst, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if ok {
				if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 && ts.Name.Name == matchName {
					found = iface
					return false
				}
			}

			return true
		})

		if found != nil {
			return found
		}
	}

	return nil
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
func (gen *codeGenerator) p(format string, args ...interface{}) {
	fmt.Fprintf(&gen.buf, format, args...)
}

func (gen *codeGenerator) generateHeader() {
	gen.p(`package %s

// Code generated by impgen. DO NOT EDIT.

import "sync"
import "testing"
import "time"

`, gen.pkgName)
}

func (gen *codeGenerator) generateMockStruct() {
	gen.p("type %s struct {\n", gen.mockName)
	gen.p("\timp *%s\n", gen.impName)
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateMainStruct() {
	gen.p("type %s struct {\n", gen.impName)
	gen.buf.WriteString("\tt *testing.T\n")
	gen.p("\tMock *%s\n", gen.mockName)
	gen.p("\tcallChan chan *%s\n", gen.callName)
	gen.p("\tExpectCallTo *%s\n", gen.expectCallToName)
	gen.p("\tcurrentCall *%s\n", gen.callName)
	gen.p("\tcallQueue []*%s\n", gen.callName)
	gen.buf.WriteString("\tqueueLock sync.Mutex\n")
	gen.buf.WriteString("}\n\n")
}

// methodCallName returns the call struct name for a method (e.g. "MyImpDoSomethingCall").
func (gen *codeGenerator) methodCallName(methodName string) string {
	return gen.impName + methodName + "Call"
}

// writeMethodSignature writes the method name and parameters (e.g., "MethodName(a int, b string)").
func (gen *codeGenerator) writeMethodSignature(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.buf.WriteString(methodName)
	gen.buf.WriteString("(")
	gen.writeMethodParams(ftype, paramNames)
	gen.buf.WriteString(")")
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
	gen.p("type %s struct {\n", callName)
	gen.p("\tresponseChan chan %sResponse\n", callName)
	gen.buf.WriteString("\tdone bool\n")

	if hasParams(ftype) {
		totalParams := countFields(ftype.Params)
		unnamedIndex := 0

		for _, param := range ftype.Params.List {
			paramType := exprToString(gen.fset, param.Type)
			if len(param.Names) > 0 {
				for i := range param.Names {
					fieldName := getParamFieldName(param, i, unnamedIndex, paramType, totalParams)
					gen.p("\t%s %s\n", fieldName, paramType)
				}
			} else {
				fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
				gen.p("\t%s %s\n", fieldName, paramType)
				unnamedIndex++
			}
		}
	}

	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateMethodResponseStruct(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	gen.p("type %sResponse struct {\n", callName)
	gen.buf.WriteString("\tType string // \"return\", \"panic\", or \"resolve\"\n")

	if hasResults(ftype) {
		returnIndex := 0

		for _, result := range ftype.Results.List {
			resultType := exprToString(gen.fset, result.Type)
			if len(result.Names) > 0 {
				for _, name := range result.Names {
					gen.p("\t%s %s\n", name.Name, resultType)

					returnIndex++
				}
			} else {
				gen.p("\tResult%d %s\n", returnIndex, resultType)
				returnIndex++
			}
		}
	}

	gen.buf.WriteString("\tPanicValue interface{}\n")
	gen.buf.WriteString("}\n\n")
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

	gen.buf.WriteString("\n")
}

func (gen *codeGenerator) generateInjectResultMethod(methodCallName string, ftype *ast.FuncType) {
	resultType := exprToString(gen.fset, ftype.Results.List[0].Type)
	gen.p("func (c *%s) InjectResult(result %s) {\n", methodCallName, resultType)
	gen.buf.WriteString("\tc.done = true\n")
	gen.p("\tc.responseChan <- %sResponse{Type: \"return\"", methodCallName)

	if len(ftype.Results.List[0].Names) > 0 {
		gen.p(", %s: result", ftype.Results.List[0].Names[0].Name)
	} else {
		gen.buf.WriteString(", Result0: result")
	}

	gen.buf.WriteString("}\n")
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateInjectResultsMethod(methodCallName string, ftype *ast.FuncType) {
	gen.p("func (c *%s) InjectResults(", methodCallName)

	returnIndex := 0

	returnParamNames := make([]string, 0)

	for _, result := range ftype.Results.List {
		resultType := exprToString(gen.fset, result.Type)
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				if returnIndex > 0 {
					gen.buf.WriteString(", ")
				}

				gen.p("%s %s", name.Name, resultType)
				returnParamNames = append(returnParamNames, name.Name)
				returnIndex++
			}
		} else {
			if returnIndex > 0 {
				gen.buf.WriteString(", ")
			}

			paramName := fmt.Sprintf("result%d", returnIndex)
			gen.p("%s %s", paramName, resultType)
			returnParamNames = append(returnParamNames, paramName)
			returnIndex++
		}
	}

	gen.buf.WriteString(") {\n")
	gen.buf.WriteString("\tc.done = true\n")
	gen.p("\tresp := %sResponse{Type: \"return\"", methodCallName)

	returnIndex = 0

	for _, result := range ftype.Results.List {
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				gen.p(", %s: %s", name.Name, returnParamNames[returnIndex])
				returnIndex++
			}
		} else {
			gen.p(", Result%d: %s", returnIndex, returnParamNames[returnIndex])
			returnIndex++
		}
	}

	gen.buf.WriteString("}\n")
	gen.buf.WriteString("\tc.responseChan <- resp\n")
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateInjectPanicMethod(methodCallName string) {
	gen.p("func (c *%s) InjectPanic(msg interface{}) {\n", methodCallName)
	gen.buf.WriteString("\tc.done = true\n")
	gen.p("\tc.responseChan <- %sResponse{Type: \"panic\", PanicValue: msg}\n", methodCallName)
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateResolveMethod(methodCallName string) {
	gen.p("func (c *%s) Resolve() {\n", methodCallName)
	gen.buf.WriteString("\tc.done = true\n")
	gen.p("\tc.responseChan <- %sResponse{Type: \"resolve\"}\n", methodCallName)
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateMockMethods() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateMockMethod(methodName, ftype)
	})
}

func (gen *codeGenerator) generateMockMethod(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	paramNames := extractParamNames(ftype)

	gen.p("func (m *%s) ", gen.mockName)
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.buf.WriteString(renderFieldList(gen.fset, ftype.Results))
	gen.buf.WriteString(" {\n")

	gen.p("\tresponseChan := make(chan %sResponse, 1)\n", callName)
	gen.buf.WriteString("\n")

	gen.p("\tcall := &%s{\n", callName)
	gen.buf.WriteString("\t\tresponseChan: responseChan,\n")
	gen.writeCallStructFields(ftype, paramNames)
	gen.buf.WriteString("\t}\n\n")

	gen.p("\tcallEvent := &%s{\n", gen.callName)
	gen.p("\t\t%s: call,\n", methodName)
	gen.buf.WriteString("\t}\n\n")

	gen.buf.WriteString("\tm.imp.callChan <- callEvent\n\n")
	gen.buf.WriteString("\tresp := <-responseChan\n\n")

	gen.buf.WriteString("\tif resp.Type == \"panic\" {\n")
	gen.buf.WriteString("\t\tpanic(resp.PanicValue)\n")
	gen.buf.WriteString("\t}\n\n")

	gen.writeReturnStatement(ftype)
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) writeMethodParams(ftype *ast.FuncType, paramNames []string) {
	if !hasParams(ftype) {
		return
	}

	paramNameIndex := 0

	for i, param := range ftype.Params.List {
		if i > 0 {
			gen.buf.WriteString(", ")
		}

		paramType := exprToString(gen.fset, param.Type)
		if len(param.Names) > 0 {
			for j, name := range param.Names {
				if j > 0 {
					gen.buf.WriteString(", ")
				}

				gen.p("%s %s", name.Name, paramType)

				paramNameIndex++
			}
		} else {
			gen.p("%s %s", paramNames[paramNameIndex], paramType)
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
				gen.p("\t\t%s: %s,\n", fieldName, name.Name)
				paramNameIndex++
			}
		} else {
			fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
			gen.p("\t\t%s: %s,\n", fieldName, paramNames[paramNameIndex])
			paramNameIndex++
			unnamedIndex++
		}
	}
}

func (gen *codeGenerator) writeReturnStatement(ftype *ast.FuncType) {
	if !hasResults(ftype) {
		gen.buf.WriteString("\treturn\n")
		return
	}

	gen.buf.WriteString("\treturn")

	returnIndex := 0

	for _, result := range ftype.Results.List {
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				if returnIndex > 0 {
					gen.buf.WriteString(", ")
				}

				gen.buf.WriteString(" resp." + name.Name)

				returnIndex++
			}
		} else {
			if returnIndex > 0 {
				gen.buf.WriteString(", ")
			}

			gen.p(" resp.Result%d", returnIndex)
			returnIndex++
		}
	}

	gen.buf.WriteString("\n")
}

func (gen *codeGenerator) generateCallStruct(methodNames []string) {
	gen.p("type %s struct {\n", gen.callName)

	for _, methodName := range methodNames {
		gen.p("\t%s *%s\n", methodName, gen.methodCallName(methodName))
	}

	gen.buf.WriteString("}\n\n")

	gen.generateCallNameMethod(methodNames)
	gen.generateCallDoneMethod(methodNames)
	gen.generateCallAsMethod(methodNames)
}

func (gen *codeGenerator) generateCallNameMethod(methodNames []string) {
	gen.p("func (c *%s) Name() string {\n", gen.callName)

	for _, methodName := range methodNames {
		gen.p("\tif c.%s != nil {\n", methodName)
		gen.p("\t\treturn %q\n", methodName)
		gen.buf.WriteString("\t}\n")
	}

	gen.buf.WriteString("\treturn \"\"\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateCallDoneMethod(methodNames []string) {
	gen.p("func (c *%s) Done() bool {\n", gen.callName)

	for _, methodName := range methodNames {
		gen.p("\tif c.%s != nil {\n", methodName)
		gen.p("\t\treturn c.%s.done\n", methodName)
		gen.buf.WriteString("\t}\n")
	}

	gen.buf.WriteString("\treturn false\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateCallAsMethod(methodNames []string) {
	for _, methodName := range methodNames {
		gen.p("func (c *%s) As%s() *%s { return c.%s }\n\n",
			gen.callName, methodName, gen.methodCallName(methodName), methodName)
	}
}

func (gen *codeGenerator) generateExpectCallToStruct() {
	gen.p("type %s struct {\n", gen.expectCallToName)
	gen.p("\timp *%s\n", gen.impName)
	gen.buf.WriteString("\ttimeout time.Duration\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateExpectCallToMethods() {
	gen.forEachMethod(func(methodName string, ftype *ast.FuncType) {
		gen.generateExpectCallToMethod(methodName, ftype)
	})
}

func (gen *codeGenerator) generateExpectCallToMethod(methodName string, ftype *ast.FuncType) {
	callName := gen.methodCallName(methodName)
	paramNames := extractParamNames(ftype)

	gen.p("func (e *%s) ", gen.expectCallToName)
	gen.writeMethodSignature(methodName, ftype, paramNames)
	gen.p(" *%s {\n", callName)

	gen.generateValidatorFunction(methodName, ftype, paramNames)

	gen.buf.WriteString("\tcall := e.imp.GetCall(e.timeout, validator)\n")
	gen.p("\treturn call.As%s()\n", methodName)
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateValidatorFunction(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.p("\tvalidator := func(c *%s) bool {\n", gen.callName)
	gen.p("\t\tif c.Name() != %q {\n", methodName)
	gen.buf.WriteString("\t\t\treturn false\n")
	gen.buf.WriteString("\t\t}\n")

	if hasParams(ftype) {
		gen.p("\t\tmethodCall := c.As%s()\n", methodName)
		gen.writeValidatorChecks(ftype, paramNames)
	}

	gen.buf.WriteString("\t\treturn true\n")
	gen.buf.WriteString("\t}\n\n")
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
				gen.p("\t\tif methodCall.%s != %s {\n", fieldName, name.Name)
				gen.buf.WriteString("\t\t\treturn false\n")
				gen.buf.WriteString("\t\t}\n")
				paramNameIndex++
			}
		} else {
			fieldName := getParamFieldName(param, 0, unnamedIndex, paramType, totalParams)
			gen.p("\t\tif methodCall.%s != %s {\n", fieldName, paramNames[paramNameIndex])
			gen.buf.WriteString("\t\t\treturn false\n")
			gen.buf.WriteString("\t\t}\n")
			paramNameIndex++
			unnamedIndex++
		}
	}
}

func (gen *codeGenerator) generateTimedStruct() {
	gen.p(`type %s struct {
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
	gen.p(`func (i *%s) GetCall(d time.Duration, validator func(*%s) bool) *%s {
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
	gen.p(`func (i *%s) GetCurrentCall() *%s {
	if i.currentCall != nil && !i.currentCall.Done() {
		return i.currentCall
	}
	i.currentCall = i.GetCall(0, func(c *%s) bool { return true })
	return i.currentCall
}

`, gen.impName, gen.callName, gen.callName)
}

func (gen *codeGenerator) generateConstructor() {
	gen.p(`func New%s(t *testing.T) *%s {
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
