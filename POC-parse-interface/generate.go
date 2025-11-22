//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

func main() {
	fmt.Printf("Running %s go on %s\n", os.Args[0], os.Getenv("GOFILE"))

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("  cwd = %s\n", cwd)
	fmt.Printf("  os.Args = %#v\n", os.Args)

	for _, ev := range []string{"GOARCH", "GOOS", "GOFILE", "GOLINE", "GOPACKAGE", "DOLLAR"} {
		fmt.Println("  ", ev, "=", os.Getenv(ev))
	}

	// Open GOFILE (if set) and print its contents.
	gofile := os.Getenv("GOFILE")
	if gofile == "" {
		fmt.Println("  GOFILE not set; skipping file read")
		return
	}

	fullPath := gofile
	if !filepath.IsAbs(gofile) {
		fullPath = filepath.Join(cwd, gofile)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Printf("  error reading GOFILE %q: %v\n", fullPath, err)
		return
	}

	fmt.Printf("----- Contents of %s -----\n%s\n----- end %s -----\n", fullPath, string(data), fullPath)

	// Parse the file and pretty print its AST as a tree of nodes
	fset := token.NewFileSet()
	fileAst, err := parser.ParseFile(fset, fullPath, data, parser.ParseComments)
	if err != nil {
		fmt.Printf("  error parsing GOFILE %q: %v\n", fullPath, err)
		return
	}

	fmt.Printf("----- AST tree of %s -----\n", fullPath)
	printAstTree(fileAst, "")
	fmt.Printf("----- end AST tree %s -----\n", fullPath)
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
