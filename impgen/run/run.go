// Package run implements the main logic for the impgen tool in a testable way.
package run

import (
	"fmt"
	"go/ast"
	"go/token"
	go_types "go/types" // Aliased import
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/toejough/imptest/impgen/reorder"
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

	pkgImportPath, err := getInterfacePackagePath(info.interfaceName, pkgLoader)
	if err != nil {
		return err
	}

	// If it's a local package, we should use the full name for symbol lookup
	// (e.g. "MyType.MyMethod" instead of just "MyMethod")
	if pkgImportPath == "." {
		info.localInterfaceName = info.interfaceName
	}

	astFiles, fset, typesInfo, err := loadPackage(pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	code, err := generateCode(info, astFiles, fset, typesInfo, pkgImportPath, pkgLoader)
	if err != nil {
		return err
	}

	err = writeGeneratedCodeToFile(code, info.impName, info.pkgName, fileWriter, output)
	if err != nil {
		return err
	}

	return nil
}

// Vars.

// Functions - Public

// WithCache executes the impgen tool with caching support. It checks if a cached version exists based on the
// package signature and uses it if available. Otherwise, it generates new code and caches the result.
func WithCache(
	args []string,
	getEnv func(string) string,
	fileSys FileSystem,
	pkgLoader PackageLoader,
	cacheFS CacheFileSystem,
	output io.Writer,
) error {
	// 1. Calculate current signature
	sig, err := CalculatePackageSignature(args, fileSys)
	if err != nil {
		// If signature calculation fails, run without cache
		return Run(args, getEnv, fileSys, pkgLoader, output)
	}

	// 2. Find project root and cache file
	root, err := FindProjectRoot(cacheFS)
	if err != nil {
		// If project root not found, run without cache
		return Run(args, getEnv, fileSys, pkgLoader, output)
	}

	cachePath := filepath.Join(root, CacheDirName, "cache.json")
	cache := LoadDiskCache(cachePath, cacheFS)

	// 3. Check cache
	key := strings.Join(args[1:], " ")
	if entry, ok := cache.Entries[key]; ok && entry.Signature == sig {
		// Cache hit! Just write the file if it doesn't exist or differs
		err = fileSys.WriteFile(entry.Filename, []byte(entry.Content), FilePerm)
		if err != nil {
			return fmt.Errorf("error writing from cache: %w", err)
		}

		return nil
	}

	// 4. Cache miss - run and record
	capturingSys := &capturingFileSystem{underlying: fileSys}

	err = Run(args, getEnv, capturingSys, pkgLoader, output)
	if err != nil {
		return err
	}

	// 5. Update cache
	if capturingSys.writtenName != "" {
		if cache.Entries == nil {
			cache.Entries = make(map[string]CacheEntry)
		}

		cache.Entries[key] = CacheEntry{
			Signature: sig,
			Content:   capturingSys.writtenContent,
			Filename:  capturingSys.writtenName,
		}
		SaveDiskCache(cachePath, cache, cacheFS)
	}

	return nil
}

// capturingFileSystem wraps a FileWriter and captures what was written for caching.
type capturingFileSystem struct {
	underlying     FileWriter
	writtenContent string
	writtenName    string
}

// WriteFile implements FileWriter by capturing the written data and delegating to underlying.
func (c *capturingFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	c.writtenContent = string(data)
	c.writtenName = name

	err := c.underlying.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("underlying write failed: %w", err)
	}

	return nil
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

func generateCode(
	info generatorInfo,
	astFiles []*ast.File,
	fset *token.FileSet,
	typesInfo *go_types.Info,
	pkgImportPath string,
	pkgLoader PackageLoader,
) (string, error) {
	// Auto-detect the symbol type
	symbol, err := findSymbol(astFiles, fset, info.localInterfaceName, pkgImportPath)
	if err != nil {
		return "", err
	}

	if symbol.kind == symbolFunction {
		return generateCallableWrapperCode(astFiles, info, fset, typesInfo, pkgImportPath, pkgLoader)
	}

	return generateImplementationCode(astFiles, info, fset, typesInfo, pkgImportPath, pkgLoader, symbol.iface)
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

// writeGeneratedCodeToFile writes the generated code to generated_<impName>.go.
func writeGeneratedCodeToFile(
	code string, impName string, pkgName string, fileWriter FileWriter, output io.Writer,
) error {
	const generatedFilePermissions = 0o600

	filename := "generated_" + impName
	// If we're in a test package, append _test to the filename
	if strings.HasSuffix(pkgName, "_test") && !strings.HasSuffix(impName, "_test") {
		filename = "generated_" + strings.TrimSuffix(impName, ".go") + "_test.go"
	} else if !strings.HasSuffix(filename, ".go") {
		filename += ".go"
	}

	// Reorder declarations according to project conventions
	reordered, err := reorder.Source(code)
	if err != nil {
		// If reordering fails, log but continue with original code
		fmt.Fprintf(output, "Warning: failed to reorder %s: %v\n", filename, err)

		reordered = code
	}

	err = fileWriter.WriteFile(filename, []byte(reordered), generatedFilePermissions)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	fmt.Fprintf(output, "%s written successfully.\n", filename)

	return nil
}
