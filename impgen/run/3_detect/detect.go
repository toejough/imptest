// Package detect provides symbol detection and type resolution for Go packages.
package detect

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"go/types"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dave/dst"
	astutil "github.com/toejough/imptest/impgen/run/0_util"
	load "github.com/toejough/imptest/impgen/run/2_load"
)

// SymbolType identifies the kind of symbol found.
type SymbolType int

// SymbolType values.
const (
	SymbolInterface SymbolType = iota
	SymbolFunction
	SymbolFunctionType
	SymbolStructType
)

// FuncTypeWithDetails is a helper struct to return both the function type and its type parameters.
type FuncTypeWithDetails struct {
	FuncType   *dst.FuncType
	TypeParams *dst.FieldList
	TypeName   string // The name of the type (e.g., "WalkFunc")
}

// IfaceWithDetails is a helper struct to return both the interface and its type parameters.
type IfaceWithDetails struct {
	Iface         *dst.InterfaceType
	TypeParams    *dst.FieldList
	SourceImports []*dst.ImportSpec // imports from the file containing the interface
	IsStructType  bool              // true if this was synthesized from a struct type
}

// PackageLoader defines an interface for loading Go packages.
type PackageLoader interface {
	Load(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error)
}

// StructWithDetails is a helper struct to return struct type information.
type StructWithDetails struct {
	TypeParams    *dst.FieldList
	TypeName      string            // The name of the struct type (e.g., "Calculator")
	SourceImports []*dst.ImportSpec // imports from the file containing the struct
}

// SymbolDetails holds information about the detected symbol.
type SymbolDetails struct {
	Kind SymbolType

	Iface      IfaceWithDetails
	FuncDecl   *dst.FuncDecl
	FuncType   FuncTypeWithDetails
	StructType StructWithDetails
	// PkgPath tracks which package the symbol was found in.
	// For symbols found via dot imports, this differs from the search package.
	PkgPath string
}

// CollectStructMethods collects all methods for a given struct type from AST files.
func CollectStructMethods(
	astFiles []*dst.File,
	fset *token.FileSet,
	structName string,
) map[string]*dst.FuncType {
	visited := make(map[string]bool)
	return collectStructMethodsRecursive(astFiles, fset, structName, visited)
}

// ExtractPackageName extracts the package name from a fully qualified name.
func ExtractPackageName(qualifiedName string) string {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) > 1 {
		return parts[0]
	}

	return ""
}

// FindImportPath finds the import path for a given package name by parsing the provided AST files.
//
//nolint:cyclop // Import path resolution requires checking multiple sources
func FindImportPath(
	astFiles []*dst.File, pkgName string, pkgLoader PackageLoader,
) (string, error) {
	// For stdlib package names, check if there's a local package shadowing it first.
	if IsStdlibPackage(pkgName) {
		files, fset, _, err := pkgLoader.Load(pkgName)
		if err == nil && len(files) > 0 {
			fullPath, err := GetImportPathFromFiles(files, fset, pkgName)

			isLocalPkg := err == nil && fullPath != pkgName &&
				(strings.HasSuffix(fullPath, "/"+pkgName) || strings.Contains(fullPath, "/"+pkgName+"/"))
			if isLocalPkg {
				return fullPath, nil
			}

			return pkgName, nil
		}
	}

	// Check existing imports in the current package.
	for _, file := range astFiles {
		for _, imp := range file.Imports {
			path, err := checkImport(imp, pkgName, pkgLoader)
			if err == nil {
				return path, nil
			}
		}
	}

	// As a last resort, try loading the package by name.
	files, fset, _, err := pkgLoader.Load(pkgName)
	if err == nil && len(files) > 0 {
		fullPath, err := GetImportPathFromFiles(files, fset, pkgName)
		if err == nil {
			return fullPath, nil
		}

		return pkgName, nil
	}

	return "", fmt.Errorf("%w: %s", errPackageNotFound, pkgName)
}

