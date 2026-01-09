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
