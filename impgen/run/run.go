// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	go_types "go/types" // Aliased import
	"os"
	"strings"

	"github.com/alexflint/go-arg"
)

// Vars.
var (
	errGOPACKAGENotSet = errors.New("GOPACKAGE environment variable not set")
)

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

	astFiles, fset, typesInfo, err := loadPackage(pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	code, err := generateCode(info, astFiles, fset, typesInfo, pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	err = writeGeneratedCodeToFile(code, info.impName, info.pkgName, fileSys)
	if err != nil {
		return err
	}

	return nil
}

// generateCode generates the Go code based on the generatorInfo and AST files.
func generateCode(
	info generatorInfo,
	astFiles []*ast.File,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
) (string, error) {
	if info.isCallable {
		return generateCallableWrapperCode(astFiles, info, fset, typesInfo, pkgImportPath, pkgLoader)
	}

	ifaceWithDetails, err := getMatchingInterfaceFromAST(astFiles, info.localInterfaceName, pkgImportPath)
	if err != nil {
		return "", err
	}

	return generateImplementationCode(astFiles, info, fset, typesInfo, pkgImportPath, pkgLoader, ifaceWithDetails)
}

// Interfaces - Public

// FileSystem interface for mocking.
type FileSystem interface {
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// Structs - Private

// cliArgs defines the command-line arguments for the generator.
type cliArgs struct {
	Interface string `arg:"positional,required" help:"interface name to implement (e.g. MyInterface or pkg.MyInterface)"`
	Name      string `arg:"--name"              help:"name for the generated implementation (defaults to <Interface>Imp)"`
	Call      bool   `arg:"--call"              help:"generate a type-safe callable wrapper instead of interface mock"`
}

// generatorInfo holds information gathered for generation.
type generatorInfo struct {
	pkgName, interfaceName, localInterfaceName, impName string
	isCallable                                          bool
}

// Functions - Private

// getGeneratorCallInfo returns basic information about the current call to the generator.
func getGeneratorCallInfo(args []string, getEnv func(string) string) (generatorInfo, error) {
	pkgName := getEnv("GOPACKAGE")
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

	// set impname if not provided
	if impName == "" {
		impName = localInterfaceName + "Imp" // default implementation name
	}

	return generatorInfo{
		pkgName:            pkgName,
		interfaceName:      interfaceName,
		localInterfaceName: localInterfaceName,
		impName:            impName,
		isCallable:         parsed.Call,
	}, nil
}

// getInterfacePackagePath resolves the import path for the package containing the target interface.
func getInterfacePackagePath(interfaceName string, pkgLoader PackageLoader) (string, error) {
	if !strings.Contains(interfaceName, ".") {
		return ".", nil
	}

	pkgName := extractPackageName(interfaceName)

	astFiles, _, _, err := pkgLoader.Load(".")
	if err != nil {
		return "", fmt.Errorf("failed to load local package to resolve import: %w", err)
	}

	return findImportPath(astFiles, pkgName, pkgLoader)
}

// getLocalInterfaceName extracts the name of the interface without the package prefix.
func getLocalInterfaceName(interfaceName string) string {
	parts := strings.Split(interfaceName, ".")
	if len(parts) > 1 {
		return strings.Join(parts[1:], ".")
	}

	return interfaceName
}

// loadPackage loads the AST and type info for the package at the given path.
func loadPackage(pkgPath string, pkgLoader PackageLoader) ([]*ast.File, *token.FileSet, *go_types.Info, error) {
	astFiles, fset, typesInfo, err := pkgLoader.Load(pkgPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load package %s: %w", pkgPath, err)
	}

	return astFiles, fset, typesInfo, nil
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

// writeGeneratedCodeToFile writes the generated code to <impName>.go.
func writeGeneratedCodeToFile(code string, impName string, pkgName string, fileSys FileSystem) error {
	const generatedFilePermissions = 0o600

	filename := impName
	// If we're in a test package, append _test to the filename
	if strings.HasSuffix(pkgName, "_test") && !strings.HasSuffix(impName, "_test") {
		filename = strings.TrimSuffix(impName, ".go") + "_test.go"
	} else if !strings.HasSuffix(filename, ".go") {
		filename += ".go"
	}

	err := fileSys.WriteFile(filename, []byte(code), generatedFilePermissions)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	fmt.Printf("%s written successfully.\n", filename)

	return nil
}
