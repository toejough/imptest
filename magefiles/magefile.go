//go:build mage
// +build mage

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/akedrou/textdiff"
	"github.com/magefile/mage/mg"
	"github.com/toejough/go-reorder"
)

// BaselineTestSpec specifies a test or set of tests to include in the baseline coverage.
type BaselineTestSpec struct {
	Package     string // Package path (e.g., "./impgen/run" or "./UAT/...")
	TestPattern string // Test name pattern for -run flag (empty string runs all tests in package)
}

// RedundancyConfig configures the redundant test analysis.
type RedundancyConfig struct {
	BaselineTests     []BaselineTestSpec // Tests that form the baseline coverage
	CoverageThreshold float64            // Percentage threshold (e.g., 80.0 for 80%)
	PackageToAnalyze  string             // Package containing tests to analyze (e.g., "./impgen/run")
}

// Build builds the local impgen binary.
func Build(c context.Context) error {
	fmt.Println("Building impgen...")

	// Create bin directory if it doesn't exist
	err := os.MkdirAll("bin", 0o755)
	if err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Build impgen to ./bin/impgen
	return run(c, "go", "build", "-o", "bin/impgen", "./impgen")
}

// Public Functions (Mage Targets)

// Run all checks & fixes on the code, in order of correctness.
func Check(c context.Context) error {
	fmt.Println("Checking...")

	mg.SerialCtxDeps(c,
		Tidy, // clean up the module dependencies
		CheckCoverage,
		CheckNils,
		Deadcode,
		Lint,
		Modernize,
		ReorderDecls,
	)

	return nil
}

// CheckCoverage checks that function coverage meets the minimum threshold.
func CheckCoverage(c context.Context) error {
	fmt.Println("Checking coverage...")
	mg.SerialCtxDeps(c, Test) // Ensure tests have run to generate coverage.out

	// Merge duplicate coverage blocks from cross-package testing
	err := mergeCoverageBlocks()
	if err != nil {
		return fmt.Errorf("failed to merge coverage blocks: %w", err)
	}

	out, err := output(c, "go", "tool", "cover", "-func=coverage.out")
	if err != nil {
		return err
	}

	lines := strings.Split(out, "\n")
	linesAndCoverage := []lineAndCoverage{}

	for _, line := range lines {
		percentString := regexp.MustCompile(`\d+\.\d`).FindString(line)

		percent, err := strconv.ParseFloat(percentString, 64)
		if err != nil {
			return err
		}

		if strings.Contains(line, "_string.go") {
			continue
		}

		if strings.Contains(line, "main.go") {
			continue
		}

		if strings.Contains(line, "generated_") {
			continue
		}

		// TODO(Phase 2): Remove this exclusion when v2 generators are tested
		if strings.Contains(line, "WriteV2") {
			continue
		}

		if strings.Contains(line, "total:") {
			continue
		}

		linesAndCoverage = append(linesAndCoverage, lineAndCoverage{line, percent})
	}

	slices.SortStableFunc(linesAndCoverage, func(a, b lineAndCoverage) int {
		if a.coverage < b.coverage {
			return -1
		}

		if a.coverage > b.coverage {
			return 1
		}

		return 0
	})
	lc := linesAndCoverage[0]

	sortedLines := make([]string, len(linesAndCoverage))
	for i := range linesAndCoverage {
		sortedLines[i] = linesAndCoverage[i].line
	}

	fmt.Println(strings.Join(sortedLines, "\n"))

	coverage := 80.0
	if lc.coverage < coverage {
		return fmt.Errorf("function coverage was less than the limit of %.1f:\n  %s", coverage, lc.line)
	}

	return nil
}

// Run all checks on the code for determining whether any fail.
func CheckForFail(c context.Context) error {
	fmt.Println("Checking...")

	// Checks from fastest to slowest
	mg.SerialCtxDeps(c,
		ReorderDeclsCheck,
		LintForFail,
		Deadcode,
		TestForFail,
		CheckNilsForFail,
		CheckCoverage,
	)

	// for _, cmd := range []func(context.Context) error{LintForFail, TestForFail} {
	// 	err := cmd(c)
	// 	if err != nil {
	// 		return fmt.Errorf("unable to finish checking: %w", err)
	// 	}
	// }

	return nil
}

// Check for nils and fix what you can
func CheckNils(c context.Context) error {
	fmt.Println("Running check for nils...")
	return run(c, "nilaway", "-fix", "./...")
}

// Check for nils, just for failure
func CheckNilsForFail(c context.Context) error {
	fmt.Println("Running check for nils...")
	return run(c, "nilaway", "./...")
}

// Clean up the dev env.
func Clean() {
	fmt.Println("Cleaning...")
	os.Remove("coverage.out")
}

// Deadcode checks that there's no dead code in codebase.
func Deadcode(c context.Context) error {
	fmt.Println("Checking for dead code...")

	out, err := output(c, "deadcode", "-test", "./...")
	if err != nil {
		return err
	}

	// Filter out functions that are used by magefiles (separate build context)
	// These appear as dead code to the deadcode tool but are actually used
	excludePatterns := []string{
		"impgen/reorder/reorder.go:.*: unreachable func: AnalyzeSectionOrder",
		"impgen/reorder/reorder.go:.*: unreachable func: identifySection",
		// Quicktemplate generates both Write* and string-returning functions
		// We use the Write* versions, so the string-returning ones appear dead
		"impgen/run/.*\\.qtpl:.*: unreachable func:",
	}

	lines := strings.Split(out, "\n")
	filteredLines := []string{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		excluded := false
		for _, pattern := range excludePatterns {
			matched, _ := regexp.MatchString(pattern, line)
			if matched {
				excluded = true
				break
			}
		}
		if !excluded {
			filteredLines = append(filteredLines, line)
		}
	}

	if len(filteredLines) > 0 {
		fmt.Println(strings.Join(filteredLines, "\n"))
		return errors.New("found dead code")
	}

	return nil
}

