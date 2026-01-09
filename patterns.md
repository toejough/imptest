# Patterns and Lessons Learned

## Meta

### Always record lessons in this file
**Added:** 2026-01-09

When identifying a lesson learned, immediately add it to this file. Don't just mention it in conversation - that's not durable. The point of lessons learned is to prevent repeating mistakes, which only works if they're recorded somewhere persistent.

## Git Workflow

### Never amend pushed commits
**Added:** 2026-01-09

When a commit has been pushed to remote, never use `git commit --amend`. This causes local and remote branches to diverge, requiring force push to fix. Instead, create a new commit with the fix.

**Context:** Amended a commit marking Issue #48 complete after it had already been pushed, causing branch divergence during Issue #24 work.

## Issue Tracking

### Fully update issue tracker after completing work
**Added:** 2026-01-09

After completing work on an issue, ensure the issue tracker is fully updated - not just the status field, but also moved to the correct section if the tracker requires it. Verify the issue appears where users would expect to find completed work.

**Context:** Completed Issue #25 but only updated the status field, leaving it visually in Backlog instead of Done.
