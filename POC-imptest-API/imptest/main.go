//go:build ignore

package main

import (
	"bytes"
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

func main() {
	info := getGeneratorInfo()
	fmt.Printf("Generator info: %+v\n", info)

	pkgImportPath, matchName := getPackageAndMatchName(info)
	fmt.Printf("Target package import path: %q, matchName: %q\n", pkgImportPath, matchName)

	astFiles, fset := parsePackageAST(pkgImportPath, info.pkgDir)
	fmt.Printf("Parsed %d AST files for package %q\n", len(astFiles), pkgImportPath)

	iface := getMatchingInterfaceFromAST(astFiles, matchName)
	if iface == nil {
		fmt.Printf("No interface named %q found in package %q.\n", matchName, pkgImportPath)
		return
	}
	fmt.Printf("Found interface %q in package %q:\n", matchName, pkgImportPath)
	printAstTree(iface, "  ")

	code := generateImplementationCode(iface, info, fset)
	fmt.Printf("Generated implementation code:\n%s\n", code)

	writeGeneratedCodeToFile(code, info.impName)
}

// getGeneratorInfo gathers basic information about the generator call
func getGeneratorInfo() struct {
	cwd, pkgDir, pkgName, goFilePath, matchName, impName string
} {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	pkgName := os.Getenv("GOPACKAGE")
	goFile := os.Getenv("GOFILE")
	goFilePath := ""
	if goFile != "" {
		if filepath.IsAbs(goFile) {
			goFilePath = goFile
		} else {
			goFilePath = filepath.Join(cwd, goFile)
		}
	}
	pkgDir := cwd // assume current dir is the package dir
	matchName := ""
	impName := ""
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--name" && i+1 < len(args) {
			impName = args[i+1]
			i++
		} else {
			matchName = args[i]
		}
	}
	return struct {
		cwd, pkgDir, pkgName, goFilePath, matchName, impName string
	}{cwd, pkgDir, pkgName, goFilePath, matchName, impName}
}

// getPackageAndMatchName determines the import path and interface name to match
func getPackageAndMatchName(info struct {
	cwd, pkgDir, pkgName, goFilePath, matchName, impName string
}) (string, string) {
	matchName := info.matchName
	// Check if matchName contains a dot, e.g. "run.ExampleInt"
	if dot := strings.Index(matchName, "."); dot != -1 {
		targetPkgImport := matchName[:dot]
		matchName = matchName[dot+1:]
		// Resolve the full import path for the target package
		astFiles, _ := parsePackageFiles(info.pkgDir)
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

// parsePackageAST loads and parses the AST for the given package import path
func parsePackageAST(pkgImportPath, pkgDir string) ([]*ast.File, *token.FileSet) {
	if pkgImportPath == pkgDir || pkgImportPath == "" {
		return parsePackageFiles(pkgDir)
	}
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, pkgImportPath)
	if err != nil || len(pkgs) == 0 {
		fmt.Printf("error loading package %q: %v\n", pkgImportPath, err)
		return nil, token.NewFileSet()
	}
	return pkgs[0].Syntax, pkgs[0].Fset
}

// parsePackageFiles reads and parses all Go files in the package directory
func parsePackageFiles(pkgDir string) ([]*ast.File, *token.FileSet) {
	entries, err := os.ReadDir(pkgDir)
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
	var astFiles []*ast.File
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("  error reading file %q: %v\n", file, err)
			continue
		}
		f, err := parser.ParseFile(fset, file, data, parser.ParseComments)
		if err != nil {
			fmt.Printf("  error parsing file %q: %v\n", file, err)
			continue
		}
		astFiles = append(astFiles, f)
	}
	return astFiles, fset
}

// getMatchingInterfaceFromAST finds the interface by name in the ASTs
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

