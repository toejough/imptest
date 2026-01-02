// Package handlers_test demonstrates that impgen can wrap interfaces with --target flag.
//
// INTERFACE AS TARGET vs INTERFACE AS DEPENDENCY:
// - Interface as dependency (--dependency): Creates mocks for testing code that depends on the interface
// - Interface as target (--target): Wraps an existing implementation to intercept/observe calls
//
// This UAT tests whether interfaces can be used with --target flag for wrapping.
// The Capability Matrix shows "?" for "Interface type as Target", so this test will
// determine if this capability is supported or should be marked as unsupported.
package handlers_test

import (
	"context"
	"errors"
	"testing"

	handlers "github.com/toejough/imptest/UAT/32-interface-as-target"
)

// simpleLogger is a basic implementation of Logger for testing.
type simpleLogger struct {
	logs []string
}

func (l *simpleLogger) Log(msg string) error {
	l.logs = append(l.logs, msg)
	return nil
}

func (l *simpleLogger) LogWithContext(_ context.Context, msg string) error {
	l.logs = append(l.logs, msg)
	return nil
}

// errorLogger always returns errors for testing error paths.
type errorLogger struct{}

func (l *errorLogger) Log(_ string) error {
	return errors.New("log error")
}

func (l *errorLogger) LogWithContext(_ context.Context, _ string) error {
	return errors.New("log error")
}

// TestWrapLogger_BasicWrapping demonstrates basic interface wrapping with --target.
// This test verifies that we can wrap a Logger implementation to intercept Log calls.
func TestWrapLogger_BasicWrapping(t *testing.T) {
	t.Parallel()

	// Create a logger implementation
	logger := &simpleLogger{}

	// Wrap the logger interface to intercept calls
	// This is the key test: Can impgen wrap an interface with --target?
	wrapper := WrapLogger(t, logger)

	// Call Log through the wrapped interface
	testMsg := "test message"

	// Get the wrapped interface and call Log on it
	wrappedLogger := wrapper.Interface()
	err := wrappedLogger.Log(testMsg)

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the call was intercepted and recorded
	calls := wrapper.Log.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 Log call, got %d", len(calls))
	}

	if calls[0].Params.Msg != testMsg {
		t.Errorf("expected message %q, got %q", testMsg, calls[0].Params.Msg)
	}

	// Verify the underlying logger received the call
	if len(logger.logs) != 1 {
		t.Fatalf("expected 1 log in underlying logger, got %d", len(logger.logs))
	}

	if logger.logs[0] != testMsg {
		t.Errorf("expected log %q, got %q", testMsg, logger.logs[0])
	}
}

// TestWrapLogger_InterceptCalls demonstrates intercepting and modifying calls.
// This test verifies that wrapped interfaces allow observing calls before they reach the target.
func TestWrapLogger_InterceptCalls(t *testing.T) {
	t.Parallel()

	logger := &simpleLogger{}
	wrapper := WrapLogger(t, logger)

	wrappedLogger := wrapper.Interface()

	// Make multiple calls through the wrapped interface
	_ = wrappedLogger.Log("first")
	_ = wrappedLogger.Log("second")
	_ = wrappedLogger.Log("third")

	// Verify all calls were intercepted
	calls := wrapper.Log.GetCalls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 Log calls, got %d", len(calls))
	}

	expectedMessages := []string{"first", "second", "third"}
	for i, call := range calls {
		if call.Params.Msg != expectedMessages[i] {
			t.Errorf("call %d: expected message %q, got %q", i, expectedMessages[i], call.Params.Msg)
		}
	}

	// Verify the underlying logger received all calls
	if len(logger.logs) != 3 {
		t.Fatalf("expected 3 logs in underlying logger, got %d", len(logger.logs))
	}
}

// TestWrapLogger_WithContext demonstrates wrapping methods that accept context.
// This test verifies that interface wrapping works with methods that have context parameters.
func TestWrapLogger_WithContext(t *testing.T) {
	t.Parallel()

	logger := &simpleLogger{}
	wrapper := WrapLogger(t, logger)

	wrappedLogger := wrapper.Interface()

	ctx := context.Background()
	testMsg := "context message"

	err := wrappedLogger.LogWithContext(ctx, testMsg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the call was intercepted
	calls := wrapper.LogWithContext.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 LogWithContext call, got %d", len(calls))
	}

	if calls[0].Params.Msg != testMsg {
		t.Errorf("expected message %q, got %q", testMsg, calls[0].Params.Msg)
	}

	// Verify underlying logger received the call
	if len(logger.logs) != 1 {
		t.Fatalf("expected 1 log in underlying logger, got %d", len(logger.logs))
	}
}

// TestWrapLogger_ErrorHandling demonstrates wrapping error returns.
// This test verifies that errors from the wrapped implementation are properly propagated.
func TestWrapLogger_ErrorHandling(t *testing.T) {
	t.Parallel()

	logger := &errorLogger{}
	wrapper := WrapLogger(t, logger)

	wrappedLogger := wrapper.Interface()

	err := wrappedLogger.Log("test")

	// Verify error was returned
	if err == nil {
		t.Fatal("expected error from Log call")
	}

	if err.Error() != "log error" {
		t.Errorf("expected error message %q, got %q", "log error", err.Error())
	}

	// Verify the call was still intercepted even though it errored
	calls := wrapper.Log.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 Log call (even though it errored), got %d", len(calls))
	}
}

// TestService_WithWrappedLogger demonstrates using wrapped logger in Service.
// This test shows a real-world use case: wrapping a dependency to observe its usage.
func TestService_WithWrappedLogger(t *testing.T) {
	t.Parallel()

	logger := &simpleLogger{}
	wrapper := WrapLogger(t, logger)

	// Create a service that uses the wrapped logger
	// Note: We pass the wrapped interface, not the original logger
	wrappedLogger := wrapper.Interface()
	service := handlers.NewService(wrappedLogger)

	// Use the service
	err := service.Process("test data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the service made the expected log calls through the wrapper
	calls := wrapper.Log.GetCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 Log calls (start and complete), got %d", len(calls))
	}

	// Verify the log messages
	expectedMessages := []string{
		"Processing: test data",
		"Completed: test data",
	}

	for i, call := range calls {
		if call.Params.Msg != expectedMessages[i] {
			t.Errorf("call %d: expected message %q, got %q", i, expectedMessages[i], call.Params.Msg)
		}
	}

	// Verify the underlying logger also received the calls
	if len(logger.logs) != 2 {
		t.Fatalf("expected 2 logs in underlying logger, got %d", len(logger.logs))
	}
}

// Generate target wrapper for Logger interface
// This is the key directive: attempting to wrap an interface with --target
//
//go:generate impgen handlers.Logger --target
