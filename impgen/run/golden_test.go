package run_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/dst"
	"github.com/toejough/imptest/impgen/run"
)

// TestUATConsistency ensures that the generated files in the UAT directory
// are exactly what the current generator code produces.
//
//nolint:paralleltest // This test must be sequential because it uses t.Chdir which is not thread-safe.
func TestUATConsistency(t *testing.T) {
	// Project root relative to this test file.
	cfs := realCacheFS{}

	projectRoot, err := run.FindProjectRoot(cfs)
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

// testPackageLoader implements PackageLoader using direct DST parsing.
type testPackageLoader struct {
	ProjectRoot string
	ScenarioDir string
}

// Load loads a package by import path and returns its DST files and FileSet.
// Uses the shared LoadPackageDST function for direct DST parsing.
func (pl *testPackageLoader) Load(importPath string) ([]*dst.File, *token.FileSet, *types.Info, error) {
	files, fset, err := run.LoadPackageDST(importPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load package %q: %w", importPath, err)
	}

	// Return nil for typesInfo - we use syntax-based type detection
	return files, fset, nil, nil
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
	loader *testPackageLoader,
	testCase uatTestCase,
) {
	t.Helper()

	fullArgs := append([]string{"impgen"}, testCase.args...)

	// Change directory to the scenario dir to run matching how CLI is used
	t.Chdir(testCase.dir)

	getEnv := func(key string) string {
		if key == "GOPACKAGE" { //nolint:goconst // Test file can't access unexported constant from run package
			return testCase.pkgName
		}

		if key == "GOFILE" {
			return testCase.sourceFile
		}

		return ""
	}

	// Mock FileSystem: Instead of writing, we read the existing file and compare
	fileSystem := &verifyingFileSystem{
		t:       t,
		baseDir: ".",
	}

	err := run.Run(fullArgs, getEnv, fileSystem, loader, io.Discard)
	if err != nil {
		t.Errorf("Run failed for %v: %v", testCase.name, err)
	}
}
