// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
)

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

	code, err := generateCode(info, astFiles, fset, pkgImportPath, pkgLoader)
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
	pkgImportPath string,
	pkgLoader PackageLoader,
) (string, error) {
	if info.isCallable {
		return generateCallableWrapperCode(astFiles, info, fset, pkgImportPath, pkgLoader)
	}

	return generateImplementationCode(astFiles, info, fset, pkgImportPath)
}

// Functions - Private

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
		isCallable:         parsed.Call,
	}, nil
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
