package handlers

import "context"

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
	if err := s.logger.Log("Processing: " + data); err != nil {
		return err
	}
	// ... actual processing would happen here ...
	return s.logger.Log("Completed: " + data)
}

// ProcessWithContext performs work with context and logs the activity.
func (s *Service) ProcessWithContext(ctx context.Context, data string) error {
	if err := s.logger.LogWithContext(ctx, "Processing: "+data); err != nil {
		return err
	}
	// ... actual processing would happen here ...
	return s.logger.LogWithContext(ctx, "Completed: "+data)
}
