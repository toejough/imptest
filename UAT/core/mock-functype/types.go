// Package handlers contains function type definitions for testing
// direct function type mocking with --dependency flag.
package handlers

// Validator is a function type for validating data.
// Simple function types like this are common for callback patterns.
type Validator func(data string) error