// FindSymbol looks for either an interface, struct type, function type, or function/method in the given AST files.
func FindSymbol(
	astFiles []*dst.File, fset *token.FileSet, symbolName string, pkgImportPath string, pkgLoader PackageLoader,
) (SymbolDetails, error) {
	// 1. Try finding it as an interface first
	iface, err := GetMatchingInterfaceFromAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return SymbolDetails{
			Kind:    SymbolInterface,
			Iface:   iface,
			PkgPath: pkgImportPath,
		}, nil
	}

	// 2. Try finding it as a struct type
	structType, err := findStructTypeInAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return SymbolDetails{
			Kind:       SymbolStructType,
			StructType: structType,
			PkgPath:    pkgImportPath,
		}, nil
	}

	// 3. Try finding it as a function type
	funcType, err := findFunctionTypeInAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return SymbolDetails{
			Kind:     SymbolFunctionType,
			FuncType: funcType,
			PkgPath:  pkgImportPath,
		}, nil
	}

	// 4. Try finding it as a function or method
	funcDecl, err := findFunctionInAST(astFiles, fset, symbolName, pkgImportPath)
	if err == nil {
		return SymbolDetails{
			Kind:     SymbolFunction,
			FuncDecl: funcDecl,
			PkgPath:  pkgImportPath,
		}, nil
	}

	// 5. If not found in current package and this is the current package (".")
	// check dot-imported packages
	if pkgImportPath == "." {
		dotImports := getDotImportPaths(astFiles)

		for _, dotImportPath := range dotImports {
			// Load the dot-imported package
			dotFiles, dotFset, _, err := pkgLoader.Load(dotImportPath)
			if err != nil {
				continue // Skip if we can't load this package
			}

			// Try to find the symbol in the dot-imported package
			symbol, err := FindSymbol(dotFiles, dotFset, symbolName, dotImportPath, pkgLoader)
			if err == nil {
				return symbol, nil
			}
		}
	}

	return SymbolDetails{}, fmt.Errorf("%w: %s in package %s", errSymbolNotFound, symbolName, pkgImportPath)
}

// GetImportPathFromFiles determines the import path of a package by examining its loaded files.
func GetImportPathFromFiles(files []*dst.File, fset *token.FileSet, _ string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("%w: no files provided", errPackageNotFound)
	}

	var filePath string

	fset.Iterate(func(f *token.File) bool {
		filePath = f.Name()
		return false
	})

	if filePath == "" {
		return "", fmt.Errorf("%w: cannot determine file path from FileSet", errPackageNotFound)
	}

	dir := filepath.Dir(filePath)

	goListCacheMu.RLock()

	if cached, ok := goListCache[dir]; ok {
		goListCacheMu.RUnlock()
		return cached, nil
	}

	goListCacheMu.RUnlock()

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

	goListCacheMu.Lock()

	goListCache[dir] = importPath

	goListCacheMu.Unlock()

	return importPath, nil
}

// GetMatchingInterfaceFromAST extracts the target interface declaration from AST files.
func GetMatchingInterfaceFromAST(
	astFiles []*dst.File, interfaceName string, pkgImportPath string,
) (IfaceWithDetails, error) {
	var (
		targetIface   *dst.InterfaceType
		typeParams    *dst.FieldList
		sourceImports []*dst.ImportSpec
	)

	for _, file := range astFiles {
		dst.Inspect(file, func(node dst.Node) bool {
			genDecl, ok := node.(*dst.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				return true
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*dst.TypeSpec)
				if !isTypeSpec || typeSpec.Name.Name != interfaceName {
					continue
				}

				iface, isInterfaceType := typeSpec.Type.(*dst.InterfaceType)
				if !isInterfaceType {
					continue
				}

				targetIface = iface
				typeParams = typeSpec.TypeParams
				sourceImports = file.Imports

				return false
			}

			return true
		})

		if targetIface != nil {
			break
		}
	}

	if targetIface == nil {
		return IfaceWithDetails{}, fmt.Errorf("%w: %s in package %s", errInterfaceNotFound, interfaceName, pkgImportPath)
	}

	return IfaceWithDetails{Iface: targetIface, TypeParams: typeParams, SourceImports: sourceImports}, nil
}

// InferImportPathFromTestFile parses the given test file and attempts to find
// the import path for a package with the given name.
func InferImportPathFromTestFile(goFilePath string, pkgName string) (string, error) {
	if goFilePath == "" {
		return "", errGOFILENotSet
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, goFilePath, nil, parser.ImportsOnly)
	if err != nil {
		return "", fmt.Errorf("failed to parse test file %s: %w", goFilePath, err)
	}

	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}

		importPath := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil && imp.Name.Name == pkgName {
			return importPath, nil
		}

		if strings.HasSuffix(importPath, "/"+pkgName) || importPath == pkgName {
			return importPath, nil
		}
	}

	return "", fmt.Errorf("%w: %s in %s", errPackageNotInImports, pkgName, goFilePath)
}