// FindRedundantTests identifies unit tests that don't provide unique coverage beyond golden+UAT tests.
// This is a convenience wrapper for this repository's specific configuration.
func FindRedundantTests(c context.Context) error {
	config := RedundancyConfig{
		BaselineTests: []BaselineTestSpec{
			{Package: "./impgen/run", TestPattern: "TestUATConsistency"},
			{Package: "./UAT/...", TestPattern: ""},
		},
		CoverageThreshold: 80.0,
		PackageToAnalyze:  "./...",
	}
	return FindRedundantTestsWithConfig(c, config)
}

// FindRedundantTestsWithConfig identifies unit tests that don't provide unique coverage beyond baseline tests.
// This generic version can be used in any repository by providing appropriate configuration.
func FindRedundantTestsWithConfig(c context.Context, config RedundancyConfig) error {
	fmt.Println("Finding redundant tests...")
	fmt.Println()

	// Step 1-N: Run baseline tests and merge coverage
	var coverageFiles []string
	baselineTestNames := make(map[string]bool) // Track baseline test names to exclude later

	for i, spec := range config.BaselineTests {
		stepNum := i + 1
		coverageFile := fmt.Sprintf("baseline_%d.out", stepNum)
		coverageFiles = append(coverageFiles, coverageFile)

		if spec.TestPattern != "" {
			fmt.Printf("Step %d: Running baseline test %s in %s...\n", stepNum, spec.TestPattern, spec.Package)
			err := runQuiet(c, "go", "test", "-coverprofile="+coverageFile, "-coverpkg=./...",
				"-run", spec.TestPattern, spec.Package)
			if err != nil {
				return fmt.Errorf("baseline test %s failed: %w", spec.TestPattern, err)
			}
			baselineTestNames[spec.TestPattern] = true
		} else {
			fmt.Printf("Step %d: Running all tests in %s...\n", stepNum, spec.Package)
			err := runQuiet(c, "go", "test", "-coverprofile="+coverageFile, "-coverpkg=./...", spec.Package)
			if err != nil {
				return fmt.Errorf("baseline tests in %s failed: %w", spec.Package, err)
			}

			// List and record all test functions in this package
			pkgTests, err := listTestFunctions(c, spec.Package)
			if err != nil {
				// Non-fatal: we'll just skip tracking these test names
				fmt.Printf("  Warning: couldn't list tests in %s: %v\n", spec.Package, err)
			} else {
				for _, testName := range pkgTests {
					baselineTestNames[testName] = true
				}
			}
		}
	}

	// Merge all baseline coverage files
	mergeStep := len(config.BaselineTests) + 1
	fmt.Printf("Step %d: Merging baseline coverage files...\n", mergeStep)
	baselineFile := "baseline.out"
	if len(coverageFiles) == 1 {
		// Filter .qtpl files even from single coverage file
		err := filterQtplFromCoverage(coverageFiles[0], baselineFile)
		if err != nil {
			return fmt.Errorf("failed to filter coverage: %w", err)
		}
		os.Remove(coverageFiles[0])
	} else {
		err := mergeMultipleCoverageFiles(coverageFiles, baselineFile)
		if err != nil {
			return fmt.Errorf("failed to merge coverage: %w", err)
		}
		// Clean up individual coverage files
		for _, f := range coverageFiles {
			os.Remove(f)
		}
	}

	// Get baseline function coverage
	analysisStep := mergeStep + 1
	fmt.Printf("Step %d: Analyzing baseline coverage...\n", analysisStep)
	baselineFuncs, err := getFunctionsAboveThreshold(baselineFile, config.CoverageThreshold)
	if err != nil {
		return fmt.Errorf("failed to get baseline coverage: %w", err)
	}
	fmt.Printf("  Baseline covers %d functions at %.0f%%+\n", len(baselineFuncs), config.CoverageThreshold)

	// Show baseline functions in verbose mode
	if mg.Verbose() {
		fmt.Println("\n  Baseline functions:")
		baselineFuncList := make([]string, 0, len(baselineFuncs))
		for fn := range baselineFuncs {
			baselineFuncList = append(baselineFuncList, fn)
		}
		sort.Strings(baselineFuncList)
		for _, fn := range baselineFuncList {
			fmt.Printf("    - %s\n", fn)
		}
		fmt.Println()
	}

	// List all test functions, excluding baseline tests
	listStep := analysisStep + 1
	fmt.Printf("Step %d: Listing unit tests...\n", listStep)
	allTestFuncs, err := listTestFunctions(c, config.PackageToAnalyze)
	if err != nil {
		return fmt.Errorf("failed to list tests: %w", err)
	}

	// Filter out baseline tests
	var testFuncs []string
	for _, testName := range allTestFuncs {
		if !baselineTestNames[testName] {
			testFuncs = append(testFuncs, testName)
		}
	}
	fmt.Printf("  Found %d unit tests (%d total, %d baseline excluded)\n\n",
		len(testFuncs), len(allTestFuncs), len(allTestFuncs)-len(testFuncs))

	// Get coverage with ALL unit tests (baseline + unit tests)
	allWithUnitStep := listStep + 1
	fmt.Printf("Step %d: Getting coverage with all unit tests...\n", allWithUnitStep)
	allTestsOut := "all_with_unit_tests.out"
	allTestsOutRaw := "all_with_unit_tests_raw.out"
	err = runQuiet(c, "go", "test", "-coverprofile="+allTestsOutRaw, "-coverpkg=./...", config.PackageToAnalyze)
	if err != nil {
		return fmt.Errorf("failed to run all unit tests: %w", err)
	}

	// Filter .qtpl files from coverage
	err = filterQtplFromCoverage(allTestsOutRaw, allTestsOut)
	if err != nil {
		return fmt.Errorf("failed to filter coverage: %w", err)
	}
	os.Remove(allTestsOutRaw)

	allWithUnitFuncs, err := getFunctionsAboveThreshold(allTestsOut, config.CoverageThreshold)
	if err != nil {
		return fmt.Errorf("failed to get all-tests coverage: %w", err)
	}
	fmt.Printf("  All tests (baseline + unit) cover %d functions at %.0f%%+\n",
		len(allWithUnitFuncs), config.CoverageThreshold)

	// Check each test for unique coverage by excluding it
	checkStep := allWithUnitStep + 1
	fmt.Printf("Step %d: Analyzing each test by exclusion...\n", checkStep)

	type testResult struct {
		name         string
		uniqueCount  int
		uniqueFuncs  []string
		coveredFuncs []string
	}

	var redundantTests []testResult
	var uniqueTests []testResult

	for _, testName := range testFuncs {
		fmt.Printf("  Checking %s... ", testName)

		// Build regex that matches all tests except this one
		var otherTests []string
		for _, t := range testFuncs {
			if t != testName {
				otherTests = append(otherTests, t)
			}
		}

		// Run all tests EXCEPT this one
		testOut := fmt.Sprintf("test_without_%s.out", testName)
		var testErr error
		if len(otherTests) == 0 {
			// If this is the only test, compare against baseline
			testErr = runQuiet(c, "go", "test", "-coverprofile="+testOut, "-coverpkg=./...",
				"-run", "^$", config.PackageToAnalyze) // Match nothing
		} else {
			// Build pattern like "^(TestA|TestB|TestC)$"
			pattern := "^(" + strings.Join(otherTests, "|") + ")$"
			testErr = runQuiet(c, "go", "test", "-coverprofile="+testOut, "-coverpkg=./...",
				"-run", pattern, config.PackageToAnalyze)
		}

		if testErr != nil {
			fmt.Printf("FAILED (test error)\n")
			continue
		}

		// Filter .qtpl files from coverage
		testOutFiltered := testOut + ".filtered"
		err := filterQtplFromCoverage(testOut, testOutFiltered)
		if err != nil {
			fmt.Printf("FAILED (filter error)\n")
			os.Remove(testOut)
			continue
		}
		os.Remove(testOut)
		os.Rename(testOutFiltered, testOut)

		// Get functions covered without this test
		withoutTestFuncs, err := getFunctionsAboveThreshold(testOut, config.CoverageThreshold)
		if err != nil {
			fmt.Printf("FAILED (coverage error)\n")
			continue
		}

		// Find functions that dropped below threshold when we excluded this test
		var uniqueFuncs []string
		for fn := range allWithUnitFuncs {
			if !withoutTestFuncs[fn] {
				// This function was at 80%+ with all tests, but below 80% without this test
				uniqueFuncs = append(uniqueFuncs, fn)
			}
		}
		sort.Strings(uniqueFuncs)

		result := testResult{
			name:        testName,
			uniqueCount: len(uniqueFuncs),
			uniqueFuncs: uniqueFuncs,
		}

		if len(uniqueFuncs) > 0 {
			fmt.Printf("KEEP (%d unique)\n", len(uniqueFuncs))
			uniqueTests = append(uniqueTests, result)
		} else {
			fmt.Printf("REDUNDANT (0 unique)\n")
			redundantTests = append(redundantTests, result)
		}

		// Show function details in verbose mode
		if mg.Verbose() && len(uniqueFuncs) > 0 {
			fmt.Printf("    Functions that would drop below %.0f%% if removed:\n", config.CoverageThreshold)
			for _, fn := range uniqueFuncs {
				fmt.Printf("      - %s\n", fn)
			}
		}

		// Clean up test coverage file
		os.Remove(testOut)
	}

	// Report results
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("RESULTS")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Printf("\nTests required for coverage (%d):\n", len(uniqueTests))
	for _, test := range uniqueTests {
		fmt.Printf("  ✓ %s (%d function", test.name, test.uniqueCount)
		if test.uniqueCount != 1 {
			fmt.Printf("s")
		}
		fmt.Printf(" would drop below %.0f%%)\n", config.CoverageThreshold)

		// Show affected functions in verbose mode
		if mg.Verbose() && len(test.uniqueFuncs) > 0 {
			for _, fn := range test.uniqueFuncs {
				fmt.Printf("      - %s\n", fn)
			}
		}
	}
	fmt.Printf("\nRedundant tests (%d):\n", len(redundantTests))

	// Group redundant tests by file
	testsByFile := make(map[string][]string)
	for _, test := range redundantTests {
		file, err := findTestFile(c, config.PackageToAnalyze, test.name)
		if err != nil {
			// If we can't find the file, use "unknown"
			file = "unknown"
		}
		testsByFile[file] = append(testsByFile[file], test.name)
	}

	// Sort files for consistent output
	var files []string
	for file := range testsByFile {
		files = append(files, file)
	}
	sort.Strings(files)

	// Display by file
	for _, file := range files {
		tests := testsByFile[file]
		fmt.Printf("\n  %s:\n", file)
		for _, testName := range tests {
			fmt.Printf("    ✗ %s\n", testName)
		}
	}
	fmt.Println()

	return nil
}

