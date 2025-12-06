package main

import (
	"fmt"
	"os"

	"github.com/toejough/imptest/generator/run"
)

// RealFileSystem implements FileSystem using os package.
type RealFileSystem struct{}

func (fs *RealFileSystem) Getwd() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return wd, nil
}

func (fs *RealFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", name, err)
	}

	return entries, nil
}

func (fs *RealFileSystem) ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", name, err)
	}

	return data, nil
}

func (fs *RealFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", name, err)
	}

	return nil
}

func main() {
	err := run.Run(os.Args, os.Getenv, &RealFileSystem{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
