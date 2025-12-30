package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dave/dst"
)

// PackageLoader defines an interface for loading Go packages.

type PackageLoader interface {
	Load(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error)
}

// ResolveLocalPackagePath checks if importPath refers to a local subdirectory package.
// For simple package names (no slashes), it checks if there's a local subdirectory
// with that name containing .go files. This handles cases where local packages
// shadow stdlib packages (e.g., a local "time" package shadowing stdlib "time").
//
// Returns the absolute path to the local package directory if found, or the
// original importPath if it should be resolved normally.
//
//nolint:cyclop // Early returns for different resolution paths
func ResolveLocalPackagePath(importPath string) string {
	// Only check for simple package names (no slashes, not ".", not absolute paths)
	if importPath == "." || strings.HasPrefix(importPath, "/") || strings.Contains(importPath, "/") {
		return importPath
	}

	srcDir, err := os.Getwd()
	if err != nil {
		return importPath
	}

	localDir := filepath.Join(srcDir, importPath)

	info, err := os.Stat(localDir)
	if err != nil || !info.IsDir() {
		return importPath
	}

	// Check if it contains .go files
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return importPath
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") && !e.IsDir() {
			// Found a local package - return the absolute path
			return localDir
		}
	}

	return importPath
}

// symbolType identifies the kind of symbol found.
type symbolType int

// symbolType values.
const (
	symbolInterface symbolType = iota
	symbolFunction
	symbolFunctionType
)

// unexported variables.
var (
	errFunctionNotFound  = errors.New("function or method not found")
	errInterfaceNotFound = errors.New("interface not found")
	errPackageNotFound   = errors.New("package not found in imports or as a loadable package")
	errSymbolNotFound    = errors.New("symbol (interface or function) not found")
	// goListCache caches results of 'go list' subprocess calls to avoid redundant execution.
	//nolint:gochecknoglobals // Cache for performance optimization
	goListCache = make(map[string]string)
	//nolint:gochecknoglobals // Mutex for goListCache
	goListCacheMu sync.RWMutex
	// stdlibPackages contains common stdlib packages that might be shadowed by local packages.
	// This list doesn't need to be exhaustive - just common cases.
	//nolint:gochecknoglobals // Global constant map for stdlib package detection
	stdlibPackages = map[string]bool{
		"archive": true, "bufio": true, "bytes": true, "compress": true,
		"container": true, "context": true, "crypto": true, "database": true,
		"debug": true, "embed": true, "encoding": true, "errors": true,
		"expvar": true, "flag": true, "fmt": true, "go": true,
		"hash": true, "html": true, "image": true, "index": true,
		"io": true, "log": true, "math": true, "mime": true,
		"net": true, "os": true, "path": true, "plugin": true,
		"reflect": true, "regexp": true, "runtime": true, "sort": true,
		"strconv": true, "strings": true, "sync": true, "syscall": true,
		"testing": true, "text": true, "time": true, "unicode": true,
		"unsafe": true,
	}
)

// funcTypeWithDetails is a helper struct to return both the function type and its type parameters.
type funcTypeWithDetails struct {
	funcType   *dst.FuncType
	typeParams *dst.FieldList
	typeName   string // The name of the type (e.g., "WalkFunc")
}

// ifaceWithDetails is a helper struct to return both the interface and its type parameters.
type ifaceWithDetails struct {
	iface      *dst.InterfaceType
	typeParams *dst.FieldList
}

// symbolDetails holds information about the detected symbol.

type symbolDetails struct {
	kind symbolType

	iface    ifaceWithDetails
	funcDecl *dst.FuncDecl
	funcType funcTypeWithDetails
}

// checkImport checks if an import matches the target package name.