// generateImplementationCode creates the Go code for the interface implementation
func generateImplementationCode(identifiedInterface *ast.InterfaceType, info struct {
	cwd, pkgDir, pkgName, goFilePath, matchName, impName string
}, fset *token.FileSet) string {
	impName := info.impName
	if impName == "" {
		impName = "interfaceImplementation"
	}
	mockName := impName + "Mock"
	var buf bytes.Buffer
	buf.WriteString("package main\n\n")
	buf.WriteString("// Code generated by generate.go. DO NOT EDIT.\n\n")

	// Generate the Mock struct with the interface methods
	buf.WriteString(fmt.Sprintf("type %s struct{}\n\n", mockName))
	for _, field := range identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}
			buf.WriteString(fmt.Sprintf("func (%s) ", mockName))
			buf.WriteString(name.Name)
			buf.WriteString(renderFieldList(fset, ftype.Params, true))
			buf.WriteString(renderFieldList(fset, ftype.Results, false))
			buf.WriteString(" { panic(\"not implemented\") }\n\n")
		}
	}

	// Generate the main implementation struct with Mock field
	buf.WriteString(fmt.Sprintf("type %s struct {\n", impName))
	buf.WriteString(fmt.Sprintf("\tMock *%s\n", mockName))
	buf.WriteString("}\n\n")

	// Generate method-specific call structs first
	callName := impName + "Call"
	var methodNames []string
	for _, field := range identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}
		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			// Collect method name for later use
			methodNames = append(methodNames, methodName.Name)

			// Generate method-specific call struct (e.g., IntOpsImpAddCall)
			methodCallName := impName + methodName.Name + "Call"
			buf.WriteString(fmt.Sprintf("type %s struct {\n", methodCallName))

			// Generate fields for parameters
			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				// Count total number of parameters
				totalParams := 0
				for _, param := range ftype.Params.List {
					if len(param.Names) > 0 {
						totalParams += len(param.Names)
					} else {
						totalParams++
					}
				}

				paramIndex := 0
				for _, param := range ftype.Params.List {
					paramType := exprToString(fset, param.Type)
					if len(param.Names) > 0 {
						// Named parameters
						for _, name := range param.Names {
							buf.WriteString(fmt.Sprintf("\t%s %s\n", name.Name, paramType))
						}
					} else {
						// Unnamed parameters - generate names
						fieldName := generateParamName(paramIndex, paramType, totalParams)
						buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, paramType))
						paramIndex++
					}
				}
			}

			buf.WriteString("}\n\n")

			// Generate InjectResult, InjectPanic, and Resolve methods for the method call struct
			if ftype.Results != nil && len(ftype.Results.List) > 0 {
				// Count total return values
				totalReturns := 0
				for _, result := range ftype.Results.List {
					if len(result.Names) > 0 {
						totalReturns += len(result.Names)
					} else {
						totalReturns++
					}
				}

				if totalReturns == 1 {
					// Single return value - generate InjectResult
					resultType := exprToString(fset, ftype.Results.List[0].Type)
					buf.WriteString(fmt.Sprintf("func (c *%s) InjectResult(result %s) {}\n", methodCallName, resultType))
				} else {
					// Multiple return values - generate InjectResults
					buf.WriteString(fmt.Sprintf("func (c *%s) InjectResults(", methodCallName))
					returnIndex := 0
					for _, result := range ftype.Results.List {
						resultType := exprToString(fset, result.Type)
						if len(result.Names) > 0 {
							for _, name := range result.Names {
								if returnIndex > 0 {
									buf.WriteString(", ")
								}
								buf.WriteString(fmt.Sprintf("%s %s", name.Name, resultType))
								returnIndex++
							}
						} else {
							if returnIndex > 0 {
								buf.WriteString(", ")
							}
							buf.WriteString(fmt.Sprintf("result%d %s", returnIndex, resultType))
							returnIndex++
						}
					}
					buf.WriteString(") {}\n")
				}
				// Generate InjectPanic for methods with return values
				buf.WriteString(fmt.Sprintf("func (c *%s) InjectPanic(msg interface{}) {}\n", methodCallName))
			} else {
				// No return values - generate Resolve
				buf.WriteString(fmt.Sprintf("func (c *%s) Resolve() {}\n", methodCallName))
				// Generate InjectPanic for methods without return values
				buf.WriteString(fmt.Sprintf("func (c *%s) InjectPanic(msg interface{}) {}\n", methodCallName))
			}
			buf.WriteString("\n")
		}
	}

	// Generate the Call struct with fields for each method-specific call struct
	buf.WriteString(fmt.Sprintf("type %s struct {\n", callName))
	for _, methodName := range methodNames {
		methodCallName := impName + methodName + "Call"
		buf.WriteString(fmt.Sprintf("\t%s *%s\n", methodName, methodCallName))
	}
	buf.WriteString("}\n\n")
	buf.WriteString(fmt.Sprintf("func (c *%s) Name() string { return \"\" }\n\n", callName))

	// Generate As[MethodName] methods on the Call struct
	for _, methodName := range methodNames {
		methodCallName := impName + methodName + "Call"
		buf.WriteString(fmt.Sprintf("func (c *%s) As%s() *%s { return c.%s }\n\n", callName, methodName, methodCallName, methodName))
	}

	// Generate GetCurrentCall method
	buf.WriteString(fmt.Sprintf("func (i *%s) GetCurrentCall() *%s {\n", impName, callName))
	buf.WriteString(fmt.Sprintf("\treturn &%s{\n", callName))
	for _, methodName := range methodNames {
		methodCallName := impName + methodName + "Call"
		buf.WriteString(fmt.Sprintf("\t\t%s: &%s{},\n", methodName, methodCallName))
	}
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	// New[impName] constructor with *testing.T arg
	buf.WriteString(fmt.Sprintf("func New%s(t *testing.T) *%s {\n", impName, impName))
	buf.WriteString(fmt.Sprintf("\treturn &%s{\n", impName))
	buf.WriteString(fmt.Sprintf("\t\tMock: &%s{},\n", mockName))
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Printf("error formatting generated code: %v\n", err)
		return buf.String()
	}
	return string(formatted)
}

