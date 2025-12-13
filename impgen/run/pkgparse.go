package run

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// Vars.
var (
	errInterfaceNotFound = errors.New("interface not found")
	errPackageNotFound   = errors.New("package not found in imports")
)

// Interfaces

// PackageLoader interface for loading external packages.
type PackageLoader interface {
	Load(importPath string) ([]*ast.File, *token.FileSet, error)
}

// Functions

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

// getInterfacePackagePath determines the import path for the interface. Returns "." for local interfaces, or resolves
// the full import path for qualified names like "pkg.Interface".
func getInterfacePackagePath(qualifiedName string, pkgLoader PackageLoader) (string, error) {
	if isLocalInterface(qualifiedName) {
		return getLocalPackagePath(), nil
	}

	return getNonLocalPackagePath(qualifiedName, pkgLoader)
}

// getLocalInterfaceName extracts the local interface name from a possibly qualified name
// (e.g., "MyInterface" from "pkg.MyInterface").
func getLocalInterfaceName(name string) string {
	if _, after, ok := strings.Cut(name, "."); ok {
		return after
	}

	return name
}

// getLocalPackagePath returns the path for local package interfaces.
func getLocalPackagePath() string {
	return "."
}

// getMatchingInterfaceFromAST finds the interface by name in the ASTs.
func getMatchingInterfaceFromAST(
	astFiles []*ast.File, localInterfaceName, pkgImportPath string,
) (*ast.InterfaceType, error) {
	for _, fileAst := range astFiles {
		if iface := searchFileForInterface(fileAst, localInterfaceName); iface != nil {
			return iface, nil
		}
	}

	return nil, fmt.Errorf("%w: named %q in package %q", errInterfaceNotFound, localInterfaceName, pkgImportPath)
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

// importPathMatchesPackageName checks if the last segment of an import path matches the target package name.
func importPathMatchesPackageName(importPath, targetPkgImport string) bool {
	parts := strings.Split(importPath, "/")
	return len(parts) > 0 && parts[len(parts)-1] == targetPkgImport
}

// isLocalInterface checks if the interface name is local (no package qualifier).
func isLocalInterface(qualifiedName string) bool {
	return !strings.Contains(qualifiedName, ".")
}

func loadPackage(pkgImportPath string, pkgLoader PackageLoader) ([]*ast.File, *token.FileSet, error) {
	astFiles, fset, err := pkgLoader.Load(pkgImportPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load package %q: %w", pkgImportPath, err)
	}

	return astFiles, fset, nil
}

// searchFileForInterface searches a single AST file for an interface with the given name.
// Returns the interface if found, nil otherwise.
func searchFileForInterface(fileAst *ast.File, interfaceName string) *ast.InterfaceType {
	var found *ast.InterfaceType

	ast.Inspect(fileAst, func(n ast.Node) bool {
		typeSpec, isTypeSpec := n.(*ast.TypeSpec)
		if !isTypeSpec {
			return true
		}

		if typeSpec.Name.Name != interfaceName {
			return true
		}

		iface, isInterface := typeSpec.Type.(*ast.InterfaceType)
		if !isInterface {
			return true
		}

		found = iface

		return false
	})

	return found
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
