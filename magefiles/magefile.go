//go:build mage
// +build mage

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/target"
)

// Types

type lineAndCoverage struct {
	line     string
	coverage float64
}

// Public Functions (Mage Targets)

// Run all checks on the code.
func Check(c context.Context) error {
	fmt.Println("Checking...")

	for _, cmd := range []func(context.Context) error{
		Tidy,          // clean up the module dependencies
		Generate,      // generate code
		Test,          // verify the stuff you explicitly care about works
		Deadcode,      // verify there's no dead code
		Lint,          // make it follow the standards you care about
		CheckNils,     // suss out nils
		CheckCoverage, // verify desired coverage
		Mutate,        // check for untested code
		Fuzz,          // suss out unsafe assumptions about your function inputs
		TodoCheck,     // look for any fixme's or todos
	} {
		err := cmd(c)
		if err != nil {
			return fmt.Errorf("unable to finish checking: %w", err)
		}
	}

	return nil
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

// CheckCoverage checks that function coverage meets the minimum threshold.
func CheckCoverage(c context.Context) error {
	fmt.Println("Checking coverage...")

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

		if strings.Contains(line, "Imp.go") || strings.Contains(line, "Imp_test.go") {
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

	for _, cmd := range []func(context.Context) error{LintForFail, TestForFail} {
		err := cmd(c)
		if err != nil {
			return fmt.Errorf("unable to finish checking: %w", err)
		}
	}

	return nil
}

// Check for nils.
func CheckNils(c context.Context) error {
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

	fmt.Println(out)

	lines := strings.Split(out, "\n")
	if len(lines) > 0 && len(lines[0]) > 0 {
		return errors.New("found dead code")
	}

	return nil
}

// Run the fuzz tests.
func Fuzz(c context.Context) error {
	fmt.Println("Running fuzz tests...")
	return run(c, "./dev/fuzz.fish")
}

// Generate runs go generate on all packages.
func Generate(c context.Context) error {
	fmt.Println("Generating...")

	return run(
		c,
		"go",
		"generate",
		"./...",
	)
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
				"-tags=mutation",
				"./...",
				"-run=TestMutation",
				// "-ooze.v",
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

// Run the unit tests.
func Test(c context.Context) error {
	fmt.Println("Running unit tests...")
	err := run(
		c,
		"go",
		"test",
		"-timeout=60s",
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

	// Strip main.go coverage lines from coverage.out
	data, err := os.ReadFile("coverage.out")
	if err != nil {
		return fmt.Errorf("failed to read coverage.out: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, "/main.go:") {
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

	return run(
		c,
		"go",
		"test",
		"-timeout=60s",
		"./...",
		// "-rapid.nofailfile",
		"-failfast",
		"-shuffle=on",
		"-race",
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

	// when did this last finish?
	var lastFinishedTime time.Time // never

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

		lastFinishedTime = time.Now()
	}

	// run the check
	go checkFunc(ctx)

	for {
		// don't run more than 1x/sec
		time.Sleep(time.Second)

		// check for change
		paths, err := globs(dir, []string{".go", ".fish", ".toml"})
		if err != nil {
			return fmt.Errorf("unable to monitor effectively due to error resolving globs: %w", err)
		}

		changeDetected, err := target.PathNewer(lastFinishedTime, paths...)
		if err != nil {
			return fmt.Errorf("unable to monitor effectively due to error checking for path updates: %w", err)
		}

		// cancel & re-run if we got a change
		if changeDetected {
			fmt.Println("Change detected...")

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

// Private Functions

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

func output(c context.Context, command string, arg ...string) (string, error) {
	buf := &bytes.Buffer{}
	cmd := exec.CommandContext(c, command, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	return strings.TrimSuffix(buf.String(), "\n"), err
}

func run(c context.Context, command string, arg ...string) error {
	cmd := exec.CommandContext(c, command, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
