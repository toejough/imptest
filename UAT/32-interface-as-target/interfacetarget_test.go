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

// Generate target wrapper for Logger interface
// This is the key directive: attempting to wrap an interface with --target
//
//go:generate impgen handlers.Logger --target
