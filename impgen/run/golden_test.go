package run_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toejough/imptest/impgen/run"
	"golang.org/x/tools/go/packages"
)

// TestUATConsistency ensures that the generated files in the UAT directory
// are exactly what the current generator code produces.
// This serves two purposes:
// 1. It provides high code coverage for the generator logic (since we call Run directly).
// 2. It ensures the UAT examples are always up-to-date.
func TestUATConsistency(t *testing.T) {
	t.Parallel()

	uatDir, err := filepath.Abs("../../UAT/run")
	if err != nil {
		t.Fatalf("failed to get absolute path for UAT directory: %v", err)
	}

	testCases, err := scanUATDirectives(uatDir)
	if err != nil {
		t.Fatalf("failed to scan UAT directives: %v", err)
	}

	if len(testCases) == 0 {
		t.Fatal("no //go:generate imptest directives found in UAT directory")
	}

	loader := &testPackageLoader{Dir: uatDir}

	for _, testCase := range testCases {
		testCaseScoped := testCase
		t.Run(testCaseScoped.name, func(t *testing.T) {
			t.Parallel()
			verifyUATFile(t, uatDir, loader, testCaseScoped)
		})
	}
}

type uatTestCase struct {
	name    string
	args    []string
	pkgName string
}

func verifyUATFile(
	t *testing.T,
	uatDir string,
	loader run.PackageLoader,
	testCase uatTestCase,
) {
	t.Helper()

	getEnv := func(key string) string {
		if key == "GOPACKAGE" {
			return testCase.pkgName
		}

		return ""
	}

	// Mock FileSystem: Instead of writing, we read the existing file and compare
	fileSystem := &verifyingFileSystem{
		t:       t,
		baseDir: uatDir,
	}

	// Note: program name is expected as the first argument by go-arg
	fullArgs := append([]string{"impgen"}, testCase.args...)

	err := run.Run(fullArgs, getEnv, fileSystem, loader)
	if err != nil {
		t.Errorf("Run failed for %v: %v", testCase.name, err)
	}
}

func scanUATDirectives(dir string) ([]uatTestCase, error) {
	var testCases []uatTestCase

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()

		fileAst, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil //nolint:nilerr // skip files that don't parse
		}

		for _, cg := range fileAst.Comments {
			for _, c := range cg.List {
				if tc, ok := parseGenerateComment(c.Text, fileAst.Name.Name); ok {
					testCases = append(testCases, tc)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk UAT directory: %w", err)
	}

	return testCases, nil
}

func parseGenerateComment(text, pkgName string) (uatTestCase, bool) {
	if !strings.HasPrefix(text, "//go:generate") {
		return uatTestCase{}, false
	}

	if !strings.Contains(text, "impgen/main.go") {
		return uatTestCase{}, false
	}

	fields := strings.Fields(text)

	var args []string

	foundMain := false
	for _, field := range fields {
		if foundMain {
			args = append(args, field)
		} else if strings.HasSuffix(field, "impgen/main.go") {
			foundMain = true
		}
	}

	if len(args) == 0 {
		return uatTestCase{}, false
	}

	// Use the interface/function name as the test case name
	return uatTestCase{
		name:    args[0],
		args:    args,
		pkgName: pkgName,
	}, true
}

// verifyingFileSystem implements FileSystem.
// It reads the file from disk that *would* be overwritten and compares content.
type verifyingFileSystem struct {
	t       *testing.T
	baseDir string
}

func (v *verifyingFileSystem) WriteFile(name string, data []byte, _ os.FileMode) error {
	path := filepath.Join(v.baseDir, name)
	// Read the actual committed file from the UAT directory
	expectedData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read expected file %s: %w", path, err)
	}

	// Compare generated content with committed content
	if string(expectedData) != string(data) {
		v.t.Errorf("Generated code differs from UAT golden file: %s", path)
	}

	return nil
}

// testPackageLoader implements PackageLoader using golang.org/x/tools/go/packages.
type testPackageLoader struct {
	Dir string
}

var (
	errNoPackagesFound = errors.New("no packages found")
	errPackageErrors   = errors.New("package errors")
)

// Load loads a package by import path and returns its AST files, FileSet, and type information.
func (pl *testPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
		Dir:   pl.Dir,
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	}

	var (
		allFiles  []*ast.File
		fset      *token.FileSet
		typesInfo *types.Info
	)

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		if fset == nil {
			fset = pkg.Fset
		}

		if typesInfo == nil && pkg.TypesInfo != nil {
			typesInfo = pkg.TypesInfo
		}

		allFiles = append(allFiles, pkg.Syntax...)
	}

	if len(allFiles) == 0 {
		if len(pkgs[0].Errors) > 0 {
			return nil, nil, nil, fmt.Errorf("%w: %v", errPackageErrors, pkgs[0].Errors)
		}

		return nil, nil, nil, fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	}

	return allFiles, fset, typesInfo, nil
}
