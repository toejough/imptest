//go:build mage
// +build mage

package main

import (
	"os"
	"strings"
	"testing"
)

// TestIssuesFix_MoveIssueFromBacklogToSelected verifies that when an issue
// has status "selected" but is in the "## Backlog" section, it gets moved
// to the "## Selected" section.
//
// This is the basic case for issue movement.
func TestIssuesFix_MoveIssueFromBacklogToSelected(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

Issues to choose from for future work.

### 30. Add UAT for function as dependency

#### Universal

**Status**
selected

**Description**
Verify mocking bare package-level functions

## Selected

Issues selected for upcoming work.

*No issues currently selected*

---

## Done

Completed issues.
`

	// Write test file
	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	// Verify issue #30 is no longer in Backlog section
	backlogStart := strings.Index(result, "## Backlog")
	selectedStart := strings.Index(result, "## Selected")
	issue30Start := strings.Index(result, "### 30.")

	if backlogStart == -1 || selectedStart == -1 || issue30Start == -1 {
		t.Fatal("Missing expected sections or issue in result")
	}

	// Issue should be AFTER "## Selected" and BEFORE next section
	if issue30Start < selectedStart {
		t.Error("Issue #30 should be in Selected section, but found before it")
	}

	doneStart := strings.Index(result, "## Done")
	if doneStart != -1 && issue30Start > doneStart {
		t.Error("Issue #30 should be in Selected section, but found after it")
	}

	// Verify Backlog section doesn't contain issue #30
	backlogContent := result[backlogStart:selectedStart]
	if strings.Contains(backlogContent, "### 30.") {
		t.Error("Issue #30 should not be in Backlog section anymore")
	}

	// Verify Selected section contains issue #30
	selectedEnd := doneStart
	if selectedEnd == -1 {
		selectedEnd = len(result)
	}
	selectedContent := result[selectedStart:selectedEnd]
	if !strings.Contains(selectedContent, "### 30.") {
		t.Error("Issue #30 should be in Selected section")
	}

	// Verify the placeholder text "*No issues currently selected*" is removed
	if strings.Contains(selectedContent, "*No issues currently selected*") {
		t.Error("Selected section should not have placeholder when issue is present")
	}
}

// TestIssuesFix_MoveIssueFromMigratedToCancelled verifies that when an issue
// has status "cancelled" but is in the "## Migrated" section, it gets moved
// to the "## Cancelled" section.
//
// This tests the same move logic with different sections.
func TestIssuesFix_MoveIssueFromMigratedToCancelled(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Migrated

Issues moved to other projects.

### 21. Rename --target/--dependency flags to --wrap/--mock (TOE-108)

#### Universal

**Status**
cancelled

**Description**
Current flag names describe user intent rather than what generator does.

---

## Cancelled

Issues that will not be completed.

*No cancelled issues*

---

## Blocked

Issues waiting on dependencies.
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	migratedStart := strings.Index(result, "## Migrated")
	cancelledStart := strings.Index(result, "## Cancelled")
	issue21Start := strings.Index(result, "### 21.")

	if migratedStart == -1 || cancelledStart == -1 || issue21Start == -1 {
		t.Fatal("Missing expected sections or issue in result")
	}

	// Issue should be AFTER "## Cancelled"
	if issue21Start < cancelledStart {
		t.Error("Issue #21 should be in Cancelled section, but found before it")
	}

	blockedStart := strings.Index(result, "## Blocked")
	if blockedStart != -1 && issue21Start > blockedStart {
		t.Error("Issue #21 should be in Cancelled section, but found after it")
	}

	// Verify Migrated section doesn't contain issue #21
	migratedContent := result[migratedStart:cancelledStart]
	if strings.Contains(migratedContent, "### 21.") {
		t.Error("Issue #21 should not be in Migrated section anymore")
	}

	// Verify Cancelled section contains issue #21
	cancelledEnd := blockedStart
	if cancelledEnd == -1 {
		cancelledEnd = len(result)
	}
	cancelledContent := result[cancelledStart:cancelledEnd]
	if !strings.Contains(cancelledContent, "### 21.") {
		t.Error("Issue #21 should be in Cancelled section")
	}
}

// TestIssuesFix_MoveIssueToEmptySection verifies that when moving an issue
// to an empty section (one with only a placeholder like "*No issues*"),
// the issue is inserted correctly and the placeholder is removed.
//
// This tests the special case of inserting into an empty section.
func TestIssuesFix_MoveIssueToEmptySection(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

### 42. Test issue

#### Universal

**Status**
review

## Review

Issues ready for review/testing.

*No issues currently in review*

---

## Done

Completed issues.
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	reviewStart := strings.Index(result, "## Review")
	if reviewStart == -1 {
		t.Fatal("Missing Review section")
	}

	doneStart := strings.Index(result, "## Done")
	reviewEnd := doneStart
	if reviewEnd == -1 {
		reviewEnd = len(result)
	}

	reviewContent := result[reviewStart:reviewEnd]

	// Verify issue is in Review section
	if !strings.Contains(reviewContent, "### 42.") {
		t.Error("Issue #42 should be in Review section")
	}

	// Verify placeholder is removed
	if strings.Contains(reviewContent, "*No issues currently in review*") {
		t.Error("Placeholder should be removed when issue is present")
	}

	// Verify proper formatting: section header, blank line, issue
	lines := strings.Split(reviewContent, "\n")
	if len(lines) < 4 {
		t.Error("Review section should have proper formatting")
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "## Review") {
		t.Error("First line should be section header")
	}
	// There should be blank lines and descriptive text before the issue
	foundIssue := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "### 42.") {
			foundIssue = true
			break
		}
	}
	if !foundIssue {
		t.Error("Issue should be present in section")
	}
}

// TestIssuesFix_MoveLastIssueInSection verifies that when the last (and only)
// issue in a section is moved out, the section is properly cleaned up with
// a placeholder message.
//
// This tests cleanup of source section after extraction.
func TestIssuesFix_MoveLastIssueInSection(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

### 99. Only issue here

#### Universal

**Status**
done

**Description**
This is the only issue in Backlog

## Done

Completed issues.

### 1. Some existing done issue

#### Universal

**Status**
done
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	backlogStart := strings.Index(result, "## Backlog")
	doneStart := strings.Index(result, "## Done")

	if backlogStart == -1 || doneStart == -1 {
		t.Fatal("Missing expected sections")
	}

	backlogContent := result[backlogStart:doneStart]

	// Verify issue #99 is NOT in Backlog
	if strings.Contains(backlogContent, "### 99.") {
		t.Error("Issue #99 should have been moved out of Backlog")
	}

	// Verify Backlog section has placeholder text
	// The exact placeholder text may vary, but should indicate emptiness
	if !strings.Contains(backlogContent, "*No") && !strings.Contains(backlogContent, "empty") {
		t.Error("Backlog section should have placeholder text when empty")
	}

	// Verify issue #99 is in Done section
	if !strings.Contains(result[doneStart:], "### 99.") {
		t.Error("Issue #99 should be in Done section")
	}
}

// TestIssuesFix_NoMidLineInsertion verifies that issues are never inserted
// mid-line (e.g., partway through another issue's content).
//
// This tests the corruption prevention requirement: insertions must be
// on new lines at proper boundaries.
func TestIssuesFix_NoMidLineInsertion(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

### 50. First issue

#### Universal

**Status**
selected

**Description**
This is a multi-line
description that spans
several lines.

### 51. Second issue

#### Universal

**Status**
backlog

## Selected

*No issues currently selected*
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	// Verify issue #51 content is intact (not corrupted by insertion)
	if !strings.Contains(result, "### 51. Second issue") {
		t.Error("Issue #51 should be intact")
	}

	issue51Pattern := `### 51\. Second issue

#### Universal

\*\*Status\*\*
backlog`
	if !strings.Contains(result, strings.ReplaceAll(issue51Pattern, `\`, "")) {
		t.Error("Issue #51 content should be completely intact, not split by insertion")
	}

	// Verify issue #50 is properly extracted (not leaving fragments)
	backlogStart := strings.Index(result, "## Backlog")
	selectedStart := strings.Index(result, "## Selected")
	backlogContent := result[backlogStart:selectedStart]

	if strings.Contains(backlogContent, "### 50.") {
		t.Error("Issue #50 should not be in Backlog section")
	}

	// Verify no orphaned fragments like "description that spans"
	if strings.Contains(backlogContent, "multi-line") && !strings.Contains(backlogContent, "### 50.") {
		t.Error("Found orphaned content from issue #50 in Backlog section")
	}
}

// TestIssuesFix_NoOrphanedContent verifies that when an issue is moved,
// ALL of its content moves with it - no lines are left behind.
//
// This tests the extraction validation requirement.
func TestIssuesFix_NoOrphanedContent(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

### 60. Complex issue

#### Universal

**Status**
done

**Description**
First paragraph.

Second paragraph with more detail.

#### Planning

**Effort**
Medium

**Priority**
High

#### Special Fields

**Note**
Important note here

## Done

Completed issues.
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	backlogStart := strings.Index(result, "## Backlog")
	doneStart := strings.Index(result, "## Done")
	backlogContent := result[backlogStart:doneStart]

	// Verify NO parts of issue #60 remain in Backlog
	orphanedPhrases := []string{
		"Complex issue",
		"First paragraph",
		"Second paragraph",
		"Important note here",
		"#### Planning",
		"**Effort**",
		"**Priority**",
	}

	for _, phrase := range orphanedPhrases {
		if strings.Contains(backlogContent, phrase) {
			t.Errorf("Found orphaned content in Backlog: %q", phrase)
		}
	}

	// Verify ALL parts are in Done section
	doneContent := result[doneStart:]
	if !strings.Contains(doneContent, "### 60. Complex issue") {
		t.Error("Issue #60 should be in Done section")
	}

	for _, phrase := range orphanedPhrases {
		if !strings.Contains(doneContent, phrase) {
			t.Errorf("Missing content in Done section: %q", phrase)
		}
	}
}

// TestIssuesFix_BlankLinesPreserved verifies that the structure and
// formatting of the file is preserved, including blank lines between
// sections and around issues.
//
// This tests the formatting preservation requirement.
func TestIssuesFix_BlankLinesPreserved(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

Issues to choose from for future work.

### 70. Test issue

#### Universal

**Status**
selected

---

## Selected

Issues selected for upcoming work.

*No issues currently selected*

---

## Done

Completed issues.
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	// Verify section separators (---) are preserved
	separatorCount := strings.Count(result, "---")
	if separatorCount < 2 {
		t.Errorf("Expected at least 2 section separators, found %d", separatorCount)
	}

	// Verify descriptive text is preserved
	if !strings.Contains(result, "Issues selected for upcoming work.") {
		t.Error("Section description should be preserved")
	}

	// Verify proper spacing after section headers
	selectedSection := result[strings.Index(result, "## Selected"):]
	lines := strings.Split(selectedSection[:strings.Index(selectedSection, "---")], "\n")

	// Should have: section header, blank line, description, blank line, issue
	if len(lines) < 4 {
		t.Error("Selected section should have proper formatting with blank lines")
	}

	// Verify blank line after section header
	if strings.TrimSpace(lines[1]) != "" && !strings.Contains(lines[1], "Issues selected") {
		t.Error("Should have blank line or description after section header")
	}
}

// TestIssuesFix_MultipleIssuesMove verifies that multiple issues can be
// moved in a single operation, and all are moved correctly.
//
// This tests batch processing without interference.
func TestIssuesFix_MultipleIssuesMove(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

### 80. First mismatch

#### Universal

**Status**
done

### 81. Second mismatch

#### Universal

**Status**
selected

### 82. Correct placement

#### Universal

**Status**
backlog

## Selected

*No issues currently selected*

---

## Done

*No completed issues*
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	selectedStart := strings.Index(result, "## Selected")
	doneStart := strings.Index(result, "## Done")

	// Verify issue #80 in Done section
	if !strings.Contains(result[doneStart:], "### 80.") {
		t.Error("Issue #80 should be in Done section")
	}

	// Verify issue #81 in Selected section
	selectedContent := result[selectedStart:doneStart]
	if !strings.Contains(selectedContent, "### 81.") {
		t.Error("Issue #81 should be in Selected section")
	}

	// Verify issue #82 stays in Backlog
	backlogStart := strings.Index(result, "## Backlog")
	backlogContent := result[backlogStart:selectedStart]
	if !strings.Contains(backlogContent, "### 82.") {
		t.Error("Issue #82 should remain in Backlog section")
	}

	// Verify issues that moved are NOT in Backlog
	if strings.Contains(backlogContent, "### 80.") {
		t.Error("Issue #80 should not be in Backlog")
	}
	if strings.Contains(backlogContent, "### 81.") {
		t.Error("Issue #81 should not be in Backlog")
	}
}

// TestIssuesFix_StatusMatchesSection verifies that issues already in the
// correct section (status matches section) are not moved or modified.
//
// This tests the idempotency and correctness of the move logic.
func TestIssuesFix_StatusMatchesSection(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

### 90. Correctly placed

#### Universal

**Status**
backlog

**Description**
This issue is already in the right place.

## Done

### 91. Also correct

#### Universal

**Status**
done
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	// Content should be unchanged (or only whitespace normalized)
	// Since no moves needed
	if !strings.Contains(result, "### 90. Correctly placed") {
		t.Error("Issue #90 should still exist")
	}
	if !strings.Contains(result, "### 91. Also correct") {
		t.Error("Issue #91 should still exist")
	}

	// Verify sections still contain their issues
	backlogStart := strings.Index(result, "## Backlog")
	doneStart := strings.Index(result, "## Done")

	backlogContent := result[backlogStart:doneStart]
	if !strings.Contains(backlogContent, "### 90.") {
		t.Error("Issue #90 should remain in Backlog")
	}

	doneContent := result[doneStart:]
	if !strings.Contains(doneContent, "### 91.") {
		t.Error("Issue #91 should remain in Done")
	}
}

// TestIssuesFix_SectionBoundaryDetection verifies that section boundaries
// are correctly detected including trailing blank lines and separators.
//
// This tests root cause #1: Boundary detection must include trailing blank
// lines and separators.
func TestIssuesFix_SectionBoundaryDetection(t *testing.T) {
	t.Parallel()

	// Arrange
	content := `# Issue Tracker

## Backlog

Issues to choose from for future work.

### 95. Test with trailing separator

#### Universal

**Status**
done

**Description**
Issue followed by separator

---

## Done

Completed issues.

### 1. Existing done issue

#### Universal

**Status**
done
`

	testFile := t.TempDir() + "/issues.md"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Act
	_, err = moveIssuesToCorrectSections(testFile)
	if err != nil {
		t.Fatalf("moveIssuesToCorrectSections failed: %v", err)
	}

	// Assert
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	// Verify issue #95 moved to Done section
	doneStart := strings.Index(result, "## Done")
	if doneStart == -1 {
		t.Fatal("Done section not found")
	}

	doneContent := result[doneStart:]
	if !strings.Contains(doneContent, "### 95.") {
		t.Error("Issue #95 should be in Done section")
	}

	// Verify Backlog section doesn't contain issue #95
	backlogStart := strings.Index(result, "## Backlog")
	backlogEnd := doneStart
	backlogContent := result[backlogStart:backlogEnd]

	if strings.Contains(backlogContent, "### 95.") {
		t.Error("Issue #95 should not be in Backlog section")
	}

	// Verify the separator (---) is NOT orphaned in Backlog
	// It should either move with the issue or be removed from empty section
	backlogSeparatorCount := strings.Count(backlogContent, "---")

	// If Backlog is empty (only has header + description + placeholder),
	// it shouldn't have a trailing separator
	if strings.Contains(backlogContent, "*No") {
		if backlogSeparatorCount > 0 {
			t.Error("Empty Backlog section should not have trailing separator")
		}
	}

	// Verify Done section has proper structure with separator
	// between sections (before next section header)
	if !strings.Contains(result, "---") {
		t.Error("Section separator should be preserved between sections")
	}
}
