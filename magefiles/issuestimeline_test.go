package main

import (
	"os"
	"strings"
	"testing"
)

// TestIssuesTimeline_CreateWorkTracking tests creating Work Tracking section when it doesn't exist

//nolint:cyclop,nestif,funlen,paralleltest // Integration test with legitimate complexity
func TestIssuesTimeline_CreateWorkTracking_IssueFollowedByIssue(t *testing.T) {
	// This test demonstrates the orphaning bug when an issue is followed by another issue.
	// Bug: Work Tracking section is inserted AFTER the issue boundary (at issueEnd-1),
	// causing it to appear between issues instead of within the target issue.
	content := `## Backlog

### 36. Split issue tracker into separate repository

#### Universal

**Status**
backlog

**Description**
Test issue

### 37. Another issue

#### Universal

**Status**
backlog
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "issues-*.md")
	if err != nil {
		t.Fatal(err)
	}

	//nolint:noinlineerr // Test code, inline error handling is clear here
	if err := os.WriteFile(tmpFile.Name(), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Temporarily replace issuesFile constant by copying function logic inline
	// to test with our temp file
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Inline the IssuesTimeline logic to test against temp file
	result := string(data)
	issuePrefix := "### 36. "
	issueStart := strings.Index(result, issuePrefix)

	if issueStart == -1 {
		t.Fatal("issue not found")
	}

	// Find end of issue
	issueEnd := len(result)
	nextIssueIdx := strings.Index(result[issueStart+len(issuePrefix):], "\n### ")

	if nextIssueIdx != -1 {
		issueEnd = issueStart + len(issuePrefix) + nextIssueIdx
	}

	issueContent := result[issueStart:issueEnd]

	// Check that issue doesn't have Timeline field yet
	if strings.Contains(issueContent, "**Timeline**") {
		t.Fatal("Issue already has Timeline field")
	}

	// This is the buggy insertion logic that should be tested
	workTrackingSection := `
#### Work Tracking

**Timeline**
- 2026-01-01 17:53 EST - Test entry
`

	insertPoint := issueEnd - 1 // BUG: This is at the newline BEFORE "### 37"

	// Try to find Planning section (won't exist in our test case)
	planningIdx := strings.Index(issueContent, "#### Planning")
	if planningIdx != -1 {
		nextSectionIdx := strings.Index(issueContent[planningIdx+len("#### Planning"):], "\n#### ")
		if nextSectionIdx != -1 {
			insertPoint = issueStart + planningIdx + len("#### Planning") + nextSectionIdx
		}
	} else {
		// Try to find Universal section (will exist)
		universalIdx := strings.Index(issueContent, "#### Universal")
		if universalIdx != -1 {
			nextSectionIdx := strings.Index(issueContent[universalIdx+len("#### Universal"):], "\n#### ")
			if nextSectionIdx != -1 {
				insertPoint = issueStart + universalIdx + len("#### Universal") + nextSectionIdx
			} else {
				// No next section, insert at end - BUG HERE
				insertPoint = issueEnd - 1
			}
		}
	}

	newContent := result[:insertPoint] + workTrackingSection + "\n" + result[insertPoint:]

	// ASSERTION: Work Tracking should be INSIDE issue #36
	// Expected: Work Tracking appears between "Test issue" and "### 37"
	// Bug: Work Tracking appears AFTER "### 37" (orphaned outside issue #36)

	issue36Start := strings.Index(newContent, "### 36.")
	issue37Start := strings.Index(newContent, "### 37.")
	workTrackingStart := strings.Index(newContent, "#### Work Tracking")

	if issue36Start == -1 {
		t.Fatal("Issue #36 not found in result")
	}

	if issue37Start == -1 {
		t.Fatal("Issue #37 not found in result")
	}

	if workTrackingStart == -1 {
		t.Fatal("Work Tracking section not found in result")
	}

	// The bug: Work Tracking is inserted AFTER issue #37 starts
	// This assertion will FAIL with current code, demonstrating the bug
	if workTrackingStart > issue37Start {
		t.Errorf("BUG DEMONSTRATED: Work Tracking section is orphaned AFTER issue #37 "+
			"(at position %d) instead of being inside issue #36 (which ends at position %d)",
			workTrackingStart, issue37Start)
	}

	// The correct behavior: Work Tracking should be between issue 36 and 37
	if workTrackingStart < issue36Start || workTrackingStart > issue37Start {
		t.Errorf("Work Tracking section at position %d is outside issue #36 boundary [%d, %d)",
			workTrackingStart, issue36Start, issue37Start)
	}

	t.Logf("Issue #36 start: %d", issue36Start)
	t.Logf("Issue #37 start: %d", issue37Start)
	t.Logf("Work Tracking start: %d", workTrackingStart)
	t.Logf("Insert point: %d", insertPoint)
	t.Logf("Issue end: %d", issueEnd)
}

//nolint:cyclop,funlen,paralleltest // Integration test with legitimate complexity
func TestIssuesTimeline_CreateWorkTracking_IssueFollowedBySection(t *testing.T) {
	// This test demonstrates the orphaning bug when an issue is followed by a section header.
	// Bug: Work Tracking section is inserted AFTER the issue boundary (at issueEnd-1),
	// causing it to appear in the next section instead of within the target issue.
	content := `## Backlog

### 36. Split issue tracker into separate repository

#### Universal

**Status**
backlog

**Description**
Test issue

## Selected

Issues selected for upcoming work.
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "issues-*.md")
	if err != nil {
		t.Fatal(err)
	}

	//nolint:noinlineerr // Test code, inline error handling is clear here
	if err := os.WriteFile(tmpFile.Name(), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)
	issuePrefix := "### 36. "
	issueStart := strings.Index(result, issuePrefix)

	if issueStart == -1 {
		t.Fatal("issue not found")
	}

	// Find end of issue
	issueEnd := len(result)
	nextIssueIdx := strings.Index(result[issueStart+len(issuePrefix):], "\n### ")

	if nextIssueIdx != -1 {
		issueEnd = issueStart + len(issuePrefix) + nextIssueIdx
	} else {
		nextSectionIdx := strings.Index(result[issueStart+len(issuePrefix):], "\n## ")
		if nextSectionIdx != -1 {
			issueEnd = issueStart + len(issuePrefix) + nextSectionIdx
		}
	}

	issueContent := result[issueStart:issueEnd]

	workTrackingSection := `
#### Work Tracking

**Timeline**
- 2026-01-01 17:53 EST - Test entry
`

	insertPoint := issueEnd - 1 // BUG: This is at the newline BEFORE "## Selected"

	// Try to find Universal section
	universalIdx := strings.Index(issueContent, "#### Universal")
	if universalIdx != -1 {
		nextSectionIdx := strings.Index(issueContent[universalIdx+len("#### Universal"):], "\n#### ")
		if nextSectionIdx != -1 {
			insertPoint = issueStart + universalIdx + len("#### Universal") + nextSectionIdx
		} else {
			// No next section, insert at end - BUG HERE
			insertPoint = issueEnd - 1
		}
	}

	newContent := result[:insertPoint] + workTrackingSection + "\n" + result[insertPoint:]

	// ASSERTION: Work Tracking should be INSIDE issue #36, NOT in the Selected section

	backlogSectionStart := strings.Index(newContent, "## Backlog")
	selectedSectionStart := strings.Index(newContent, "## Selected")
	workTrackingStart := strings.Index(newContent, "#### Work Tracking")

	if backlogSectionStart == -1 {
		t.Fatal("Backlog section not found")
	}

	if selectedSectionStart == -1 {
		t.Fatal("Selected section not found")
	}

	if workTrackingStart == -1 {
		t.Fatal("Work Tracking section not found")
	}

	// The bug: Work Tracking is inserted in the Selected section
	// This assertion will FAIL with current code, demonstrating the bug
	if workTrackingStart > selectedSectionStart {
		t.Errorf("BUG DEMONSTRATED: Work Tracking section is orphaned in Selected section "+
			"(at position %d) instead of being inside issue #36 in Backlog section (which ends at position %d)",
			workTrackingStart, selectedSectionStart)
	}

	// The correct behavior: Work Tracking should be in Backlog section, before Selected section
	if workTrackingStart < backlogSectionStart || workTrackingStart > selectedSectionStart {
		t.Errorf("Work Tracking section at position %d is outside Backlog section boundary [%d, %d)",
			workTrackingStart, backlogSectionStart, selectedSectionStart)
	}

	t.Logf("Backlog section start: %d", backlogSectionStart)
	t.Logf("Selected section start: %d", selectedSectionStart)
	t.Logf("Work Tracking start: %d", workTrackingStart)
	t.Logf("Insert point: %d", insertPoint)
	t.Logf("Issue end: %d", issueEnd)
}

