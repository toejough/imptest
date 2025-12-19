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
	"sync"
	"testing"

	"github.com/toejough/imptest/impgen/run"
	"golang.org/x/tools/go/packages"
)

var (
	// globalLoadCache persists across all test runs in this process to minimize expensive go/packages.Load calls.
	//
	//nolint:gochecknoglobals // This global cache is intentional to speed up tests across the entire package.
	globalLoadCache = make(map[string]loadResult)
	//nolint:gochecknoglobals // Global mutex to protect the global cache.
	globalLoadMu sync.RWMutex
)

type loadResult struct {
	files []*ast.File
	fset  *token.FileSet
	info  *types.Info
	err   error
}

type uatTestCase struct {
	name    string
	args    []string
	pkgName string
}

// TestUATConsistency ensures that the generated files in the UAT directory
// are exactly what the current generator code produces.
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
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			verifyUATFile(t, uatDir, loader, tc)
		})
	}
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
// It uses a global cache to avoid redundant work across different test cases.
func (pl *testPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
	cacheKey := fmt.Sprintf("%s|%s", pl.Dir, importPath)

	globalLoadMu.RLock()

	res, ok := globalLoadCache[cacheKey]

	globalLoadMu.RUnlock()

	if ok {
		return res.files, res.fset, res.info, res.err
	}

	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
		Dir:   pl.Dir,
	}

	pkgs, err := packages.Load(cfg, importPath)

	var (
		resFiles []*ast.File
		resFset  *token.FileSet
		resInfo  *types.Info
		resErr   error
	)

	switch {
	case err != nil:
		resErr = fmt.Errorf("failed to load package: %w", err)
	case len(pkgs) == 0:
		resErr = fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	default:
		resFiles, resFset, resInfo, resErr = pl.processPackages(pkgs, importPath)
	}

	globalLoadMu.Lock()

	globalLoadCache[cacheKey] = loadResult{
		files: resFiles,
		fset:  resFset,
		info:  resInfo,
		err:   resErr,
	}

	globalLoadMu.Unlock()

	return resFiles, resFset, resInfo, resErr
}

func (pl *testPackageLoader) processPackages(
	pkgs []*packages.Package, importPath string,
) ([]*ast.File, *token.FileSet, *types.Info, error) {
	var (
		allFiles  []*ast.File
		fset      *token.FileSet
		typesInfo *types.Info
	)

	for _, pkg := range pkgs {
		if fset == nil {
			fset = pkg.Fset
		}

		if typesInfo == nil && pkg.TypesInfo != nil {
			typesInfo = pkg.TypesInfo
		}

		allFiles = append(allFiles, pkg.Syntax...)
	}

	if len(allFiles) == 0 {
		var err error
		if len(pkgs[0].Errors) > 0 {
			err = fmt.Errorf("%w: %v", errPackageErrors, pkgs[0].Errors)
		} else {
			err = fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
		}

		return nil, nil, nil, err
	}

	return allFiles, fset, typesInfo, nil
}