// IsStdlibPackage checks if a package name is a standard library package.
func IsStdlibPackage(pkgName string) bool {
	if !strings.Contains(pkgName, "/") {
		return stdlibPackages[pkgName]
	}

	return false
}

// PackageAmbiguity checks if a package name is ambiguous.
func PackageAmbiguity(pkgName string) (hasStdlib bool, hasLocal bool, localPath string) {
	hasStdlib = IsStdlibPackage(pkgName)
	localPath = load.ResolveLocalPackagePath(pkgName)
	hasLocal = localPath != pkgName

	return hasStdlib, hasLocal, localPath
}

// unexported constants.
const (
	// defaultMethodCapacity is the initial capacity for method maps.
	defaultMethodCapacity = 8
)

// unexported variables.
var (
	errFunctionNotFound    = errors.New("function or method not found")
	errGOFILENotSet        = errors.New("GOFILE environment variable not set")
	errInterfaceNotFound   = errors.New("interface not found")
	errPackageNotFound     = errors.New("package not found in imports or as a loadable package")
	errPackageNotInImports = errors.New("package not found in imports")
	errSymbolNotFound      = errors.New("symbol (interface or function) not found")
	// goListCache caches results of 'go list' subprocess calls to avoid redundant execution.
	//nolint:gochecknoglobals // Cache for performance optimization
	goListCache = make(map[string]string)
	//nolint:gochecknoglobals // Mutex for goListCache
	goListCacheMu sync.RWMutex
	// stdlibPackages contains common stdlib packages that might be shadowed by local packages.
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

// checkImport checks if an import matches the target package name.
func checkImport(imp *dst.ImportSpec, pkgName string, pkgLoader PackageLoader) (string, error) {
	path := strings.Trim(imp.Path.Value, `"`)

	if imp.Name != nil && imp.Name.Name == pkgName {
		return path, nil
	}

	if strings.HasSuffix(path, "/"+pkgName) || path == pkgName {
		return path, nil
	}

	importedFiles, _, _, err := pkgLoader.Load(path)

	if err == nil && len(importedFiles) > 0 && importedFiles[0].Name.Name == pkgName {
		return path, nil
	}

	return "", errPackageNotFound
}

// collectStructMethodsRecursive recursively collects methods including promoted methods from embedded structs.
//
//nolint:cyclop // Recursive embedded struct handling requires multiple code paths
func collectStructMethodsRecursive(
	astFiles []*dst.File,
	fset *token.FileSet,
	structName string,
	visited map[string]bool,
) map[string]*dst.FuncType {
	if visited[structName] {
		return make(map[string]*dst.FuncType)
	}

	visited[structName] = true

	methods := make(map[string]*dst.FuncType, defaultMethodCapacity)

	for _, file := range astFiles {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*dst.FuncDecl)
			if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			recvType := astutil.ExprToString(fset, funcDecl.Recv.List[0].Type)
			normalizedRecvType := strings.TrimPrefix(recvType, "*")

			if normalizedRecvType == structName {
				methods[funcDecl.Name.Name] = funcDecl.Type
			}
		}
	}

	embeddedTypes := findEmbeddedStructTypes(astFiles, structName)
	for _, embeddedName := range embeddedTypes {
		embeddedMethods := collectStructMethodsRecursive(astFiles, fset, embeddedName, visited)
		for name, funcType := range embeddedMethods {
			if _, exists := methods[name]; !exists {
				methods[name] = funcType
			}
		}
	}

	return methods
}

