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
	"testing"
)

// Generate target wrapper for Logger interface
// This is the key directive: attempting to wrap an interface with --target
//
//go:generate impgen handlers.Logger --target

// ConsoleLogger is a simple implementation of Logger for testing.
type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) error {
	// In real implementation, would write to console
	_ = msg
	return nil
}

func (c *ConsoleLogger) LogWithContext(ctx context.Context, msg string) error {
	// In real implementation, would write to console with context
	_, _ = ctx, msg
	return nil
}

// TestWrapLogger_BasicUsage demonstrates basic interface wrapping with call handles.
func TestWrapLogger_BasicUsage(t *testing.T) {
	t.Parallel()

	logger := &ConsoleLogger{}
	wrapper := StartLogger(t, logger)

	// Start() returns call handle, ExpectReturn() verifies and waits
	wrapper.Log.Start("test message").ExpectReturn(nil)
}

// TestWrapLogger_WithContext demonstrates wrapping methods with context.
func TestWrapLogger_WithContext(t *testing.T) {
	t.Parallel()

	logger := &ConsoleLogger{}
	wrapper := StartLogger(t, logger)

	ctx := context.Background()
	wrapper.LogWithContext.Start(ctx, "context message").ExpectReturn(nil)
}
