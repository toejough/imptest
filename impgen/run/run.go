// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Run executes the impgen tool logic. It takes command-line arguments, an environment variable getter, and a FileSystem
// interface for file operations. It returns an error if any step fails. On success, it generates a Go source file
// implementing the specified interface, in the calling test package.
func Run(args []string, getEnv func(string) string, fileSys FileSystem) error {
	info := getGeneratorInfo(args, getEnv, fileSys)
	// fmt.Printf("Generator info: %+v\n", info)

	pkgImportPath, matchName := getPackageAndMatchName(info, fileSys)
	// fmt.Printf("Target package import path: %q, matchName: %q\n", pkgImportPath, matchName)

	// set impname if not provided
	if info.impName == "" {
		info.impName = matchName + "Imp" // default implementation name
	}

	astFiles, fset := parsePackageAST(pkgImportPath, info.pkgDir, fileSys)
	// fmt.Printf("Parsed %d AST files for package %q\n", len(astFiles), pkgImportPath)

	iface := getMatchingInterfaceFromAST(astFiles, matchName)
	if iface == nil {
		return fmt.Errorf("%w: named %q in package %q", errInterfaceNotFound, matchName, pkgImportPath)
	}

	// fmt.Printf("Found interface %q in package %q:\n", matchName, pkgImportPath)
	// printAstTree(iface, "  ")

	code := generateImplementationCode(iface, info, fset)
	// fmt.Printf("Generated implementation code:\n%s\n", code)

	return writeGeneratedCodeToFile(code, info.impName, info.pkgName, fileSys)
}

