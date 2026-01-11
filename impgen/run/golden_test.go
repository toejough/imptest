package run_test

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/toejough/imptest/impgen/run"
	cache "github.com/toejough/imptest/impgen/run/1_cache"
)

const goPackageEnvVar = "GOPACKAGE"

// TestUATConsistency ensures that the generated files in the UAT directory
// are exactly what the current generator code produces.
func TestUATConsistency(t *testing.T) {
	t.Parallel()
	// Project root relative to this test file.
	cfs := realCacheFS{}

	projectRoot, err := cache.FindProjectRoot(cfs)
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
			t.Parallel()
			verifyUATFile(t, loader, tc)
		})
	}
}

// unexported variables.
var (
	packageCache   = make(map[string]cachedPackage)
	packageCacheMu sync.RWMutex
)

// cachedPackage holds parsed package data.
type cachedPackage struct {
	files []*dst.File
	fset  *token.FileSet
}

// realCacheFS implements cache.FileSystem for production use in tests.
type realCacheFS struct{}

func (realCacheFS) Create(path string) (io.WriteCloser, error) { return os.Create(path) }

func (realCacheFS) Getwd() (string, error) { return os.Getwd() }

func (realCacheFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

func (realCacheFS) Open(path string) (io.ReadCloser, error) { return os.Open(path) }

func (realCacheFS) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

// testPackageLoader implements PackageLoader using direct DST parsing.
type testPackageLoader struct {
	ProjectRoot string
	ScenarioDir string
	WorkDir     string // Working directory for package resolution
}

// Load loads a package by import path and returns its DST files and FileSet.
// Uses in-memory caching to avoid re-parsing the same packages across test cases.
func (pl *testPackageLoader) Load(
	importPath string,
) ([]*dst.File, *token.FileSet, *types.Info, error) {
	// Use WorkDir if set, otherwise fall back to current directory
	workDir := pl.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	// Create cache key that includes working directory
	// since package resolution depends on the source directory
	cacheKey := workDir + "|" + importPath

	// Check cache first
	packageCacheMu.RLock()

	if cached, ok := packageCache[cacheKey]; ok {
		packageCacheMu.RUnlock()
		return cached.files, cached.fset, nil, nil
	}

	packageCacheMu.RUnlock()

	// Cache miss - load package from the specified working directory
	files, fset, err := pl.loadPackageFromDir(importPath, workDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(
			"failed to load package %q from %q: %w",
			importPath,
			workDir,
			err,
		)
	}

	// Store in cache
	packageCacheMu.Lock()

	packageCache[cacheKey] = cachedPackage{files: files, fset: fset}

	packageCacheMu.Unlock()

	// Return nil for typesInfo - we use syntax-based type detection
	return files, fset, nil, nil
}

// loadPackageFromDir loads a package from a specific working directory.
// This is similar to run.LoadPackageDST but works from an explicit directory.
//
//nolint:cyclop,funlen // Package loading mirrors production code structure
func (pl *testPackageLoader) loadPackageFromDir(
	importPath, workDir string,
) ([]*dst.File, *token.FileSet, error) {
	// Resolve import path to directory
	var dir string

	//nolint:nestif // Path resolution requires conditional logic
	if importPath == "." {
		// Use the specified working directory
		dir = workDir
	} else {
		// Check if it's a local subdirectory package relative to workDir
		resolvedPath := pl.resolveLocalPackageFromDir(importPath, workDir)

		if resolvedPath != importPath {
			// It's a local package - use the resolved absolute path
			dir = resolvedPath
		} else {
			// Use go/build to resolve the import path from the specified working directory
			pkg, err := build.Import(importPath, workDir, build.FindOnly)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to find package %q from %q: %w", importPath, workDir, err)
			}

			dir = pkg.Dir
		}
	}

	// Find all .go files
	// For local packages (importPath == "."), include test files
	// For external/stdlib packages, exclude test files to avoid parse errors
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	includeTests := (importPath == ".")

	goFiles := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		// Skip test files for non-local packages
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			continue
		}

		goFiles = append(goFiles, filepath.Join(dir, name))
	}

	if len(goFiles) == 0 {
		return nil, nil, fmt.Errorf("no .go files in %s", dir)
	}

	// Parse all files using DST
	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	allFiles := make([]*dst.File, 0, len(goFiles))

	for _, goFile := range goFiles {
		dstFile, err := dec.ParseFile(goFile, nil, 0)
		if err != nil {
			// Skip files with parse errors
			continue
		}

		allFiles = append(allFiles, dstFile)
	}

	if len(allFiles) == 0 {
		return nil, nil, fmt.Errorf("failed to parse any .go files in %s", dir)
	}

	return allFiles, fset, nil
}

