// Package handlers contains function type definitions for testing
// direct function type mocking with --dependency flag.
package handlers

type Validator func(data string) error
