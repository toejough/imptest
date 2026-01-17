// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/token"
	"io"
	"strings"
	"time"

	"github.com/dave/dst"
	"github.com/toejough/targ"

	astutil "github.com/toejough/imptest/impgen/run/0_util"
	load "github.com/toejough/imptest/impgen/run/2_load"
	detect "github.com/toejough/imptest/impgen/run/3_detect"
	generate "github.com/toejough/imptest/impgen/run/5_generate"
	output "github.com/toejough/imptest/impgen/run/6_output"
)

// Exported variables.
var (
	// ErrFunctionModeRequired indicates that a mode flag is required for functions.
	ErrFunctionModeRequired = errors.New("mode flag required: use --target or --dependency")
	// ErrInterfaceModeRequired indicates that a mode flag is required for interfaces.
	ErrInterfaceModeRequired = errors.New("mode flag required: use --dependency or --target")
)

// Interfaces - Public

// FileReader interface for reading files during signature calculation.
type FileReader interface {
	Glob(pattern string) ([]string, error)
	ReadFile(name string) ([]byte, error)
}

// FileSystem interface combines reading and writing for convenience.
type FileSystem interface {
	FileReader
	output.Writer
}

// Run executes the impgen tool logic. It takes command-line arguments, an environment variable getter, an output.Writer
// interface for file operations, a PackageLoader for package operations, and an io.Writer for output messages. It
// returns an error if any step fails. On success, it generates a Go source file implementing the specified interface,
// in the calling test package.
//
//nolint:cyclop,funlen // Main orchestration function with timing instrumentation
func Run(
	args []string,
	getEnv func(string) string,
	fileSystem FileSystem,
	pkgLoader detect.PackageLoader,
	out io.Writer,
) error {
	timing := getEnv("IMPGEN_TIMING") != ""

	var start time.Time

	if timing {
		start = time.Now()
	}

	info, err := getGeneratorCallInfo(args, getEnv)
	if err != nil {
		return err
	}

	pkgImportPath, err := getInterfacePackagePath(
		info.InterfaceName,
		pkgLoader,
		info.ImportPathFlag,
		getEnv,
	)
	if err != nil {
		return err
	}

	// If it's a local package, we should use the full name for symbol lookup
	// (e.g. "MyType.MyMethod" instead of just "MyMethod")
	if pkgImportPath == "." {
		info.LocalInterfaceName = info.InterfaceName
		// Recalculate impName with the corrected localInterfaceName if not user-provided
		if !info.NameProvided {
			info.ImpName = determineGeneratedTypeName(info.Mode, info.LocalInterfaceName)
		}
	}

	if timing {
		_, _ = fmt.Fprintf(out, "[%s] Args/resolve: %v\n", info.ImpName, time.Since(start))
		start = time.Now()
	}

	astFiles, fset, err := loadPackage(pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	if timing {
		_, _ = fmt.Fprintf(out, "[%s] Load package: %v\n", info.ImpName, time.Since(start))
		start = time.Now()
	}

	// Find the symbol to generate code for
	result, err := findSymbol(info, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	if timing {
		_, _ = fmt.Fprintf(out, "[%s] Find symbol: %v\n", info.ImpName, time.Since(start))
		start = time.Now()
	}

	// Compute hash and check cache (unless disabled)
	typeHash := computeTypeHash(result.symbol, info, result.fset)
	outputFile := getOutputFilename(info.ImpName, info.PkgName, getEnv)
	noCache := getEnv("IMPGEN_NO_CACHE") != ""

	if !noCache && checkCachedHash(outputFile, typeHash, fileSystem) {
		if timing {
			_, _ = fmt.Fprintf(out, "[%s] Cache hit: %v\n", info.ImpName, time.Since(start))
		}

		_, _ = fmt.Fprintf(out, "%s unchanged (cached).\n", outputFile)

		return nil
	}

	if timing {
		_, _ = fmt.Fprintf(out, "[%s] Cache miss: %v\n", info.ImpName, time.Since(start))
		start = time.Now()
	}

	code, err := generateCode(info, result, pkgLoader)
	if err != nil {
		return err
	}

	// Add hash to generated code for future cache checks
	code = addHashToCode(code, typeHash)

	if timing {
		_, _ = fmt.Fprintf(out, "[%s] Generate code: %v\n", info.ImpName, time.Since(start))
		start = time.Now()
	}

	err = output.WriteGeneratedCode(code, info.ImpName, info.PkgName, getEnv, fileSystem, out)
	if err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	if timing {
		_, _ = fmt.Fprintf(out, "[%s] Write output: %v\n", info.ImpName, time.Since(start))
	}

	return nil
}

// unexported constants.
const (
	goPackageEnvVarName = "GOPACKAGE"
	hashHeaderLines     = 10
	hashPrefix          = "// impgen:hash:"
)

// unexported variables.
var (
	errAmbiguousPackage = errors.New(
		"package is ambiguous: both stdlib and local package exist",
	)
	errGOPACKAGENotSet        = errors.New(goPackageEnvVarName + " environment variable not set")
	errMutuallyExclusiveFlags = errors.New("--target and --dependency flags are mutually exclusive")
)

// Structs - Private

// cliArgs defines the command-line arguments for the generator.
type cliArgs struct {
	Interface  string `targ:"positional,required,desc=interface or function name to wrap/mock"`
	Name       string `targ:"flag,desc=name for the generated code (overrides default naming)"`
	Target     bool   `targ:"flag,desc=generate target wrapper (WrapXxx) instead of dependency mock"`
	Dependency bool   `targ:"flag,desc=generate dependency mock (MockXxx) - this is the default behavior"`
	ImportPath string `targ:"flag,name=import-path,desc=explicit import path when ambiguous"`
}

// Run is required by targ but not used - parsing only.
func (c *cliArgs) Run() { _ = c }

// symbolResult holds the result of symbol detection for caching purposes.
type symbolResult struct {
	symbol   detect.SymbolDetails
	astFiles []*dst.File
	fset     *token.FileSet
	pkgPath  string
}

// addHashToCode inserts the hash comment after the "DO NOT EDIT" line.
func addHashToCode(code, hash string) string {
	const doNotEdit = "// Code generated by impgen. DO NOT EDIT."
	if idx := strings.Index(code, doNotEdit); idx != -1 {
		insertPos := idx + len(doNotEdit)

		return code[:insertPos] + "\n" + hashPrefix + hash + code[insertPos:]
	}
	// Fallback: prepend hash

	return hashPrefix + hash + "\n" + code
}

// checkCachedHash reads the existing generated file and checks if its hash matches.
func checkCachedHash(filename, expectedHash string, fs FileReader) bool {
	data, err := fs.ReadFile(filename)
	if err != nil {
		return false // File doesn't exist or can't be read
	}

	// Look for hash line in first few lines
	lines := strings.SplitN(string(data), "\n", hashHeaderLines)
	for _, line := range lines {
		if after, ok := strings.CutPrefix(line, hashPrefix); ok {
			storedHash := after

			return storedHash == expectedHash
		}
	}

	return false // No hash found
}

// computeTypeHash computes a hash of the symbol details and generator info.
// This hash changes when the type definition or generator settings change.
func computeTypeHash(
	symbol detect.SymbolDetails,
	info generate.GeneratorInfo,
	fset *token.FileSet,
) string {
	var builder strings.Builder

	// Include generator settings that affect output
	fmt.Fprintf(&builder, "mode:%d\n", info.Mode)
	fmt.Fprintf(&builder, "pkg:%s\n", info.PkgName)
	fmt.Fprintf(&builder, "imp:%s\n", info.ImpName)
	fmt.Fprintf(&builder, "kind:%d\n", symbol.Kind)
	fmt.Fprintf(&builder, "pkgpath:%s\n", symbol.PkgPath)

	// Include type-specific details
	switch symbol.Kind {
	case detect.SymbolInterface:
		serializeInterface(&builder, symbol.Iface, fset)
	case detect.SymbolFunction:
		serializeFunc(&builder, symbol.FuncDecl, fset)
	case detect.SymbolFunctionType:
		serializeFuncType(&builder, symbol.FuncType, fset)
	case detect.SymbolStructType:
		serializeStruct(&builder, symbol.StructType, fset)
	}

	hash := sha256.Sum256([]byte(builder.String()))

	return hex.EncodeToString(hash[:8]) // Use first 8 bytes (16 hex chars)
}

// determineGeneratedTypeName generates the type name based on the naming mode and interface name.
func determineGeneratedTypeName(mode generate.NamingMode, localInterfaceName string) string {
	// Remove dots from localInterfaceName to create valid Go type names
	// e.g., "Calculator.Add" -> "CalculatorAdd", "MyInterface" -> "MyInterface"
	typeName := strings.ReplaceAll(localInterfaceName, ".", "")

	switch mode {
	case generate.NamingModeTarget:
		return "Start" + typeName
	case generate.NamingModeDependency:
		return "Mock" + typeName
	case generate.NamingModeDefault:
		// Default: backward compatible (Imp suffix, no prefix)
		// For methods (contains dots in original), use name as-is
		// For interfaces, append "Imp" suffix
		if strings.Contains(localInterfaceName, ".") {
			return typeName // methods: TypeMethod
		}

		return typeName + "Imp" // interfaces: InterfaceImp
	default:
		// Should never reach here
		return typeName + "Imp"
	}
}

func findSymbol(
	info generate.GeneratorInfo,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
) (symbolResult, error) {
	// Auto-detect the symbol type
	symbol, err := detect.FindSymbol(
		astFiles,
		fset,
		info.LocalInterfaceName,
		pkgImportPath,
		pkgLoader,
	)
	if err != nil {
		return symbolResult{}, fmt.Errorf(
			"failed to find symbol %s: %w",
			info.LocalInterfaceName,
			err,
		)
	}

	// Use the actual package path where the symbol was found
	// (important for dot imports where symbol.PkgPath differs from pkgImportPath)
	actualPkgPath := symbol.PkgPath

	// If symbol was found via dot import, we need to load that package's AST
	if actualPkgPath != pkgImportPath {
		astFiles, fset, _, err = pkgLoader.Load(actualPkgPath)
		if err != nil {
			return symbolResult{}, fmt.Errorf("failed to load package %s: %w", actualPkgPath, err)
		}
	}

	return symbolResult{
		symbol:   symbol,
		astFiles: astFiles,
		fset:     fset,
		pkgPath:  actualPkgPath,
	}, nil
}

func generateCode(
	info generate.GeneratorInfo,
	result symbolResult,
	pkgLoader detect.PackageLoader,
) (string, error) {
	// Route to appropriate generator based on symbol type and mode
	return routeToGenerator(
		result.astFiles,
		info,
		result.fset,
		result.pkgPath,
		pkgLoader,
		result.symbol,
	)
}

// Functions - Private

// getGeneratorCallInfo returns basic information about the current call to the generator.
func getGeneratorCallInfo(
	args []string,
	getEnv func(string) string,
) (generate.GeneratorInfo, error) {
	pkgName := getEnv(goPackageEnvVarName)
	if pkgName == "" {
		return generate.GeneratorInfo{}, errGOPACKAGENotSet
	}

	parsed, err := parseArgs(args)
	if err != nil {
		return generate.GeneratorInfo{}, err
	}

	interfaceName := parsed.Interface
	localInterfaceName := getLocalInterfaceName(interfaceName)
	impName := parsed.Name
	nameProvided := parsed.Name != ""

	// Validate mutually exclusive flags
	if parsed.Target && parsed.Dependency {
		return generate.GeneratorInfo{}, errMutuallyExclusiveFlags
	}

	// Determine naming mode based on flags
	mode := generate.NamingModeDefault
	if parsed.Target {
		mode = generate.NamingModeTarget
	} else if parsed.Dependency {
		mode = generate.NamingModeDependency
	}

	// set impname if not provided
	if impName == "" {
		impName = determineGeneratedTypeName(mode, localInterfaceName)
	}

	return generate.GeneratorInfo{
		PkgName:            pkgName,
		InterfaceName:      interfaceName,
		LocalInterfaceName: localInterfaceName,
		ImpName:            impName,
		Mode:               mode,
		ImportPathFlag:     parsed.ImportPath,
		NameProvided:       nameProvided,
	}, nil
}

// getInterfacePackagePath resolves the import path for the package containing the target interface.
// It uses a 4-tier resolution strategy:
// 1. Explicit --import-path flag takes precedence
// 2. Infer from test file imports if package is imported
// 3. Detect ambiguity (stdlib + local package) and error with helpful message
// 4. Fallback to existing import resolution logic
func getInterfacePackagePath(
	interfaceName string,
	pkgLoader detect.PackageLoader,
	importPathFlag string,
	getEnv func(string) string,
) (string, error) {
	if !strings.Contains(interfaceName, ".") {
		return ".", nil
	}

	pkgName := detect.ExtractPackageName(interfaceName)

	// Tier 1: Explicit --import-path flag
	if importPathFlag != "" {
		// Validate the path is loadable
		_, _, _, err := pkgLoader.Load(importPathFlag)
		if err != nil {
			return "", fmt.Errorf("--import-path=%q is not loadable: %w", importPathFlag, err)
		}

		return importPathFlag, nil
	}

	// Tier 2: Infer from test file imports
	goFile := getEnv("GOFILE")
	if goFile != "" {
		inferredPath, err := detect.InferImportPathFromTestFile(goFile, pkgName)
		if err == nil {
			// Found in imports - use it
			return inferredPath, nil
		}
		// Not found in imports - continue to next tier
	}

	// Tier 3: Ambiguity detection
	hasStdlib := detect.IsStdlibPackage(pkgName)
	localPath := load.ResolveLocalPackagePath(pkgName)

	hasLocal := localPath != pkgName
	if hasStdlib && hasLocal {
		return "", fmt.Errorf(
			"%w: %q\n"+
				"  Use --import-path=%s for stdlib\n"+
				"  Use --import-path=%s for local package\n"+
				"  Or import the desired package in your test file",
			errAmbiguousPackage,
			pkgName,
			pkgName,
			localPath,
		)
	}

	// Tier 4: Fallback to existing logic
	astFiles, _, _, err := pkgLoader.Load(".")
	if err != nil {
		return "", fmt.Errorf("failed to load local package to resolve import: %w", err)
	}

	// Try to find the import path for the prefix.
	// If it's not found in imports, it might be a local type/method reference (e.g. MyType.MyMethod)
	path, err := detect.FindImportPath(astFiles, pkgName, pkgLoader)
	if err != nil {
		// If it's not a package we know about, assume it's a local reference
		return ".", nil //nolint:nilerr
	}

	return path, nil
}

// getLocalInterfaceName extracts the name of the interface without the package prefix.
func getLocalInterfaceName(interfaceName string) string {
	parts := strings.Split(interfaceName, ".")
	if len(parts) > 1 {
		return strings.Join(parts[1:], ".")
	}

	return interfaceName
}

// getOutputFilename returns the filename that would be generated.
func getOutputFilename(impName, pkgName string, getEnv func(string) string) string {
	filename := "generated_" + impName
	goFile := getEnv("GOFILE")

	isTestFile := strings.HasSuffix(pkgName, "_test") || strings.HasSuffix(goFile, "_test.go")
	if isTestFile && !strings.HasSuffix(impName, "_test") {
		filename = "generated_" + strings.TrimSuffix(impName, ".go") + "_test.go"
	} else if !strings.HasSuffix(filename, ".go") {
		filename += ".go"
	}

	return filename
}

// loadPackage loads the AST for the package at the given path.
func loadPackage(
	pkgPath string,
	pkgLoader detect.PackageLoader,
) ([]*dst.File, *token.FileSet, error) {
	astFiles, fset, _, err := pkgLoader.Load(pkgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load package %s: %w", pkgPath, err)
	}

	return astFiles, fset, nil
}

// parseArgs parses command-line arguments into cliArgs.
func parseArgs(args []string) (cliArgs, error) {
	var parsed cliArgs

	_, err := targ.Execute(args, &parsed)
	if err != nil {
		return cliArgs{}, fmt.Errorf("failed to parse arguments: %w", err)
	}

	return parsed, nil
}

// routeFunctionGenerator routes to function generators based on mode.
//
//nolint:wrapcheck // internal subpackage, errors already have context
func routeFunctionGenerator(
	astFiles []*dst.File, info generate.GeneratorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader detect.PackageLoader, funcDecl *dst.FuncDecl,
) (string, error) {
	switch info.Mode {
	case generate.NamingModeTarget:
		return generate.TargetCode(astFiles, info, fset, pkgPath, pkgLoader, funcDecl)
	case generate.NamingModeDependency:
		return generate.FunctionDependencyCode(astFiles, info, fset, pkgPath, pkgLoader, funcDecl)
	case generate.NamingModeDefault:
		return "", ErrFunctionModeRequired
	}

	return "", ErrFunctionModeRequired
}

// routeFunctionTypeGenerator routes to function type generators based on mode.
//
//nolint:wrapcheck // internal subpackage, errors already have context
func routeFunctionTypeGenerator(
	astFiles []*dst.File, info generate.GeneratorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader detect.PackageLoader, funcType detect.FuncTypeWithDetails,
) (string, error) {
	switch info.Mode {
	case generate.NamingModeTarget:
		return generate.TargetCodeFromFuncType(astFiles, info, fset, pkgPath, pkgLoader, funcType)
	case generate.NamingModeDependency:
		return generate.FunctionTypeDependencyCode(
			astFiles,
			info,
			fset,
			pkgPath,
			pkgLoader,
			funcType,
		)
	case generate.NamingModeDefault:
		return "", ErrFunctionModeRequired
	}

	return "", ErrFunctionModeRequired
}

// routeInterfaceGenerator routes to interface generators based on mode.
//
//nolint:wrapcheck // internal subpackage, errors already have context
func routeInterfaceGenerator(
	astFiles []*dst.File, info generate.GeneratorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader detect.PackageLoader, iface detect.IfaceWithDetails,
) (string, error) {
	switch info.Mode {
	case generate.NamingModeDependency:
		return generate.DependencyCode(astFiles, info, fset, pkgPath, pkgLoader, iface)
	case generate.NamingModeTarget:
		return generate.InterfaceTargetCode(astFiles, info, fset, pkgPath, pkgLoader, iface, false)
	case generate.NamingModeDefault:
		return "", ErrInterfaceModeRequired
	}

	return "", ErrInterfaceModeRequired
}

// routeStructGenerator routes to struct generators based on mode.
//
//nolint:wrapcheck // internal subpackage, errors already have context
func routeStructGenerator(
	astFiles []*dst.File, info generate.GeneratorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader detect.PackageLoader, structType detect.StructWithDetails,
) (string, error) {
	switch info.Mode {
	case generate.NamingModeTarget:
		return generate.StructTargetCode(astFiles, info, fset, pkgPath, pkgLoader, structType)
	case generate.NamingModeDependency:
		return generate.StructDependencyCode(astFiles, info, fset, pkgPath, pkgLoader, structType)
	case generate.NamingModeDefault:
		return "", ErrFunctionModeRequired
	}

	return "", ErrFunctionModeRequired
}

// routeToGenerator routes to the appropriate generator based on symbol type and mode.
func routeToGenerator(
	astFiles []*dst.File,
	info generate.GeneratorInfo,
	fset *token.FileSet,
	actualPkgPath string,
	pkgLoader detect.PackageLoader,
	symbol detect.SymbolDetails,
) (string, error) {
	switch symbol.Kind {
	case detect.SymbolFunction:
		return routeFunctionGenerator(
			astFiles,
			info,
			fset,
			actualPkgPath,
			pkgLoader,
			symbol.FuncDecl,
		)
	case detect.SymbolFunctionType:
		return routeFunctionTypeGenerator(
			astFiles,
			info,
			fset,
			actualPkgPath,
			pkgLoader,
			symbol.FuncType,
		)
	case detect.SymbolStructType:
		return routeStructGenerator(
			astFiles,
			info,
			fset,
			actualPkgPath,
			pkgLoader,
			symbol.StructType,
		)
	case detect.SymbolInterface:
		return routeInterfaceGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.Iface)
	}

	return routeInterfaceGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.Iface)
}

