package run

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// CacheDirName is the name of the local cache directory.
	CacheDirName = ".impgen"
	// DirPerm is the default directory permission.
	DirPerm = 0o755
	// FilePerm is the default file permission.
	FilePerm = 0o600
)

// CacheFileSystem abstracts file operations for the cache system.
type CacheFileSystem interface {
	Open(path string) (io.ReadCloser, error)
	Create(path string) (io.WriteCloser, error)
	MkdirAll(path string, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	Getwd() (string, error)
}

// CacheData represents the structure of the persistent disk cache.
type CacheData struct {
	Entries map[string]CacheEntry `json:"entries"`
}

// CacheEntry represents a single cached mock generation result.
type CacheEntry struct {
	Signature string `json:"signature"`
	Content   string `json:"content"`
	Filename  string `json:"filename"`
}

var errProjectRootNotFound = errors.New("could not find project root (go.mod)")

// CalculatePackageSignature generates a unique hash based on CLI arguments
// and the Go source files in the current directory.
func CalculatePackageSignature(args []string, fileReader FileReader) (string, error) {
	pkgHash := sha256.New()

	// 1. Hash the arguments (skip program name)
	if len(args) > 1 {
		_, _ = pkgHash.Write([]byte(strings.Join(args[1:], "|")))
	}

	// 2. Hash all .go files in the current directory (where go:generate runs)
	files, err := fileReader.Glob("*.go")
	if err != nil {
		return "", fmt.Errorf("failed to glob go files: %w", err)
	}

	sort.Strings(files)

	for _, fileName := range files {
		// Skip generated files to avoid circular dependency in signature
		if strings.HasPrefix(fileName, "generated_") {
			continue
		}

		data, err := fileReader.ReadFile(fileName)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", fileName, err)
		}

		_, _ = pkgHash.Write(data)
	}

	return hex.EncodeToString(pkgHash.Sum(nil)), nil
}

// LoadDiskCache reads the cache from the specified path.
func LoadDiskCache(path string, cfs CacheFileSystem) CacheData {
	var data CacheData

	file, err := cfs.Open(path)
	if err != nil {
		return data
	}
	defer file.Close()

	_ = json.NewDecoder(file).Decode(&data)

	return data
}

// SaveDiskCache writes the cache to the specified path.
func SaveDiskCache(path string, data CacheData, cfs CacheFileSystem) {
	_ = cfs.MkdirAll(filepath.Dir(path), DirPerm)

	file, err := cfs.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data) //nolint:errchkjson
}

// FindProjectRoot locates the nearest directory containing a go.mod file.
func FindProjectRoot(cfs CacheFileSystem) (string, error) {
	curr, err := cfs.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		_, err = cfs.Stat(filepath.Join(curr, "go.mod"))
		if err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", errProjectRootNotFound
		}

		curr = parent
	}
}
