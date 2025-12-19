package run_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

func TestDiskCache(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.json")

	data := run.CacheData{
		Entries: map[string]run.CacheEntry{
			"test": {
				Signature: "sig",
				Content:   "content",
				Filename:  "file.go",
			},
		},
	}

	// Test saving
	run.SaveDiskCache(cachePath, data)

	// Verify file exists
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		t.Fatal("Expected cache file to be created")
	}

	// Test loading
	loaded := run.LoadDiskCache(cachePath)
	if len(loaded.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(loaded.Entries))
	}

	entry, ok := loaded.Entries["test"]
	if !ok || entry.Signature != "sig" {
		t.Errorf("Loaded data mismatch")
	}
}

//nolint:paralleltest
func TestCalculatePackageSignature(t *testing.T) {
	// Create a temp directory with some go files
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	_ = os.WriteFile("a.go", []byte("package a"), 0o600)
	_ = os.WriteFile("b.go", []byte("package b"), 0o600)
	// Should be ignored
	_ = os.WriteFile("aImp.go", []byte("package a"), 0o600)

	args := []string{"cmd", "arg1", "arg2"}

	sig1, err := run.CalculatePackageSignature(args)
	if err != nil {
		t.Fatalf("Failed to calculate signature: %v", err)
	}

	// Same args, same files -> same signature
	sig2, _ := run.CalculatePackageSignature(args)
	if sig1 != sig2 {
		t.Error("Signatures should be identical for same input")
	}

	// Different args -> different signature
	sig3, _ := run.CalculatePackageSignature([]string{"cmd", "other"})
	if sig1 == sig3 {
		t.Error("Signatures should differ for different args")
	}

	// Different files -> different signature
	_ = os.WriteFile("a.go", []byte("package a modified"), 0o600)

	sig4, _ := run.CalculatePackageSignature(args)
	if sig1 == sig4 {
		t.Error("Signatures should differ for modified files")
	}

	t.Run("unreadable file", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Chdir(tempDir)

		_ = os.WriteFile("bad.go", []byte("unreadable"), 0o000)

		_, err := run.CalculatePackageSignature([]string{"cmd"})
		if err == nil {
			t.Error("Expected error for unreadable file, got nil")
		}
	})

	t.Run("invalid directory", func(t *testing.T) {
		// CalculatePackageSignature uses filepath.Glob("*.go")
		// It's hard to make Glob fail without permissions issues,
		// but we can at least test with no args.
		_, err := run.CalculatePackageSignature(nil)
		if err != nil {
			t.Errorf("Should handle nil args, got %v", err)
		}
	})
}

//nolint:paralleltest
func TestFindProjectRoot(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "a", "b", "c")
	_ = os.MkdirAll(subDir, 0o755)

	// Create go.mod in tempDir
	_ = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test"), 0o600)

	t.Chdir(subDir)

	root, err := run.FindProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	absTempDir, _ := filepath.Abs(tempDir)
	evalRoot, _ := filepath.EvalSymlinks(root)

	evalTemp, _ := filepath.EvalSymlinks(absTempDir)
	if evalRoot != evalTemp {
		t.Errorf("Expected %s, got %s", evalTemp, evalRoot)
	}
}
