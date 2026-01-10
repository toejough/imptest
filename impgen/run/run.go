// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"errors"
	"fmt"
	"go/token"
	"io"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/dave/dst"
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
func Run(
	args []string, getEnv func(string) string, fileWriter output.Writer, pkgLoader detect.PackageLoader, out io.Writer,
) error {
	info, err := getGeneratorCallInfo(args, getEnv)
	if err != nil {
		return err
	}

	pkgImportPath, err := getInterfacePackagePath(info.InterfaceName, pkgLoader, info.ImportPathFlag, getEnv)
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

	astFiles, fset, err := loadPackage(pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	code, err := generateCode(info, astFiles, fset, pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	err = output.WriteGeneratedCode(code, info.ImpName, info.PkgName, getEnv, fileWriter, out)
	if err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	return nil
}

// unexported constants.
const (
	goPackageEnvVarName = "GOPACKAGE"
)

// unexported variables.
var (
	errAmbiguousPackage       = errors.New("package is ambiguous: both stdlib and local package exist")
	errGOPACKAGENotSet        = errors.New(goPackageEnvVarName + " environment variable not set")
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

// determineGeneratedTypeName generates the type name based on the naming mode and interface name.
func determineGeneratedTypeName(mode generate.NamingMode, localInterfaceName string) string {
	// Remove dots from localInterfaceName to create valid Go type names
	// e.g., "Calculator.Add" -> "CalculatorAdd", "MyInterface" -> "MyInterface"
	typeName := strings.ReplaceAll(localInterfaceName, ".", "")

	switch mode {
	case generate.NamingModeTarget:
		return "Wrap" + typeName
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

func generateCode(
	info generate.GeneratorInfo,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
) (string, error) {
	// Auto-detect the symbol type
	symbol, err := detect.FindSymbol(astFiles, fset, info.LocalInterfaceName, pkgImportPath, pkgLoader)
	if err != nil {
		return "", fmt.Errorf("failed to find symbol %s: %w", info.LocalInterfaceName, err)
	}

	// Use the actual package path where the symbol was found
	// (important for dot imports where symbol.PkgPath differs from pkgImportPath)
	actualPkgPath := symbol.PkgPath

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
func getGeneratorCallInfo(args []string, getEnv func(string) string) (generate.GeneratorInfo, error) {
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
	hasStdlib, hasLocal, localPath := detect.PackageAmbiguity(pkgName)
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

// loadPackage loads the AST for the package at the given path.
func loadPackage(pkgPath string, pkgLoader detect.PackageLoader) ([]*dst.File, *token.FileSet, error) {
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
		return generate.FunctionTypeDependencyCode(astFiles, info, fset, pkgPath, pkgLoader, funcType)
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
		return routeFunctionGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.FuncDecl)
	case detect.SymbolFunctionType:
		return routeFunctionTypeGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.FuncType)
	case detect.SymbolStructType:
		return routeStructGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.StructType)
	case detect.SymbolInterface:
		return routeInterfaceGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.Iface)
	}

	return routeInterfaceGenerator(astFiles, info, fset, actualPkgPath, pkgLoader, symbol.Iface)
}