// Run the fuzz tests.
func Fuzz(c context.Context) error {
	fmt.Println("Running fuzz tests...")
	return run(c, "./dev/fuzz.fish")
}

// Generate runs go generate on all packages using the locally-built impgen binary.
func Generate(c context.Context) error {
	fmt.Println("Generating...")

	// Build local impgen first
	mg.SerialCtxDeps(c, Build)

	// Get current PATH and prepend our bin directory
	currentPath := os.Getenv("PATH")
	binDir, err := filepath.Abs("bin")
	if err != nil {
		return fmt.Errorf("failed to get absolute path for bin: %w", err)
	}
	newPath := binDir + string(filepath.ListSeparator) + currentPath

	// Run go generate with modified PATH
	cmd := exec.CommandContext(c, "go", "generate", "./...")
	cmd.Env = append(os.Environ(), "PATH="+newPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Install development tooling.
func InstallTools(c context.Context) error {
	fmt.Println("Installing development tools...")
	return run(c, "./dev/dev-install.sh")
}

// Lint lints the codebase.
func Lint(c context.Context) error {
	fmt.Println("Linting...")
	// _, err := sh.Exec(nil, os.Stdout, nil, "golangci-lint", "run", "-c", "dev/golangci.toml")
	// return err
	return run(c, "golangci-lint", "run", "-c", "dev/golangci.toml")
}

// LintForFail lints the codebase purely to find out whether anything fails.
func LintForFail(c context.Context) error {
	fmt.Println("Linting to check for overall pass/fail...")
	// _, err := sh.Exec(
	// 	nil, os.Stdout, nil,
	// 	"golangci-lint", "run",
	// 	"-c", "dev/golangci.toml",
	// 	"--fix=false",
	// 	"--max-issues-per-linter=1",
	// 	"--max-same-issues=1",
	// )
	// return err
	return run(
		c,
		"golangci-lint", "run",
		"-c", "dev/golangci.toml",
		"--fix=false",
		"--max-issues-per-linter=1",
		"--max-same-issues=1",
		"--allow-parallel-runners",
	)
}

// Modernize the codebase.
func Modernize(c context.Context) error {
	fmt.Println("Modernizing codebase...")
	return run(c, "go", "run", "golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest",
		"-fix", "./...")
}

// Run the mutation tests.
func Mutate(c context.Context) error {
	fmt.Println("Running mutation tests...")

	for _, cmd := range []func(context.Context) error{
		TestForFail,
		func(c context.Context) error {
			return run(
				c,
				"go",
				"test",
				// "-v",
				"-timeout=6000s",
				"-tags=mutation",
				"-ooze.v",
				"./...",
				"-run=TestMutation",
			)
		},
	} {
		err := cmd(c)
		if err != nil {
			return fmt.Errorf("unable to finish checking: %w", err)
		}
	}

	return nil
}

// ReorderDecls reorders declarations in Go files per CLAUDE.md conventions.
// Run manually with 'mage ReorderDecls' when needed.
func ReorderDecls(c context.Context) error {
	fmt.Println("Reordering declarations...")

	files, err := globs(".", []string{".go"})
	if err != nil {
		return fmt.Errorf("failed to find Go files: %w", err)
	}

	reorderedCount := 0
	for _, file := range files {
		// Skip generated files by name pattern
		if strings.Contains(file, "generated_") {
			continue
		}
		// Skip vendor
		if strings.HasPrefix(file, "vendor/") {
			continue
		}
		// Skip hidden directories
		if strings.Contains(file, "/.") {
			continue
		}

		// Skip files with generated markers (e.g., .qtpl.go files)
		isGenerated, err := isGeneratedFile(file)
		if err != nil {
			return fmt.Errorf("failed to check if %s is generated: %w", file, err)
		}
		if isGenerated {
			continue
		}

		// Read file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Reorder
		reordered, err := reorder.Source(string(content))
		if err != nil {
			fmt.Printf("Warning: failed to reorder %s: %v\n", file, err)
			continue
		}

		// Write back if changed
		if string(content) != reordered {
			err = os.WriteFile(file, []byte(reordered), 0o600)
			if err != nil {
				return fmt.Errorf("failed to write %s: %w", file, err)
			}
			fmt.Printf("  Reordered: %s\n", file)
			reorderedCount++
		}
	}

	fmt.Printf("Reordered %d file(s).\n", reorderedCount)
	return nil
}

// ReorderDeclsCheck checks which files need reordering without modifying them.
// Run with 'mage ReorderDeclsCheck' to list out-of-order files.
// Run with 'mage -v ReorderDeclsCheck' to also see the diffs.
func ReorderDeclsCheck(c context.Context) error {
	fmt.Println("Checking declaration order...")

	files, err := globs(".", []string{".go"})
	if err != nil {
		return fmt.Errorf("failed to find Go files: %w", err)
	}

	outOfOrderFiles := 0
	filesProcessed := 0
	for _, file := range files {
		// Skip generated files by name pattern
		if strings.Contains(file, "generated_") {
			continue
		}
		// Skip vendor
		if strings.HasPrefix(file, "vendor/") {
			continue
		}
		// Skip hidden directories
		if strings.Contains(file, "/.") {
			continue
		}

		// Skip files with generated markers (e.g., .qtpl.go files)
		isGenerated, err := isGeneratedFile(file)
		if err != nil {
			return fmt.Errorf("failed to check if %s is generated: %w", file, err)
		}
		if isGenerated {
			continue
		}

		// Read file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Analyze section order
		sectionOrder, err := reorder.AnalyzeSectionOrder(string(content))
		if err != nil {
			fmt.Printf("Warning: failed to analyze %s: %v\n", file, err)
			continue
		}

		filesProcessed++

		// In verbose mode, show all files and their sections
		if mg.Verbose() {
			fmt.Printf("\nProcessing %s:\n", file)
			if len(sectionOrder.Sections) > 0 {
				fmt.Println("  Sections found:")
				for i, section := range sectionOrder.Sections {
					fmt.Printf("    %d. %s\n", i+1, section.Name)
				}
			} else {
				fmt.Println("  No sections found")
			}
		}

		// Get reordered version
		reordered, err := reorder.Source(string(content))
		if err != nil {
			fmt.Printf("Warning: failed to reorder %s: %v\n", file, err)
			continue
		}

		// Check if reordering would change the file
		if string(content) != reordered {
			outOfOrderFiles++
			// Only print filename if not already printed in verbose mode
			if !mg.Verbose() {
				fmt.Printf("\n%s:\n", file)
			}

			// Print section analysis
			fmt.Println("  Current order:")
			for i, section := range sectionOrder.Sections {
				posStr := fmt.Sprintf("%d", i+1)
				expectedNote := ""
				if section.Expected != i+1 {
					expectedNote = fmt.Sprintf(" <- should be #%d", section.Expected)
				}
				fmt.Printf("    %s. %-24s%s\n", posStr, section.Name, expectedNote)
			}

			// Identify sections that are out of place
			outOfPlace := []string{}
			for i, section := range sectionOrder.Sections {
				if section.Expected != i+1 {
					outOfPlace = append(outOfPlace, fmt.Sprintf("%s (at #%d, should be #%d)",
						section.Name, i+1, section.Expected))
				}
			}

			if len(outOfPlace) > 0 {
				fmt.Printf("  Sections out of place: %s\n", strings.Join(outOfPlace, ", "))
			}

			// If verbose, show diff
			if mg.Verbose() {
				diff := textdiff.Unified(file+" (current)", file+" (reordered)", string(content), reordered)
				if diff != "" {
					fmt.Printf("\n%s\n", diff)
				}
			}
		}
	}

	if outOfOrderFiles > 0 {
		fmt.Printf("\n%d file(s) need reordering (out of %d processed). Run 'mage ReorderDecls' to fix.\n", outOfOrderFiles, filesProcessed)
		return fmt.Errorf("%d file(s) need reordering", outOfOrderFiles)
	}

	fmt.Printf("All files are correctly ordered (%d files processed).\n", filesProcessed)
	return nil
}

// Run the unit tests.
func Test(c context.Context) error {
	fmt.Println("Running unit tests...")

	mg.SerialCtxDeps(c, Generate)

	err := run(
		c,
		"go",
		"test",
		"-timeout=2m",
		"-race",
		"-coverprofile=coverage.out",
		"-coverpkg=./...",
		// "-coverpkg=./impgen/...,./imptest/...",
		"-cover",
		"./...",
	)
	if err != nil {
		return err
	}

	// Strip main.go and .qtpl coverage lines from coverage.out
	data, err := os.ReadFile("coverage.out")
	if err != nil {
		return fmt.Errorf("failed to read coverage.out: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, "/main.go:") && !strings.Contains(line, ".qtpl:") {
			filtered = append(filtered, line)
		}
	}

	err = os.WriteFile("coverage.out", []byte(strings.Join(filtered, "\n")), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write coverage.out: %w", err)
	}

	return nil
}

// Run the unit tests purely to find out whether any fail.
func TestForFail(c context.Context) error {
	fmt.Println("Running unit tests for overall pass/fail...")

	mg.SerialCtxDeps(c, Generate)

	return run(
		c,
		"go",
		"test",
		"-timeout=10s",
		"./...",
		// "-rapid.nofailfile",
		"-failfast",
	)
}

// Tidy tidies up go.mod.
func Tidy(c context.Context) error {
	fmt.Println("Tidying go.mod...")
	return run(c, "go", "mod", "tidy")
	// return sh.RunWithV(map[string]string{"GOPRIVATE": "github.com/toejough/protest"}, "go", "mod", "tidy")
}

// TodoCheck checks for TODO and FIXME comments using golangci-lint.
func TodoCheck(c context.Context) error {
	fmt.Println("Linting...")
	// _, err := sh.Exec(nil, os.Stdout, nil, "golangci-lint", "run", "-c", "dev/golangci-todos.toml")
	// return err
	return run(c, "golangci-lint", "run", "-c", "dev/golangci-todos.toml")
}

// Watch, and re-run Check whenever the files change.
func Watch() error {
	fmt.Println("Watching...")

	// look for files that might change in the current directory
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to monitor effectively due to error getting current working directory: %w", err)
	}

	// Track file hashes to detect actual content changes
	fileHashes, err := calculateFileHashes(dir, []string{".go", ".fish", ".toml"})
	if err != nil {
		return fmt.Errorf("unable to calculate initial file hashes: %w", err)
	}

	// Track the current cancel function
	var currentCancel context.CancelFunc

	defer func() {
		if currentCancel != nil {
			currentCancel()
		}
	}()

	// cancellation context so we can cancel the check
	ctx, cancel := context.WithCancel(context.Background())
	currentCancel = cancel

	// func to run the check
	checkFunc := func(c context.Context) {
		err := Check(c)
		if err != nil {
			fmt.Printf("continuing to watch after check failure: \n  %s\n", err)
		} else {
			fmt.Println("continuing to watch after all checks passed!")
		}
	}

	// run the check
	go checkFunc(ctx)

	for {
		// don't run more than 1x/sec
		time.Sleep(time.Second)

		// check for changes by comparing file hashes
		newHashes, err := calculateFileHashes(dir, []string{".go", ".fish", ".toml"})
		if err != nil {
			return fmt.Errorf("unable to calculate file hashes: %w", err)
		}

		changeDetected := hasFileHashesChanged(fileHashes, newHashes)

		// cancel & re-run if we got a change
		if changeDetected {
			fmt.Println("Change detected...")

			// Update our baseline hashes
			fileHashes = newHashes

			// Cancel the old context
			if currentCancel != nil {
				currentCancel()
			}

			// Create new context
			ctx, cancel := context.WithCancel(context.Background())
			currentCancel = cancel

			go checkFunc(ctx)
		}
	}
}

type coverageBlock struct {
	file       string
	startLine  int
	startCol   int
	endLine    int
	endCol     int
	statements int
	count      int
}

// Types

type lineAndCoverage struct {
	line     string
	coverage float64
}

// position represents a line:column position in a file.
type position struct {
	line int
	col  int
}

// compare returns -1 if p < other, 0 if p == other, 1 if p > other.
func (p position) compare(other position) int {
	if p.line < other.line {
		return -1
	}
	if p.line > other.line {
		return 1
	}
	if p.col < other.col {
		return -1
	}
	if p.col > other.col {
		return 1
	}
	return 0
}

// segment represents a coverage segment with summed counts from all overlapping blocks.
type segment struct {
	start position
	end   position
	count int
}

// calculateFileHashes computes SHA256 hashes for all files matching the given extensions.
// Returns a map of relative file path to hash string.
func calculateFileHashes(dir string, extensions []string) (map[string]string, error) {
	hashes := make(map[string]string)

	files, err := globs(dir, extensions)
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	for _, filePath := range files {
		hash, err := hashFile(filePath)
		if err != nil {
			// If we can't read a file, skip it (might be deleted or temporary)
			continue
		}
		hashes[filePath] = hash
	}

	return hashes, nil
}

// Private Functions

// filterQtplFromCoverage removes .qtpl template file entries from a coverage file.
func filterQtplFromCoverage(inputFile, outputFile string) error {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", inputFile, err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("empty coverage file: %s", inputFile)
	}

	// Keep mode line, filter out .qtpl entries
	filtered := []string{lines[0]} // mode line
	for _, line := range lines[1:] {
		if line == "" || strings.Contains(line, ".qtpl:") {
			continue
		}
		filtered = append(filtered, line)
	}

	result := strings.Join(filtered, "\n")
	err = os.WriteFile(outputFile, []byte(result), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", outputFile, err)
	}

	return nil
}