// findEmbeddedStructTypes finds the names of all embedded struct types in a struct definition.
//
//nolint:gocognit,cyclop // Nested AST traversal requires multiple condition checks
func findEmbeddedStructTypes(astFiles []*dst.File, structName string) []string {
	var embeddedTypes []string

	for _, file := range astFiles {
		dst.Inspect(file, func(node dst.Node) bool {
			genDecl, ok := node.(*dst.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				return true
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*dst.TypeSpec)
				if !isTypeSpec || typeSpec.Name.Name != structName {
					continue
				}

				structType, isStructType := typeSpec.Type.(*dst.StructType)
				if !isStructType || structType.Fields == nil {
					continue
				}

				for _, field := range structType.Fields.List {
					if len(field.Names) > 0 {
						continue
					}

					switch typ := field.Type.(type) {
					case *dst.Ident:
						embeddedTypes = append(embeddedTypes, typ.Name)
					case *dst.StarExpr:
						if ident, ok := typ.X.(*dst.Ident); ok {
							embeddedTypes = append(embeddedTypes, ident.Name)
						}
					}
				}

				return false
			}

			return true
		})
	}

	return embeddedTypes
}

// findFunctionInAST looks for a function declaration in the given AST files.
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

			if funcDecl.Name.Name != methodName {
				continue
			}

			if funcDecl.Recv != nil {
				if len(funcDecl.Recv.List) == 0 {
					continue
				}

				recvType := astutil.ExprToString(fset, funcDecl.Recv.List[0].Type)
				normalizedRecvType := strings.TrimPrefix(recvType, "*")

				if normalizedRecvType != strings.TrimPrefix(receiverName, "*") {
					continue
				}
			} else if receiverName != "" {
				continue
			}

			return funcDecl, nil
		}
	}

	return nil, fmt.Errorf("%w: %s in package %s", errFunctionNotFound, funcName, pkgImportPath)
}

// findFunctionTypeInAST extracts the target function type declaration from AST files.
func findFunctionTypeInAST(
	astFiles []*dst.File, typeName string, pkgImportPath string,
) (FuncTypeWithDetails, error) {
	var (
		targetFuncType *dst.FuncType
		typeParams     *dst.FieldList
	)

	for _, file := range astFiles {
		dst.Inspect(file, func(node dst.Node) bool {
			genDecl, ok := node.(*dst.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				return true
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*dst.TypeSpec)
				if !isTypeSpec || typeSpec.Name.Name != typeName {
					continue
				}

				funcType, isFuncType := typeSpec.Type.(*dst.FuncType)
				if !isFuncType {
					continue
				}

				targetFuncType = funcType
				typeParams = typeSpec.TypeParams

				return false
			}

			return true
		})

		if targetFuncType != nil {
			break
		}
	}

	if targetFuncType == nil {
		//nolint:err113 // Dynamic error message required for user-facing parse errors
		return FuncTypeWithDetails{}, fmt.Errorf("function type not found: %s in package %s", typeName, pkgImportPath)
	}

	return FuncTypeWithDetails{
		FuncType:   targetFuncType,
		TypeParams: typeParams,
		TypeName:   typeName,
	}, nil
}

// findStructTypeInAST extracts the target struct type declaration from AST files.
func findStructTypeInAST(
	astFiles []*dst.File, typeName string, pkgImportPath string,
) (StructWithDetails, error) {
	var (
		targetStructType *dst.StructType
		typeParams       *dst.FieldList
		sourceImports    []*dst.ImportSpec
	)

	for _, file := range astFiles {
		found := false

		dst.Inspect(file, func(node dst.Node) bool {
			genDecl, ok := node.(*dst.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				return true
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*dst.TypeSpec)
				if !isTypeSpec || typeSpec.Name.Name != typeName {
					continue
				}

				structType, isStructType := typeSpec.Type.(*dst.StructType)
				if !isStructType {
					continue
				}

				targetStructType = structType
				typeParams = typeSpec.TypeParams
				sourceImports = file.Imports
				found = true

				return false
			}

			return true
		})

		if found {
			break
		}
	}

	if targetStructType == nil {
		//nolint:err113 // Dynamic error message required for user-facing parse errors
		return StructWithDetails{}, fmt.Errorf("struct type not found: %s in package %s", typeName, pkgImportPath)
	}

	return StructWithDetails{
		TypeParams:    typeParams,
		TypeName:      typeName,
		SourceImports: sourceImports,
	}, nil
}

// getDotImportPaths collects all dot-imported package paths from AST files.
func getDotImportPaths(astFiles []*dst.File) []string {
	var dotImports []string

	for _, file := range astFiles {
		for _, imp := range file.Imports {
			if imp.Name != nil && imp.Name.Name == "." {
				path := strings.Trim(imp.Path.Value, `"`)
				dotImports = append(dotImports, path)
			}
		}
	}

	return dotImports
}
