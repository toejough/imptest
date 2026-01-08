package run

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
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

// unexported constants.
const (
	// defaultMethodCapacity is the initial capacity for method maps.
	// This is a reasonable starting size for most struct types.
	defaultMethodCapacity = 8
)

// symbolType identifies the kind of symbol found.
type symbolType int

// symbolType values.
const (
	symbolInterface symbolType = iota
	symbolFunction
	symbolFunctionType
	symbolStructType
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
	iface         *dst.InterfaceType
	typeParams    *dst.FieldList
	sourceImports []*dst.ImportSpec // imports from the file containing the interface
	isStructType  bool              // true if this was synthesized from a struct type
}

// structWithDetails is a helper struct to return struct type information.
type structWithDetails struct {
	typeParams    *dst.FieldList
	typeName      string            // The name of the struct type (e.g., "Calculator")
	sourceImports []*dst.ImportSpec // imports from the file containing the struct
}

// symbolDetails holds information about the detected symbol.

type symbolDetails struct {
	kind symbolType

	iface      ifaceWithDetails
	funcDecl   *dst.FuncDecl
	funcType   funcTypeWithDetails
	structType structWithDetails
	// pkgPath tracks which package the symbol was found in.
	// For symbols found via dot imports, this differs from the search package.
	pkgPath string
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

// collectStructMethods collects all methods for a given struct type from AST files.
// It searches for function declarations with a receiver matching the struct type name,
// and also includes promoted methods from embedded structs.
func collectStructMethods(
	astFiles []*dst.File,
	fset *token.FileSet,
	structName string,
) map[string]*dst.FuncType {
	// Use a visited set to avoid infinite recursion (though Go doesn't allow circular embedding)
	visited := make(map[string]bool)
	return collectStructMethodsRecursive(astFiles, fset, structName, visited)
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
	// Avoid revisiting the same struct (prevents infinite loops)
	if visited[structName] {
		return make(map[string]*dst.FuncType)
	}

	visited[structName] = true

	// Preallocating with reasonable capacity
	methods := make(map[string]*dst.FuncType, defaultMethodCapacity)

	// Step 1: Collect direct methods on this struct
	for _, file := range astFiles {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*dst.FuncDecl)
			if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			// Get receiver type
			recvType := exprToString(fset, funcDecl.Recv.List[0].Type)

			// Normalize receiver type (remove pointer)
			normalizedRecvType := strings.TrimPrefix(recvType, "*")

			// Check if this method belongs to our struct
			if normalizedRecvType == structName {
				methods[funcDecl.Name.Name] = funcDecl.Type
			}
		}
	}

	// Step 2: Find embedded structs and collect their methods (promoted methods)
	embeddedTypes := findEmbeddedStructTypes(astFiles, structName)
	for _, embeddedName := range embeddedTypes {
		// Recursively collect methods from embedded struct
		embeddedMethods := collectStructMethodsRecursive(astFiles, fset, embeddedName, visited)
		// Add promoted methods (only if not already defined on the outer struct)
		for name, funcType := range embeddedMethods {
			if _, exists := methods[name]; !exists {
				methods[name] = funcType
			}
		}
	}

	return methods
}

// detectPackageAmbiguity checks if a package name is ambiguous - i.e., whether
// both a stdlib package and a local package with the same name exist.
// Returns flags indicating presence and the local path if it exists.
func detectPackageAmbiguity(pkgName string) (hasStdlib bool, hasLocal bool, localPath string) {
	// Check if it's a stdlib package
	hasStdlib = isStdlibPackage(pkgName)

	// Check if a local package exists
	localPath = ResolveLocalPackagePath(pkgName)
	hasLocal = localPath != pkgName // ResolveLocalPackagePath returns original if not found

	return hasStdlib, hasLocal, localPath
}

// extractPackageName extracts the package name from a fully qualified name (e.g., "pkg.Interface" -> "pkg").
func extractPackageName(qualifiedName string) string {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) > 1 {
		return parts[0]
	}

	return ""
}

// findEmbeddedStructTypes finds the names of all embedded struct types in a struct definition.
// Returns only local (same-package) embedded struct names.
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

				// Look for embedded fields (fields without names)
				for _, field := range structType.Fields.List {
					// Embedded fields have no names
					if len(field.Names) > 0 {
						continue
					}

					// Get the embedded type name
					switch typ := field.Type.(type) {
					case *dst.Ident:
						// Local embedded type (e.g., Logger)
						embeddedTypes = append(embeddedTypes, typ.Name)
					case *dst.StarExpr:
						// Pointer to embedded type (e.g., *Logger)
						if ident, ok := typ.X.(*dst.Ident); ok {
							embeddedTypes = append(embeddedTypes, ident.Name)
						}
						// Note: We skip SelectorExpr (external types like io.Reader)
						// for now as they require package loading. Those would need
						// additional handling similar to interfaceExpandEmbedded.
					}
				}

				return false // Found the struct, stop searching
			}

			return true
		})
	}

	return embeddedTypes
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

