package run

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// ifaceWithDetails is a helper struct to return both the interface and its type parameters.
type ifaceWithDetails struct {
	iface      *ast.InterfaceType
	typeParams *ast.FieldList
}

var (
	errFunctionNotFound = errors.New("function or method not found")

	errInterfaceNotFound = errors.New("interface not found")

	errPackageNotFound = errors.New("package not found in imports or as a loadable package")
)

// findFunctionInAST looks for a function declaration in the given AST files.

// funcName can be "MyFunc", "MyType.MyMethod", or "*MyType.MyMethod".

func findFunctionInAST(
	astFiles []*ast.File, fset *token.FileSet, funcName string, pkgImportPath string,
) (*ast.FuncDecl, error) {
	parts := strings.Split(funcName, ".")

	methodName := parts[len(parts)-1]

	receiverName := ""

	if len(parts) > 1 {
		receiverName = strings.Join(parts[0:len(parts)-1], ".")
	}

	for _, file := range astFiles {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)

			if !ok {
				continue
			}

			// Check for function name

			if funcDecl.Name.Name != methodName {
				continue
			}

			// Check for receiver if it's a method

			if funcDecl.Recv != nil {
				if len(funcDecl.Recv.List) == 0 {
					continue
				}

				recvType := exprToString(fset, funcDecl.Recv.List[0].Type)

				normalizedRecvType := strings.TrimPrefix(recvType, "*")

				if normalizedRecvType != strings.TrimPrefix(receiverName, "*") {
					continue
				}
			} else if receiverName != "" {
				continue // Function has no receiver but receiver name was specified
			}

			return funcDecl, nil
		}
	}

	return nil, fmt.Errorf("%w: %s in package %s", errFunctionNotFound, funcName, pkgImportPath)
}

// getMatchingInterfaceFromAST extracts the target interface declaration and its type parameters from AST files.

func getMatchingInterfaceFromAST(
	astFiles []*ast.File, interfaceName string, pkgImportPath string,
) (ifaceWithDetails, error) {
	var (
		targetIface *ast.InterfaceType

		typeParams *ast.FieldList
	)

	for _, file := range astFiles {
		ast.Inspect(file, func(node ast.Node) bool {
			genDecl, ok := node.(*ast.GenDecl)

			if !ok || genDecl.Tok != token.TYPE {
				return true // Not a type declaration
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*ast.TypeSpec)

				if !isTypeSpec || typeSpec.Name.Name != interfaceName {
					continue // Not the interface we're looking for
				}

				iface, isInterfaceType := typeSpec.Type.(*ast.InterfaceType)

				if !isInterfaceType {
					continue // Not an interface type
				}

				// Found it!

				targetIface = iface

				typeParams = typeSpec.TypeParams // Capture type parameters

				return false // Stop inspecting
			}

			return true
		})

		if targetIface != nil {
			break // Found in this file, no need to check others
		}
	}

	if targetIface == nil {
		return ifaceWithDetails{}, fmt.Errorf("%w: %s in package %s", errInterfaceNotFound, interfaceName, pkgImportPath)
	}

	return ifaceWithDetails{iface: targetIface, typeParams: typeParams}, nil
}

// PackageLoader defines an interface for loading Go packages.

type PackageLoader interface {
	Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error)
}

// findImportPath finds the import path for a given package name by parsing the provided AST files.

func findImportPath(
	astFiles []*ast.File, pkgName string, pkgLoader PackageLoader,
) (string, error) {
	for _, file := range astFiles {
		for _, imp := range file.Imports {
			path, err := checkImport(imp, pkgName, pkgLoader)
			if err == nil {
				return path, nil
			}
		}
	}

	// As a last resort, try loading the package by name.

	// This covers cases where the package is implicitly imported (e.g., "builtin").

	// This is also important for when the package under test is the one being referenced,

	// and therefore will not appear in the imports (e.g., "mytypes.MyStruct").

	files, _, _, err := pkgLoader.Load(pkgName)

	if err == nil && len(files) > 0 {
		return pkgName, nil
	}

	return "", fmt.Errorf("%w: %s", errPackageNotFound, pkgName)
}

// checkImport checks if an import matches the target package name.

func checkImport(imp *ast.ImportSpec, pkgName string, pkgLoader PackageLoader) (string, error) {
	path := strings.Trim(imp.Path.Value, `"`)

	if imp.Name != nil && imp.Name.Name == pkgName {
		// Aliased import: `import alias "path/to/pkg"`
		return path, nil
	}

	// Non-aliased import: `import "path/to/pkg"`

	if strings.HasSuffix(path, "/"+pkgName) || path == pkgName {
		return path, nil
	}

	// If suffix doesn't match, the package name might still match the internal package name.

	// Load the package to check.

	importedFiles, _, _, err := pkgLoader.Load(path)

	if err == nil && len(importedFiles) > 0 && importedFiles[0].Name.Name == pkgName {
		return path, nil
	}

	return "", errPackageNotFound
}

// extractPackageName extracts the package name from a fully qualified name (e.g., "pkg.Interface" -> "pkg").
func extractPackageName(qualifiedName string) string {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) > 1 {
		return parts[0]
	}

	return ""
}
