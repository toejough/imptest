# Data Race Analysis: mockTester Pattern

## Executive Summary

Three tests in the imptest package contain data races caused by unsynchronized access to closure variables shared between the test goroutine and the dispatcher goroutine. The races are detected by Go's race detector when running `go test -race`.

## Affected Tests

### 1. TestDispatchLoop_OrderedFailsOnDispatcherMismatch
**Location:** `imptest/controller_test.go:554-600`

**Race Details:**
- **Write Location:** Line 563-564, dispatcher goroutine (via `mockTester.Fatalf` closure)
- **Read Location:** Line 592, 597, test goroutine
- **Variables:** `fatalfCalled` (bool) and `fatalfMsg` (string)

**Race Detector Output:**
```
WARNING: DATA RACE
Read at 0x00c0001121df by goroutine 6 (test):
  controller_test.go:592 +0x2ec (reading fatalfCalled)

Previous write at 0x00c0001121df by goroutine 7 (dispatcher):
  controller_test.go:563 +0x44 (mockTester.Fatalf writes fatalfCalled)
```

### 2. TestGetCallOrdered_FailsOnMismatch
**Location:** `imptest/controller_test.go:200-241`

**Race Details:**
- **Write Location:** Line 209-210, when `Fatalf` closure executes
- **Read Location:** Line 233, 238, test goroutine
- **Variables:** `fatalfCalled` (bool) and `fatalfMsg` (string)

**Note:** This race may not trigger consistently because the call is already queued when `GetCallOrdered` is called, so `Fatalf` might be called synchronously in the same goroutine. However, the pattern is still racy and could fail with different timing.

### 3. TestImpGetCallOrdered_WrongMethod
**Location:** `imptest/imp_test.go:149-189`

**Race Details:**
- **Write Location:** Line 156-157, when `Fatalf` closure executes
- **Read Location:** Line 183, 186, test goroutine
- **Variables:** `fatalfCalled` (bool) and `fatalfMsg` (string)

**Note:** Similar to #2, timing may make this race intermittent.

## Root Cause Analysis

### The Pattern

All three tests follow this unsafe pattern:

```go
// Test-local variables (stack variables)
fatalfCalled := false
var fatalfMsg string

mockTester := &mockTester{
    fatalf: func(format string, args ...any) {
        // WRITE: Closure captures and writes to test-local variables
        fatalfCalled = true
        fatalfMsg = fmt.Sprintf(format, args...)
    },
}

// ... test creates controller and triggers condition that calls Fatalf ...

time.Sleep(100 * time.Millisecond) // Insufficient synchronization!

// READ: Test goroutine reads the variables
if !fatalfCalled {
    t.Error("expected Fatalf to be called")
}
if !contains(fatalfMsg, "expected") {
    t.Error("...")
}
```

### Why This Is Racy

1. **Concurrent Access:** The `Fatalf` method is called by the dispatcher goroutine (running in `Controller.dispatchLoop()`), while the test goroutine reads the variables.

2. **No Synchronization:** `time.Sleep()` does NOT provide synchronization. It only provides timing, not memory ordering guarantees.

3. **Memory Model Violation:** There is no happens-before relationship between the write in the dispatcher goroutine and the read in the test goroutine.

### Why Sleep Doesn't Help

The Go memory model requires explicit synchronization. From the Go spec:

> "Within a single goroutine, reads and writes must behave as if they executed in the order specified by the program. That is, compilers and processors may reorder the reads and writes executed within a single goroutine only when the reordering does not change the behavior within that goroutine as defined by the language specification. Because of this reordering, the execution order observed by one goroutine may differ from the order perceived by another."

`time.Sleep()` only delays execution; it doesn't create a happens-before edge.

## Correct Synchronization Patterns

### Pattern 1: Channel-Based (Recommended for Simple Cases)

```go
fatalfChan := make(chan string, 1)

mockTester := &mockTester{
    helper: func() {},
    fatalf: func(format string, args ...any) {
        msg := fmt.Sprintf(format, args...)
        fatalfChan <- msg // Channel send creates happens-before relationship
    },
}

// ... trigger Fatalf ...

// Receive from channel (creates happens-before relationship)
select {
case msg := <-fatalfChan:
    if !contains(msg, "expected") {
        t.Errorf("unexpected message: %s", msg)
    }
case <-time.After(500 * time.Millisecond):
    t.Error("timeout waiting for Fatalf")
}
```

**Advantages:**
- Simple and idiomatic Go
- Provides both synchronization and timeout handling
- Clear happens-before relationship

