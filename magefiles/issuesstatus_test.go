//go:build mage

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIssuesStatus_LastIssueInSection_NewlineBeforeNextSection is a regression test
// for the bug where moving the last issue in a section causes "section not found" error.
//
// Root cause: When the last issue in a section is removed, the code finds the issue end
// by searching for "\n## " (the newline before the next section header). When removing
// the issue with newContent[:issueStart] + newContent[issueStart+issueEnd:], the newline
// is left behind, so the next section header becomes "\n## Selected" instead of "## Selected".
// Then when searching for the target section with strings.Index(newContent, "## Selected"),
// it fails because of the leading newline.
//
// This test reproduces the exact scenario from the bug report:
// - Issue #33 is at line 411 with status "backlog"
// - Issue #33 is the last issue in the Backlog section
// - The next section header is "## Selected" at line 444
// - Running `mage issuesstatus 33 selected` should move the issue but instead fails
func TestIssuesStatus_LastIssueInSection_NewlineBeforeNextSection(t *testing.T) {
	// Create a temporary directory for this test
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create a minimal issues.md that reproduces the bug
	// Key aspects:
	// 1. Issue #33 is the LAST (and only) issue in Backlog section
	// 2. The next section is "## Selected"
	// 3. There are blank lines between the issue and the next section
	content := `# Issue Tracker

## Backlog

Issues to choose from for future work.

### 33. Add UAT for struct type as target (comprehensive)

#### Universal

**Status**
backlog

**Description**
Verify wrapping struct types with --target flag comprehensively (beyond just methods)

#### Planning

**Rationale**
Taxonomy matrix shows "?" for "Struct type as Target", partially covered in UAT-02 but needs comprehensive coverage

**Acceptance**
UAT demonstrating full struct type wrapping capabilities with --target

**Effort**
Small (1-2 hours)

**Priority**
Low

**Note**
After completing UAT, update Capability Matrix in TAXONOMY.md to mark "Struct type as Target" as "Yes" with comprehensive UAT reference

#### Special Fields

**Taxonomy Gap**
Capability Matrix - "Struct type" row, "As Target" column

---
## Selected

Issues selected for upcoming work.


### 32. Add UAT for interface as target

#### Universal

**Status**
selected

**Description**
Verify mocking interfaces with --target flag
`

	// Write the test issues.md file
	issuesFile := filepath.Join(tempDir, "issues.md")
	err = os.WriteFile(issuesFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test issues.md: %v", err)
	}

	// Act - this is the exact command from the bug report
	err = IssuesStatus(33, "selected")

	// Assert - the bug causes this to fail with "section not found: ## Selected"
	if err != nil {
		t.Fatalf("IssuesStatus(33, 'selected') failed: %v\nThis is the bug - the section exists but can't be found due to leading newline after issue removal", err)
	}

	// Verify the issue was actually moved
	resultContent, err := os.ReadFile(issuesFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	// Find section boundaries
	backlogStart := strings.Index(result, "## Backlog")
	selectedStart := strings.Index(result, "## Selected")

	if backlogStart == -1 {
		t.Fatal("Backlog section not found in result")
	}
	if selectedStart == -1 {
		t.Fatal("Selected section not found in result")
	}

	// Verify issue #33 is in Selected section (after "## Selected" header)
	issue33Pos := strings.Index(result, "### 33.")
	if issue33Pos == -1 {
		t.Fatal("Issue #33 not found in result")
	}

	if issue33Pos < selectedStart {
		t.Errorf("Issue #33 (at pos %d) should be after Selected section header (at pos %d)",
			issue33Pos, selectedStart)
	}

	// Verify issue #33 is NOT in Backlog section
	backlogContent := result[backlogStart:selectedStart]
	if strings.Contains(backlogContent, "### 33.") {
		t.Error("Issue #33 should not be in Backlog section after move")
	}

	// Verify the status was updated to "selected"
	if !strings.Contains(result, "### 33.") {
		t.Fatal("Issue #33 not found in result")
	}
	issue33Start := strings.Index(result, "### 33.")
	issue33Section := result[issue33Start:]
	if nextIssue := strings.Index(issue33Section[1:], "\n### "); nextIssue != -1 {
		issue33Section = issue33Section[:nextIssue+1]
	}

	// Check that the status field now says "selected"
	if !strings.Contains(issue33Section, "**Status**\nselected") {
		t.Error("Issue #33 status should be 'selected' after update")
	}
}

// TestIssuesStatus_LastIssueInSection_NoNewlineBeforeEOF tests moving the last issue
// when it's at the end of the file (no section after it).
//
// This is a variant where issueEnd will be len(newContent) - issueStart instead of
// being determined by "\n## ". This should work correctly since there's no newline issue.
func TestIssuesStatus_LastIssueInSection_NoNewlineBeforeEOF(t *testing.T) {
	tempDir := t.TempDir()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Issue #99 is the only issue and appears at the end of file
	content := `# Issue Tracker

## Backlog

Issues to choose from.

### 99. Last issue in file

#### Universal

**Status**
done

**Description**
This is the last issue and there's no section after it.
`

	issuesFile := filepath.Join(tempDir, "issues.md")
	err = os.WriteFile(issuesFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test issues.md: %v", err)
	}

	// This should work because there's no "\n## " to cause the newline issue
	err = IssuesStatus(99, "done")

	if err != nil {
		t.Fatalf("IssuesStatus(99, 'done') failed: %v", err)
	}
}

// TestIssuesStatus_MiddleIssueInSection tests moving a middle issue (not the last one).
//
// This should work correctly because issueEnd will find "\n### " (next issue) instead
// of "\n## " (next section), so no leading newline problem.
func TestIssuesStatus_MiddleIssueInSection(t *testing.T) {
	tempDir := t.TempDir()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Issue #50 is the first of two issues in Backlog
	content := `# Issue Tracker

## Backlog

### 50. First issue

#### Universal

**Status**
backlog

### 51. Second issue

#### Universal

**Status**
backlog

## Selected

Issues selected for upcoming work.
`

	issuesFile := filepath.Join(tempDir, "issues.md")
	err = os.WriteFile(issuesFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test issues.md: %v", err)
	}

	// This should work because issueEnd finds "\n### 51" not "\n## Selected"
	err = IssuesStatus(50, "selected")

	if err != nil {
		t.Fatalf("IssuesStatus(50, 'selected') failed: %v", err)
	}

	// Verify it moved
	resultContent, err := os.ReadFile(issuesFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}
	result := string(resultContent)

	selectedStart := strings.Index(result, "## Selected")
	if selectedStart == -1 {
		t.Fatal("Selected section not found")
	}

	issue50Pos := strings.Index(result, "### 50.")
	if issue50Pos == -1 {
		t.Fatal("Issue #50 not found")
	}

	if issue50Pos < selectedStart {
		t.Error("Issue #50 should be in Selected section")
	}
}
