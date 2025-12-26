package run_test

import (
	"strings"
	"testing"
)

// assertContainsAll verifies that content contains all expected strings.
// This helper consolidates the repeated pattern of looping through expected strings.
func assertContainsAll(t *testing.T, content string, expected []string) {
	t.Helper()

	for _, exp := range expected {
		if !strings.Contains(content, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", content)
		}
	}
}

// assertGeneratedContains is a convenience function combining getGeneratedContent and assertContainsAll.
func assertGeneratedContains(t *testing.T, mockFS *MockFileSystem, filename string, expected []string) {
	t.Helper()

	content := getGeneratedContent(t, mockFS, filename)
	assertContainsAll(t, content, expected)
}

// getGeneratedContent retrieves and returns generated file content from MockFileSystem.
// Fails the test if the file wasn't created.
func getGeneratedContent(t *testing.T, mockFS *MockFileSystem, filename string) string {
	t.Helper()

	content, ok := mockFS.files[filename]
	if !ok {
		t.Fatalf("Expected %s to be created", filename)
	}

	return string(content)
}