### Pattern 2: Atomic + Mutex (For More Complex Cases)

```go
var fatalfCalled atomic.Bool
var msgMu sync.Mutex
var fatalfMsg string

mockTester := &mockTester{
    helper: func() {},
    fatalf: func(format string, args ...any) {
        fatalfCalled.Store(true) // Atomic operation
        msgMu.Lock()
        fatalfMsg = fmt.Sprintf(format, args...)
        msgMu.Unlock()
    },
}

// ... trigger Fatalf ...
time.Sleep(100 * time.Millisecond) // Still need to wait for async call

// Synchronized reads
if !fatalfCalled.Load() { // Atomic read
    t.Error("expected Fatalf to be called")
}

msgMu.Lock()
msg := fatalfMsg
msgMu.Unlock()

if !contains(msg, "expected") {
    t.Errorf("unexpected message: %s", msg)
}
```

**Advantages:**
- More granular control
- Can check boolean without blocking
- Suitable when you need to check multiple variables

### Pattern 3: WaitGroup + Mutex (For Guaranteed Completion)

```go
var wg sync.WaitGroup
var mu sync.Mutex
var fatalfMsg string

wg.Add(1)
mockTester := &mockTester{
    helper: func() {},
    fatalf: func(format string, args ...any) {
        mu.Lock()
        fatalfMsg = fmt.Sprintf(format, args...)
        mu.Unlock()
        wg.Done()
    },
}

// ... trigger Fatalf ...

// Wait for Fatalf to complete
wg.Wait()

// Synchronized read
mu.Lock()
msg := fatalfMsg
mu.Unlock()

if !contains(msg, "expected") {
    t.Errorf("unexpected message: %s", msg)
}
```

**Advantages:**
- Guarantees the callback has completed
- No spurious timeouts
- Clear synchronization semantics

## Regression Tests

See `race_regression_test.go` for comprehensive regression tests that:

1. **Reproduce the races:** Three tests that deliberately use the racy pattern to demonstrate the issue
2. **Demonstrate correct patterns:** Two tests showing proper synchronization using channels and atomics

### Running the Regression Tests

```bash
# Detect the races (will show DATA RACE warnings but tests pass functionally)
go test -race ./imptest -run TestRaceRegression

# Verify the correct patterns (no DATA RACE warnings)
go test -race ./imptest -run TestProperSynchronization
```

## Recommended Fixes

### For All Three Tests

Replace the unsafe pattern with channel-based synchronization:

**Before (Racy):**
```go
fatalfCalled := false
var fatalfMsg string

mockTester := &mockTester{
    fatalf: func(format string, args ...any) {
        fatalfCalled = true
        fatalfMsg = fmt.Sprintf(format, args...)
    },
}

// ... test code ...

time.Sleep(100 * time.Millisecond)
if !fatalfCalled { ... }
```

**After (Safe):**
```go
fatalfChan := make(chan string, 1)

mockTester := &mockTester{
    fatalf: func(format string, args ...any) {
        fatalfChan <- fmt.Sprintf(format, args...)
    },
}

// ... test code ...

select {
case msg := <-fatalfChan:
    if !contains(msg, "expected") { ... }
case <-time.After(500 * time.Millisecond):
    t.Error("timeout")
}
```

## Impact Assessment

### Severity: Medium

- **Does NOT affect production code:** The races only exist in test code
- **Tests are functionally correct:** The races don't cause test failures (sleeps are long enough)
- **Detection:** Race detector reliably catches these issues
- **Fix complexity:** Low - straightforward pattern replacement

### Why Fix This?

1. **Best Practices:** Tests should model correct concurrent code
2. **Future Maintenance:** Someone might reduce the sleep duration and introduce flaky tests
3. **CI/CD:** If you add `-race` to CI, these will fail
4. **Educational:** Tests serve as examples; racy tests teach bad patterns

## Prevention

### Code Review Checklist

When reviewing tests that use `mockTester` or similar patterns:

- [ ] Are closure variables accessed from multiple goroutines?
- [ ] Is there explicit synchronization (channel, mutex, atomic)?
- [ ] Does the code rely on `time.Sleep()` for correctness?
- [ ] Would the race detector flag this code?

### Alternative: Use Generated Mocks

Consider using the generated `MockTester` instead of hand-rolled mocks for better type safety and patterns that encourage proper synchronization.

## References

- [Go Memory Model](https://go.dev/ref/mem)
- [Go Race Detector](https://go.dev/doc/articles/race_detector)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
