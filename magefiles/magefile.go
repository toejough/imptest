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
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/target"
)

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

// Tidy tidies up go.mod.
func Tidy(c context.Context) error {
	fmt.Println("Tidying go.mod...")
	return run(c, "go", "mod", "tidy")
	// return sh.RunWithV(map[string]string{"GOPRIVATE": "github.com/toejough/protest"}, "go", "mod", "tidy")
}

func run(c context.Context, command string, arg ...string) error {
	cmd := exec.CommandContext(c, command, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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

// Lint lints the codebase.
func Lint(c context.Context) error {
	fmt.Println("Linting...")
	// _, err := sh.Exec(nil, os.Stdout, nil, "golangci-lint", "run", "-c", "dev/golangci.toml")
	// return err
	return run(c, "golangci-lint", "run", "-c", "dev/golangci.toml")
}

func TodoCheck(c context.Context) error {
	fmt.Println("Linting...")
	// _, err := sh.Exec(nil, os.Stdout, nil, "golangci-lint", "run", "-c", "dev/golangci-todos.toml")
	// return err
	return run(c, "golangci-lint", "run", "-c", "dev/golangci-todos.toml")
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

func Generate(c context.Context) error {
	fmt.Println("Generating...")

	return run(
		c,
		"go",
		"generate",
		"./...",
	)
}

// Run the unit tests.
func Test(c context.Context) error {
	fmt.Println("Running unit tests...")
	// return sh.RunV(
	// 	"go",
	// 	"test",
	// 	"-timeout=5s",
	// 	// "-shuffle=1725149006359140000",
	// 	"-race",
	// 	"-coverprofile=coverage.out",
	// 	"-coverpkg=./imptest",
	// 	"./...",
	// 	// -test.shuffle 1725149006359140000
	// )
	return run(
		c,
		"go",
		"test",
		"-timeout=5s",
		// "-shuffle=1725149006359140000",
		// "-race",
		"-p=1",
		"-coverprofile=coverage.out",
		"-coverpkg=./UAT/run,.,./impgen/run",
		"-covermode=atomic",
		"./...",
		// -test.shuffle 1725149006359140000
	)
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

func CheckCoverage(c context.Context) error {
	fmt.Println("Checking coverage...")

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

type lineAndCoverage struct {
	line     string
	coverage float64
}

// Run the unit tests purely to find out whether any fail.
func TestForFail(c context.Context) error {
	fmt.Println("Running unit tests for overall pass/fail...")

	return run(
		c,
		"go",
		"test",
		"-timeout=1s",
		"./...",
		// "-rapid.nofailfile",
		"-failfast",
		"-shuffle=on",
		"-race",
	)
}

// Run the fuzz tests.
func Fuzz(c context.Context) error {
	fmt.Println("Running fuzz tests...")
	return run(c, "./dev/fuzz.fish")
}

// Check for nils.
func CheckNils(c context.Context) error {
	fmt.Println("Running check for nils...")
	return run(c, "nilaway", "./...")
}

// Install development tooling.
func InstallTools(c context.Context) error {
	fmt.Println("Installing development tools...")
	return run(c, "./dev/dev-install.sh")
}

// Clean up the dev env.
func Clean() {
	fmt.Println("Cleaning...")
	os.Remove("coverage.out")
}
