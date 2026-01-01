# Data Race Regression Tests - Quick Start

## TL;DR

Three tests in imptest have data races. This directory contains regression tests that reliably reproduce them.

**Quick Verification:**
```bash
# See the races
go test -race ./imptest -run TestRaceRegression_DispatcherFatalfClosure

# See the fix
go test -race ./imptest -run TestProperSynchronization_ChannelBased
```

## The Problem

Tests use unsynchronized closure variables accessed by multiple goroutines:

```go
// ❌ UNSAFE - data race!
fatalfCalled := false
mockTester := &mockTester{
    fatalf: func(format string, args ...any) {
        fatalfCalled = true  // Write in dispatcher goroutine
    },
}
// ... later ...
if !fatalfCalled { ... }  // Read in test goroutine - RACE!
```

## The Solution

Use proper synchronization:

```go
// ✅ SAFE - channel provides synchronization
fatalfChan := make(chan string, 1)
mockTester := &mockTester{
    fatalf: func(format string, args ...any) {
        fatalfChan <- fmt.Sprintf(format, args...)
    },
}
// ... later ...
select {
case msg := <-fatalfChan:
    // Process message safely
case <-time.After(500 * time.Millisecond):
    t.Error("timeout")
}
```

## Files in This Directory

| File | Purpose |
|------|---------|
| `race_regression_test.go` | Regression tests that reproduce the races |
| `RACE_ANALYSIS.md` | Detailed technical analysis of the races |
| `REGRESSION_TEST_SUMMARY.md` | Complete test inventory and results |
| `RACE_DETECTOR_README.md` | This file - quick start guide |

## Affected Tests

1. **TestDispatchLoop_OrderedFailsOnDispatcherMismatch** (controller_test.go:554-600)
2. **TestGetCallOrdered_FailsOnMismatch** (controller_test.go:200-241)
3. **TestImpGetCallOrdered_WrongMethod** (imp_test.go:149-189)

## Regression Test Summary

| Test | Without -race | With -race | Purpose |
|------|--------------|------------|---------|
| TestRaceRegression_DispatcherFatalfClosure | ✅ PASS | ❌ FAIL | Reliably reproduces main race |
| TestRaceRegression_QueuedCallFatalfClosure | ✅ PASS | ⚠️ Sometimes FAIL | Demonstrates queued call pattern |
| TestRaceRegression_ImpWrongMethodClosure | ✅ PASS | ⚠️ Sometimes FAIL | Demonstrates Imp-level race |
| TestRaceRegression_StressTest | ✅ PASS | ❌ FAIL | Aggressive race exposure |
| TestProperSynchronization_ChannelBased | ✅ PASS | ✅ PASS | Shows correct channel-based fix |
| TestProperSynchronization_AtomicBased | ✅ PASS | ✅ PASS | Shows correct atomic-based fix |

## Running the Tests

### See the races in action
```bash
# Main regression test (reliable)
go test -race ./imptest -run TestRaceRegression_DispatcherFatalfClosure -v

# Stress test (very reliable)
go test -race ./imptest -run TestRaceRegression_StressTest -v

# All regression tests
go test -race ./imptest -run TestRaceRegression -v
```

### See the proper fixes
```bash
# Channel-based synchronization
go test -race ./imptest -run TestProperSynchronization_ChannelBased -v

# Atomic-based synchronization
go test -race ./imptest -run TestProperSynchronization_AtomicBased -v

# Both fix examples
go test -race ./imptest -run TestProperSynchronization -v
```

### Verify the original tests also have races
```bash
go test -race ./imptest -run TestDispatchLoop_OrderedFailsOnDispatcherMismatch -v
go test -race ./imptest -run TestGetCallOrdered_FailsOnMismatch -v
go test -race ./imptest -run TestImpGetCallOrdered_WrongMethod -v
```

## Expected Output

### Racy Test (Regression Tests)
```
=== RUN   TestRaceRegression_DispatcherFatalfClosure
==================
WARNING: DATA RACE
Read at 0x... by goroutine 6:
  race_regression_test.go:100 +0x2ec (reading fatalfCalled)

Previous write at 0x... by goroutine 7:
  race_regression_test.go:72 +0x44 (Fatalf writes fatalfCalled)
==================
    testing.go:1617: race detected during execution of test
--- FAIL: TestRaceRegression_DispatcherFatalfClosure (0.15s)
FAIL
```

### Safe Test (Proper Synchronization)
```
=== RUN   TestProperSynchronization_ChannelBased
--- PASS: TestProperSynchronization_ChannelBased (0.05s)
PASS
```

## Understanding the Race

### Why `time.Sleep()` Doesn't Help

```go
// In dispatcher goroutine:
fatalfCalled = true  // WRITE

// In test goroutine:
time.Sleep(100 * time.Millisecond)  // ❌ This is NOT synchronization!
if !fatalfCalled { ... }  // READ - RACE!
```

From the Go Memory Model:
> "Within a single goroutine, reads and writes must behave as if they executed in the order specified by the program... Because of this reordering, the execution order observed by one goroutine may differ from the order perceived by another."

**Sleep only provides timing, not synchronization.** There's no happens-before relationship between the write and read.

### Happens-Before Relationships

The Go memory model requires explicit synchronization primitives:

✅ **Channel send/receive**
```go
ch <- value  // Write happens-before receive
<-ch         // Receive happens-after send
```

✅ **Mutex lock/unlock**
```go
mu.Lock()
x = 1       // Write happens-before unlock
mu.Unlock() // Unlock happens-before next lock
```

✅ **Atomic operations**
```go
x.Store(1)  // Store happens-before load
x.Load()    // Load sees stored value
```

❌ **Sleep (no synchronization)**
```go
time.Sleep(100 * time.Millisecond)  // No happens-before guarantee
```

## Impact

- **Severity:** Medium (test-only, functionally correct due to timing)
- **Production:** Not affected (races only in test code)
- **CI/CD:** Will fail if `-race` flag is added
- **Best Practice:** Tests should model correct concurrent code

## Next Steps

1. **Learn:** Read `RACE_ANALYSIS.md` for detailed technical analysis
2. **Verify:** Run regression tests to see the races
3. **Understand:** Study the proper synchronization examples
4. **Fix:** Apply fixes to the three original tests
5. **Validate:** Run `go test -race ./imptest` to confirm

## Quick Reference

### The Three Synchronization Patterns

#### 1. Channel (Recommended for most cases)
```go
ch := make(chan string, 1)
fatalf: func(format string, args ...any) {
    ch <- fmt.Sprintf(format, args...)
}
```

#### 2. Atomic + Mutex (For complex cases)
```go
var called atomic.Bool
var mu sync.Mutex
var msg string

fatalf: func(format string, args ...any) {
    called.Store(true)
    mu.Lock()
    msg = fmt.Sprintf(format, args...)
    mu.Unlock()
}
```

#### 3. WaitGroup + Mutex (For guaranteed completion)
```go
var wg sync.WaitGroup
var mu sync.Mutex
var msg string

wg.Add(1)
fatalf: func(format string, args ...any) {
    mu.Lock()
    msg = fmt.Sprintf(format, args...)
    mu.Unlock()
    wg.Done()
}
// Later: wg.Wait()
```

## Resources

- [Go Memory Model](https://go.dev/ref/mem)
- [Go Race Detector](https://go.dev/doc/articles/race_detector)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)

## Questions?

See the detailed documentation:
- Technical details: `RACE_ANALYSIS.md`
- Test inventory: `REGRESSION_TEST_SUMMARY.md`
- Source code: `race_regression_test.go`
