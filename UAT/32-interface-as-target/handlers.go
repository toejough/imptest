package handlers

import (
	"context"
)

// Logger is a simple interface for logging operations.
// This demonstrates wrapping an interface with --target flag to intercept calls.
// Unlike mocking (--dependency), target wrapping is meant to observe/modify behavior.
type Logger interface {
	// Log writes a log message and returns any error encountered.
	Log(msg string) error

	// LogWithContext writes a log message with context and returns any error.
	LogWithContext(ctx context.Context, msg string) error
}

// Service uses a Logger for its operations.
// This demonstrates a typical use case where we'd want to wrap the logger
// to intercept or observe log calls.
type Service struct {
	// Service fields would go here in a real implementation
}

// NewService creates a new Service with the provided logger.

// Process performs some work and logs the activity.

// ... actual processing would happen here ...

// ProcessWithContext performs work with context and logs the activity.

// ... actual processing would happen here ...