// resolveLocalPackageFromDir checks if importPath refers to a local subdirectory package
// relative to the specified workDir. This is similar to run.ResolveLocalPackagePath
// but works from an explicit directory instead of using os.Getwd().
func (pl *testPackageLoader) resolveLocalPackageFromDir(importPath, workDir string) string {
	// Only check for simple package names (no slashes, not ".", not absolute paths)
	if importPath == "." || strings.HasPrefix(importPath, "/") ||
		strings.Contains(importPath, "/") {
		return importPath
	}

	localDir := filepath.Join(workDir, importPath)

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

type uatTestCase struct {
	name       string
	args       []string
	pkgName    string
	dir        string
	sourceFile string
}

type verifyingFileSystem struct {
	t       *testing.T
	baseDir string
}

func (v *verifyingFileSystem) Glob(pattern string) ([]string, error) {
	// Delegate to real filesystem in the base directory
	matches, err := filepath.Glob(filepath.Join(v.baseDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("glob failed for pattern %s: %w", pattern, err)
	}

	// Strip the base directory prefix from results
	for idx, match := range matches {
		rel, err := filepath.Rel(v.baseDir, match)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", match, err)
		}

		matches[idx] = rel
	}

	return matches, nil
}

func (v *verifyingFileSystem) ReadFile(name string) ([]byte, error) {
	path := filepath.Join(v.baseDir, name)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return data, nil
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

func parseGenerateComment(text, pkgName, dir, sourceFile string) (uatTestCase, bool) {
	if !strings.HasPrefix(text, "//go:generate") {
		return uatTestCase{}, false
	}

	if !strings.Contains(text, "impgen") {
		return uatTestCase{}, false
	}

	fields := strings.Fields(text)

	var args []string

	// iterate till we find "impgen", then collect the rest as args
	foundImpgen := false
	for _, field := range fields {
		if foundImpgen {
			args = append(args, field)
		} else if strings.HasSuffix(field, "impgen") {
			foundImpgen = true
		}
	}

	if len(args) == 0 {
		return uatTestCase{}, false
	}

	// Use the interface/function name as the test case name
	return uatTestCase{
		name:       args[0],
		args:       args,
		pkgName:    pkgName,
		dir:        dir,
		sourceFile: sourceFile,
	}, true
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
				if tc, ok := parseGenerateComment(c.Text, fileAst.Name.Name, filepath.Dir(path), filepath.Base(path)); ok {
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

func verifyUATFile(
	t *testing.T,
	baseLoader *testPackageLoader,
	testCase uatTestCase,
) {
	t.Helper()

	fullArgs := append([]string{"impgen"}, testCase.args...)

	// Create a per-test loader with the scenario directory as WorkDir
	// This enables parallel test execution without changing the working directory
	testLoader := &testPackageLoader{
		ProjectRoot: baseLoader.ProjectRoot,
		ScenarioDir: baseLoader.ScenarioDir,
		WorkDir:     testCase.dir,
	}

	getEnv := func(key string) string {
		if key == goPackageEnvVar {
			return testCase.pkgName
		}

		if key == "GOFILE" {
			return testCase.sourceFile
		}

		return ""
	}

	// Mock FileSystem: Use the test case directory as baseDir
	fileSystem := &verifyingFileSystem{
		t:       t,
		baseDir: testCase.dir,
	}

	err := run.Run(fullArgs, getEnv, fileSystem, testLoader, io.Discard)
	if err != nil {
		t.Errorf("Run failed for %v: %v", testCase.name, err)
	}
}
