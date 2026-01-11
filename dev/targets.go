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
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

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
func Check(ctx context.Context) error {
	fmt.Println("Checking...")

	return targ.Deps(
		func() error { return Tidy(ctx) },           // clean up the module dependencies
		func() error { return DeleteDeadcode(ctx) }, // no use doing anything else to dead code
		func() error { return Fmt(ctx) },            // after dead code removal, format code including imports
		func() error { return Modernize(ctx) },      // no use doing anything else to old code patterns
		func() error { return CheckCoverage(ctx) },  // does our code work?
		func() error { return CheckNils(ctx) },      // is it nil free?
		func() error { return ReorderDecls(ctx) },   // linter will yell about declaration order if not correct
		func() error { return Lint(ctx) },
	)
}

// CheckCoverage checks that function coverage meets the minimum threshold.
func CheckCoverage(ctx context.Context) error {
	fmt.Println("Checking coverage...")

	if err := targ.Deps(func() error { return Test(ctx) }); err != nil {
		return err
	}

	// Merge duplicate coverage blocks from cross-package testing
	if err := mergeCoverageBlocks("coverage.out"); err != nil {
		return fmt.Errorf("failed to merge coverage blocks: %w", err)
	}

	out, err := output(ctx, "go", "tool", "cover", "-func=coverage.out")
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
func CheckNils(ctx context.Context) error {
	fmt.Println("Running check for nils...")
	return sh.RunContext(ctx, "nilaway", "-fix", "./...")
}

// CheckNilsForFail checks for nils, just for failure.
func CheckNilsForFail(ctx context.Context) error {
	fmt.Println("Running check for nils...")
	return sh.RunContext(ctx, "nilaway", "./...")
}

// Clean cleans up the dev env.
func Clean() {
	fmt.Println("Cleaning...")
	os.Remove("coverage.out")
}

// Deadcode checks that there's no dead code in codebase.
func Deadcode(ctx context.Context) error {
	fmt.Println("Checking for dead code...")

	out, err := output(ctx, "deadcode", "-test", "./...")
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
func DeleteDeadcode(ctx context.Context) error {
	fmt.Println("Deleting dead code...")

	out, err := output(ctx, "deadcode", "-test", "./...")
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

// Fmt formats the codebase using golangci-lint formatters.
func Fmt(ctx context.Context) error {
	fmt.Println("Formatting...")
	return sh.RunContext(ctx, "golangci-lint", "run", "-c", "dev/golangci-fmt.toml")
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
func Lint(ctx context.Context) error {
	fmt.Println("Linting...")
	return sh.RunContext(ctx, "golangci-lint", "run", "-c", "dev/golangci-lint.toml")
}

// LintForFail lints the codebase purely to find out whether anything fails.
func LintForFail() error {
	fmt.Println("Linting to check for overall pass/fail...")

	return sh.Run(
		"golangci-lint", "run",
		"-c", "dev/golangci-lint.toml",
		"--fix=false",
		"--max-issues-per-linter=1",
		"--max-same-issues=1",
		"--allow-parallel-runners",
	)
}

// Modernize updates the codebase to use modern Go patterns.
func Modernize(ctx context.Context) error {
	fmt.Println("Modernizing codebase...")

	return sh.RunContext(ctx, "go", "run", "golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest",
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
func ReorderDecls(ctx context.Context) error {
	_ = ctx // Reserved for future cancellation support
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
func Test(ctx context.Context) error {
	fmt.Println("Running unit tests...")

	if err := targ.Deps(Generate); err != nil {
		return err
	}

	// Skip TestRaceRegression tests in CI runs
	// Use -count=1 to disable caching so coverage is regenerated
	err := sh.RunContext(ctx,
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
func Tidy(ctx context.Context) error {
	fmt.Println("Tidying go.mod...")
	return sh.RunContext(ctx, "go", "mod", "tidy")
}

// TodoCheck checks for TODO and FIXME comments using golangci-lint.
func TodoCheck() error {
	fmt.Println("Checking for TODOs...")
	return sh.Run("golangci-lint", "run", "-c", "dev/golangci-todos.toml")
}

// Watch re-runs Check whenever files change.
func Watch(ctx context.Context) error {
	fmt.Println("Watching...")

	var (
		cancelCheck context.CancelFunc
		checkMu     sync.Mutex
	)

	return file.Watch(ctx, []string{"**/*.go", "**/*.fish", "**/*.toml"}, file.WatchOptions{}, func(changes file.ChangeSet) error {
		// Filter out generated files and coverage output to avoid infinite loops
		if !hasRelevantChanges(changes) {
			return nil
		}

		// Log the changed files for debugging
		fmt.Println("Change detected in:")
		for _, f := range changes.Added {
			fmt.Printf("  + %s\n", f)
		}
		for _, f := range changes.Modified {
			fmt.Printf("  ~ %s\n", f)
		}
		for _, f := range changes.Removed {
			fmt.Printf("  - %s\n", f)
		}

		checkMu.Lock()
		defer checkMu.Unlock()

		// Cancel any running check
		if cancelCheck != nil {
			fmt.Println("Cancelling previous check...")
			cancelCheck()
		}

		// Create new cancellable context for this check
		checkCtx, cancel := context.WithCancel(ctx)
		cancelCheck = cancel

		targ.ResetDeps() // Clear execution cache so targets run again

		err := Check(checkCtx)
		if errors.Is(err, context.Canceled) {
			fmt.Println("Check cancelled, restarting...")
		} else if err != nil {
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

type deadFunc struct {
	name string
	line int
}

type lineAndCoverage struct {
	line     string
	coverage float64
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

// detectParallelTests detects which tests are marked with t.Parallel().
// Returns a map of qualified test names (pkg:TestName) that are parallel-safe.
func detectParallelTests(tests []testInfo) map[string]bool {
	result := make(map[string]bool)

	// Group tests by package
	testsByPkg := make(map[string][]testInfo)
	for _, t := range tests {
		testsByPkg[t.pkg] = append(testsByPkg[t.pkg], t)
	}

	for pkg, pkgTests := range testsByPkg {
		// Get the directory for this package
		pkgDir, err := output(context.Background(), "go", "list", "-f", "{{.Dir}}", pkg)
		if err != nil {
			continue
		}

		pkgDir = strings.TrimSpace(pkgDir)

		// Find test files in this package
		testFiles, err := filepath.Glob(filepath.Join(pkgDir, "*_test.go"))
		if err != nil {
			continue
		}

		// Parse each test file and check for t.Parallel() calls
		fset := token.NewFileSet()

		for _, testFile := range testFiles {
			f, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
			if err != nil {
				continue
			}

			// Find all test functions and check if they call t.Parallel()
			ast.Inspect(f, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Name == nil {
					return true
				}

				// Only consider Test* functions with *testing.T parameter
				if !strings.HasPrefix(fn.Name.Name, "Test") {
					return true
				}

				// Check if this function is one we care about
				qualifiedName := pkg + ":" + fn.Name.Name
				isRelevant := false

				for _, t := range pkgTests {
					if t.qualifiedName() == qualifiedName {
						isRelevant = true

						break
					}
				}

				if !isRelevant {
					return true
				}

				// Check if function body contains t.Parallel() call
				if fn.Body != nil && hasParallelCall(fn.Body) {
					result[qualifiedName] = true
				}

				return true
			})
		}
	}

	return result
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

	// Step 1: Identify baseline tests (preferred tests)
	fmt.Println("Step 1: Identifying baseline tests...")
	baselineTestSet := make(map[string]bool) // key: "pkg:TestName"

	for _, spec := range config.BaselineTests {
		if spec.TestPattern != "" {
			// Resolve package path to full module path for consistent matching
			fullPkg, err := output(context.Background(), "go", "list", spec.Package)
			if err != nil {
				return fmt.Errorf("failed to resolve package %s: %w", spec.Package, err)
			}

			baselineTestSet[strings.TrimSpace(fullPkg)+":"+spec.TestPattern] = true
		} else {
			// List all test functions in package
			pkgTests, err := listTestFunctionsWithPackages(spec.Package)
			if err != nil {
				fmt.Printf("  Warning: couldn't list tests in %s: %v\n", spec.Package, err)
			} else {
				for _, t := range pkgTests {
					baselineTestSet[t.qualifiedName()] = true
				}
			}
		}
	}

	fmt.Printf("  Identified %d baseline tests\n", len(baselineTestSet))

	// Step 2: List all tests
	fmt.Println("\nStep 2: Listing all tests...")

	allTests, err := listTestFunctionsWithPackages(config.PackageToAnalyze)
	if err != nil {
		return fmt.Errorf("failed to list tests: %w", err)
	}

	// Separate into baseline and non-baseline
	var baselineTests []testInfo
	var nonBaselineTests []testInfo

	for _, t := range allTests {
		if baselineTestSet[t.qualifiedName()] {
			baselineTests = append(baselineTests, t)
		} else {
			nonBaselineTests = append(nonBaselineTests, t)
		}
	}

	fmt.Printf("  Found %d baseline tests, %d non-baseline tests (%d total)\n",
		len(baselineTests), len(nonBaselineTests), len(allTests))

	// Step 3: Run each test individually to collect coverage
	fmt.Println("\nStep 3: Running each test individually to collect coverage...")

	// Combine all tests
	allTestsToRun := append(baselineTests, nonBaselineTests...)

	// Detect which tests are marked with t.Parallel()
	fmt.Println("  Detecting parallel-safe tests...")

	parallelTests := detectParallelTests(allTestsToRun)
	fmt.Printf("  Found %d parallel-safe tests, %d serial tests\n",
		len(parallelTests), len(allTestsToRun)-len(parallelTests))

	testCoverageFiles := make(map[string]string)
	var allTestOrder []testInfo

	// Helper to run a single test and collect coverage
	runSingleTest := func(test testInfo) bool {
		coverFile := fmt.Sprintf("cov_%s_%s.out", sanitize(filepath.Base(test.pkg)), test.name)
		coverFileRaw := coverFile + ".raw"

		testErr := runQuietCoverage("go", "test", "-count=1", "-coverprofile="+coverFileRaw, "-coverpkg="+coverpkg,
			"-run", "^"+test.name+"$", test.pkg)

		if testErr != nil {
			return false
		}

		err := filterQtplFromCoverage(coverFileRaw, coverFile)
		if err != nil {
			os.Remove(coverFileRaw)

			return false
		}

		os.Remove(coverFileRaw)
		testCoverageFiles[test.qualifiedName()] = coverFile
		allTestOrder = append(allTestOrder, test)

		return true
	}

	// Separate tests into parallel-safe and serial
	var serialTests []testInfo
	var parallelSafeTests []testInfo

	for _, test := range allTestsToRun {
		if parallelTests[test.qualifiedName()] {
			parallelSafeTests = append(parallelSafeTests, test)
		} else {
			serialTests = append(serialTests, test)
		}
	}

	// Run serial tests first (sequentially)
	if len(serialTests) > 0 {
		fmt.Printf("  Running %d serial tests sequentially...\n", len(serialTests))

		for i, test := range serialTests {
			fmt.Printf("    [%d/%d] %s... ", i+1, len(serialTests), test.qualifiedName())

			if runSingleTest(test) {
				fmt.Printf("OK\n")
			} else {
				fmt.Printf("FAILED\n")
			}
		}
	}

	// Run parallel-safe tests concurrently
	if len(parallelSafeTests) > 0 {
		fmt.Printf("  Running %d parallel-safe tests concurrently...\n", len(parallelSafeTests))

		var testCoverageFilesMu sync.Mutex
		var allTestOrderMu sync.Mutex

		numWorkers := runtime.NumCPU()
		sem := make(chan struct{}, numWorkers)
		var wg sync.WaitGroup
		var completed int32

		for _, test := range parallelSafeTests {
			wg.Add(1)

			go func(test testInfo) {
				defer wg.Done()

				sem <- struct{}{}
				defer func() { <-sem }()

				coverFile := fmt.Sprintf("cov_%s_%s.out", sanitize(filepath.Base(test.pkg)), test.name)
				coverFileRaw := coverFile + ".raw"

				testErr := runQuietCoverage("go", "test", "-count=1", "-coverprofile="+coverFileRaw, "-coverpkg="+coverpkg,
					"-run", "^"+test.name+"$", test.pkg)

				current := atomic.AddInt32(&completed, 1)

				if testErr != nil {
					fmt.Printf("    [%d/%d] %s... FAILED\n", current, len(parallelSafeTests), test.qualifiedName())

					return
				}

				err := filterQtplFromCoverage(coverFileRaw, coverFile)
				if err != nil {
					fmt.Printf("    [%d/%d] %s... FAILED (filter)\n", current, len(parallelSafeTests), test.qualifiedName())
					os.Remove(coverFileRaw)

					return
				}

				os.Remove(coverFileRaw)

				testCoverageFilesMu.Lock()
				testCoverageFiles[test.qualifiedName()] = coverFile
				testCoverageFilesMu.Unlock()

				allTestOrderMu.Lock()
				allTestOrder = append(allTestOrder, test)
				allTestOrderMu.Unlock()

				fmt.Printf("    [%d/%d] %s... OK\n", current, len(parallelSafeTests), test.qualifiedName())
			}(test)
		}

		wg.Wait()
	}

	// Step 4: Compute target coverage (all tests merged)
	fmt.Println("\nStep 4: Computing target coverage with all tests...")

	var allCoverageFiles []string
	for _, f := range testCoverageFiles {
		allCoverageFiles = append(allCoverageFiles, f)
	}

	if len(allCoverageFiles) == 0 {
		return fmt.Errorf("no tests ran successfully")
	}

	totalCoverageFile := "total_coverage.out"

	err = mergeMultipleCoverageFiles(allCoverageFiles, totalCoverageFile)
	if err != nil {
		return fmt.Errorf("failed to merge total coverage: %w", err)
	}

	targetCoverage, err := getAllFunctionsCoverage(totalCoverageFile)
	if err != nil {
		return fmt.Errorf("failed to analyze target coverage: %w", err)
	}

	// Build target set: functions that reach threshold with all tests
	targetFuncs := make(map[string]bool)

	for fn, cov := range targetCoverage {
		if cov >= config.CoverageThreshold {
			targetFuncs[fn] = true
		}
	}

	fmt.Printf("  Target: %d functions at %.0f%%+ (with all tests)\n", len(targetFuncs), config.CoverageThreshold)

	os.Remove(totalCoverageFile)

	// Step 5: Greedy addition starting from 0 coverage
	fmt.Println("\nStep 5: Building minimal test set from zero (preferring baseline tests)...")
	fmt.Printf("  %-80s %6s   %s\n", "TEST", "FILLS", "DECISION")
	fmt.Printf("  %-80s %6s   %s\n", strings.Repeat("-", 80), "------", "--------")

	type testResult struct {
		name       string
		pkg        string
		isBaseline bool
		gapsFilled int
	}

	var keptTests []testResult
	keptTestFiles := []string{}

	// Track current coverage level for each target function (starts at 0)
	currentCoverage := make(map[string]float64)
	for fn := range targetFuncs {
		currentCoverage[fn] = 0
	}

	// Track which functions still need coverage (below threshold)
	remainingGaps := make(map[string]bool)
	for fn := range targetFuncs {
		remainingGaps[fn] = true
	}

	keptTestSet := make(map[string]bool)

	// Maintain a running "merged so far" file to avoid re-merging all kept files each time
	currentMergedFile := ""

	// Helper to evaluate a single test candidate
	// Returns list of functions where this test improves coverage (for functions still below threshold)
	evalCandidate := func(test testInfo, mergedSoFar string, gaps map[string]bool, currCov map[string]float64) []string {
		qName := test.qualifiedName()
		coverFile := testCoverageFiles[qName]

		if coverFile == "" {
			return nil
		}

		var improvements []string

		if mergedSoFar == "" {
			// First test - check its coverage directly
			testCov, covErr := getAllFunctionsCoverage(coverFile)
			if covErr != nil {
				return nil
			}

			for fn := range gaps {
				// Count as improvement if this test provides any coverage for an unfilled gap
				if testCov[fn] > currCov[fn] {
					improvements = append(improvements, fn)
				}
			}
		} else {
			// Merge candidate with current merged coverage (just 2 files!)
			mergedFile := fmt.Sprintf("merged_%s.out", sanitize(qName))

			mergeErr := mergeMultipleCoverageFiles([]string{mergedSoFar, coverFile}, mergedFile)
			if mergeErr != nil {
				return nil
			}

			mergedCov, covErr := getAllFunctionsCoverage(mergedFile)
			os.Remove(mergedFile)

			if covErr != nil {
				return nil
			}

			for fn := range gaps {
				// Count as improvement if merged coverage is better than current
				if mergedCov[fn] > currCov[fn] {
					improvements = append(improvements, fn)
				}
			}
		}

		return improvements
	}

	// Helper to find best test from a pool (parallelized)
	findBestTest := func(pool []testInfo) (testInfo, []string) {
		// Filter out already-kept tests
		var candidates []testInfo

		for _, test := range pool {
			if !keptTestSet[test.qualifiedName()] {
				candidates = append(candidates, test)
			}
		}

		if len(candidates) == 0 {
			return testInfo{}, nil
		}

		// Copy current state for parallel evaluation
		gapsCopy := make(map[string]bool)
		for fn := range remainingGaps {
			gapsCopy[fn] = true
		}

		covCopy := make(map[string]float64)
		for fn, cov := range currentCoverage {
			covCopy[fn] = cov
		}

		// Evaluate candidates in parallel
		type result struct {
			test         testInfo
			improvements []string
		}

		results := make([]result, len(candidates))
		var wg sync.WaitGroup
		// Use more workers for I/O-bound work (file merging + process spawning)
		sem := make(chan struct{}, runtime.NumCPU()*4)

		for i, test := range candidates {
			wg.Add(1)

			go func(idx int, t testInfo) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				improved := evalCandidate(t, currentMergedFile, gapsCopy, covCopy)
				results[idx] = result{test: t, improvements: improved}
			}(i, test)
		}

		wg.Wait()

		// Find best result (most improvements)
		var bestTest testInfo
		var bestImprovements []string

		for _, r := range results {
			if len(r.improvements) > len(bestImprovements) {
				bestTest = r.test
				bestImprovements = r.improvements
			}
		}

		return bestTest, bestImprovements
	}

	// Keep adding tests until no gaps remain
	for len(remainingGaps) > 0 {
		// First try baseline tests
		bestTest, bestGapsFilled := findBestTest(baselineTests)
		isBaseline := true

		// If no baseline test fills any gap, try non-baseline tests
		if len(bestGapsFilled) == 0 {
			bestTest, bestGapsFilled = findBestTest(nonBaselineTests)
			isBaseline = false
		}

		if len(bestGapsFilled) == 0 {
			// No test can fill any remaining gaps
			break
		}

		// Add the best test
		qName := bestTest.qualifiedName()
		marker := ""
		if isBaseline {
			marker = " (baseline)"
		}

		fmt.Printf("  %-80s %6d   KEEP%s\n", qName, len(bestGapsFilled), marker)

		keptTests = append(keptTests, testResult{
			name:       bestTest.name,
			pkg:        bestTest.pkg,
			isBaseline: isBaseline,
			gapsFilled: len(bestGapsFilled),
		})
		keptTestSet[qName] = true
		keptTestFiles = append(keptTestFiles, testCoverageFiles[qName])

		// Update the running merged coverage file
		newMergedFile := fmt.Sprintf("current_merged_%d.out", len(keptTestFiles))

		if currentMergedFile == "" {
			// First test - just copy its coverage file
			data, _ := os.ReadFile(testCoverageFiles[qName])
			_ = os.WriteFile(newMergedFile, data, 0o600)
		} else {
			// Merge with existing
			_ = mergeMultipleCoverageFiles([]string{currentMergedFile, testCoverageFiles[qName]}, newMergedFile)
			os.Remove(currentMergedFile)
		}

		currentMergedFile = newMergedFile

		// Update current coverage levels from new merged file
		newCoverage, err := getAllFunctionsCoverage(newMergedFile)
		if err == nil {
			for fn := range targetFuncs {
				if newCov, ok := newCoverage[fn]; ok {
					currentCoverage[fn] = newCov
				}
			}
		}

		// Remove gaps only when they reach threshold
		for fn := range remainingGaps {
			if currentCoverage[fn] >= config.CoverageThreshold {
				delete(remainingGaps, fn)
			}
		}
	}

	// Clean up the final merged file
	if currentMergedFile != "" {
		os.Remove(currentMergedFile)
	}

	// Mark remaining tests as redundant
	var redundantBaselineTests []testResult
	var redundantNonBaselineTests []testResult

	for _, test := range allTestOrder {
		qName := test.qualifiedName()
		if !keptTestSet[qName] {
			isBaseline := baselineTestSet[qName]
			marker := ""
			if isBaseline {
				marker = " (baseline)"
			}

			fmt.Printf("  %-80s %6d   REDUNDANT%s\n", qName, 0, marker)

			result := testResult{
				name:       test.name,
				pkg:        test.pkg,
				isBaseline: isBaseline,
			}

			if isBaseline {
				redundantBaselineTests = append(redundantBaselineTests, result)
			} else {
				redundantNonBaselineTests = append(redundantNonBaselineTests, result)
			}
		}
	}

	// Validation
	fmt.Println("\nStep 6: Validating final coverage...")

	if len(keptTestFiles) == 0 {
		fmt.Println("  WARNING: No tests kept - validation skipped")
	} else {
		finalMergedFile := "final_coverage.out"

		err = mergeMultipleCoverageFiles(keptTestFiles, finalMergedFile)
		if err != nil {
			return fmt.Errorf("failed to merge final coverage: %w", err)
		}

		finalCoverage, err := getAllFunctionsCoverage(finalMergedFile)
		if err != nil {
			os.Remove(finalMergedFile)

			return fmt.Errorf("failed to analyze final coverage: %w", err)
		}

		os.Remove(finalMergedFile)

		var validationErrors []string

		for fn := range targetFuncs {
			if finalCoverage[fn] < config.CoverageThreshold {
				validationErrors = append(validationErrors,
					fmt.Sprintf("  %s: %.1f%% (target: %.0f%%)", fn, finalCoverage[fn], config.CoverageThreshold))
			}
		}

		if len(validationErrors) > 0 {
			fmt.Printf("  VALIDATION FAILED: %d functions dropped below threshold:\n", len(validationErrors))

			for _, e := range validationErrors {
				fmt.Println(e)
			}

			return fmt.Errorf("validation failed: %d functions dropped below coverage threshold", len(validationErrors))
		}

		fmt.Printf("  VALIDATION PASSED: All %d target functions maintain %.0f%%+ coverage\n",
			len(targetFuncs), config.CoverageThreshold)
	}

	// Report unfilled gaps
	if len(remainingGaps) > 0 {
		fmt.Printf("\n  WARNING: %d functions could not reach %.0f%% even with all tests:\n",
			len(remainingGaps), config.CoverageThreshold)

		for fn := range remainingGaps {
			fmt.Printf("    %s: %.1f%%\n", fn, targetCoverage[fn])
		}
	}

	// Clean up
	for _, f := range testCoverageFiles {
		os.Remove(f)
	}

	// Report results
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("RESULTS")
	fmt.Println("=" + strings.Repeat("=", 79))

	// Count kept by type
	var keptBaseline, keptNonBaseline int

	for _, t := range keptTests {
		if t.isBaseline {
			keptBaseline++
		} else {
			keptNonBaseline++
		}
	}

	fmt.Printf("\nTests that must be kept (%d total: %d baseline, %d non-baseline):\n",
		len(keptTests), keptBaseline, keptNonBaseline)
	fmt.Printf("  %-80s %6s   %s\n", "TEST", "FILLS", "TYPE")
	fmt.Printf("  %-80s %6s   %s\n", strings.Repeat("-", 80), "------", "--------")

	for _, test := range keptTests {
		qName := test.pkg + ":" + test.name
		typeStr := "unit"
		if test.isBaseline {
			typeStr = "baseline"
		}

		fmt.Printf("  %-80s %6d   %s\n", qName, test.gapsFilled, typeStr)
	}

	// Trimming report - redundant baseline tests
	fmt.Printf("\nBaseline tests that could be trimmed (%d):\n", len(redundantBaselineTests))
	fmt.Printf("  %-80s\n", "TEST")
	fmt.Printf("  %-80s\n", strings.Repeat("-", 80))

	sort.Slice(redundantBaselineTests, func(i, j int) bool {
		if redundantBaselineTests[i].pkg != redundantBaselineTests[j].pkg {
			return redundantBaselineTests[i].pkg < redundantBaselineTests[j].pkg
		}

		return redundantBaselineTests[i].name < redundantBaselineTests[j].name
	})

	for _, test := range redundantBaselineTests {
		qName := test.pkg + ":" + test.name
		fmt.Printf("  %-80s\n", qName)
	}

	// Redundant non-baseline tests
	fmt.Printf("\nRedundant non-baseline tests (%d):\n", len(redundantNonBaselineTests))
	fmt.Printf("  %-80s\n", "TEST")
	fmt.Printf("  %-80s\n", strings.Repeat("-", 80))

	sort.Slice(redundantNonBaselineTests, func(i, j int) bool {
		if redundantNonBaselineTests[i].pkg != redundantNonBaselineTests[j].pkg {
			return redundantNonBaselineTests[i].pkg < redundantNonBaselineTests[j].pkg
		}

		return redundantNonBaselineTests[i].name < redundantNonBaselineTests[j].name
	})

	for _, test := range redundantNonBaselineTests {
		qName := test.pkg + ":" + test.name
		fmt.Printf("  %-80s\n", qName)
	}

	fmt.Println()

	return nil
}

// getAllFunctionsCoverage returns a map of function name -> coverage percentage for all functions.
func getAllFunctionsCoverage(coverageFile string) (map[string]float64, error) {
	out, err := exec.Command("go", "tool", "cover", "-func="+coverageFile).Output()
	if err != nil {
		return nil, fmt.Errorf("go tool cover failed: %w", err)
	}

	funcs := make(map[string]float64)
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
		funcs[funcName] = percent
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

// hasParallelCall checks if a block statement contains a call to t.Parallel().
func hasParallelCall(body *ast.BlockStmt) bool {
	found := false

	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check for selector expression (t.Parallel or something.Parallel)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Check if the method is "Parallel"
		if sel.Sel.Name == "Parallel" {
			found = true

			return false
		}

		return true
	})

	return found
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
	listOut, err := output(context.Background(), "go", "list", pkgPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var allTests []testInfo
	packages := strings.Split(strings.TrimSpace(listOut), "\n")

	for _, pkg := range packages {
		if pkg == "" {
			continue
		}

		out, err := output(context.Background(), "go", "test", "-list", ".", pkg)
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

	// Rebuild coverage file with deduplicated blocks
	// Note: We don't split overlapping blocks - go tool cover handles them correctly.
	// We only deduplicate identical blocks (same start/end positions) by summing counts.
	var merged []string
	merged = append(merged, mode)

	// Sort for deterministic output
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].file != blocks[j].file {
			return blocks[i].file < blocks[j].file
		}

		if blocks[i].startLine != blocks[j].startLine {
			return blocks[i].startLine < blocks[j].startLine
		}

		return blocks[i].startCol < blocks[j].startCol
	})

	for _, block := range blocks {
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
func output(ctx context.Context, command string, args ...string) (string, error) {
	buf := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, command, args...)
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