// findTestFile finds which file contains the given test function in the package.
func findTestFile(c context.Context, pkg, testName string) (string, error) {
	var searchDir string

	if pkg == "./..." {
		// When analyzing all packages, search from current directory
		searchDir = "."
	} else {
		// Get specific package directory
		out, err := output(c, "go", "list", "-f", "{{.Dir}}", pkg)
		if err != nil {
			return "", fmt.Errorf("failed to get package dir: %w", err)
		}
		searchDir = strings.TrimSpace(out)
	}

	// Search for the test function in _test.go files
	pattern := fmt.Sprintf("^func %s(", testName)
	cmd := exec.CommandContext(c, "grep", "-l", "-r", "--include=*_test.go", pattern, searchDir)
	outBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("test not found")
	}

	// Get the first matching file
	files := strings.Split(strings.TrimSpace(string(outBytes)), "\n")
	if len(files) == 0 {
		return "", fmt.Errorf("test not found")
	}

	// Return cleaned relative path (grep returns paths like ./path/to/file.go)
	path := strings.TrimPrefix(files[0], "./")
	return filepath.Clean(path), nil
}

// getFunctionsAboveThreshold returns a set of functions that have coverage >= threshold.
func getFunctionsAboveThreshold(coverageFile string, threshold float64) (map[string]bool, error) {
	out, err := exec.Command("go", "tool", "cover", "-func="+coverageFile).Output()
	if err != nil {
		return nil, fmt.Errorf("go tool cover failed: %w", err)
	}

	funcs := make(map[string]bool)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "total:") {
			continue
		}

		// Format: file:line:  functionName  percentage%
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Last field is percentage like "85.7%"
		percentStr := fields[len(fields)-1]
		percentStr = strings.TrimSuffix(percentStr, "%")
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			continue
		}

		// Function name with location (e.g., "file.go:123: funcName")
		funcName := strings.Join(fields[0:len(fields)-1], " ")

		if percent >= threshold {
			funcs[funcName] = true
		}
	}

	return funcs, nil
}

