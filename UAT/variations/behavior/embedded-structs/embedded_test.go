package embeddedstructs_test

import (
	"testing"

	embeddedstructs "github.com/toejough/imptest/UAT/variations/behavior/embedded-structs"
)

//go:generate impgen TimedLogger --dependency

// TestEmbeddedStructMethods demonstrates that promoted methods from embedded
// structs are correctly included in the generated mock.
//
// Key Requirements Met:
//  1. Automatic Expansion: Methods from embedded structs (Logger and Counter)
//     are automatically discovered and included in the mock.
//  2. Promoted Methods: Can call Log, SetPrefix (from Logger) and Inc, Value
//     (from Counter) directly on the mock.
func TestEmbeddedStructMethods(t *testing.T) {
	t.Parallel()

	mock, imp := MockTimedLogger(t)

	const expectedResult = "[APP] Hello (count: 1)"

	go func() {
		result := embeddedstructs.UseTimedLogger(mock, "Hello")
		if result != expectedResult {
			t.Errorf("expected %q, got %q", expectedResult, result)
		}
	}()

	// SetPrefix is promoted from Logger
	imp.SetPrefix.Expect("APP").Return()

	// LogWithCount is a direct method on TimedLogger
	// It internally calls Inc (from Counter) and Log (from Logger)
	imp.LogWithCount.Expect("Hello").Return(expectedResult)
}

// TestPromotedMethodsFromCounter demonstrates using promoted methods from Counter.
func TestPromotedMethodsFromCounter(t *testing.T) {
	t.Parallel()

	mock, imp := MockTimedLogger(t)

	const (
		initialValue = 0
		afterInc     = 1
	)

	go func() {
		// Directly use Counter's promoted methods through the interface
		_ = mock.Value()
		_ = mock.Inc()
	}()

	// Both methods are promoted from embedded Counter
	imp.Value.Match().Return(initialValue)
	imp.Inc.Match().Return(afterInc)
}

// TestPromotedMethodsFromLogger demonstrates using promoted methods from Logger.
func TestPromotedMethodsFromLogger(t *testing.T) {
	t.Parallel()

	mock, imp := MockTimedLogger(t)

	const expectedLog = "[INFO] Test message"

	go func() {
		// UseLogger only uses Logger methods
		result := embeddedstructs.UseLogger(mock, "Test message")
		if result != expectedLog {
			t.Errorf("expected %q, got %q", expectedLog, result)
		}
	}()

	// Both methods are promoted from embedded Logger
	imp.SetPrefix.Expect("INFO").Return()
	imp.Log.Expect("Test message").Return(expectedLog)
}

// TestRealTimedLogger exercises the actual TimedLogger implementation.
// This ensures the struct methods are not marked as dead code.
func TestRealTimedLogger(t *testing.T) {
	t.Parallel()

	timedLogger := &embeddedstructs.TimedLogger{}

	// Exercise all methods including promoted ones
	timedLogger.SetPrefix("TEST")
	_ = timedLogger.Log("message")
	_ = timedLogger.Inc()
	_ = timedLogger.Value()
	_ = timedLogger.LogWithCount("counted message")
}
