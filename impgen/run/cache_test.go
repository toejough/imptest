package run_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

// realCacheFS implements CacheFileSystem using os package for testing.
type realCacheFS struct{}

//nolint:wrapcheck // test helper
func (realCacheFS) Open(path string) (io.ReadCloser, error) { return os.Open(path) }

//nolint:wrapcheck // test helper
func (realCacheFS) Create(path string) (io.WriteCloser, error) { return os.Create(path) }

//nolint:wrapcheck // test helper
func (realCacheFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

//nolint:wrapcheck // test helper
func (realCacheFS) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

//nolint:wrapcheck // test helper
func (realCacheFS) Getwd() (string, error) { return os.Getwd() }

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
	cfs := realCacheFS{}
	run.SaveDiskCache(cachePath, data, cfs)

	// Verify file exists
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		t.Fatal("Expected cache file to be created")
	}

	// Test loading
	loaded := run.LoadDiskCache(cachePath, cfs)
	if len(loaded.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(loaded.Entries))
	}

	entry, ok := loaded.Entries["test"]
	if !ok || entry.Signature != "sig" {
		t.Errorf("Loaded data mismatch")
	}
}