// better glob expansion
// https://stackoverflow.com/a/26809999
func globs(dir string, ext []string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("unable to find all glob matches: %w", err)
		}

		for _, each := range ext {
			if filepath.Ext(path) == each {
				files = append(files, path)
				return nil
			}
		}

		return nil
	})

	return files, err
}

// hasFileHashesChanged compares two hash maps and returns true if they differ.
// This includes detecting new files, deleted files, or modified files.
func hasFileHashesChanged(oldHashes, newHashes map[string]string) bool {
	// Check if the number of files changed
	if len(oldHashes) != len(newHashes) {
		return true
	}

	// Check if any file's hash changed
	for path, newHash := range newHashes {
		oldHash, exists := oldHashes[path]
		if !exists || oldHash != newHash {
			return true
		}
	}

	// Check if any files were deleted
	for path := range oldHashes {
		if _, exists := newHashes[path]; !exists {
			return true
		}
	}

	return false
}

// hashFile computes the SHA256 hash of a file's contents.
func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// isGeneratedFile checks if a file is generated by looking for the standard marker.
func isGeneratedFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	// Read first 200 bytes to check for generated marker
	buf := make([]byte, 200)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read %s: %w", path, err)
	}

	content := string(buf[:n])
	// Standard generated file marker per https://golang.org/s/generatedcode
	return strings.Contains(content, "Code generated") || strings.Contains(content, "DO NOT EDIT"), nil
}

