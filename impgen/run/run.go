// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strconv"
	"strings"

	"github.com/alexflint/go-arg"
)

// Vars.
var (
	errInterfaceNotFound = errors.New("interface not found")
	errPackageNotFound   = errors.New("package not found in imports")
)

// Interfaces - Public

// FileSystem interface for mocking.
type FileSystem interface {
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// PackageLoader interface for loading external packages.
type PackageLoader interface {
	Load(importPath string) ([]*ast.File, *token.FileSet, error)
}

// Structs - Private

// cliArgs defines the command-line arguments for the generator.
type cliArgs struct {
	Interface string `arg:"positional,required" help:"interface name to implement (e.g. MyInterface or pkg.MyInterface)"`
	Name      string `arg:"--name"              help:"name for the generated implementation (defaults to <Interface>Imp)"`
}

// generatorInfo holds information gathered for generation.
type generatorInfo struct {
	pkgName, interfaceName, localInterfaceName, impName string
}

// Functions - Public

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

// Functions - Private

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
