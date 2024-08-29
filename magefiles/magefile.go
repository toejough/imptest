//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
	//"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
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

// Run all checks on the code whenever a relevant file changes independently
func Monitor() error {
	fmt.Println("Monitoring...")

	err := Check()
	if err != nil {
		fmt.Printf("continuing to monitor after check failure: %s\n", err)
	} else {
		fmt.Println("continuing to monitor after all checks passed!")
	}

	lastFinishedTime := time.Now()

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to monitor effectively due to error getting current working directory: %w", err)
	}

	for {
		time.Sleep(time.Second)

		paths, err := globs(dir, []string{".go", ".fish", ".toml"})
		if err != nil {
			return fmt.Errorf("unable to monitor effectively due to error resolving globs: %w", err)
		}

		changeDetected, err := target.PathNewer(lastFinishedTime, paths...)
		if err != nil {
			return fmt.Errorf("unable to monitor effectively due to error checking for path updates: %w", err)
		}

		if changeDetected {
			fmt.Println("Change detected...")
			err = Check()
			if err != nil {
				fmt.Printf("continuing to monitor after check failure: %s\n", err)
			} else {
				fmt.Println("continuing to monitor after all checks passed!")
			}

			lastFinishedTime = time.Now()
		}
	}
}

// Run all checks on the code
func Check() error {
	fmt.Println("Checking...")
	for _, cmd := range []func() error{
		Tidy,          // clean up the module dependencies
		Test,          // verify the stuff you explicitly care about works
		Lint,          // make it follow the standards you care about
		CheckNils,     // suss out nils
		CheckCoverage, // verify desired coverage
		Mutate,        // check for untested code
		Fuzz,          // suss out unsafe assumptions about your function inputs
		TodoCheck,     // look for any fixme's or todos
	} {
		err := cmd()
		if err != nil {
			return fmt.Errorf("unable to finish checking: %w", err)
		}
	}
	return nil
}

// Run all checks on the code for determining whether any fail
func CheckForFail() error {
	fmt.Println("Checking...")
	for _, cmd := range []func() error{LintForFail, TestForFail} {
		err := cmd()
		if err != nil {
			return fmt.Errorf("unable to finish checking: %w", err)
		}
	}
	return nil
}

// Tidy tidies up go.mod
func Tidy() error {
	fmt.Println("Tidying go.mod...")
	return sh.RunWithV(map[string]string{"GOPRIVATE": "github.com/toejough/protest"}, "go", "mod", "tidy")
}

// Lint lints the codebase
func Lint() error {
	fmt.Println("Linting...")
	_, err := sh.Exec(nil, os.Stdout, nil, "golangci-lint", "run", "-c", "dev/golangci.toml")
	return err
}

func TodoCheck() error {
	fmt.Println("Linting...")
	_, err := sh.Exec(nil, os.Stdout, nil, "golangci-lint", "run", "-c", "dev/golangci-todos.toml")
	return err
}

// LintForFail lints the codebase purely to find out whether anything fails
func LintForFail() error {
	fmt.Println("Linting to check for overall pass/fail...")
	_, err := sh.Exec(
		nil, os.Stdout, nil,
		"golangci-lint", "run",
		"-c", "dev/golangci.toml",
		"--fix=false",
		"--max-issues-per-linter=1",
		"--max-same-issues=1",
	)
	return err
}

// Run the unit tests
func Test() error {
	fmt.Println("Running unit tests...")
	return sh.RunV(
		"go",
		"test",
		"-timeout=5s",
		"-shuffle=on",
		"-race",
		"-coverprofile=coverage.out",
		"-coverpkg=./imptest",
		"./...",
	)
}

// Run the mutation tests
func Mutate() error {
	// TODO: add a run of the testForFail func, which is what the mutator runs
	fmt.Println("Running mutation tests...")
	return sh.RunV(
		"go",
		"test",
		"-v",
		"-tags=mutation",
		"./...",
		"-run=TestMutation",
	)
}

func CheckCoverage() error {
	fmt.Println("Checking coverage...")
	out, err := sh.Output("go", "tool", "cover", "-func=coverage.out")
	if err != nil {
		return err
	}
	fmt.Println(out)
	lines := strings.Split(out, "\n")
	lastLine := lines[len(lines)-1]
	percentString := regexp.MustCompile(`\d+\.\d`).FindString(lastLine)
	percent, err := strconv.ParseFloat(percentString, 64)
	if err != nil {
		return err
	}
	if percent < 95 {
		return fmt.Errorf("coverage was less than the limit of 95%%")
	}
	return nil
}

// Run the unit tests purely to find out whether any fail
func TestForFail() error {
	fmt.Println("Running unit tests for overall pass/fail...")

	return sh.RunV(
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

// Run the fuzz tests
func Fuzz() error {
	fmt.Println("Running fuzz tests...")
	return sh.RunV("./dev/fuzz.fish")
}

// Check for nils
func CheckNils() error {
	fmt.Println("Running check for nils...")
	return sh.RunV("nilaway", "./...")
}

// Install development tooling
func InstallTools() error {
	fmt.Println("Installing development tools...")
	return sh.RunV("./dev/dev-install.sh")
}

// Clean up the dev env
func Clean() {
	fmt.Println("Cleaning...")
	os.Remove("coverage.out")
}
