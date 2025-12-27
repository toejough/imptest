package visitor

import (
	"fmt"
	"io/fs"
)

// WalkFunc is a function type for walking directory entries.
type WalkFunc func(path string, d fs.DirEntry, err error) error

// TreeWalker walks a directory tree, calling fn for each entry.
type TreeWalker interface {
	Walk(root string, fn func(path string, d fs.DirEntry, err error) error) error
	WalkWithNamedType(root string, fn WalkFunc) error
}

// CountFiles counts regular files using the walker.
func CountFiles(walker TreeWalker, root string) (int, error) {
	count := 0

	err := walker.Walk(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			count++
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walking directory tree: %w", err)
	}

	return count, nil
}
