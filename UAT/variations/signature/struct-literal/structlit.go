// Package structlit demonstrates mocking interfaces and functions with anonymous
// struct literal parameters and return types.
package structlit

import "errors"

// Exported variables.
var (
	ErrInvalidConfig = errors.New("invalid config")
	ErrProcessFailed = errors.New("process failed")
)

// Default configuration values.
const (
	defaultPort       = 8080
	defaultMaxRetries = 3
	defaultTimeout    = 30
)

// ConfigManager demonstrates a struct with methods using struct literal returns.
type ConfigManager struct{}

// Load returns a struct literal containing configuration data.
// Struct literal returns avoid creating named types for simple data structures.
func (c ConfigManager) Load(_ string) struct {
	Host string
	Port int
	TLS  bool
} {
	// Implementation would load from path
	return struct {
		Host string
		Port int
		TLS  bool
	}{
		Host: "localhost",
		Port: defaultPort,
		TLS:  false,
	}
}

// DataProcessor demonstrates an interface with struct literal parameters and returns.
// Struct literals are anonymous struct types defined inline, commonly used for
// configuration options and API responses where a full named type is overkill.
type DataProcessor interface {
	// Process accepts a struct literal parameter with single field.
	// Single-field struct literals are common for simple configuration options.
	Process(cfg struct{ Timeout int }) error

	// Transform accepts a struct literal with multiple fields.
	// Multi-field struct literals are used for more complex option patterns.
	Transform(opts struct {
		Debug bool
		Level int
	}) (string, error)

	// GetConfig returns a struct literal type.
	// Struct literal returns are common for API responses and configuration data.
	GetConfig() struct {
		Host string
		Port int
	}

	// Apply uses struct literals in both parameter and return.
	// This pattern is common in middleware and request/response handlers.
	Apply(req struct{ Method string }) struct{ Status int }
}

// GetDefaults demonstrates a function returning a struct literal.
// This pattern is common for providing default configuration values.
func GetDefaults() struct {
	MaxRetries int
	Timeout    int
} {
	return struct {
		MaxRetries int
		Timeout    int
	}{
		MaxRetries: defaultMaxRetries,
		Timeout:    defaultTimeout,
	}
}

// ValidateRequest demonstrates a standalone function with struct literal parameter.
// This pattern is common for validation functions that need multiple configuration values.
func ValidateRequest(req struct {
	APIKey  string
	Timeout int
},
) error {
	if req.APIKey == "" {
		return ErrInvalidConfig
	}

	if req.Timeout <= 0 {
		return ErrInvalidConfig
	}

	return nil
}