// FileSystem interface for mocking.
type FileSystem interface {
	Getwd() (string, error)
	ReadDir(name string) ([]os.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// GeneratorInfo holds information gathered for generation.
type GeneratorInfo struct {
	pkgDir, pkgName, matchName, impName string
}

// getGeneratorInfo gathers basic information about the generator call.
func getGeneratorInfo(args []string, getEnv func(string) string, fileSys FileSystem) GeneratorInfo {
	cwd, err := fileSys.Getwd()
	if err != nil {
		panic(err)
	}

	pkgName := getEnv("GOPACKAGE")

	pkgDir := cwd // assume current dir is the package dir
	matchName := ""
	impName := ""

	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	for i := 0; i < len(cmdArgs); i++ {
		if cmdArgs[i] == "--name" && i+1 < len(cmdArgs) {
			impName = cmdArgs[i+1]
			i++
		} else {
			matchName = cmdArgs[i]
		}
	}

	return GeneratorInfo{pkgDir: pkgDir, pkgName: pkgName, matchName: matchName, impName: impName}
}

// getPackageAndMatchName determines the import path and interface name to match.
func getPackageAndMatchName(info GeneratorInfo, fileSys FileSystem) (string, string) {
	matchName := info.matchName
	// Check if matchName contains a dot, e.g. "run.ExampleInt"
	if dot := strings.Index(matchName, "."); dot != -1 {
		targetPkgImport := matchName[:dot]
		matchName = matchName[dot+1:]
		// Resolve the full import path for the target package
		astFiles, _ := parsePackageFiles(info.pkgDir, fileSys)
		for _, fileAst := range astFiles {
			for _, imp := range fileAst.Imports {
				importPath, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					continue
				}
				// Check if the last segment matches the targetPkgImport
				parts := strings.Split(importPath, "/")
				if len(parts) > 0 && parts[len(parts)-1] == targetPkgImport {
					return importPath, matchName
				}
			}
		}

		return "", matchName
	}

	return info.pkgDir, matchName
}

// parsePackageAST loads and parses the AST for the given package import path.
func parsePackageAST(pkgImportPath, pkgDir string, fileSys FileSystem) ([]*ast.File, *token.FileSet) {
	if pkgImportPath == pkgDir || pkgImportPath == "" {
		return parsePackageFiles(pkgDir, fileSys)
	}

	cfg := &packages.Config{Mode: packages.LoadAllSyntax}

	pkgs, err := packages.Load(cfg, pkgImportPath)
	if err != nil || len(pkgs) == 0 || pkgs[0].Errors != nil {
		fmt.Printf("error loading package %q: %v\n", pkgImportPath, err)
		return nil, token.NewFileSet()
	}

	return pkgs[0].Syntax, pkgs[0].Fset
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

var errInterfaceNotFound = errors.New("interface not found")

// generateImplementationCode creates the Go code for the interface implementation.
func generateImplementationCode(
	identifiedInterface *ast.InterfaceType,
	info GeneratorInfo,
	fset *token.FileSet,
) string {
	impName := info.impName
	if impName == "" {
		impName = "interfaceImplementation"
	}

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
		fmt.Printf("error formatting generated code: %v\n", err)
		return gen.buf.String()
	}

	return string(formatted)
}

// writeGeneratedCodeToFile writes the generated code to <impName>.go.
func writeGeneratedCodeToFile(code string, impName string, pkgName string, fileSys FileSystem) error {
	const generatedFilePermissions = 0o600

	filename := "generated.go"
	if impName != "" {
		filename = impName
		// If we're in a test package, append _test to the filename
		if strings.HasSuffix(pkgName, "_test") && !strings.HasSuffix(impName, "_test") {
			filename = strings.TrimSuffix(impName, ".go") + "_test.go"
		} else if !strings.HasSuffix(filename, ".go") {
			filename += ".go"
		}
	}

	err := fileSys.WriteFile(filename, []byte(code), generatedFilePermissions)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	fmt.Printf("%s written successfully.\n", filename)

	return nil
}

// parsePackageFiles reads and parses all Go files in the package directory.
func parsePackageFiles(pkgDir string, fileSys FileSystem) ([]*ast.File, *token.FileSet) {
	entries, err := fileSys.ReadDir(pkgDir)
	if err != nil {
		fmt.Printf("  error reading package dir %q: %v\n", pkgDir, err)
		return nil, token.NewFileSet()
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) > 3 && name[len(name)-3:] == ".go" && name != "generated.go" {
			files = append(files, filepath.Join(pkgDir, name))
		}
	}

	fset := token.NewFileSet()

	astFiles := make([]*ast.File, 0, len(files))

	for _, file := range files {
		data, err := fileSys.ReadFile(file)
		if err != nil {
			fmt.Printf("  error reading file %q: %v\n", file, err)
			continue
		}

		parsedFile, err := parser.ParseFile(fset, file, data, parser.ParseComments)
		if err != nil {
			fmt.Printf("  error parsing file %q: %v\n", file, err)
			continue
		}

		astFiles = append(astFiles, parsedFile)
	}

	return astFiles, fset
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

func (gen *codeGenerator) generateHeader() {
	gen.buf.WriteString(fmt.Sprintf("package %s\n\n", gen.pkgName))
	gen.buf.WriteString("// Code generated by generate.go. DO NOT EDIT.\n\n")
	gen.buf.WriteString("import \"sync\"\n")
	gen.buf.WriteString("import \"testing\"\n")
	gen.buf.WriteString("import \"time\"\n\n")
}

func (gen *codeGenerator) generateMockStruct() {
	gen.buf.WriteString(fmt.Sprintf("type %s struct {\n", gen.mockName))
	gen.buf.WriteString(fmt.Sprintf("\timp *%s\n", gen.impName))
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateMainStruct() {
	gen.buf.WriteString(fmt.Sprintf("type %s struct {\n", gen.impName))
	gen.buf.WriteString("\tt *testing.T\n")
	gen.buf.WriteString(fmt.Sprintf("\tMock *%s\n", gen.mockName))
	gen.buf.WriteString(fmt.Sprintf("\tcallChan chan *%s\n", gen.callName))
	gen.buf.WriteString(fmt.Sprintf("\tExpectCallTo *%s\n", gen.expectCallToName))
	gen.buf.WriteString(fmt.Sprintf("\tcurrentCall *%s\n", gen.callName))
	gen.buf.WriteString(fmt.Sprintf("\tcallQueue []*%s\n", gen.callName))
	gen.buf.WriteString("\tqueueLock sync.Mutex\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateMethodStructs() []string {
	var methodNames []string

	for _, field := range gen.identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			methodNames = append(methodNames, methodName.Name)
			gen.generateMethodCallStruct(methodName.Name, ftype)
			gen.generateMethodResponseStruct(methodName.Name, ftype)
			gen.generateMethodResponseMethods(methodName.Name, ftype)
		}
	}

	return methodNames
}

func (gen *codeGenerator) generateMethodCallStruct(methodName string, ftype *ast.FuncType) {
	methodCallName := gen.impName + methodName + "Call"
	gen.buf.WriteString(fmt.Sprintf("type %s struct {\n", methodCallName))
	gen.buf.WriteString(fmt.Sprintf("\tresponseChan chan %sResponse\n", methodCallName))
	gen.buf.WriteString("\tdone bool\n")

	if ftype.Params != nil && len(ftype.Params.List) > 0 {
		totalParams := countTotalParams(ftype.Params)
		paramIndex := 0

		for _, param := range ftype.Params.List {
			paramType := exprToString(gen.fset, param.Type)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					gen.buf.WriteString(fmt.Sprintf("\t%s %s\n", name.Name, paramType))
				}
			} else {
				fieldName := generateParamName(paramIndex, paramType, totalParams)
				gen.buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, paramType))

				paramIndex++
			}
		}
	}

	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateMethodResponseStruct(methodName string, ftype *ast.FuncType) {
	methodCallName := gen.impName + methodName + "Call"
	gen.buf.WriteString(fmt.Sprintf("type %sResponse struct {\n", methodCallName))
	gen.buf.WriteString("\tType string // \"return\", \"panic\", or \"resolve\"\n")

	if ftype.Results != nil && len(ftype.Results.List) > 0 {
		returnIndex := 0

		for _, result := range ftype.Results.List {
			resultType := exprToString(gen.fset, result.Type)
			if len(result.Names) > 0 {
				for _, name := range result.Names {
					gen.buf.WriteString(fmt.Sprintf("\t%s %s\n", name.Name, resultType))

					returnIndex++
				}
			} else {
				gen.buf.WriteString(fmt.Sprintf("\tResult%d %s\n", returnIndex, resultType))
				returnIndex++
			}
		}
	}

	gen.buf.WriteString("\tPanicValue interface{}\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateMethodResponseMethods(methodName string, ftype *ast.FuncType) {
	methodCallName := gen.impName + methodName + "Call"

	if ftype.Results != nil && len(ftype.Results.List) > 0 {
		totalReturns := countTotalReturns(ftype.Results)

		if totalReturns == 1 {
			gen.generateInjectResultMethod(methodCallName, ftype)
		} else {
			gen.generateInjectResultsMethod(methodCallName, ftype)
		}

		gen.generateInjectPanicMethod(methodCallName)
	} else {
		gen.generateResolveMethod(methodCallName)
		gen.generateInjectPanicMethod(methodCallName)
	}

	gen.buf.WriteString("\n")
}

func (gen *codeGenerator) generateInjectResultMethod(methodCallName string, ftype *ast.FuncType) {
	resultType := exprToString(gen.fset, ftype.Results.List[0].Type)
	gen.buf.WriteString(fmt.Sprintf("func (c *%s) InjectResult(result %s) {\n", methodCallName, resultType))
	gen.buf.WriteString("\tc.done = true\n")
	gen.buf.WriteString(fmt.Sprintf("\tc.responseChan <- %sResponse{Type: \"return\"", methodCallName))

	if len(ftype.Results.List[0].Names) > 0 {
		gen.buf.WriteString(fmt.Sprintf(", %s: result", ftype.Results.List[0].Names[0].Name))
	} else {
		gen.buf.WriteString(", Result0: result")
	}

	gen.buf.WriteString("}\n")
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateInjectResultsMethod(methodCallName string, ftype *ast.FuncType) {
	gen.buf.WriteString(fmt.Sprintf("func (c *%s) InjectResults(", methodCallName))

	returnIndex := 0

	returnParamNames := make([]string, 0)

	for _, result := range ftype.Results.List {
		resultType := exprToString(gen.fset, result.Type)
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				if returnIndex > 0 {
					gen.buf.WriteString(", ")
				}

				gen.buf.WriteString(fmt.Sprintf("%s %s", name.Name, resultType))
				returnParamNames = append(returnParamNames, name.Name)
				returnIndex++
			}
		} else {
			if returnIndex > 0 {
				gen.buf.WriteString(", ")
			}

			paramName := fmt.Sprintf("result%d", returnIndex)
			gen.buf.WriteString(fmt.Sprintf("%s %s", paramName, resultType))
			returnParamNames = append(returnParamNames, paramName)
			returnIndex++
		}
	}

	gen.buf.WriteString(") {\n")
	gen.buf.WriteString("\tc.done = true\n")
	gen.buf.WriteString(fmt.Sprintf("\tresp := %sResponse{Type: \"return\"", methodCallName))

	returnIndex = 0

	for _, result := range ftype.Results.List {
		if len(result.Names) > 0 {
			for _, name := range result.Names {
				gen.buf.WriteString(fmt.Sprintf(", %s: %s", name.Name, returnParamNames[returnIndex]))
				returnIndex++
			}
		} else {
			gen.buf.WriteString(fmt.Sprintf(", Result%d: %s", returnIndex, returnParamNames[returnIndex]))
			returnIndex++
		}
	}

	gen.buf.WriteString("}\n")
	gen.buf.WriteString("\tc.responseChan <- resp\n")
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateInjectPanicMethod(methodCallName string) {
	gen.buf.WriteString(fmt.Sprintf("func (c *%s) InjectPanic(msg interface{}) {\n", methodCallName))
	gen.buf.WriteString("\tc.done = true\n")
	gen.buf.WriteString(fmt.Sprintf("\tc.responseChan <- %sResponse{Type: \"panic\", PanicValue: msg}\n", methodCallName))
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateResolveMethod(methodCallName string) {
	gen.buf.WriteString(fmt.Sprintf("func (c *%s) Resolve() {\n", methodCallName))
	gen.buf.WriteString("\tc.done = true\n")
	gen.buf.WriteString(fmt.Sprintf("\tc.responseChan <- %sResponse{Type: \"resolve\"}\n", methodCallName))
	gen.buf.WriteString("}\n")
}

func (gen *codeGenerator) generateMockMethods() {
	for _, field := range gen.identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			gen.generateMockMethod(methodName.Name, ftype)
		}
	}
}

