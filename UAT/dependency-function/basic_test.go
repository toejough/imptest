// Package dependencyfunction_test demonstrates mocking function dependencies with imptest v2.
//
// Test Taxonomy Coverage:
//
//	What:     Target x | Dependency ✓
//	Type:     Function ✓ | Interface x
//	Mode:     Ordered ✓ | Unordered x
//	Matching: Exact ✓ | Matcher ✓
//	Outcome:  Return ✓ | Panic ✓
//	Source:   Type ✓ | Definition x
//
// Mock Sources (function types used for code generation):
//
//	MockFetcher   ← type Fetcher func(int) (string, error)
//	MockValidator ← type Validator func(int) bool
package dependencyfunction_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/imptest"
)

// Fetcher is a function type that fetches data by ID
type Fetcher func(int) (string, error)

// Validator is a function type that validates an integer
type Validator func(int) bool

// ProcessData is a function under test that depends on a fetcher function
func ProcessData(id int, fetcher Fetcher) (string, error) {
	data, err := fetcher(id)
	if err != nil {
		return "", err
	}
	return "processed: " + data, nil
}

// TestDependencyFunction_Ordered_Exact_Args demonstrates the conversational pattern
// with function dependencies
func TestDependencyFunction_Ordered_Exact_Args(t *testing.T) {
	imp := imptest.NewImp(t)

	// Create mock for the dependency function
	fetcher := MockFetcher(imp)

	// Start execution (runs in goroutine)
	result := WrapProcessData(imp, ProcessData).Start(42, fetcher.Func())

	// THEN verify the dependency was called and inject response
	call := fetcher.ExpectCalledWithExactly(42)
	call.InjectReturnValues("test data", nil)

	// Verify the business logic result
	result.ExpectReturnsEqual("processed: test data", nil)
}

// TestDependencyFunction_Ordered_Matcher_Args demonstrates using matchers for args
func TestDependencyFunction_Ordered_Matcher_Args(t *testing.T) {
	imp := imptest.NewImp(t)
	fetcher := MockFetcher(imp)

	result := WrapProcessData(imp, ProcessData).Start(99, fetcher.Func())

	// Expect call with argument matching a condition using gomega matcher
	call := fetcher.ExpectCalledWithMatches(BeNumerically(">", 0))
	call.InjectReturnValues("test data", nil)

	result.ExpectReturnsEqual("processed: test data", nil)
}

// TestDependencyFunction_Ordered_InjectPanic demonstrates injecting a panic
func TestDependencyFunction_Ordered_InjectPanic(t *testing.T) {
	imp := imptest.NewImp(t)
	fetcher := MockFetcher(imp)

	result := WrapProcessData(imp, ProcessData).Start(42, fetcher.Func())

	call := fetcher.ExpectCalledWithExactly(42)
	call.InjectPanicValue("simulated error")

	result.ExpectPanicEquals("simulated error")
}

// TestDependencyFunction_Ordered_GetArgs demonstrates getting actual arguments
func TestDependencyFunction_Ordered_GetArgs(t *testing.T) {
	imp := imptest.NewImp(t)
	fetcher := MockFetcher(imp)

	result := WrapProcessData(imp, ProcessData).Start(42, fetcher.Func())

	call := fetcher.ExpectCalledWithExactly(42)
	call.InjectReturnValues("data", nil)

	result.ExpectReturnsEqual("processed: data", nil)

	// Get the actual arguments that were passed
	args := call.GetArgs()
	if args.A1 != 42 {
		t.Errorf("expected arg 42, got %d", args.A1)
	}
}

// ValidateAndProcess uses a validator function and returns whether it succeeded
func ValidateAndProcess(value int, validator Validator) bool {
	return validator(value)
}

// TestDependencyFunction_BoolReturn demonstrates mocking functions with bool returns
func TestDependencyFunction_BoolReturn(t *testing.T) {
	imp := imptest.NewImp(t)
	validator := MockValidator(imp)

	result := WrapValidateAndProcess(imp, ValidateAndProcess).Start(10, validator.Func())

	call := validator.ExpectCalledWithExactly(10)
	call.InjectReturnValues(true)

	result.ExpectReturnsEqual(true)
}