// listTestFunctions lists all test functions in the given package.
func listTestFunctions(c context.Context, pkg string) ([]string, error) {
	out, err := output(c, "go", "test", "-list", ".", pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to list tests: %w", err)
	}

	var tests []string
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") {
			tests = append(tests, line)
		}
	}

	return tests, nil
}

// mergeCoverageBlocks merges coverage blocks, splitting overlapping blocks into segments.
// When Go's coverage tool instruments code, it creates overlapping blocks that can
// cause incorrect coverage percentages. This function:
// 1. Sums counts for identical blocks from different test packages
// 2. Splits overlapping (but non-identical) blocks into non-overlapping segments
// 3. Sums execution counts for each segment from all blocks that cover it
func mergeCoverageBlocks() error {
	// TODO: take the coverage file name as an arg
	data, err := os.ReadFile("coverage.out")
	if err != nil {
		return fmt.Errorf("failed to read coverage.out: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return nil
	}

	// Keep the mode line
	mode := lines[0]

	// Parse all blocks
	var blocks []coverageBlock
	blockCounts := make(map[string]int) // Sum counts for identical blocks

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}

		blockID := parts[0]
		numStmts, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		file, startLine, startCol, endLine, endCol, err := parseBlockID(blockID)
		if err != nil {
			continue
		}

		// Sum counts for identical blocks
		blockCounts[blockID] += count

		// Store block for deduplication
		found := false
		for i, b := range blocks {
			if b.file == file && b.startLine == startLine && b.startCol == startCol &&
				b.endLine == endLine && b.endCol == endCol {
				blocks[i].count = blockCounts[blockID]
				found = true
				break
			}
		}

		if !found {
			blocks = append(blocks, coverageBlock{
				file:       file,
				startLine:  startLine,
				startCol:   startCol,
				endLine:    endLine,
				endCol:     endCol,
				statements: numStmts,
				count:      blockCounts[blockID],
			})
		}
	}

	// For each file, split overlapping blocks into non-overlapping segments
	fileBlocks := make(map[string][]coverageBlock)
	for _, block := range blocks {
		fileBlocks[block.file] = append(fileBlocks[block.file], block)
	}

	var finalBlocks []coverageBlock
	for _, blocks := range fileBlocks {
		// Split overlapping blocks into segments and sum their counts
		segments := splitBlocksIntoSegments(blocks)
		finalBlocks = append(finalBlocks, segments...)
	}

	// Rebuild coverage file
	var merged []string
	merged = append(merged, mode)

	// Sort for deterministic output
	sort.Slice(finalBlocks, func(i, j int) bool {
		if finalBlocks[i].file != finalBlocks[j].file {
			return finalBlocks[i].file < finalBlocks[j].file
		}
		if finalBlocks[i].startLine != finalBlocks[j].startLine {
			return finalBlocks[i].startLine < finalBlocks[j].startLine
		}
		return finalBlocks[i].startCol < finalBlocks[j].startCol
	})

	for _, block := range finalBlocks {
		blockID := fmt.Sprintf("%s:%d.%d,%d.%d",
			block.file, block.startLine, block.startCol, block.endLine, block.endCol)
		merged = append(merged, fmt.Sprintf("%s %d %d", blockID, block.statements, block.count))
	}

	// Write merged coverage
	err = os.WriteFile("coverage.out", []byte(strings.Join(merged, "\n")+"\n"), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write merged coverage: %w", err)
	}

	return nil
}

