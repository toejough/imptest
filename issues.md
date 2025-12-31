# Issue Tracker

A simple md issue tracker.

## Statuses

- backlog (to choose from)
- selected (to work on next)
- in progress (currently being worked on)
- review (ready for review/testing)
- done (completed)
- cancelled (not going to be done, for whatever reason, should have a reason)
- blocked (waiting on something else)

## Issues

1. remove the deprecation messages and any hint of V1/V2
   - status: backlog
2. Fix stdlib package shadowing ambiguity (UAT-11)
   - status: backlog
   - description: When a local package shadows a stdlib package name (e.g., local `time` package shadowing stdlib `time`), and the test file doesn't import the shadowed package, impgen cannot determine which package to use. Currently accepts syntax like `impgen time.Timer --dependency` but makes incorrect assumptions about which `time` package is intended.
   - current behavior: Accepts ambiguous syntax, generates incorrect code
   - expected behavior: Either require explicit package qualification (e.g., `--package=stdlib` flag) or detect and report ambiguity error
   - affected: UAT-11 demonstrates this issue
