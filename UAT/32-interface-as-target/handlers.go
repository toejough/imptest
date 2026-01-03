package handlers

import (
	"context"
	"fmt"
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
	logger Logger
}

// NewService creates a new Service with the provided logger.
func NewService(logger Logger) *Service {
	return &Service{logger: logger}
}

// Process performs some work and logs the activity.
func (s *Service) Process(data string) error {
	err := s.logger.Log("Processing: " + data)
	if err != nil {
		return fmt.Errorf("failed to log processing: %w", err)
	}
	// ... actual processing would happen here ...
	err = s.logger.Log("Completed: " + data)
	if err != nil {
		return fmt.Errorf("failed to log completion: %w", err)
	}

	return nil
}

// ProcessWithContext performs work with context and logs the activity.

// ... actual processing would happen here ...