func (gen *codeGenerator) generateMockMethod(methodName string, ftype *ast.FuncType) {
	methodCallName := gen.impName + methodName + "Call"
	paramNames := extractParamNames(ftype)

	gen.buf.WriteString(fmt.Sprintf("func (m *%s) ", gen.mockName))
	gen.buf.WriteString(methodName)
	gen.buf.WriteString("(")
	gen.writeMethodParams(ftype, paramNames)
	gen.buf.WriteString(")")
	gen.buf.WriteString(renderFieldList(gen.fset, ftype.Results, false))
	gen.buf.WriteString(" {\n")

	gen.buf.WriteString(fmt.Sprintf("\tresponseChan := make(chan %sResponse, 1)\n", methodCallName))
	gen.buf.WriteString("\n")

	gen.buf.WriteString(fmt.Sprintf("\tcall := &%s{\n", methodCallName))
	gen.buf.WriteString("\t\tresponseChan: responseChan,\n")
	gen.writeCallStructFields(ftype, paramNames)
	gen.buf.WriteString("\t}\n\n")

	gen.buf.WriteString(fmt.Sprintf("\tcallEvent := &%s{\n", gen.callName))
	gen.buf.WriteString(fmt.Sprintf("\t\t%s: call,\n", methodName))
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
	if ftype.Params == nil || len(ftype.Params.List) == 0 {
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

				gen.buf.WriteString(fmt.Sprintf("%s %s", name.Name, paramType))

				paramNameIndex++
			}
		} else {
			gen.buf.WriteString(fmt.Sprintf("%s %s", paramNames[paramNameIndex], paramType))
			paramNameIndex++
		}
	}
}