// writeGeneratedCodeToFile writes the generated code to <impName>.go
func writeGeneratedCodeToFile(code string, impName string) {
	filename := "generated.go"
	if impName != "" {
		filename = impName + ".go"
	}
	err := os.WriteFile(filename, []byte(code), 0644)
	if err != nil {
		fmt.Printf("error writing %s: %v\n", filename, err)
		return
	}
	fmt.Printf("%s written successfully.\n", filename)
}

// printAstTree recursively prints the AST node tree with indentation
func printAstTree(node interface{}, indent string) {
	switch n := node.(type) {
	case nil:
		return
	case *ast.Ident:
		typeName := fmt.Sprintf("%T", n)
		fmt.Printf("%s%s (Name: %q)\n", indent, typeName, n.Name)
		return
	case ast.Node:
		typeName := fmt.Sprintf("%T", n)
		fmt.Printf("%s%s\n", indent, typeName)
		indent2 := indent + "  "
		ast.Inspect(n, func(child ast.Node) bool {
			if child != n && child != nil {
				printAstTree(child, indent2)
				return false
			}
			return true
		})
	}
}

// printTypeSpecsWithInterface prints TypeSpec nodes whose immediate child is an InterfaceType
func printTypeSpecsWithInterface(node ast.Node, indent string) {
	ast.Inspect(node, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if ok {
			if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 {
				fmt.Printf("%s*ast.TypeSpec (Name: %q)\n", indent, ts.Name.Name)
				printAstTree(iface, indent+"  ")
				return false // don't descend into children again
			}
		}
		return true
	})
}

// printTypeSpecsWithInterfaceName prints TypeSpec nodes whose immediate child is an InterfaceType and whose name matches
func printTypeSpecsWithInterfaceName(node ast.Node, indent, matchName string) {
	ast.Inspect(node, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if ok {
			if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 && ts.Name.Name == matchName {
				fmt.Printf("%s*ast.TypeSpec (Name: %q)\n", indent, ts.Name.Name)
				printAstTree(iface, indent+"  ")
				return false // don't descend into children again
			}
		}
		return true
	})
}

// renderFieldList renders a *ast.FieldList as Go code (params/results)
func renderFieldList(fset *token.FileSet, fl *ast.FieldList, isParams bool) string {
	if fl == nil || len(fl.List) == 0 {
		if isParams {
			return "()"
		}
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString("(")
	for i, field := range fl.List {
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

// exprToString renders an ast.Expr to Go code
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
