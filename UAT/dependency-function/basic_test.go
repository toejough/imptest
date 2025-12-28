// Package dependencyfunction_test demonstrates mocking function dependencies with imptest v2.
//
// Test Taxonomy Coverage:
//
//	What:     Target x | Dependency ✓
//	Type:     Function ✓ | Interface x
//	Mode:     Ordered ✓ | Unordered x
//	Matching: Exact ✓ | Matcher ✓
//	Outcome:  Return ✓ | Panic ✓
//	Source:   Type x | Definition ✓
//
// Mock Sources (function types used for code generation):
//
//	MockFetcher   ← func(int) (string, error)
//	MockValidator ← func(int) bool
package dependencyfunction_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/imptest"
)

// ProcessData is a function under test that depends on a fetcher function
func ProcessData(id int, fetcher func(int) (string, error)) (string, error) {
	data, err := fetcher(id)
	if err != nil {
		return "", err
	}
	return "processed: " + data, nil
}

// TestDependencyFunction_Ordered_Exact_Args demonstrates mocking a function dependency
// with ordered expectations and exact argument matching
func TestDependencyFunction_Ordered_Exact_Args(t *testing.T) {
	// Create mock for the dependency function
	fetcher := MockFetcher(t)

	// Expect the dependency to be called with exact arguments
	call := fetcher.ExpectCalledWithExactly(42)

	// Inject the return values the mock should provide
	call.InjectReturnValues("test data", nil)

	// Execute the function under test with the mock
	result, err := ProcessData(42, fetcher.Func())

	// Verify the business logic result
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "processed: test data" {
		t.Errorf("expected 'processed: test data', got %q", result)
	}
}

// TestDependencyFunction_Ordered_Matcher_Args demonstrates using matchers for args
func TestDependencyFunction_Ordered_Matcher_Args(t *testing.T) {
	fetcher := MockFetcher(t)

	// Expect call with argument matching a condition using gomega matcher
	call := fetcher.ExpectCalledWithMatches(BeNumerically(">", 0))

	call.InjectReturnValues("test data", nil)

	result, err := ProcessData(99, fetcher.Func())

	if err != nil || result != "processed: test data" {
		t.Errorf("unexpected result: %q, %v", result, err)
	}
}

// TestDependencyFunction_Ordered_InjectPanic demonstrates injecting a panic
func TestDependencyFunction_Ordered_InjectPanic(t *testing.T) {
	fetcher := MockFetcher(t)

	call := fetcher.ExpectCalledWithExactly(42)

	// Inject a panic instead of return values
	call.InjectPanicValue("simulated error")

	// Expect the function under test to panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, but none occurred")
		} else if r != "simulated error" {
			t.Errorf("expected panic 'simulated error', got %v", r)
		}
	}()

	ProcessData(42, fetcher.Func())
}

// TestDependencyFunction_Ordered_GetArgs demonstrates getting actual arguments
func TestDependencyFunction_Ordered_GetArgs(t *testing.T) {
	fetcher := MockFetcher(t)

	call := fetcher.ExpectCalledWithExactly(42)
	call.InjectReturnValues("data", nil)

	ProcessData(42, fetcher.Func())

	// Get the actual arguments that were passed
	args := call.GetArgs()
	if args.A1 != 42 {
		t.Errorf("expected arg 42, got %d", args.A1)
	}
}

// ValidateAndProcess uses a validator function and returns whether it succeeded
func ValidateAndProcess(value int, validator func(int) bool) bool {
	return validator(value)
}

// TestDependencyFunction_BoolReturn demonstrates mocking functions with bool returns
func TestDependencyFunction_BoolReturn(t *testing.T) {
	validator := MockValidator(t)

	call := validator.ExpectCalledWithExactly(10)
	call.InjectReturnValues(true)

	result := ValidateAndProcess(10, validator.Func())

	if !result {
		t.Error("expected true, got false")
	}
}