func checkImport(imp *dst.ImportSpec, pkgName string, pkgLoader PackageLoader) (string, error) {
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

// findFunctionInAST looks for a function declaration in the given AST files.

// funcName can be "MyFunc", "MyType.MyMethod", or "*MyType.MyMethod".

func findFunctionInAST(
	astFiles []*dst.File, fset *token.FileSet, funcName string, pkgImportPath string,
) (*dst.FuncDecl, error) {
	parts := strings.Split(funcName, ".")

	methodName := parts[len(parts)-1]

	receiverName := ""

	if len(parts) > 1 {
		receiverName = strings.Join(parts[0:len(parts)-1], ".")
	}

	for _, file := range astFiles {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*dst.FuncDecl)

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

// findFunctionTypeInAST extracts the target function type declaration and its type parameters from AST files.
func findFunctionTypeInAST(
	astFiles []*dst.File, typeName string, pkgImportPath string,
) (funcTypeWithDetails, error) {
	var (
		targetFuncType *dst.FuncType
		typeParams     *dst.FieldList
	)

	for _, file := range astFiles {
		dst.Inspect(file, func(node dst.Node) bool {
			genDecl, ok := node.(*dst.GenDecl)

			if !ok || genDecl.Tok != token.TYPE {
				return true // Not a type declaration
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*dst.TypeSpec)

				if !isTypeSpec || typeSpec.Name.Name != typeName {
					continue // Not the type we're looking for
				}

				funcType, isFuncType := typeSpec.Type.(*dst.FuncType)

				if !isFuncType {
					continue // Not a function type
				}

				// Found it!
				targetFuncType = funcType
				typeParams = typeSpec.TypeParams // Capture type parameters

				return false // Stop inspecting
			}

			return true
		})

		if targetFuncType != nil {
			break // Found in this file, no need to check others
		}
	}

	if targetFuncType == nil {
		//nolint:err113 // Dynamic error message required for user-facing parse errors
		return funcTypeWithDetails{}, fmt.Errorf("function type not found: %s in package %s", typeName, pkgImportPath)
	}

	return funcTypeWithDetails{
		funcType:   targetFuncType,
		typeParams: typeParams,
		typeName:   typeName,
	}, nil
}

// findImportPath finds the import path for a given package name by parsing the provided AST files.
//
//nolint:cyclop // Import path resolution requires checking multiple sources
func findImportPath(
	astFiles []*dst.File, pkgName string, pkgLoader PackageLoader,
) (string, error) {
	// For stdlib package names, check if there's a local package shadowing it first.
	// This ensures local packages shadow stdlib packages with the same name (e.g., local "time" vs stdlib "time").
	// Only do this for actual stdlib packages to avoid expensive loads for module packages.
	if isStdlibPackage(pkgName) {
		files, fset, _, err := pkgLoader.Load(pkgName)
		if err == nil && len(files) > 0 {
			// Try to get the full import path from the loaded package directory.
			// For stdlib packages loaded from their actual location, getImportPathFromFiles will fail
			// (because stdlib packages are typically in GOROOT). That's expected - we just want
			// to detect LOCAL packages that shadow stdlib packages.
			fullPath, err := getImportPathFromFiles(files, fset, pkgName)

			isLocalPackage := err == nil && fullPath != pkgName &&
				(strings.HasSuffix(fullPath, "/"+pkgName) || strings.Contains(fullPath, "/"+pkgName+"/"))
			if isLocalPackage {
				// Got a different path that's valid for this package - this means a local package is shadowing the stdlib package
				return fullPath, nil
			}

			// Either getImportPathFromFiles failed (normal for stdlib) or returned an invalid path.
			// In both cases, use the package name as-is (it's the stdlib package).
			return pkgName, nil
		}
	}

	// Check existing imports in the current package.
	// This covers cases where the package is already imported or has an alias.
	for _, file := range astFiles {
		for _, imp := range file.Imports {
			path, err := checkImport(imp, pkgName, pkgLoader)
			if err == nil {
				return path, nil
			}
		}
	}

	// As a last resort, try loading the package by name.
	// This covers cases where the package is implicitly imported (e.g., "builtin")
	// or when the package under test is the one being referenced.
	files, fset, _, err := pkgLoader.Load(pkgName)
	if err == nil && len(files) > 0 {
		// Try to get the full import path from the loaded package directory
		fullPath, err := getImportPathFromFiles(files, fset, pkgName)
		if err == nil {
			return fullPath, nil
		}

		// Fall back to the short name if import path resolution fails
		return pkgName, nil
	}

	return "", fmt.Errorf("%w: %s", errPackageNotFound, pkgName)
}

// findSymbol looks for either an interface, function type, or function/method in the given AST files.
func findSymbol(
	astFiles []*dst.File, fset *token.FileSet, symbolName string, pkgImportPath string,
) (symbolDetails, error) {
	// 1. Try finding it as an interface first
	iface, err := getMatchingInterfaceFromAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return symbolDetails{
			kind:  symbolInterface,
			iface: iface,
		}, nil
	}

	// 2. Try finding it as a function type
	funcType, err := findFunctionTypeInAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return symbolDetails{
			kind:     symbolFunctionType,
			funcType: funcType,
		}, nil
	}

	// 3. Try finding it as a function or method
	funcDecl, err := findFunctionInAST(astFiles, fset, symbolName, pkgImportPath)
	if err == nil {
		return symbolDetails{
			kind:     symbolFunction,
			funcDecl: funcDecl,
		}, nil
	}

	return symbolDetails{}, fmt.Errorf("%w: %s in package %s", errSymbolNotFound, symbolName, pkgImportPath)
}

