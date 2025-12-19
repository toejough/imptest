package run_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/imptest/impgen/run"
	"golang.org/x/tools/go/packages"
)

type uatTestCase struct {
	generatedFile string
	args          []string
}

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

	loader := &testPackageLoader{Dir: uatDir}

	for _, testCase := range getUATTestCases() {
		verifyUATFile(t, uatDir, loader, testCase)
	}
}

func verifyUATFile(
	t *testing.T,
	uatDir string,
	loader run.PackageLoader,
	testCase uatTestCase,
) {
	t.Helper()
	t.Run(testCase.generatedFile, func(t *testing.T) {
		t.Parallel()

		getEnv := func(key string) string {
			if key == "GOPACKAGE" {
				return "run_test"
			}

			return ""
		}

		fileSystem := &verifyingFileSystem{
			t:            t,
			expectedPath: filepath.Join(uatDir, testCase.generatedFile),
		}

		err := run.Run(testCase.args, getEnv, fileSystem, loader)
		if err != nil {
			t.Errorf("Run failed: %v", err)
		}
	})
}

func getUATTestCases() []uatTestCase {
	return []uatTestCase{
		{
			generatedFile: "TrackerImp_test.go",
			args:          []string{"program_name", "run.Tracker", "--name", "TrackerImp"},
		},
		{
			generatedFile: "CoinFlipperImp_test.go",
			args:          []string{"program_name", "run.CoinFlipper", "--name", "CoinFlipperImp"},
		},
		{
			generatedFile: "PingPongPlayerImp_test.go",
			args:          []string{"program_name", "run.PingPongPlayer.Play", "--name", "PingPongPlayerImp", "--call"},
		},
		{
			generatedFile: "IntOpsImp_test.go",
			args:          []string{"program_name", "run.IntOps", "--name", "IntOpsImp"},
		},
		{
			generatedFile: "PrintSumImp_test.go",
			args:          []string{"program_name", "run.PrintSum", "--name", "PrintSumImp", "--call"},
		},
	}
}

// verifyingFileSystem implements FileSystem.
// It reads the file from disk that *would* be overwritten and compares content.
type verifyingFileSystem struct {
	t            *testing.T
	expectedPath string
}

func (v *verifyingFileSystem) WriteFile(_ string, data []byte, _ os.FileMode) error {
	// Read the actual committed file from the UAT directory
	expectedData, err := os.ReadFile(v.expectedPath)
	if err != nil {
		return fmt.Errorf("failed to read expected file %s: %w", v.expectedPath, err)
	}

	// Compare generated content with committed content
	if string(expectedData) != string(data) {
		v.t.Errorf("Generated code differs from UAT golden file: %s", v.expectedPath)
		// Note: We could print a diff here, but strict equality check is sufficient for CI.
		// If this fails, the user should inspect the file manually or use a diff tool.
	}

	return nil
}

// testPackageLoader implements PackageLoader using golang.org/x/tools/go/packages.
// Duplicated here for testing purposes to avoid importing main.
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
		Dir:   pl.Dir, // Use the configured directory
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: %q", errNoPackagesFound, importPath)
	}

	// Collect all AST files from all packages (including test packages)
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

		// Use type info from the first valid package
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
