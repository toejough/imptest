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