// getImportPathFromFiles determines the import path of a package by examining its loaded files.
// It gets the directory from the FileSet and runs `go list` on that directory.
// Results are cached to avoid redundant subprocess calls.
func getImportPathFromFiles(files []*dst.File, fset *token.FileSet, _ string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("%w: no files provided", errPackageNotFound)
	}

	// Iterate through the FileSet to find a file path
	var filePath string

	fset.Iterate(func(f *token.File) bool {
		filePath = f.Name()
		return false // Stop after first file
	})

	if filePath == "" {
		return "", fmt.Errorf("%w: cannot determine file path from FileSet", errPackageNotFound)
	}

	// Get the directory containing the file
	dir := filepath.Dir(filePath)

	// Check cache first
	goListCacheMu.RLock()

	if cached, ok := goListCache[dir]; ok {
		goListCacheMu.RUnlock()
		return cached, nil
	}

	goListCacheMu.RUnlock()

	// Cache miss - run go list
	//nolint:noctx // context not needed for simple command
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", dir)

	var out bytes.Buffer

	cmd.Stdout = &out
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get import path for directory %s: %w", dir, err)
	}

	importPath := strings.TrimSpace(out.String())

	// Store in cache
	goListCacheMu.Lock()

	goListCache[dir] = importPath

	goListCacheMu.Unlock()

	return importPath, nil
}

// getMatchingInterfaceFromAST extracts the target interface declaration and its type parameters from AST files.

func getMatchingInterfaceFromAST(
	astFiles []*dst.File, interfaceName string, pkgImportPath string,
) (ifaceWithDetails, error) {
	var (
		targetIface *dst.InterfaceType

		typeParams *dst.FieldList
	)

	for _, file := range astFiles {
		dst.Inspect(file, func(node dst.Node) bool {
			genDecl, ok := node.(*dst.GenDecl)

			if !ok || genDecl.Tok != token.TYPE {
				return true // Not a type declaration
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*dst.TypeSpec)

				if !isTypeSpec || typeSpec.Name.Name != interfaceName {
					continue // Not the interface we're looking for
				}

				iface, isInterfaceType := typeSpec.Type.(*dst.InterfaceType)

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

// isStdlibPackage checks if a package name is a standard library package.
func isStdlibPackage(pkgName string) bool {
	// Simple packages without slashes that are in our stdlib list
	if !strings.Contains(pkgName, "/") {
		return stdlibPackages[pkgName]
	}

	return false
}