// mergeCoverageBlocksFile merges coverage blocks in the specified file (in-place).
func mergeCoverageBlocksFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return nil
	}

	// Keep the mode line
	mode := lines[0]

	// Parse all blocks
	var blocks []coverageBlock
	blockCounts := make(map[string]int)

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}

		blockID := parts[0]
		numStmts, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		file, startLine, startCol, endLine, endCol, err := parseBlockID(blockID)
		if err != nil {
			continue
		}

		// Sum counts for identical blocks
		blockCounts[blockID] += count

		// Store block for deduplication
		found := false
		for i, b := range blocks {
			if b.file == file && b.startLine == startLine && b.startCol == startCol &&
				b.endLine == endLine && b.endCol == endCol {
				blocks[i].count = blockCounts[blockID]
				found = true
				break
			}
		}

		if !found {
			blocks = append(blocks, coverageBlock{
				file:       file,
				startLine:  startLine,
				startCol:   startCol,
				endLine:    endLine,
				endCol:     endCol,
				statements: numStmts,
				count:      blockCounts[blockID],
			})
		}
	}

	// For each file, split overlapping blocks into non-overlapping segments
	fileBlocks := make(map[string][]coverageBlock)
	for _, block := range blocks {
		fileBlocks[block.file] = append(fileBlocks[block.file], block)
	}

	var finalBlocks []coverageBlock
	for _, blocks := range fileBlocks {
		segments := splitBlocksIntoSegments(blocks)
		finalBlocks = append(finalBlocks, segments...)
	}

	// Rebuild coverage file
	var merged []string
	merged = append(merged, mode)

	// Sort for deterministic output
	sort.Slice(finalBlocks, func(i, j int) bool {
		if finalBlocks[i].file != finalBlocks[j].file {
			return finalBlocks[i].file < finalBlocks[j].file
		}
		if finalBlocks[i].startLine != finalBlocks[j].startLine {
			return finalBlocks[i].startLine < finalBlocks[j].startLine
		}
		return finalBlocks[i].startCol < finalBlocks[j].startCol
	})

	for _, block := range finalBlocks {
		blockID := fmt.Sprintf("%s:%d.%d,%d.%d",
			block.file, block.startLine, block.startCol, block.endLine, block.endCol)
		merged = append(merged, fmt.Sprintf("%s %d %d", blockID, block.statements, block.count))
	}

	// Write merged coverage
	return os.WriteFile(filename, []byte(strings.Join(merged, "\n")+"\n"), 0o600)
}