// serializeFieldList writes a field list to the builder for hashing.
func serializeFieldList(
	builder *strings.Builder,
	label string,
	fields *dst.FieldList,
	fset *token.FileSet,
) {
	if fields == nil || len(fields.List) == 0 {
		return
	}

	fmt.Fprintf(builder, "%s:", label)

	for _, field := range fields.List {
		for _, name := range field.Names {
			fmt.Fprintf(builder, "%s,", name.Name)
		}

		fmt.Fprintf(builder, "%s;", astutil.ExprToString(fset, field.Type))
	}

	builder.WriteString("\n")
}

// serializeFunc writes a stable string representation of a function declaration.
func serializeFunc(builder *strings.Builder, funcDecl *dst.FuncDecl, fset *token.FileSet) {
	if funcDecl == nil {
		return
	}

	fmt.Fprintf(builder, "name:%s\n", funcDecl.Name.Name)
	fmt.Fprintf(builder, "sig:%s\n", astutil.ExprToString(fset, funcDecl.Type))
	serializeFieldList(builder, "recv", funcDecl.Recv, fset)
}

// serializeFuncType writes a stable string representation of a function type.
func serializeFuncType(
	builder *strings.Builder,
	funcType detect.FuncTypeWithDetails,
	fset *token.FileSet,
) {
	fmt.Fprintf(builder, "name:%s\n", funcType.TypeName)
	fmt.Fprintf(builder, "sig:%s\n", astutil.ExprToString(fset, funcType.FuncType))
	serializeFieldList(builder, "typeparams", funcType.TypeParams, fset)
}

// serializeInterface writes a stable string representation of an interface.
func serializeInterface(
	builder *strings.Builder,
	iface detect.IfaceWithDetails,
	fset *token.FileSet,
) {
	if iface.Iface == nil || iface.Iface.Methods == nil {
		return
	}

	for _, method := range iface.Iface.Methods.List {
		for _, name := range method.Names {
			fmt.Fprintf(builder, "method:%s\n", name.Name)
		}

		fmt.Fprintf(builder, "type:%s\n", astutil.ExprToString(fset, method.Type))
	}

	serializeFieldList(builder, "typeparams", iface.TypeParams, fset)
}

// serializeStruct writes a stable string representation of a struct type.
func serializeStruct(
	builder *strings.Builder,
	structType detect.StructWithDetails,
	fset *token.FileSet,
) {
	fmt.Fprintf(builder, "name:%s\n", structType.TypeName)
	serializeFieldList(builder, "typeparams", structType.TypeParams, fset)
}
