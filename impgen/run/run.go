// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/dave/dst"
	"github.com/toejough/go-reorder"
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
	FileWriter
}

// FileWriter interface for writing generated code.
type FileWriter interface {
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// Run executes the impgen tool logic. It takes command-line arguments, an environment variable getter, a FileWriter
// interface for file operations, a PackageLoader for package operations, and an io.Writer for output messages. It
// returns an error if any step fails. On success, it generates a Go source file implementing the specified interface,
// in the calling test package.
func Run(
	args []string, getEnv func(string) string, fileWriter FileWriter, pkgLoader PackageLoader, output io.Writer,
) error {
	info, err := getGeneratorCallInfo(args, getEnv)
	if err != nil {
		return err
	}

	pkgImportPath, err := getInterfacePackagePath(info.interfaceName, pkgLoader, info.importPathFlag, getEnv)
	if err != nil {
		return err
	}

	// If it's a local package, we should use the full name for symbol lookup
	// (e.g. "MyType.MyMethod" instead of just "MyMethod")
	if pkgImportPath == "." {
		info.localInterfaceName = info.interfaceName
	}

	astFiles, fset, err := loadPackage(pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	code, err := generateCode(info, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	err = writeGeneratedCodeToFile(code, info.impName, info.pkgName, getEnv, fileWriter, output)
	if err != nil {
		return err
	}

	return nil
}

// Functions - Public

// WithCache executes the impgen tool with caching support. It checks if a cached version exists based on the
// package signature and uses it if available. Otherwise, it generates new code and caches the result.

// 1. Calculate current signature

// If signature calculation fails, run without cache

// 2. Find project root and cache file

// If project root not found, run without cache

// 3. Check cache

// Cache hit! Just write the file if it doesn't exist or differs

// 4. Cache miss - run and record

// 5. Update cache

// namingMode represents the different modes for generating type names.
type namingMode int

// namingMode values.
const (
	namingModeDefault namingMode = iota
	namingModeTarget
	namingModeDependency
)

// unexported variables.
var (
	errAmbiguousPackage       = errors.New("package is ambiguous: both stdlib and local package exist")
	errMutuallyExclusiveFlags = errors.New("--target and --dependency flags are mutually exclusive")
)

// Structs - Private

// cliArgs defines the command-line arguments for the generator.
type cliArgs struct {
	Interface  string `arg:"positional,required" help:"interface or function name to wrap/mock"`
	Name       string `arg:"--name"              help:"name for the generated code (overrides default naming)"`
	Target     bool   `arg:"--target"            help:"generate target wrapper (WrapXxx) instead of dependency mock"`
	Dependency bool   `arg:"--dependency"        help:"generate dependency mock (MockXxx) - this is the default behavior"`
	ImportPath string `arg:"--import-path"       help:"explicit import path when ambiguous"`
}

// generatorInfo holds information gathered for generation.
type generatorInfo struct {
	pkgName, interfaceName, localInterfaceName, impName string
	mode                                                namingMode
	importPathFlag                                      string
}

// determineGeneratedTypeName generates the type name based on the naming mode and interface name.
func determineGeneratedTypeName(mode namingMode, localInterfaceName string) string {
	// Remove dots from localInterfaceName to create valid Go type names
	// e.g., "Calculator.Add" -> "CalculatorAdd", "MyInterface" -> "MyInterface"
	typeName := strings.ReplaceAll(localInterfaceName, ".", "")

	switch mode {
	case namingModeTarget:
		return "Wrap" + typeName
	case namingModeDependency:
		return "Mock" + typeName
	case namingModeDefault:
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

func generateCode(
	info generatorInfo,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
) (string, error) {
	// Auto-detect the symbol type
	symbol, err := findSymbol(astFiles, fset, info.localInterfaceName, pkgImportPath, pkgLoader)
	if err != nil {
		return "", err
	}

	// Use the actual package path where the symbol was found
	// (important for dot imports where symbol.pkgPath differs from pkgImportPath)
	actualPkgPath := symbol.pkgPath

	// If symbol was found via dot import, we need to load that package's AST
	if actualPkgPath != pkgImportPath {
		astFiles, fset, _, err = pkgLoader.Load(actualPkgPath)
		if err != nil {
			return "", fmt.Errorf("failed to load package %s: %w", actualPkgPath, err)
		}
	}

	// Route to appropriate generator based on symbol type and mode
	return routeToGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol)
}

// Functions - Private

// getGeneratorCallInfo returns basic information about the current call to the generator.
func getGeneratorCallInfo(args []string, getEnv func(string) string) (generatorInfo, error) {
	pkgName := getEnv(goPackageEnvVarName)
	if pkgName == "" {
		return generatorInfo{}, errGOPACKAGENotSet
	}

	parsed, err := parseArgs(args)
	if err != nil {
		return generatorInfo{}, err
	}

	interfaceName := parsed.Interface
	localInterfaceName := getLocalInterfaceName(interfaceName)
	impName := parsed.Name

	// Validate mutually exclusive flags
	if parsed.Target && parsed.Dependency {
		return generatorInfo{}, errMutuallyExclusiveFlags
	}

	// Determine naming mode based on flags
	mode := namingModeDefault
	if parsed.Target {
		mode = namingModeTarget
	} else if parsed.Dependency {
		mode = namingModeDependency
	}

	// set impname if not provided
	if impName == "" {
		impName = determineGeneratedTypeName(mode, localInterfaceName)
	}

	return generatorInfo{
		pkgName:            pkgName,
		interfaceName:      interfaceName,
		localInterfaceName: localInterfaceName,
		impName:            impName,
		mode:               mode,
		importPathFlag:     parsed.ImportPath,
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
	pkgLoader PackageLoader,
	importPathFlag string,
	getEnv func(string) string,
) (string, error) {
	if !strings.Contains(interfaceName, ".") {
		return ".", nil
	}

	pkgName := extractPackageName(interfaceName)

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
		inferredPath, err := inferImportPathFromTestFile(goFile, pkgName)
		if err == nil {
			// Found in imports - use it
			return inferredPath, nil
		}
		// Not found in imports - continue to next tier
	}

	// Tier 3: Ambiguity detection
	hasStdlib, hasLocal, localPath := detectPackageAmbiguity(pkgName)
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
	path, err := findImportPath(astFiles, pkgName, pkgLoader)
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

// loadPackage loads the AST for the package at the given path.
func loadPackage(pkgPath string, pkgLoader PackageLoader) ([]*dst.File, *token.FileSet, error) {
	astFiles, fset, _, err := pkgLoader.Load(pkgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load package %s: %w", pkgPath, err)
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

// routeFunctionGenerator routes to function generators based on mode.
func routeFunctionGenerator(
	astFiles []*dst.File, info generatorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader PackageLoader, funcDecl *dst.FuncDecl,
) (string, error) {
	switch info.mode {
	case namingModeTarget:
		return generateTargetCode(astFiles, info, fset, pkgPath, pkgLoader, funcDecl)
	case namingModeDependency:
		return generateFunctionDependencyCode(astFiles, info, fset, pkgPath, pkgLoader, funcDecl)
	case namingModeDefault:
		return "", ErrFunctionModeRequired
	}

	return "", ErrFunctionModeRequired
}

// routeFunctionTypeGenerator routes to function type generators based on mode.
func routeFunctionTypeGenerator(
	astFiles []*dst.File, info generatorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader PackageLoader, funcType funcTypeWithDetails,
) (string, error) {
	switch info.mode {
	case namingModeTarget:
		return generateTargetCodeFromFuncType(astFiles, info, fset, pkgPath, pkgLoader, funcType)
	case namingModeDependency:
		return generateFunctionTypeDependencyCode(astFiles, info, fset, pkgPath, pkgLoader, funcType)
	case namingModeDefault:
		return "", ErrFunctionModeRequired
	}

	return "", ErrFunctionModeRequired
}

// routeInterfaceGenerator routes to interface generators based on mode.
func routeInterfaceGenerator(
	astFiles []*dst.File, info generatorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader PackageLoader, iface ifaceWithDetails,
) (string, error) {
	switch info.mode {
	case namingModeDependency:
		return generateDependencyCode(astFiles, info, fset, pkgPath, pkgLoader, iface)
	case namingModeTarget:
		return generateInterfaceTargetCode(astFiles, info, fset, pkgPath, pkgLoader, iface, false)
	case namingModeDefault:
		return "", ErrInterfaceModeRequired
	}

	return "", ErrInterfaceModeRequired
}

// routeStructGenerator routes to struct generator (only supports target mode).
func routeStructGenerator(
	astFiles []*dst.File, info generatorInfo, fset *token.FileSet,
	pkgPath string, pkgLoader PackageLoader, structType structWithDetails,
) (string, error) {
	if info.mode != namingModeTarget {
		//nolint:err113 // Dynamic error message for user-facing validation
		return "", fmt.Errorf("struct types only support --target flag (use 'impgen %s --target')", info.localInterfaceName)
	}

	return generateStructTargetCode(astFiles, info, fset, pkgPath, pkgLoader, structType)
}

// routeToGenerator routes to the appropriate generator based on symbol type and mode.
func routeToGenerator(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	actualPkgPath string,
	pkgLoader PackageLoader,
	symbol symbolDetails,
) (string, error) {
	switch symbol.kind {
	case symbolFunction:
		return routeFunctionGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.funcDecl)
	case symbolFunctionType:
		return routeFunctionTypeGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.funcType)
	case symbolStructType:
		return routeStructGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.structType)
	case symbolInterface:
		return routeInterfaceGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.iface)
	}

	return routeInterfaceGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.iface)
}

// writeGeneratedCodeToFile writes the generated code to generated_<impName>.go.
func writeGeneratedCodeToFile(
	code string, impName string, pkgName string, getEnv func(string) string, fileWriter FileWriter, output io.Writer,
) error {
	const generatedFilePermissions = 0o600

	filename := "generated_" + impName
	// If we're in a test package OR the source file is a test file, append _test to the filename
	// This handles both blackbox testing (package xxx_test) and whitebox testing (package xxx in xxx_test.go)
	goFile := getEnv("GOFILE")

	isTestFile := strings.HasSuffix(pkgName, "_test") || strings.HasSuffix(goFile, "_test.go")
	if isTestFile && !strings.HasSuffix(impName, "_test") {
		filename = "generated_" + strings.TrimSuffix(impName, ".go") + "_test.go"
	} else if !strings.HasSuffix(filename, ".go") {
		filename += ".go"
	}

	// Reorder declarations according to project conventions
	reordered, err := reorder.Source(code)
	if err != nil {
		// If reordering fails, log but continue with original code
		_, _ = fmt.Fprintf(output, "Warning: failed to reorder %s: %v\n", filename, err)

		reordered = code
	}

	err = fileWriter.WriteFile(filename, []byte(reordered), generatedFilePermissions)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	_, _ = fmt.Fprintf(output, "%s written successfully.\n", filename)

	return nil
}
