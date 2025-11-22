//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fmt.Printf("Running %s go on %s\n", os.Args[0], os.Getenv("GOFILE"))

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("  cwd = %s\n", cwd)
	fmt.Printf("  os.Args = %#v\n", os.Args)

	for _, ev := range []string{"GOARCH", "GOOS", "GOFILE", "GOLINE", "GOPACKAGE", "DOLLAR"} {
		fmt.Println("  ", ev, "=", os.Getenv(ev))
	}

	// Open GOFILE (if set) and print its contents.
	gofile := os.Getenv("GOFILE")
	if gofile == "" {
		fmt.Println("  GOFILE not set; skipping file read")
		return
	}

	fullPath := gofile
	if !filepath.IsAbs(gofile) {
		fullPath = filepath.Join(cwd, gofile)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Printf("  error reading GOFILE %q: %v\n", fullPath, err)
		return
	}

	fmt.Printf("----- Contents of %s -----\n%s\n----- end %s -----\n", fullPath, string(data), fullPath)
}
