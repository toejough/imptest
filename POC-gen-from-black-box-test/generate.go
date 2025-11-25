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
	if info.pkgName == "" {
		fmt.Println("  GOPACKAGE not set; cannot search package")
		return
	}

	matchName := info.matchName
	// Check if matchName contains a dot, e.g. "run.ExampleInt"
	if dot := strings.Index(matchName, "."); dot != -1 {
		targetPkgImport := matchName[:dot]
		matchName = matchName[dot+1:]
		// Resolve the full import path for the target package
		astFiles, _ := parsePackageFiles(info.pkgDir)
		var resolvedImportPath string
		for _, fileAst := range astFiles {
			for _, imp := range fileAst.Imports {
				importPath, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					continue
				}
				// Check if the last segment matches the targetPkgImport
				parts := strings.Split(importPath, "/")
				if len(parts) > 0 && parts[len(parts)-1] == targetPkgImport {
					resolvedImportPath = importPath
					break
				}
			}
			if resolvedImportPath != "" {
				break
			}
		}
		if resolvedImportPath == "" {
			fmt.Printf("could not resolve import path for package %q\n", targetPkgImport)
			return
		}
		// Load the target package by resolved import path
		cfg := &packages.Config{Mode: packages.LoadAllSyntax}
		pkgs, err := packages.Load(cfg, resolvedImportPath)
		if err != nil || len(pkgs) == 0 {
			fmt.Printf("error loading package %q: %v\n", resolvedImportPath, err)
			return
		}
		fmt.Printf("----- Interfaces in package %s -----\n", pkgs[0].PkgPath)
		// Pretty print the AST of each file in the target package
		for i, fileAst := range pkgs[0].Syntax {
			fmt.Printf("----- AST tree of file %d in %s -----\n", i+1, pkgs[0].PkgPath)
			printAstTree(fileAst, "")
			fmt.Printf("----- end AST tree of file %d in %s -----\n", i+1, pkgs[0].PkgPath)
		}
		found := false
		for _, fileAst := range pkgs[0].Syntax {
			ast.Inspect(fileAst, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if ok {
					if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 && ts.Name.Name == matchName {
						fmt.Printf("*ast.TypeSpec (Name: %q)\n", ts.Name.Name)
						printAstTree(iface, "  ")
						found = true
						return false
					}
				}
				return true
			})
		}
		if !found {
			fmt.Printf("No interface named %q found in package %s.\n", matchName, pkgs[0].PkgPath)
		}
		fmt.Printf("----- end interfaces in package %s -----\n", pkgs[0].PkgPath)
		return
	}

	astFiles, fset := parsePackageFiles(info.pkgDir)

	// Pretty print the AST of the GOFILE
	if info.goFilePath != "" {
		data, err := os.ReadFile(info.goFilePath)
		if err == nil {
			fsetSingle := token.NewFileSet()
			fileAst, err := parser.ParseFile(fsetSingle, info.goFilePath, data, parser.ParseComments)
			if err == nil {
				fmt.Printf("----- AST tree of %s -----\n", info.goFilePath)
				printAstTree(fileAst, "")
				fmt.Printf("----- end AST tree %s -----\n", info.goFilePath)
			}
		}
	}

	// Pretty print all interfaces in the package
	fmt.Printf("----- All interfaces in package %s -----\n", info.pkgName)
	for _, fileAst := range astFiles {
		ast.Inspect(fileAst, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if ok {
				if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 {
					fmt.Printf("*ast.TypeSpec (Name: %q)\n", ts.Name.Name)
					printAstTree(iface, "  ")
					return false
				}
			}
			return true
		})
	}
	fmt.Printf("----- end interfaces in package %s -----\n", info.pkgName)

	iface := findInterfaceInPackage(astFiles, info.matchName)
	writeImplementationFile(iface, info, fset)
}

// getGeneratorInfo gathers basic information about the generator call
func getGeneratorInfo() struct {
	cwd, pkgDir, pkgName, goFilePath, matchName string
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
	if len(os.Args) > 0 {
		matchName = os.Args[len(os.Args)-1]
	}
	return struct {
		cwd, pkgDir, pkgName, goFilePath, matchName string
	}{cwd, pkgDir, pkgName, goFilePath, matchName}
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

// findInterfaceInPackage searches for the interface by name in the package ASTs
func findInterfaceInPackage(astFiles []*ast.File, matchName string) *ast.InterfaceType {
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

// writeImplementationFile writes the implementation of the found interface to generated.go
func writeImplementationFile(identifiedInterface *ast.InterfaceType, info struct {
	cwd, pkgDir, pkgName, goFilePath, matchName string
}, fset *token.FileSet) {
	if identifiedInterface == nil {
		fmt.Printf("No interface named %q found in package.\n", info.matchName)
		return
	}
	var buf bytes.Buffer
	buf.WriteString("package main\n\n")
	buf.WriteString("// Code generated by generate.go. DO NOT EDIT.\n\n")
	buf.WriteString("type interfaceImplementation struct{}\n\n")
	for _, field := range identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}
			buf.WriteString("func (interfaceImplementation) ")
			buf.WriteString(name.Name)
			buf.WriteString(renderFieldList(fset, ftype.Params, true))
			buf.WriteString(renderFieldList(fset, ftype.Results, false))
			buf.WriteString(" { panic(\"not implemented\") }\n\n")
		}
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Printf("error formatting generated code: %v\n", err)
		return
	}
	if err := os.WriteFile("generated.go", formatted, 0644); err != nil {
		fmt.Printf("error writing generated.go: %v\n", err)
		return
	}
	fmt.Println("generated.go written successfully.")
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
