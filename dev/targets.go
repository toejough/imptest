//go:build targ

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/akedrou/textdiff"
	"github.com/toejough/go-reorder"
	"github.com/toejough/targ"
	"github.com/toejough/targ/file"
	"github.com/toejough/targ/sh"
)

// Types

// BaselineTestSpec specifies a baseline test for redundancy analysis.
type BaselineTestSpec struct {
	Package     string // Package path (e.g., "./impgen/run" or "./UAT/...")
	TestPattern string // Test name pattern for -run flag (empty string runs all tests in package)
}

// RedundancyConfig configures the redundant test analysis.
type RedundancyConfig struct {
	BaselineTests     []BaselineTestSpec // Tests that form the baseline coverage
	CoverageThreshold float64            // Percentage threshold (e.g., 80.0 for 80%)
	PackageToAnalyze  string             // Package containing tests to analyze (e.g., "./impgen/run")
	CoveragePackages  string             // Packages to measure coverage for (e.g., "./impgen/...,./imptest/...")
}

// Build builds the local impgen binary.
func Build() error {
	fmt.Println("Building impgen...")

	if err := os.MkdirAll("bin", 0o755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	return sh.Run("go", "build", "-o", "bin/impgen", "./impgen")
}

// Check runs all checks & fixes on the code, in order of correctness.
func Check() error {
	fmt.Println("Checking...")

	return targ.Deps(
		Tidy,           // clean up the module dependencies
		DeleteDeadcode, // no use doing anything else to dead code
		FixImports,     // after dead code removal, fix imports to remove unused ones
		Modernize,      // no use doing anything else to old code patterns
		CheckCoverage,  // does our code work?
		CheckNils,      // is it nil free?
		ReorderDecls,   // linter will yell about declaration order if not correct
		Lint,
	)
}

// CheckCoverage checks that function coverage meets the minimum threshold.
func CheckCoverage() error {
	fmt.Println("Checking coverage...")

	if err := targ.Deps(Test); err != nil {
		return err
	}

	// Merge duplicate coverage blocks from cross-package testing
	if err := mergeCoverageBlocks("coverage.out"); err != nil {
		return fmt.Errorf("failed to merge coverage blocks: %w", err)
	}

	out, err := output("go", "tool", "cover", "-func=coverage.out")
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

// CheckForFail runs all checks on the code for determining whether any fail.
func CheckForFail() error {
	fmt.Println("Checking...")

	// Checks from fastest to slowest
	return targ.Deps(
		ReorderDeclsCheck,
		LintForFail,
		Deadcode,
		TestForFail,
		CheckNilsForFail,
		CheckCoverage,
	)
}

// CheckNils checks for nils and fixes what it can.
func CheckNils() error {
	fmt.Println("Running check for nils...")
	return sh.Run("nilaway", "-fix", "./...")
}

// CheckNilsForFail checks for nils, just for failure.
func CheckNilsForFail() error {
	fmt.Println("Running check for nils...")
	return sh.Run("nilaway", "./...")
}

// Clean cleans up the dev env.
func Clean() {
	fmt.Println("Cleaning...")
	os.Remove("coverage.out")
}

// Deadcode checks that there's no dead code in codebase.
func Deadcode() error {
	fmt.Println("Checking for dead code...")

	out, err := output("deadcode", "-test", "./...")
	if err != nil {
		return err
	}

	// Filter out functions that are used by targ files (separate build context)
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

// DeleteDeadcode removes unreachable functions from the codebase.
func DeleteDeadcode() error {
	fmt.Println("Deleting dead code...")

	out, err := output("deadcode", "-test", "./...")
	if err != nil {
		return err
	}

	// Parse deadcode output: "file.go:123: unreachable func: FuncName"
	// Group by file
	fileToFuncs := make(map[string][]deadFunc)

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse: "impgen/run/codegen_interface.go:42: unreachable func: callStructData"
		parts := strings.Split(line, ": unreachable func: ")
		if len(parts) != 2 {
			continue
		}

		fileParts := strings.Split(parts[0], ":")
		if len(fileParts) < 2 {
			continue
		}

		file := fileParts[0]
		funcName := parts[1]

		// Skip generated files and test files
		if strings.Contains(file, "generated_") || strings.HasSuffix(file, ".qtpl.go") || strings.HasSuffix(file, "_test.go") {
			continue
		}

		lineNum, err := strconv.Atoi(fileParts[1])
		if err != nil {
			continue
		}

		fileToFuncs[file] = append(fileToFuncs[file], deadFunc{name: funcName, line: lineNum})
	}

	// Process each file
	totalDeleted := 0

	for file, funcs := range fileToFuncs {
		deleted, err := deleteDeadFunctionsFromFile(file, funcs)
		if err != nil {
			fmt.Printf("Warning: failed to process %s: %v\n", file, err)

			continue
		}

		totalDeleted += deleted
	}

	fmt.Printf("Deleted %d unreachable functions from %d files\n", totalDeleted, len(fileToFuncs))

	return nil
}

// FindRedundantTests identifies unit tests that don't provide unique coverage beyond golden+UAT tests.
// This is a convenience wrapper for this repository's specific configuration.
func FindRedundantTests() error {
	config := RedundancyConfig{
		BaselineTests: []BaselineTestSpec{
			{Package: "./impgen/run", TestPattern: "TestUATConsistency"},
			{Package: "./UAT/core/...", TestPattern: ""},
			{Package: "./UAT/variations/...", TestPattern: ""},
		},
		CoverageThreshold: 80.0,
		PackageToAnalyze:  "./...",
		// Only measure coverage of impgen and imptest packages, not test fixtures
		CoveragePackages: "./impgen/...,./imptest/...",
	}

	return findRedundantTestsWithConfig(config)
}

// FixImports fixes all imports in the codebase.
func FixImports() error {
	fmt.Println("Fixing imports...")
	return sh.Run("goimports", "-w", ".")
}

// Fuzz runs the fuzz tests.
func Fuzz() error {
	fmt.Println("Running fuzz tests...")
	return sh.Run("./dev/fuzz.fish")
}

// Generate runs go generate on all packages using the locally-built impgen binary.
func Generate() error {
	fmt.Println("Generating...")

	if err := targ.Deps(Build); err != nil {
		return err
	}

	// Get current PATH and prepend our bin directory
	currentPath := os.Getenv("PATH")

	binDir, err := filepath.Abs("bin")
	if err != nil {
		return fmt.Errorf("failed to get absolute path for bin: %w", err)
	}

	newPath := binDir + string(filepath.ListSeparator) + currentPath

	// Run go generate with modified PATH
	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = append(os.Environ(), "PATH="+newPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// InstallTools installs development tooling.
func InstallTools() error {
	fmt.Println("Installing development tools...")
	return sh.Run("./dev/dev-install.sh")
}

// Lint lints the codebase.
func Lint() error {
	fmt.Println("Linting...")
	return sh.Run("golangci-lint", "run", "-c", "dev/golangci.toml")
}

// LintForFail lints the codebase purely to find out whether anything fails.
func LintForFail() error {
	fmt.Println("Linting to check for overall pass/fail...")

	return sh.Run(
		"golangci-lint", "run",
		"-c", "dev/golangci.toml",
		"--fix=false",
		"--max-issues-per-linter=1",
		"--max-same-issues=1",
		"--allow-parallel-runners",
	)
}

// Modernize updates the codebase to use modern Go patterns.
func Modernize() error {
	fmt.Println("Modernizing codebase...")

	return sh.Run("go", "run", "golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest",
		"-fix", "./...")
}

// Mutate runs the mutation tests.
func Mutate() error {
	fmt.Println("Running mutation tests...")

	if err := targ.Deps(TestForFail); err != nil {
		return err
	}

	return sh.Run(
		"go",
		"test",
		"-timeout=6000s",
		"-tags=mutation",
		"-ooze.v",
		"./...",
		"-run=TestMutation",
	)
}

// ReorderDecls reorders declarations in Go files per conventions.
func ReorderDecls() error {
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
func ReorderDeclsCheck() error {
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

		// Get reordered version
		reordered, err := reorder.Source(string(content))
		if err != nil {
			fmt.Printf("Warning: failed to reorder %s: %v\n", file, err)

			continue
		}

		// Check if reordering would change the file
		if string(content) != reordered {
			outOfOrderFiles++
			fmt.Printf("\n%s:\n", file)

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

			// Show diff
			diff := textdiff.Unified(file+" (current)", file+" (reordered)", string(content), reordered)
			if diff != "" {
				fmt.Printf("\n%s\n", diff)
			}
		}
	}

	if outOfOrderFiles > 0 {
		fmt.Printf("\n%d file(s) need reordering (out of %d processed). Run 'targ reorder-decls' to fix.\n", outOfOrderFiles, filesProcessed)

		return fmt.Errorf("%d file(s) need reordering", outOfOrderFiles)
	}

	fmt.Printf("All files are correctly ordered (%d files processed).\n", filesProcessed)

	return nil
}

// Test runs the unit tests.
func Test() error {
	fmt.Println("Running unit tests...")

	if err := targ.Deps(Generate); err != nil {
		return err
	}

	// Skip TestRaceRegression tests in CI runs
	// Use -count=1 to disable caching so coverage is regenerated
	err := sh.Run(
		"go",
		"test",
		"-timeout=2m",
		"-race",
		"-count=1",
		"-skip=TestRaceRegression",
		"-coverprofile=coverage.out",
		"-coverpkg=./impgen/...,./imptest/...",
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

// TestForFail runs the unit tests purely to find out whether any fail.
func TestForFail() error {
	fmt.Println("Running unit tests for overall pass/fail...")

	if err := targ.Deps(Generate); err != nil {
		return err
	}

	return sh.Run(
		"go",
		"test",
		"-timeout=10s",
		"./...",
		"-failfast",
	)
}

// Tidy tidies up go.mod.
func Tidy() error {
	fmt.Println("Tidying go.mod...")
	return sh.Run("go", "mod", "tidy")
}

// TodoCheck checks for TODO and FIXME comments using golangci-lint.
func TodoCheck() error {
	fmt.Println("Checking for TODOs...")
	return sh.Run("golangci-lint", "run", "-c", "dev/golangci-todos.toml")
}

// Watch re-runs Check whenever files change.
func Watch(ctx context.Context) error {
	fmt.Println("Watching...")

	return file.Watch(ctx, []string{"**/*.go", "**/*.fish", "**/*.toml"}, file.WatchOptions{}, func(changes file.ChangeSet) error {
		// Filter out generated files and coverage output to avoid infinite loops
		if !hasRelevantChanges(changes) {
			return nil
		}

		fmt.Println("Change detected...")

		targ.ResetDeps() // Clear execution cache so targets run again

		err := Check()
		if err != nil {
			fmt.Println("continuing to watch after check failure (see errors above)")
		} else {
			fmt.Println("continuing to watch after all checks passed!")
		}

		return nil // Don't stop watching on error
	})
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

// coveredLine represents a single line covered by a test.
type coveredLine struct {
	file string
	line int
}

type deadFunc struct {
	name string
	line int
}

type lineAndCoverage struct {
	line     string
	coverage float64
}

type position struct {
	line int
	col  int
}

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

type segment struct {
	start position
	end   position
	count int
}

// testInfo holds a test function name with its package.
type testInfo struct {
	pkg  string
	name string
}

// qualifiedName returns the package-qualified test name (pkg:TestName).
func (t testInfo) qualifiedName() string {
	return t.pkg + ":" + t.name
}

// Helper Functions

func deleteDeadFunctionsFromFile(filename string, funcs []deadFunc) (int, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return 0, fmt.Errorf("failed to parse file: %w", err)
	}

	toDelete := make(map[string]bool)
	for _, f := range funcs {
		toDelete[f.name] = true
	}

	newDecls := []ast.Decl{}
	deleted := 0

	for _, decl := range file.Decls {
		keep := true

		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			funcName := funcDecl.Name.Name

			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				recvType := funcDecl.Recv.List[0].Type
				var typeName string

				switch t := recvType.(type) {
				case *ast.StarExpr:
					if ident, ok := t.X.(*ast.Ident); ok {
						typeName = ident.Name
					}
				case *ast.Ident:
					typeName = t.Name
				}

				fullName := typeName + "." + funcName
				if toDelete[fullName] || toDelete[funcName] {
					keep = false
				}
			} else {
				if toDelete[funcName] {
					keep = false
				}
			}
		}

		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if toDelete[typeSpec.Name.Name] {
						keep = false
					}
				}
			}
		}

		if keep {
			newDecls = append(newDecls, decl)
		} else {
			deleted++
		}
	}

	if deleted == 0 {
		return 0, nil
	}

	file.Decls = newDecls

	var buf bytes.Buffer

	err = printer.Fprint(&buf, fset, file)
	if err != nil {
		return 0, fmt.Errorf("failed to print AST: %w", err)
	}

	err = os.WriteFile(filename, buf.Bytes(), 0o600)
	if err != nil {
		return 0, fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("  %s: deleted %d declarations\n", filename, deleted)

	return deleted, nil
}

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

// findRedundantTestsWithConfig identifies unit tests that don't provide unique coverage beyond baseline tests.
// This generic version can be used in any repository by providing appropriate configuration.
func findRedundantTestsWithConfig(config RedundancyConfig) error {
	fmt.Println("Finding redundant tests...")
	fmt.Println()

	// Default to ./... if not specified
	coverpkg := config.CoveragePackages
	if coverpkg == "" {
		coverpkg = "./..."
	}

	// Step 1-N: Run baseline tests and merge coverage
	var coverageFiles []string
	// Track baseline tests with package-qualified names to avoid collisions across packages
	baselineTests := make(map[string]bool) // key: "pkg:TestName"

	for i, spec := range config.BaselineTests {
		stepNum := i + 1
		coverageFile := fmt.Sprintf("baseline_%d.out", stepNum)
		coverageFiles = append(coverageFiles, coverageFile)

		if spec.TestPattern != "" {
			fmt.Printf("Step %d: Running baseline test %s in %s...\n", stepNum, spec.TestPattern, spec.Package)

			err := runQuietCoverage("go", "test", "-coverprofile="+coverageFile, "-coverpkg="+coverpkg,
				"-run", spec.TestPattern, spec.Package)
			if err != nil {
				return fmt.Errorf("baseline test %s failed: %w", spec.TestPattern, err)
			}

			// Resolve package path to full module path for consistent matching
			fullPkg, err := output("go", "list", spec.Package)
			if err != nil {
				return fmt.Errorf("failed to resolve package %s: %w", spec.Package, err)
			}

			baselineTests[strings.TrimSpace(fullPkg)+":"+spec.TestPattern] = true
		} else {
			fmt.Printf("Step %d: Running all tests in %s ...\n", stepNum, spec.Package)

			err := runQuietCoverage("go", "test", "-coverprofile="+coverageFile, "-coverpkg="+coverpkg, spec.Package)
			if err != nil {
				return fmt.Errorf("baseline tests in %s failed: %w", spec.Package, err)
			}

			// List and record all test functions with their packages
			pkgTests, err := listTestFunctionsWithPackages(spec.Package)
			if err != nil {
				// Non-fatal: we'll just skip tracking these test names
				fmt.Printf("  Warning: couldn't list tests in %s: %v\n", spec.Package, err)
			} else {
				for _, t := range pkgTests {
					baselineTests[t.qualifiedName()] = true
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

	// List all test functions, excluding baseline tests
	listStep := analysisStep + 1
	fmt.Printf("Step %d: Listing unit tests...\n", listStep)

	allTestsWithPkgs, err := listTestFunctionsWithPackages(config.PackageToAnalyze)
	if err != nil {
		return fmt.Errorf("failed to list tests: %w", err)
	}

	// Filter out baseline tests using package-qualified names
	var testFuncs []testInfo

	for _, t := range allTestsWithPkgs {
		if !baselineTests[t.qualifiedName()] {
			testFuncs = append(testFuncs, t)
		}
	}

	fmt.Printf("  Found %d unit tests (%d total, %d baseline excluded)\n\n",
		len(testFuncs), len(allTestsWithPkgs), len(allTestsWithPkgs)-len(testFuncs))

	// Run each test individually to collect coverage
	checkStep := listStep + 1
	fmt.Printf("Step %d: Running each test individually to collect coverage...\n", checkStep)

	// Map from test qualified name to its coverage file
	testCoverageFiles := make(map[string]string)
	var testOrder []testInfo // preserve order for consistent output

	for _, test := range testFuncs {
		fmt.Printf("  Running %s... ", test.qualifiedName())

		// Run just this one test in its package
		coverFile := fmt.Sprintf("cov_%s_%s.out", sanitize(filepath.Base(test.pkg)), test.name)
		coverFileRaw := coverFile + ".raw"

		testErr := runQuietCoverage("go", "test", "-coverprofile="+coverFileRaw, "-coverpkg="+coverpkg,
			"-run", "^"+test.name+"$", test.pkg)

		if testErr != nil {
			fmt.Printf("FAILED\n")

			continue
		}

		// Filter .qtpl files from coverage
		err := filterQtplFromCoverage(coverFileRaw, coverFile)
		if err != nil {
			fmt.Printf("FAILED (filter)\n")
			os.Remove(coverFileRaw)

			continue
		}

		os.Remove(coverFileRaw)

		testCoverageFiles[test.qualifiedName()] = coverFile
		testOrder = append(testOrder, test)
		fmt.Printf("OK\n")
	}

	// Compute initial total coverage (baseline + all tests)
	analyzeStep := checkStep + 1
	fmt.Printf("Step %d: Computing total coverage with all tests...\n", analyzeStep)

	// Build initial set of coverage files
	currentCoverageFiles := make([]string, 0, 1+len(testCoverageFiles))
	currentCoverageFiles = append(currentCoverageFiles, baselineFile)

	for _, f := range testCoverageFiles {
		currentCoverageFiles = append(currentCoverageFiles, f)
	}

	totalCoverageFile := "total_coverage.out"

	err = mergeMultipleCoverageFiles(currentCoverageFiles, totalCoverageFile)
	if err != nil {
		return fmt.Errorf("failed to merge total coverage: %w", err)
	}

	totalFuncs, err := getFunctionsAboveThreshold(totalCoverageFile, config.CoverageThreshold)
	if err != nil {
		return fmt.Errorf("failed to analyze total coverage: %w", err)
	}

	fmt.Printf("  Total coverage: %d functions at %.0f%%+\n", len(totalFuncs), config.CoverageThreshold)
	os.Remove(totalCoverageFile)

	// Calculate unique line coverage per test to prioritize removal order
	sortStep := analyzeStep + 1
	fmt.Printf("Step %d: Sorting tests by unique line coverage...\n", sortStep)

	// Build map of line -> tests that cover it
	type fileLine struct {
		file string
		line int
	}

	lineCoverage := make(map[fileLine][]string) // line -> list of test qualified names

	for _, test := range testOrder {
		coverFile := testCoverageFiles[test.qualifiedName()]

		lines, err := getCoveredLines(coverFile)
		if err != nil {
			continue // Skip tests we can't analyze
		}

		for _, line := range lines {
			fl := fileLine{file: line.file, line: line.line}
			lineCoverage[fl] = append(lineCoverage[fl], test.qualifiedName())
		}
	}

	// Count uniquely covered lines per test (lines covered by ONLY that test)
	uniqueLineCount := make(map[string]int)

	for _, tests := range lineCoverage {
		if len(tests) == 1 {
			uniqueLineCount[tests[0]]++
		}
	}

	// Sort tests by unique line count (ascending - fewest unique lines first)
	sortedTests := make([]testInfo, len(testOrder))
	copy(sortedTests, testOrder)

	sort.Slice(sortedTests, func(i, j int) bool {
		return uniqueLineCount[sortedTests[i].qualifiedName()] < uniqueLineCount[sortedTests[j].qualifiedName()]
	})

	// Report sorting results
	zeroUnique := 0

	for _, test := range sortedTests {
		if uniqueLineCount[test.qualifiedName()] == 0 {
			zeroUnique++
		}
	}

	fmt.Printf("  %d tests have 0 unique lines (prime redundancy candidates)\n", zeroUnique)

	// Iterative greedy removal
	iterativeStep := sortStep + 1
	fmt.Printf("Step %d: Iteratively identifying redundant tests...\n", iterativeStep)
	fmt.Printf("  %-80s %6s   %s\n", "TEST", "UNIQUE", "DECISION")
	fmt.Printf("  %-80s %6s   %s\n", strings.Repeat("-", 80), "------", "--------")

	type testResult struct {
		name        string
		pkg         string
		uniqueCount int
	}

	var redundantTests []testResult

	// Start with sorted tests as candidates for removal
	remainingTests := sortedTests
	printedAsKeep := make(map[string]bool) // Track tests already printed as KEEP

	for {
		foundRedundant := false

		for i, test := range remainingTests {
			qName := test.qualifiedName()
			uniqueLines := uniqueLineCount[qName]

			// Skip evaluation if already marked as KEEP in previous iteration
			if printedAsKeep[qName] {
				continue
			}

			fmt.Printf("  %-80s %6d   ", qName, uniqueLines)

			// Build list of coverage files WITHOUT this test
			filesWithout := make([]string, 0, len(remainingTests))
			filesWithout = append(filesWithout, baselineFile)
			testFile := testCoverageFiles[test.qualifiedName()]

			for _, t := range remainingTests {
				f := testCoverageFiles[t.qualifiedName()]
				if f != testFile {
					filesWithout = append(filesWithout, f)
				}
			}

			// Merge coverage without this test
			withoutFile := "without_test.out"

			err := mergeMultipleCoverageFiles(filesWithout, withoutFile)
			if err != nil {
				fmt.Printf("FAILED (merge)\n")

				continue
			}

			withoutFuncs, err := getFunctionsAboveThreshold(withoutFile, config.CoverageThreshold)
			if err != nil {
				fmt.Printf("FAILED (analyze)\n")
				os.Remove(withoutFile)

				continue
			}

			os.Remove(withoutFile)

			// Check if any function dropped below threshold
			var droppedFuncs []string

			for fn := range totalFuncs {
				if !withoutFuncs[fn] {
					droppedFuncs = append(droppedFuncs, fn)
				}
			}

			if len(droppedFuncs) > 0 {
				fmt.Printf("KEEP (%d funcs drop)\n", len(droppedFuncs))
				printedAsKeep[qName] = true // Mark as KEEP to skip in future iterations

				continue // Check next test - this one is needed
			}

			// Test is redundant - removing it doesn't drop any function below threshold
			fmt.Printf("REDUNDANT\n")
			redundantTests = append(redundantTests, testResult{
				name:        test.name,
				pkg:         test.pkg,
				uniqueCount: uniqueLines,
			})

			// Remove this test from remaining tests
			remainingTests = append(remainingTests[:i], remainingTests[i+1:]...)
			foundRedundant = true

			break // restart the loop with updated set
		}

		if !foundRedundant {
			// No more redundant tests found - we're done
			break
		}
	}

	// Build uniqueTests from remaining tests
	var uniqueTests []testResult

	for _, test := range remainingTests {
		uniqueTests = append(uniqueTests, testResult{
			name:        test.name,
			pkg:         test.pkg,
			uniqueCount: uniqueLineCount[test.qualifiedName()],
		})
	}

	// Clean up individual coverage files
	for _, f := range testCoverageFiles {
		os.Remove(f)
	}

	// Report results
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("RESULTS")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Printf("\nTests that must be kept (%d):\n", len(uniqueTests))
	fmt.Printf("  %-80s %6s\n", "TEST", "UNIQUE")
	fmt.Printf("  %-80s %6s\n", strings.Repeat("-", 80), "------")

	for _, test := range uniqueTests {
		qName := test.pkg + ":" + test.name
		fmt.Printf("  %-80s %6d\n", qName, test.uniqueCount)
	}

	fmt.Printf("\nRedundant tests (%d):\n", len(redundantTests))
	fmt.Printf("  %-80s %6s\n", "TEST", "UNIQUE")
	fmt.Printf("  %-80s %6s\n", strings.Repeat("-", 80), "------")

	// Sort redundant tests by package for grouped display
	sort.Slice(redundantTests, func(i, j int) bool {
		if redundantTests[i].pkg != redundantTests[j].pkg {
			return redundantTests[i].pkg < redundantTests[j].pkg
		}

		return redundantTests[i].name < redundantTests[j].name
	})

	for _, test := range redundantTests {
		qName := test.pkg + ":" + test.name
		fmt.Printf("  %-80s %6d\n", qName, test.uniqueCount)
	}

	fmt.Println()

	return nil
}

// getCoveredLines parses a coverage file and returns all lines that were executed.
// A line is considered covered if any block covering it has count > 0.
func getCoveredLines(coverageFile string) ([]coveredLine, error) {
	data, err := os.ReadFile(coverageFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	coveredSet := make(map[coveredLine]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		// Format: file:startLine.startCol,endLine.endCol statements count
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		blockID := parts[0]
		count, _ := strconv.Atoi(parts[2])

		if count == 0 {
			continue // Not covered
		}

		file, startLine, _, endLine, _, err := parseBlockID(blockID)
		if err != nil {
			continue
		}

		// Mark all lines in the block as covered
		for lineNum := startLine; lineNum <= endLine; lineNum++ {
			coveredSet[coveredLine{file: file, line: lineNum}] = true
		}
	}

	// Convert set to slice
	result := make([]coveredLine, 0, len(coveredSet))

	for cl := range coveredSet {
		result = append(result, cl)
	}

	return result, nil
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

// hasRelevantChanges returns true if the changeset contains files we care about.
// Filters out generated files and build artifacts that Check() itself creates.
func hasRelevantChanges(changes file.ChangeSet) bool {
	allFiles := append(append(changes.Added, changes.Removed...), changes.Modified...)

	for _, f := range allFiles {
		// Skip generated test files
		if strings.Contains(f, "generated_") {
			continue
		}
		// Skip coverage output
		if strings.HasSuffix(f, "coverage.out") {
			continue
		}
		// Found a relevant change
		return true
	}

	return false
}

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

func isGeneratedFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	buf := make([]byte, 200)

	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read %s: %w", path, err)
	}

	content := string(buf[:n])

	return strings.Contains(content, "Code generated") || strings.Contains(content, "DO NOT EDIT"), nil
}

// listTestFunctionsWithPackages lists all test functions with their packages.
func listTestFunctionsWithPackages(pkgPattern string) ([]testInfo, error) {
	// First, expand the package pattern to get actual packages
	listOut, err := output("go", "list", pkgPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var allTests []testInfo
	packages := strings.Split(strings.TrimSpace(listOut), "\n")

	for _, pkg := range packages {
		if pkg == "" {
			continue
		}

		out, err := output("go", "test", "-list", ".", pkg)
		if err != nil {
			// Package may have no tests, skip it
			continue
		}

		lines := strings.Split(out, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Test") {
				allTests = append(allTests, testInfo{pkg: pkg, name: line})
			}
		}
	}

	return allTests, nil
}

// mergeCoverageBlocks merges duplicate coverage blocks in a coverage file.
// This handles the case where multiple test packages cover the same code.
func mergeCoverageBlocks(coverageFile string) error {
	data, err := os.ReadFile(coverageFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return nil
	}

	// First line is mode
	modeLine := lines[0]

	// Merge blocks by key (file:start,end statements)
	blocks := make(map[string]coverageBlock)

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		block, err := parseCoverageBlock(line)
		if err != nil {
			continue
		}

		key := fmt.Sprintf("%s:%d.%d,%d.%d %d",
			block.file, block.startLine, block.startCol,
			block.endLine, block.endCol, block.statements)

		if existing, ok := blocks[key]; ok {
			existing.count += block.count
			blocks[key] = existing
		} else {
			blocks[key] = block
		}
	}

	// Write merged blocks
	var result strings.Builder

	result.WriteString(modeLine)
	result.WriteString("\n")

	for _, block := range blocks {
		fmt.Fprintf(&result, "%s:%d.%d,%d.%d %d %d\n",
			block.file, block.startLine, block.startCol,
			block.endLine, block.endCol, block.statements, block.count)
	}

	return os.WriteFile(coverageFile, []byte(result.String()), 0o600)
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

	for _, blks := range fileBlocks {
		segments := splitBlocksIntoSegments(blks)
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
func mergeMultipleCoverageFiles(files []string, outputFile string) error {
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

	err := os.WriteFile(outputFile, []byte(combined), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", outputFile, err)
	}

	// Merge overlapping blocks using existing logic
	return mergeCoverageBlocksFile(outputFile)
}

// output runs a command and captures stdout only (stderr goes to os.Stderr).
func output(command string, args ...string) (string, error) {
	buf := &bytes.Buffer{}
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	return strings.TrimSuffix(buf.String(), "\n"), err
}

func parseBlockID(blockID string) (file string, startLine, startCol, endLine, endCol int, err error) {
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

func parseCoverageBlock(line string) (coverageBlock, error) {
	// Format: file:startLine.startCol,endLine.endCol statements count
	parts := strings.Fields(line)
	if len(parts) != 3 {
		return coverageBlock{}, fmt.Errorf("invalid line format")
	}

	blockID := parts[0]
	statements, _ := strconv.Atoi(parts[1])
	count, _ := strconv.Atoi(parts[2])

	file, startLine, startCol, endLine, endCol, err := parseBlockID(blockID)
	if err != nil {
		return coverageBlock{}, err
	}

	return coverageBlock{
		file:       file,
		startLine:  startLine,
		startCol:   startCol,
		endLine:    endLine,
		endCol:     endCol,
		statements: statements,
		count:      count,
	}, nil
}

// runQuietCoverage runs a go test command with coverage, filtering out expected warnings
// about packages not matching coverage patterns.
func runQuietCoverage(command string, arg ...string) error {
	cmd := exec.Command(command, arg...)
	cmd.Stdin = os.Stdin

	// Capture stderr to filter out coverage warnings
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	// Filter and display stderr, removing expected coverage warnings
	stderrLines := strings.Split(stderrBuf.String(), "\n")

	for _, line := range stderrLines {
		// Skip the "no packages being tested depend on matches" warning
		if strings.Contains(line, "no packages being tested depend on matches") {
			continue
		}
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Show other stderr output
		fmt.Fprintln(os.Stderr, line)
	}

	return err
}

// sanitize makes a string safe for use in filenames.
func sanitize(s string) string {
	// Replace characters that are problematic in filenames
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		".", "_",
	)

	return replacer.Replace(s)
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