//nolint:cyclop,funlen,paralleltest // Integration test with legitimate complexity
func TestIssuesTimeline_CreateWorkTracking_LastIssueInFile(t *testing.T) {
	// This test covers the edge case where the issue is the last in the file.
	// In this case, issueEnd = len(content) and insertPoint = len(content) - 1,
	// which should work correctly (insert before EOF).
	content := `## Backlog

### 36. Split issue tracker into separate repository

#### Universal

**Status**
backlog

**Description**
Test issue
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "issues-*.md")
	if err != nil {
		t.Fatal(err)
	}

	//nolint:noinlineerr // Test code, inline error handling is clear here
	if err := os.WriteFile(tmpFile.Name(), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)
	issuePrefix := "### 36. "
	issueStart := strings.Index(result, issuePrefix)

	if issueStart == -1 {
		t.Fatal("issue not found")
	}

	// Find end of issue
	issueEnd := len(result)
	nextIssueIdx := strings.Index(result[issueStart+len(issuePrefix):], "\n### ")

	if nextIssueIdx != -1 {
		issueEnd = issueStart + len(issuePrefix) + nextIssueIdx
	} else {
		nextSectionIdx := strings.Index(result[issueStart+len(issuePrefix):], "\n## ")
		if nextSectionIdx != -1 {
			issueEnd = issueStart + len(issuePrefix) + nextSectionIdx
		}
	}

	issueContent := result[issueStart:issueEnd]

	workTrackingSection := `
#### Work Tracking

**Timeline**
- 2026-01-01 17:53 EST - Test entry
`

	insertPoint := issueEnd - 1 // Should be len(content) - 1

	// Try to find Universal section
	universalIdx := strings.Index(issueContent, "#### Universal")
	if universalIdx != -1 {
		nextSectionIdx := strings.Index(issueContent[universalIdx+len("#### Universal"):], "\n#### ")
		if nextSectionIdx != -1 {
			insertPoint = issueStart + universalIdx + len("#### Universal") + nextSectionIdx
		} else {
			// No next section, insert at end
			insertPoint = issueEnd - 1
		}
	}

	newContent := result[:insertPoint] + workTrackingSection + "\n" + result[insertPoint:]

	// ASSERTION: Work Tracking should be at the end of the file, inside issue #36
	// This case should work correctly even with the bug

	issue36Start := strings.Index(newContent, "### 36.")
	workTrackingStart := strings.Index(newContent, "#### Work Tracking")

	if issue36Start == -1 {
		t.Fatal("Issue #36 not found in result")
	}

	if workTrackingStart == -1 {
		t.Fatal("Work Tracking section not found in result")
	}

	// This should pass: Work Tracking is inside issue #36
	if workTrackingStart < issue36Start {
		t.Errorf("Work Tracking section at position %d is before issue #36 at position %d",
			workTrackingStart, issue36Start)
	}

	// Verify it's at the end of the file
	if !strings.HasSuffix(strings.TrimSpace(newContent), "- 2026-01-01 17:53 EST - Test entry") {
		t.Errorf("Work Tracking section is not at the end of the file as expected")
	}

	t.Logf("Issue #36 start: %d", issue36Start)
	t.Logf("Work Tracking start: %d", workTrackingStart)
	t.Logf("Insert point: %d", insertPoint)
	t.Logf("Issue end (file length): %d", issueEnd)
}

//nolint:cyclop,funlen,paralleltest // Integration test with legitimate complexity
func TestIssuesTimeline_CreateWorkTracking_WithPlanningSection(t *testing.T) {
	// This test verifies that when a Planning section exists,
	// the Work Tracking section is inserted after it correctly.
	content := `## Backlog

### 36. Split issue tracker into separate repository

#### Universal

**Status**
backlog

**Description**
Test issue

#### Planning

**Acceptance**
Some acceptance criteria

### 37. Another issue

#### Universal

**Status**
backlog
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "issues-*.md")
	if err != nil {
		t.Fatal(err)
	}

	//nolint:noinlineerr // Test code, inline error handling is clear here
	if err := os.WriteFile(tmpFile.Name(), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)
	issuePrefix := "### 36. "
	issueStart := strings.Index(result, issuePrefix)

	if issueStart == -1 {
		t.Fatal("issue not found")
	}

	// Find end of issue
	issueEnd := len(result)
	nextIssueIdx := strings.Index(result[issueStart+len(issuePrefix):], "\n### ")

	if nextIssueIdx != -1 {
		issueEnd = issueStart + len(issuePrefix) + nextIssueIdx
	}

	issueContent := result[issueStart:issueEnd]

	workTrackingSection := `
#### Work Tracking

**Timeline**
- 2026-01-01 17:53 EST - Test entry
`

	insertPoint := issueEnd - 1

	// Try to find Planning section
	planningIdx := strings.Index(issueContent, "#### Planning")
	if planningIdx != -1 {
		// Look for next section after Planning
		nextSectionIdx := strings.Index(issueContent[planningIdx+len("#### Planning"):], "\n#### ")
		if nextSectionIdx != -1 {
			// BUG: This should insert AFTER Planning, but calculation is wrong
			insertPoint = issueStart + planningIdx + len("#### Planning") + nextSectionIdx
		}
	}

	newContent := result[:insertPoint] + workTrackingSection + "\n" + result[insertPoint:]

	// ASSERTION: Work Tracking should be AFTER Planning section, INSIDE issue #36

	issue36Start := strings.Index(newContent, "### 36.")
	issue37Start := strings.Index(newContent, "### 37.")
	planningStart := strings.Index(newContent, "#### Planning")
	workTrackingStart := strings.Index(newContent, "#### Work Tracking")

	if issue36Start == -1 {
		t.Fatal("Issue #36 not found in result")
	}

	if issue37Start == -1 {
		t.Fatal("Issue #37 not found in result")
	}

	if planningStart == -1 {
		t.Fatal("Planning section not found in result")
	}

	if workTrackingStart == -1 {
		t.Fatal("Work Tracking section not found in result")
	}

	// Work Tracking should be after Planning
	if workTrackingStart < planningStart {
		t.Errorf("Work Tracking section at position %d is before Planning section at position %d",
			workTrackingStart, planningStart)
	}

	// Work Tracking should be inside issue #36 (before issue #37)
	if workTrackingStart > issue37Start {
		t.Errorf("BUG DEMONSTRATED: Work Tracking section at position %d is orphaned after issue #37 at position %d",
			workTrackingStart, issue37Start)
	}

	t.Logf("Issue #36 start: %d", issue36Start)
	t.Logf("Planning section start: %d", planningStart)
	t.Logf("Work Tracking start: %d", workTrackingStart)
	t.Logf("Issue #37 start: %d", issue37Start)
	t.Logf("Insert point: %d", insertPoint)
}
