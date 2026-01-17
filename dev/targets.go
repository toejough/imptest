//go:build targ

package dev

import (
	"bytes"
	"context"
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
	"strconv"
	"strings"
	"sync"

	"github.com/akedrou/textdiff"
	"github.com/toejough/go-reorder"
	"github.com/toejough/targ"
	"github.com/toejough/targ/file"
	"github.com/toejough/targ/sh"
	"github.com/toejough/testredundancy"
)

type Coverage struct {
	HTML bool `targ:"flag,desc=Open HTML report in browser"`
}

func (c *Coverage) Description() string {
	return "Display coverage report"
}

func (c *Coverage) Run() error {
	if c.HTML {
		return sh.RunV("go", "tool", "cover", "-html=coverage.out")
	}
	return sh.RunV("go", "tool", "cover", "-func=coverage.out")
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

		// Exclude init() functions - they run at package load time and often
		// contain defensive code that can't be tested.
		if strings.Contains(line, "\tinit\t") {
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

// CheckCoverageForFail checks coverage from existing coverage.out (doesn't run tests).
// Must be run after TestForFail which generates coverage.out.
func CheckCoverageForFail(ctx context.Context) error {
	fmt.Println("Checking coverage...")

	// Merge duplicate coverage blocks from cross-package testing
	if err := mergeCoverageBlocks("coverage.out"); err != nil {
		return fmt.Errorf("failed to merge coverage blocks: %w", err)
	}

	out, err := sh.OutputContext(ctx, "go", "tool", "cover", "-func=coverage.out")
	if err != nil {
		return err
	}

	lines := strings.Split(out, "\n")
	var minCoverage float64 = 100
	var minLine string

	for _, line := range lines {
		// Skip empty lines and files we don't care about
		if line == "" ||
			strings.Contains(line, "_string.go") ||
			strings.Contains(line, "main.go") ||
			strings.Contains(line, "generated_") ||
			strings.Contains(line, "total:") ||
			strings.Contains(line, "\tinit\t") { // init() has untestable defensive code
			continue
		}

		percentString := regexp.MustCompile(`\d+\.\d`).FindString(line)
		if percentString == "" {
			continue
		}

		percent, err := strconv.ParseFloat(percentString, 64)
		if err != nil {
			return err
		}

		if percent < minCoverage {
			minCoverage = percent
			minLine = line
		}
	}

	threshold := 80.0
	if minCoverage < threshold {
		return fmt.Errorf("function coverage was less than the limit of %.1f:\n  %s", threshold, minLine)
	}

	fmt.Printf("Coverage OK (min: %.1f%%)\n", minCoverage)

	return nil
}

// CheckForFail runs all checks on the code for determining whether any fail.
func CheckForFail(ctx context.Context) error {
	fmt.Println("Checking...")

	return targ.Deps(
		ReorderDeclsCheck,
		LintFast,
		LintForFail,
		Deadcode,
		func() error { return targ.Deps(TestForFail, CheckCoverageForFail, targ.WithContext(ctx)) },
		CheckNilsForFail,
		targ.Parallel(),
		targ.WithContext(ctx),
	)
}

// CheckNils checks for nils: applies fixes, then validates.
func CheckNils(ctx context.Context) error {
	return targ.Deps(
		CheckNilsFix,
		CheckNilsForFail,
		targ.WithContext(ctx),
	)
}

// CheckNilsFix applies any auto-fixable nil issues.
func CheckNilsFix(ctx context.Context) error {
	fmt.Println("Fixing nil issues...")
	return sh.RunContext(ctx, "nilaway", "-fix", "./...")
}

// CheckNilsForFail checks for nils and fails on any issues.
func CheckNilsForFail(ctx context.Context) error {
	fmt.Println("Checking for nil issues...")
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

	out, err := sh.OutputContext(ctx, "deadcode", "-test", "./...")
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
	config := testredundancy.Config{
		BaselineTests: []testredundancy.BaselineTestSpec{
			{Package: "./impgen/run", TestPattern: "TestUATConsistency"},
			{Package: "./UAT/core/...", TestPattern: ""},
			{Package: "./UAT/variations/...", TestPattern: ""},
		},
		CoverageThreshold: 80.0,
		PackageToAnalyze:  "./...",
		// Only measure coverage of impgen and imptest packages, not test fixtures
		CoveragePackages: "./impgen/...,./imptest/...",
	}

	return testredundancy.Find(config)
}

// Fmt formats the codebase using golangci-lint formatters.
func Fmt(ctx context.Context) error {
	fmt.Println("Formatting...")
	return sh.RunContext(ctx, "golangci-lint", "run", "-c", "dev/golangci-fmt.toml")
}

// Fuzz runs the fuzz tests.
// Discovers all Fuzz* functions in *_test.go files and runs each for 1000 iterations.
func Fuzz() error {
	fmt.Println("Running fuzz tests...")

	// Find all test files
	testFiles, err := globs(".", []string{".go"})
	if err != nil {
		return fmt.Errorf("failed to find test files: %w", err)
	}

	// Filter to *_test.go files and find fuzz functions
	fuzzPattern := regexp.MustCompile(`^func (Fuzz\w+)\(`)
	type fuzzTest struct {
		dir  string
		name string
	}

	var fuzzTests []fuzzTest

	for _, file := range testFiles {
		if !strings.HasSuffix(file, "_test.go") {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			matches := fuzzPattern.FindStringSubmatch(line)
			if matches != nil {
				fuzzTests = append(fuzzTests, fuzzTest{
					dir:  "./" + filepath.Dir(file),
					name: matches[1],
				})
			}
		}
	}

	if len(fuzzTests) == 0 {
		fmt.Println("No fuzz tests found.")
		return nil
	}

	fmt.Printf("Found %d fuzz tests.\n", len(fuzzTests))

	// Run each fuzz test
	for _, test := range fuzzTests {
		fmt.Printf("  Fuzzing %s in %s...\n", test.name, test.dir)

		err := sh.Run("go", "test", test.dir, "-fuzz=^"+test.name+"$", "-fuzztime=1000x")
		if err != nil {
			return fmt.Errorf("fuzz test %s failed: %w", test.name, err)
		}
	}

	fmt.Println("All fuzz tests passed.")

	return nil
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

// LintFast runs only fast linters for quick fail-fast checks.
func LintFast(ctx context.Context) error {
	fmt.Println("Running fast linters...")

	return sh.RunContext(ctx,
		"golangci-lint", "run",
		"-c", "dev/golangci-fast.toml",
		"--allow-parallel-runners",
	)
}

// LintForFail lints the codebase purely to find out whether anything fails.
func LintForFail(ctx context.Context) error {
	fmt.Println("Linting to check for overall pass/fail...")

	return sh.RunContext(ctx,
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

	if err := targ.Deps(CheckForFail); err != nil {
		return err
	}

	return sh.Run(
		"go",
		"test",
		"-timeout=0",
		"-tags=mutation",
		"-v",
		"./dev/...",
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
func ReorderDeclsCheck(ctx context.Context) error {
	_ = ctx // Reserved for future cancellation support
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

	// Use -count=1 to disable caching so coverage is regenerated
	err := sh.RunContext(ctx,
		"go",
		"test",
		"-timeout=2m",
		"-race",
		"-count=1",
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
// Also generates coverage.out for CheckCoverageForFail.
func TestForFail(ctx context.Context) error {
	fmt.Println("Running unit tests for overall pass/fail...")

	if err := targ.Deps(Generate); err != nil {
		return err
	}

	return sh.RunContext(ctx,
		"go",
		"test",
		"-buildvcs=false",
		"-timeout=30s",
		"-coverprofile=coverage.out",
		"-coverpkg=./impgen/...,./imptest/...",
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
