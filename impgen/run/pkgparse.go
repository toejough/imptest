package run

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

// Vars.
var (
	errFunctionNotFound             = errors.New("function not found")
	errInterfaceNotFound            = errors.New("interface not found")
	errPackageNotFound              = errors.New("package not found in imports")
	errUnsupportedEmbeddedType      = errors.New("unsupported embedded interface type")
	errExternalEmbeddedNotSupported = errors.New("embedded interface from external package is not yet supported")
)

// Interfaces

// PackageLoader interface for loading external packages.
type PackageLoader interface {
	Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error)
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
// Returns the interface along with its type parameters.
func getMatchingInterfaceFromAST(
	astFiles []*ast.File, localInterfaceName, pkgImportPath string,
) (*interfaceWithParams, error) {
	for _, fileAst := range astFiles {
		ifaceWithParams := searchFileForInterface(fileAst, localInterfaceName)
		if ifaceWithParams != nil {
			return ifaceWithParams, nil
		}
	}

	return nil, fmt.Errorf("%w: named %q in package %q", errInterfaceNotFound, localInterfaceName, pkgImportPath)
}

// getNonLocalPackagePath resolves the full import path for a qualified interface name.
func getNonLocalPackagePath(qualifiedName string, pkgLoader PackageLoader) (string, error) {
	targetPkgImport := extractPackageName(qualifiedName)

	astFiles, _, _, err := pkgLoader.Load(".")
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

// loadPackage loads a package and returns its AST files, file set, and type info.
func loadPackage(pkgImportPath string, pkgLoader PackageLoader) ([]*ast.File, *token.FileSet, *types.Info, error) {
	astFiles, fset, typesInfo, err := pkgLoader.Load(pkgImportPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load package %q: %w", pkgImportPath, err)
	}

	return astFiles, fset, typesInfo, nil
}

// interfaceWithParams holds an interface type along with its type parameters.
type interfaceWithParams struct {
	iface      *ast.InterfaceType
	typeParams *ast.FieldList
}

// searchFileForInterface searches a single AST file for an interface with the given name.
// Returns the interface with its type parameters if found, nil if not found.
func searchFileForInterface(fileAst *ast.File, interfaceName string) *interfaceWithParams {
	var found *interfaceWithParams

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

		// Capture type parameters (Go 1.18+ generics)
		found = &interfaceWithParams{
			iface:      iface,
			typeParams: typeSpec.TypeParams,
		}

		return false
	})

	return found
}

// searchFileImports searches a single AST file's imports for a matching package name.
// Returns the full import path if found, empty string if not found, or an error if import paths are malformed.
// Handles both aliased imports (e.g., `import foo "github.com/bar/baz"`) and regular imports.
func searchFileImports(fileAst *ast.File, targetPkgImport string) (string, error) {
	for _, imp := range fileAst.Imports {
		// Check for aliased import first
		if imp.Name != nil && imp.Name.Name == targetPkgImport {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return "", fmt.Errorf("failed to unquote import path %q: %w", imp.Path.Value, err)
			}

			return importPath, nil
		}

		// Fall back to path-based matching for non-aliased imports
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

// findFunctionInAST finds a function or method declaration in the AST files.
// funcName can be a plain function name like "PrintSum" or a method reference like "PingPongPlayer.Play".
func findFunctionInAST(astFiles []*ast.File, funcName string, pkgImportPath string) (*ast.FuncDecl, error) {
	typeName, methodName, isMethod := strings.Cut(funcName, ".")

	for _, file := range astFiles {
		if found := findFunctionInFile(file, typeName, methodName, isMethod); found != nil {
			return found, nil
		}
	}

	return nil, fmt.Errorf("%w: named %q in package %q", errFunctionNotFound, funcName, pkgImportPath)
}

// findFunctionInFile searches a single file for a matching function or method declaration.
func findFunctionInFile(file *ast.File, typeName, methodName string, isMethod bool) *ast.FuncDecl {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if isMethod {
			if matchesMethod(funcDecl, typeName, methodName) {
				return funcDecl
			}
		} else {
			if matchesFunction(funcDecl, typeName) {
				return funcDecl
			}
		}
	}

	return nil
}

// matchesMethod checks if a function declaration is a method with the given receiver type and method name.
func matchesMethod(funcDecl *ast.FuncDecl, typeName, methodName string) bool {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return false
	}

	if funcDecl.Name.Name != methodName {
		return false
	}

	return matchesReceiverType(funcDecl.Recv.List[0].Type, typeName)
}

// matchesFunction checks if a function declaration is a plain function (no receiver) with the given name.
func matchesFunction(funcDecl *ast.FuncDecl, funcName string) bool {
	return funcDecl.Recv == nil && funcDecl.Name.Name == funcName
}

// matchesReceiverType checks if the receiver type expression matches the given type name.
// Handles both value receivers (T) and pointer receivers (*T).
func matchesReceiverType(expr ast.Expr, typeName string) bool {
	switch recv := expr.(type) {
	case *ast.Ident:
		return recv.Name == typeName
	case *ast.StarExpr:
		// Pointer receiver - check the underlying type
		if ident, ok := recv.X.(*ast.Ident); ok {
			return ident.Name == typeName
		}
	}

	return false
}

// isComparableExpr checks if an AST expression represents a comparable type.
// Uses Go's type system to accurately determine comparability.
func isComparableExpr(expr ast.Expr, typesInfo *types.Info) bool {
	if typesInfo == nil {
		return false // Conservative: assume non-comparable if no type info
	}

	tv, ok := typesInfo.Types[expr]
	if !ok {
		return false // Type not found, be conservative
	}

	return types.Comparable(tv.Type)
}