func (gen *codeGenerator) writeCallStructFields(ftype *ast.FuncType, paramNames []string) {
	if ftype.Params == nil || len(ftype.Params.List) == 0 {
		return
	}

	totalParams := countTotalParams(ftype.Params)
	paramNameIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(gen.fset, param.Type)
		if len(param.Names) > 0 {
			for _, name := range param.Names {
				gen.buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n", name.Name, name.Name))

				paramNameIndex++
			}
		} else {
			unnamedIndex := calculateUnnamedIndex(ftype.Params, param)
			fieldName := generateParamName(unnamedIndex, paramType, totalParams)
			gen.buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n", fieldName, paramNames[paramNameIndex]))
			paramNameIndex++
		}
	}
}

func (gen *codeGenerator) writeReturnStatement(ftype *ast.FuncType) {
	if ftype.Results == nil || len(ftype.Results.List) == 0 {
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

			gen.buf.WriteString(fmt.Sprintf(" resp.Result%d", returnIndex))
			returnIndex++
		}
	}

	gen.buf.WriteString("\n")
}

func (gen *codeGenerator) generateCallStruct(methodNames []string) {
	gen.buf.WriteString(fmt.Sprintf("type %s struct {\n", gen.callName))

	for _, methodName := range methodNames {
		methodCallName := gen.impName + methodName + "Call"
		gen.buf.WriteString(fmt.Sprintf("\t%s *%s\n", methodName, methodCallName))
	}

	gen.buf.WriteString("}\n\n")

	gen.generateCallNameMethod(methodNames)
	gen.generateCallDoneMethod(methodNames)
	gen.generateCallAsMethod(methodNames)
}