// mergeMultipleCoverageFiles merges multiple coverage files into a single output file.
func mergeMultipleCoverageFiles(files []string, output string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to merge")
	}

	var mode string
	var allBlocks []string

	for i, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		lines := strings.Split(string(data), "\n")
		if len(lines) == 0 {
			continue
		}

		// Use mode from first file
		if i == 0 {
			mode = lines[0]
		}

		// Append blocks from this file (skip mode line and .qtpl files)
		for _, line := range lines[1:] {
			// Skip empty lines and lines referencing .qtpl template files
			if line == "" || strings.Contains(line, ".qtpl:") {
				continue
			}
			allBlocks = append(allBlocks, line)
		}
	}

	// Write combined file
	combined := mode + "\n" + strings.Join(allBlocks, "\n")
	err := os.WriteFile(output, []byte(combined), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", output, err)
	}

	// Merge overlapping blocks using existing logic
	return mergeCoverageBlocksFile(output)
}

func output(c context.Context, command string, arg ...string) (string, error) {
	buf := &bytes.Buffer{}
	cmd := exec.CommandContext(c, command, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	return strings.TrimSuffix(buf.String(), "\n"), err
}

// parseBlockID parses a block ID like "file.go:10.5,20.10" into components.
func parseBlockID(blockID string) (file string, startLine, startCol, endLine, endCol int, err error) {
	// Format: file:startLine.startCol,endLine.endCol
	fileParts := strings.Split(blockID, ":")
	if len(fileParts) != 2 {
		return "", 0, 0, 0, 0, fmt.Errorf("invalid block ID format: %s", blockID)
	}
	file = fileParts[0]

	rangeParts := strings.Split(fileParts[1], ",")
	if len(rangeParts) != 2 {
		return "", 0, 0, 0, 0, fmt.Errorf("invalid range format: %s", blockID)
	}

	startParts := strings.Split(rangeParts[0], ".")
	if len(startParts) != 2 {
		return "", 0, 0, 0, 0, fmt.Errorf("invalid start position: %s", blockID)
	}

	endParts := strings.Split(rangeParts[1], ".")
	if len(endParts) != 2 {
		return "", 0, 0, 0, 0, fmt.Errorf("invalid end position: %s", blockID)
	}

	startLine, _ = strconv.Atoi(startParts[0])
	startCol, _ = strconv.Atoi(startParts[1])
	endLine, _ = strconv.Atoi(endParts[0])
	endCol, _ = strconv.Atoi(endParts[1])

	return file, startLine, startCol, endLine, endCol, nil
}

func run(c context.Context, command string, arg ...string) error {
	cmd := exec.CommandContext(c, command, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// runQuiet runs a command, suppressing stdout unless mg.Verbose() is true.
// Stderr is always shown to display errors.
func runQuiet(c context.Context, command string, arg ...string) error {
	cmd := exec.CommandContext(c, command, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Only show stdout if verbose mode is enabled
	if mg.Verbose() {
		cmd.Stdout = os.Stdout
	}

	return cmd.Run()
}

// splitBlocksIntoSegments splits overlapping blocks into non-overlapping segments,
// summing counts for each segment from all blocks that cover it.
func splitBlocksIntoSegments(blocks []coverageBlock) []coverageBlock {
	if len(blocks) == 0 {
		return nil
	}

	// Collect all unique boundary positions
	boundarySet := make(map[position]bool)
	for _, block := range blocks {
		boundarySet[position{block.startLine, block.startCol}] = true
		boundarySet[position{block.endLine, block.endCol}] = true
	}

	// Convert to sorted slice
	var boundaries []position
	for pos := range boundarySet {
		boundaries = append(boundaries, pos)
	}
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].compare(boundaries[j]) < 0
	})

	// Create segments between consecutive boundaries
	var segments []segment
	for i := 0; i < len(boundaries)-1; i++ {
		seg := segment{
			start: boundaries[i],
			end:   boundaries[i+1],
			count: 0,
		}

		// Sum counts from all blocks that cover this segment
		for _, block := range blocks {
			blockStart := position{block.startLine, block.startCol}
			blockEnd := position{block.endLine, block.endCol}

			// Check if block covers this segment
			// Segment is covered if: blockStart <= segStart AND segEnd <= blockEnd
			if blockStart.compare(seg.start) <= 0 && seg.end.compare(blockEnd) <= 0 {
				seg.count += block.count
			}
		}

		// Only keep segments with non-zero count
		if seg.count > 0 {
			segments = append(segments, seg)
		}
	}

	// Convert segments back to coverageBlocks
	// We need to estimate the number of statements in each segment
	var result []coverageBlock
	for _, seg := range segments {
		// For simplicity, use 1 statement per segment
		// The actual number doesn't affect coverage percentage calculations
		result = append(result, coverageBlock{
			file:       blocks[0].file, // All blocks in input have same file
			startLine:  seg.start.line,
			startCol:   seg.start.col,
			endLine:    seg.end.line,
			endCol:     seg.end.col,
			statements: 1,
			count:      seg.count,
		})
	}

	return result
}
