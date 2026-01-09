package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/toejough/go-reorder"
)

// Writer interface for writing generated code.
type Writer interface {
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// WriteGeneratedCode writes the generated code to generated_<impName>.go.
func WriteGeneratedCode(
	code string, impName string, pkgName string, getEnv func(string) string, fileWriter Writer, out io.Writer,
) error {
	const generatedFilePermissions = 0o600

	filename := "generated_" + impName
	// If we're in a test package OR the source file is a test file, append _test to the filename
	// This handles both blackbox testing (package xxx_test) and whitebox testing (package xxx in xxx_test.go)
	goFile := getEnv("GOFILE")

	isTestFile := strings.HasSuffix(pkgName, "_test") || strings.HasSuffix(goFile, "_test.go")
	if isTestFile && !strings.HasSuffix(impName, "_test") {
		filename = "generated_" + strings.TrimSuffix(impName, ".go") + "_test.go"
	} else if !strings.HasSuffix(filename, ".go") {
		filename += ".go"
	}

	// Reorder declarations according to project conventions
	reordered, err := reorder.Source(code)
	if err != nil {
		// If reordering fails, log but continue with original code
		_, _ = fmt.Fprintf(out, "Warning: failed to reorder %s: %v\n", filename, err)

		reordered = code
	}

	err = fileWriter.WriteFile(filename, []byte(reordered), generatedFilePermissions)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	_, _ = fmt.Fprintf(out, "%s written successfully.\n", filename)

	return nil
}
