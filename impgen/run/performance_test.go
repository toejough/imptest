package run_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

// BenchmarkCallableGeneration measures the performance of generating a simple callable.
// This benchmark establishes a baseline for callable template data construction and execution.
func BenchmarkCallableGeneration(b *testing.B) {
	projectRoot := mustFindProjectRoot(b)
	scenarioDir := filepath.Join(projectRoot, "UAT", "01-basic-interface-mocking")

	loader := &testPackageLoader{
		ProjectRoot: projectRoot,
		ScenarioDir: scenarioDir,
	}

	// Warm up the package loader cache
	_, _, _, err := loader.Load(scenarioDir) //nolint:dogsled // Only checking for error during warmup
	if err != nil {
		b.Fatalf("failed to load package: %v", err)
	}

	for b.Loop() {
		var buf bytes.Buffer

		// Simulate the generation by calling Run with the PerformOps callable
		args := []string{"impgen", "PerformOps"}
		getEnv := func(key string) string {
			if key == "GOPACKAGE" {
				return "basic"
			}

			return ""
		}

		fileSystem := &discardFileSystem{}

		// Change to scenario dir
		oldCwd := mustGetCwd(b)
		mustChdir(b, scenarioDir)

		err := run.Run(args, getEnv, fileSystem, loader, &buf)
		if err != nil {
			b.Fatalf("Run failed: %v", err)
		}

		mustChdir(b, oldCwd)
	}
}

// BenchmarkInterfaceGeneration measures the performance of generating a simple interface.
// This benchmark establishes a baseline for template data construction and execution.
func BenchmarkInterfaceGeneration(b *testing.B) {
	projectRoot := mustFindProjectRoot(b)
	scenarioDir := filepath.Join(projectRoot, "UAT", "01-basic-interface-mocking")

	loader := &testPackageLoader{
		ProjectRoot: projectRoot,
		ScenarioDir: scenarioDir,
	}

	// Warm up the package loader cache
	_, _, _, err := loader.Load(scenarioDir) //nolint:dogsled // Only checking for error during warmup
	if err != nil {
		b.Fatalf("failed to load package: %v", err)
	}

	for b.Loop() {
		var buf bytes.Buffer

		// Simulate the generation by calling Run with the Ops interface
		args := []string{"impgen", "Ops"}
		getEnv := func(key string) string {
			if key == "GOPACKAGE" {
				return "basic"
			}

			return ""
		}

		fileSystem := &discardFileSystem{}

		// Change to scenario dir (using SetCwd instead of Chdir for benchmarks)
		oldCwd := mustGetCwd(b)
		mustChdir(b, scenarioDir)

		err := run.Run(args, getEnv, fileSystem, loader, &buf)
		if err != nil {
			b.Fatalf("Run failed: %v", err)
		}

		mustChdir(b, oldCwd)
	}
}

// discardFileSystem implements run.FileSystem, discarding all writes.
type discardFileSystem struct{}

func (d *discardFileSystem) Glob(_ string) ([]string, error) {
	return nil, nil
}

func (d *discardFileSystem) ReadFile(_ string) ([]byte, error) {
	return nil, io.EOF
}

func (d *discardFileSystem) WriteFile(_ string, _ []byte, _ os.FileMode) error {
	// Discard the write
	return nil
}

func mustChdir(b *testing.B, dir string) {
	b.Helper()
	b.Chdir(dir)
}

// Helper functions for benchmarks.
func mustFindProjectRoot(b *testing.B) string {
	b.Helper()

	cfs := realCacheFS{}

	projectRoot, err := run.FindProjectRoot(cfs)
	if err != nil {
		b.Fatalf("failed to find project root: %v", err)
	}

	return projectRoot
}

func mustGetCwd(b *testing.B) string {
	b.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		b.Fatalf("failed to get current directory: %v", err)
	}

	return cwd
}