func (gen *codeGenerator) generateCallNameMethod(methodNames []string) {
	gen.buf.WriteString(fmt.Sprintf("func (c *%s) Name() string {\n", gen.callName))

	for _, methodName := range methodNames {
		gen.buf.WriteString(fmt.Sprintf("\tif c.%s != nil {\n", methodName))
		gen.buf.WriteString(fmt.Sprintf("\t\treturn %q\n", methodName))
		gen.buf.WriteString("\t}\n")
	}

	gen.buf.WriteString("\treturn \"\"\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateCallDoneMethod(methodNames []string) {
	gen.buf.WriteString(fmt.Sprintf("func (c *%s) Done() bool {\n", gen.callName))

	for _, methodName := range methodNames {
		gen.buf.WriteString(fmt.Sprintf("\tif c.%s != nil {\n", methodName))
		gen.buf.WriteString(fmt.Sprintf("\t\treturn c.%s.done\n", methodName))
		gen.buf.WriteString("\t}\n")
	}

	gen.buf.WriteString("\treturn false\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateCallAsMethod(methodNames []string) {
	for _, methodName := range methodNames {
		methodCallName := gen.impName + methodName + "Call"
		gen.buf.WriteString(fmt.Sprintf(
			"func (c *%s) As%s() *%s { return c.%s }\n\n",
			gen.callName, methodName, methodCallName, methodName,
		))
	}
}

func (gen *codeGenerator) generateExpectCallToStruct() {
	gen.buf.WriteString(fmt.Sprintf("type %s struct {\n", gen.expectCallToName))
	gen.buf.WriteString(fmt.Sprintf("\timp *%s\n", gen.impName))
	gen.buf.WriteString("\ttimeout time.Duration\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateExpectCallToMethods() {
	for _, field := range gen.identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			gen.generateExpectCallToMethod(methodName.Name, ftype)
		}
	}
}

func (gen *codeGenerator) generateExpectCallToMethod(methodName string, ftype *ast.FuncType) {
	methodCallName := gen.impName + methodName + "Call"
	paramNames := extractParamNames(ftype)

	gen.buf.WriteString(fmt.Sprintf("func (e *%s) ", gen.expectCallToName))
	gen.buf.WriteString(methodName)
	gen.buf.WriteString("(")
	gen.writeMethodParams(ftype, paramNames)
	gen.buf.WriteString(")")
	gen.buf.WriteString(fmt.Sprintf(" *%s {\n", methodCallName))

	gen.generateValidatorFunction(methodName, ftype, paramNames)

	gen.buf.WriteString("\tcall := e.imp.GetCall(e.timeout, validator)\n")
	gen.buf.WriteString(fmt.Sprintf("\treturn call.As%s()\n", methodName))
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateValidatorFunction(methodName string, ftype *ast.FuncType, paramNames []string) {
	gen.buf.WriteString(fmt.Sprintf("\tvalidator := func(c *%s) bool {\n", gen.callName))
	gen.buf.WriteString(fmt.Sprintf("\t\tif c.Name() != %q {\n", methodName))
	gen.buf.WriteString("\t\t\treturn false\n")
	gen.buf.WriteString("\t\t}\n")

	if ftype.Params != nil && len(ftype.Params.List) > 0 {
		gen.buf.WriteString(fmt.Sprintf("\t\tmethodCall := c.As%s()\n", methodName))
		gen.writeValidatorChecks(ftype, paramNames)
	}

	gen.buf.WriteString("\t\treturn true\n")
	gen.buf.WriteString("\t}\n\n")
}

func (gen *codeGenerator) writeValidatorChecks(ftype *ast.FuncType, paramNames []string) {
	totalParams := countTotalParams(ftype.Params)
	paramNameIndex := 0

	for _, param := range ftype.Params.List {
		paramType := exprToString(gen.fset, param.Type)
		if len(param.Names) > 0 {
			for _, name := range param.Names {
				gen.buf.WriteString(fmt.Sprintf("\t\tif methodCall.%s != %s {\n", name.Name, name.Name))
				gen.buf.WriteString("\t\t\treturn false\n")
				gen.buf.WriteString("\t\t}\n")

				paramNameIndex++
			}
		} else {
			unnamedIndex := calculateUnnamedIndex(ftype.Params, param)
			fieldName := generateParamName(unnamedIndex, paramType, totalParams)
			gen.buf.WriteString(fmt.Sprintf("\t\tif methodCall.%s != %s {\n", fieldName, paramNames[paramNameIndex]))
			gen.buf.WriteString("\t\t\treturn false\n")
			gen.buf.WriteString("\t\t}\n")

			paramNameIndex++
		}
	}
}

func (gen *codeGenerator) generateTimedStruct() {
	gen.buf.WriteString(fmt.Sprintf("type %s struct {\n", gen.timedName))
	gen.buf.WriteString(fmt.Sprintf("\tExpectCallTo *%s\n", gen.expectCallToName))
	gen.buf.WriteString("}\n\n")

	gen.buf.WriteString(fmt.Sprintf("func (i *%s) Within(d time.Duration) *%s {\n", gen.impName, gen.timedName))
	gen.buf.WriteString(fmt.Sprintf("\treturn &%s{\n", gen.timedName))
	gen.buf.WriteString(fmt.Sprintf("\t\tExpectCallTo: &%s{imp: i, timeout: d},\n", gen.expectCallToName))
	gen.buf.WriteString("\t}\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateGetCallMethod() {
	gen.buf.WriteString(fmt.Sprintf(
		"func (i *%s) GetCall(d time.Duration, validator func(*%s) bool) *%s {\n",
		gen.impName, gen.callName, gen.callName,
	))
	gen.buf.WriteString("\ti.queueLock.Lock()\n")
	gen.buf.WriteString("\tdefer i.queueLock.Unlock()\n\n")

	gen.buf.WriteString("\tfor index, call := range i.callQueue {\n")
	gen.buf.WriteString("\t\tif validator(call) {\n")
	gen.buf.WriteString("\t\t\t// Remove from queue\n")
	gen.buf.WriteString("\t\t\ti.callQueue = append(i.callQueue[:index], i.callQueue[index+1:]...)\n")
	gen.buf.WriteString("\t\t\treturn call\n")
	gen.buf.WriteString("\t\t}\n")
	gen.buf.WriteString("\t}\n\n")

	gen.buf.WriteString("\tvar timeoutChan <-chan time.Time\n")
	gen.buf.WriteString("\tif d > 0 {\n")
	gen.buf.WriteString("\t\ttimeoutChan = time.After(d)\n")
	gen.buf.WriteString("\t}\n\n")

	gen.buf.WriteString("\tfor {\n")
	gen.buf.WriteString("\t\tselect {\n")
	gen.buf.WriteString("\t\tcase call := <-i.callChan:\n")
	gen.buf.WriteString("\t\t\tif validator(call) {\n")
	gen.buf.WriteString("\t\t\t\treturn call\n")
	gen.buf.WriteString("\t\t\t}\n")
	gen.buf.WriteString("\t\t\t// Queue it\n")
	gen.buf.WriteString("\t\t\ti.callQueue = append(i.callQueue, call)\n")
	gen.buf.WriteString("\t\tcase <-timeoutChan:\n")
	gen.buf.WriteString("\t\t\ti.t.Fatalf(\"timeout waiting for call matching validator\")\n")
	gen.buf.WriteString("\t\t\treturn nil\n")
	gen.buf.WriteString("\t\t}\n")
	gen.buf.WriteString("\t}\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateGetCurrentCallMethod() {
	gen.buf.WriteString(fmt.Sprintf("func (i *%s) GetCurrentCall() *%s {\n", gen.impName, gen.callName))
	gen.buf.WriteString("\tif i.currentCall != nil && !i.currentCall.Done() {\n")
	gen.buf.WriteString("\t\treturn i.currentCall\n")
	gen.buf.WriteString("\t}\n")
	gen.buf.WriteString(fmt.Sprintf("\ti.currentCall = i.GetCall(0, func(c *%s) bool { return true })\n", gen.callName))
	gen.buf.WriteString("\treturn i.currentCall\n")
	gen.buf.WriteString("}\n\n")
}

func (gen *codeGenerator) generateConstructor() {
	gen.buf.WriteString(fmt.Sprintf("func New%s(t *testing.T) *%s {\n", gen.impName, gen.impName))
	gen.buf.WriteString(fmt.Sprintf("\timp := &%s{\n", gen.impName))
	gen.buf.WriteString("\t\tt: t,\n")
	gen.buf.WriteString(fmt.Sprintf("\t\tcallChan: make(chan *%s, 1),\n", gen.callName))
	gen.buf.WriteString("\t}\n")
	gen.buf.WriteString(fmt.Sprintf("\timp.Mock = &%s{imp: imp}\n", gen.mockName))
	gen.buf.WriteString(fmt.Sprintf("\timp.ExpectCallTo = &%s{imp: imp}\n", gen.expectCallToName))
	gen.buf.WriteString("\treturn imp\n")
	gen.buf.WriteString("}\n\n")
}

// Helper functions

func countTotalParams(params *ast.FieldList) int {
	totalParams := 0

	for _, param := range params.List {
		if len(param.Names) > 0 {
			totalParams += len(param.Names)
		} else {
			totalParams++
		}
	}

	return totalParams
}

func countTotalReturns(results *ast.FieldList) int {
	totalReturns := 0

	for _, result := range results.List {
		if len(result.Names) > 0 {
			totalReturns += len(result.Names)
		} else {
			totalReturns++
		}
	}

	return totalReturns
}

func extractParamNames(ftype *ast.FuncType) []string {
	paramNames := make([]string, 0)
	if ftype.Params == nil || len(ftype.Params.List) == 0 {
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

func calculateUnnamedIndex(params *ast.FieldList, targetParam *ast.Field) int {
	unnamedIndex := 0

	for _, p := range params.List {
		if len(p.Names) == 0 {
			if p == targetParam {
				break
			}

			unnamedIndex++
		}
	}

	return unnamedIndex
}

// renderFieldList renders a *ast.FieldList as Go code (params/results).
func renderFieldList(fset *token.FileSet, fieldList *ast.FieldList, isParams bool) string {
	if fieldList == nil || len(fieldList.List) == 0 {
		if isParams {
			return "()"
		}

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
			return "Input"
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