// findStructTypeInAST extracts the target struct type declaration and its type parameters from AST files.
func findStructTypeInAST(
	astFiles []*dst.File, typeName string, pkgImportPath string,
) (structWithDetails, error) {
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
				return true // Not a type declaration
			}

			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*dst.TypeSpec)

				if !isTypeSpec || typeSpec.Name.Name != typeName {
					continue // Not the type we're looking for
				}

				structType, isStructType := typeSpec.Type.(*dst.StructType)

				if !isStructType {
					continue // Not a struct type
				}

				// Found it!
				targetStructType = structType
				typeParams = typeSpec.TypeParams // Capture type parameters
				sourceImports = file.Imports     // Capture imports from this file
				found = true

				return false // Stop inspecting
			}

			return true
		})

		if found {
			break // Found in this file, no need to check others
		}
	}

	if targetStructType == nil {
		//nolint:err113 // Dynamic error message required for user-facing parse errors
		return structWithDetails{}, fmt.Errorf("struct type not found: %s in package %s", typeName, pkgImportPath)
	}

	return structWithDetails{
		typeParams:    typeParams,
		typeName:      typeName,
		sourceImports: sourceImports,
	}, nil
}

// findSymbol looks for either an interface, struct type, function type, or function/method in the given AST files.
func findSymbol(
	astFiles []*dst.File, fset *token.FileSet, symbolName string, pkgImportPath string, pkgLoader PackageLoader,
) (symbolDetails, error) {
	// 1. Try finding it as an interface first
	iface, err := getMatchingInterfaceFromAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return symbolDetails{
			kind:    symbolInterface,
			iface:   iface,
			pkgPath: pkgImportPath,
		}, nil
	}

	// 2. Try finding it as a struct type
	structType, err := findStructTypeInAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return symbolDetails{
			kind:       symbolStructType,
			structType: structType,
			pkgPath:    pkgImportPath,
		}, nil
	}

	// 3. Try finding it as a function type
	funcType, err := findFunctionTypeInAST(astFiles, symbolName, pkgImportPath)
	if err == nil {
		return symbolDetails{
			kind:     symbolFunctionType,
			funcType: funcType,
			pkgPath:  pkgImportPath,
		}, nil
	}

	// 4. Try finding it as a function or method
	funcDecl, err := findFunctionInAST(astFiles, fset, symbolName, pkgImportPath)
	if err == nil {
		return symbolDetails{
			kind:     symbolFunction,
			funcDecl: funcDecl,
			pkgPath:  pkgImportPath,
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
			// The recursive call will set pkgPath to dotImportPath
			symbol, err := findSymbol(dotFiles, dotFset, symbolName, dotImportPath, pkgLoader)
			if err == nil {
				return symbol, nil
			}
		}
	}

	return symbolDetails{}, fmt.Errorf("%w: %s in package %s", errSymbolNotFound, symbolName, pkgImportPath)
}

// getDotImportPaths collects all dot-imported package paths from AST files.
// Dot imports use the syntax: import . "path/to/package"
func getDotImportPaths(astFiles []*dst.File) []string {
	var dotImports []string

	for _, file := range astFiles {
		for _, imp := range file.Imports {
			// Check if this is a dot import (imp.Name.Name == ".")
			if imp.Name != nil && imp.Name.Name == "." {
				path := strings.Trim(imp.Path.Value, `"`)
				dotImports = append(dotImports, path)
			}
		}
	}

	return dotImports
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
		targetIface   *dst.InterfaceType
		typeParams    *dst.FieldList
		sourceImports []*dst.ImportSpec
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
				sourceImports = file.Imports     // Capture imports from THIS file

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

	return ifaceWithDetails{iface: targetIface, typeParams: typeParams, sourceImports: sourceImports}, nil
}

// inferImportPathFromTestFile parses the given test file and attempts to find
// the import path for a package with the given name. It checks both aliased
// and non-aliased imports. Returns the import path if found, or an error if
// the package is not imported in the test file.
func inferImportPathFromTestFile(goFilePath string, pkgName string) (string, error) {
	if goFilePath == "" {
		return "", errGOFILENotSet
	}

	// Parse the test file
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, goFilePath, nil, parser.ImportsOnly)
	if err != nil {
		return "", fmt.Errorf("failed to parse test file %s: %w", goFilePath, err)
	}

	// Search through imports for the package name
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}

		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check for aliased import: `import alias "path/to/pkg"`
		if imp.Name != nil && imp.Name.Name == pkgName {
			return importPath, nil
		}

		// Check for non-aliased import: `import "path/to/pkg"`
		// Match if the last component of the path equals pkgName
		if strings.HasSuffix(importPath, "/"+pkgName) || importPath == pkgName {
			return importPath, nil
		}
	}

	return "", fmt.Errorf("%w: %s in %s", errPackageNotInImports, pkgName, goFilePath)
}

// isStdlibPackage checks if a package name is a standard library package.
func isStdlibPackage(pkgName string) bool {
	// Simple packages without slashes that are in our stdlib list
	if !strings.Contains(pkgName, "/") {
		return stdlibPackages[pkgName]
	}

	return false
}
