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

	// loadSemaphore limits concurrent packages.Load calls to prevent GC thrashing.
	// We limit to 2 concurrent loads as they are extremely memory intensive.
	//
	//nolint:gochecknoglobals
	loadSemaphore = make(chan struct{}, 2)
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
	dir     string
}

// TestUATConsistency ensures that the generated files in the UAT directory
// are exactly what the current generator code produces.
//
//nolint:paralleltest // This test must be sequential because it uses t.Chdir which is not thread-safe.
func TestUATConsistency(t *testing.T) {
	// Project root relative to this test file.
	projectRoot, err := run.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	uatDir := filepath.Join(projectRoot, "UAT")

	testCases, err := scanUATDirectives(uatDir)
	if err != nil {
		t.Fatalf("failed to scan UAT directives: %v", err)
	}

	if len(testCases) == 0 {
		t.Fatal("no //go:generate imptest directives found in UAT directory")
	}

	loader := &testPackageLoader{
		ProjectRoot: projectRoot,
		ScenarioDir: uatDir,
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			verifyUATFile(t, loader, tc)
		})
	}
}

func verifyUATFile(
	t *testing.T,
	loader *testPackageLoader,
	testCase uatTestCase,
) {
	t.Helper()

	// 1. Check disk cache if possible
	fullArgs := append([]string{"impgen"}, testCase.args...)

	if tryDiskCache(t, loader.ProjectRoot, testCase.dir, fullArgs) {
		return
	}

	// Change directory to the scenario dir to run matching how CLI is used
	t.Chdir(testCase.dir)

	getEnv := func(key string) string {
		if key == "GOPACKAGE" {
			return testCase.pkgName
		}

		return ""
	}

	// Mock FileSystem: Instead of writing, we read the existing file and compare
	fileSystem := &verifyingFileSystem{
		t:       t,
		baseDir: ".",
	}

	err := run.Run(fullArgs, getEnv, fileSystem, loader)
	if err != nil {
		t.Errorf("Run failed for %v: %v", testCase.name, err)
	}
}

func tryDiskCache(t *testing.T, projectRoot, scenarioDir string, fullArgs []string) bool {
	t.Helper()

	// Change to scenario dir for signature calculation (it globs for files)
	t.Chdir(scenarioDir)

	sig, err := run.CalculatePackageSignature(fullArgs)
	if err != nil {
		return false
	}

	cachePath := filepath.Join(projectRoot, run.CacheDirName, "cache.json")
	cache := run.LoadDiskCache(cachePath)
	key := strings.Join(fullArgs[1:], " ")

	entry, ok := cache.Entries[key]
	if !ok || entry.Signature != sig {
		return false
	}

	// Cache hit! Verify content matches disk.
	path := filepath.Join(scenarioDir, entry.Filename)

	actualData, err := os.ReadFile(path)
	if err != nil {
		// If file is missing, we can't verify consistency via cache hit.
		// Return false to fall back to full Run().
		return false
	}

	if string(actualData) != entry.Content {
		t.Errorf("Cached content for %s differs from disk file", path)
	}

	return true
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
				if tc, ok := parseGenerateComment(c.Text, fileAst.Name.Name, filepath.Dir(path)); ok {
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

func parseGenerateComment(text, pkgName, dir string) (uatTestCase, bool) {
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
		dir:     dir,
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
	ProjectRoot string
	ScenarioDir string
}

var (
	errNoPackagesFound = errors.New("no packages found")
	errPackageErrors   = errors.New("package errors")
)

// Load loads a package by import path and returns its AST files, FileSet, and type information.
// It uses a shared cache to avoid redundant work across different test cases.
func (pl *testPackageLoader) Load(importPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
	// Consolidate Dir to project root for better go/packages internal caching.
	// If the path is ".", use the ScenarioDir.
	loadPath := importPath
	if importPath == "." {
		loadPath = pl.ScenarioDir
	}

	cacheKey := fmt.Sprintf("%s|%s", pl.ProjectRoot, loadPath)

	globalLoadMu.RLock()

	res, ok := globalLoadCache[cacheKey]

	globalLoadMu.RUnlock()

	if ok {
		return res.files, res.fset, res.info, res.err
	}

	resFiles, resFset, resInfo, resErr := pl.loadWithThrottle(loadPath)

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

func (pl *testPackageLoader) loadWithThrottle(loadPath string) ([]*ast.File, *token.FileSet, *types.Info, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax,
		Tests: true,
		Dir:   pl.ProjectRoot,
	}

	// Acquire semaphore
	loadSemaphore <- struct{}{}

	pkgs, err := packages.Load(cfg, loadPath)
	// Release semaphore
	<-loadSemaphore

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
		resErr = fmt.Errorf("%w: %q", errNoPackagesFound, loadPath)
	default:
		resFiles, resFset, resInfo, resErr = pl.processPackages(pkgs, loadPath)
	}

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
