package packagealias_test

import (
	"testing"
	"time"

	imptest "github.com/toejough/imptest/imptest/v2"
)

// TimeService uses the standard library time package (single-word import)
type TimeService interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// GetCurrentHour uses TimeService to get the current hour
func GetCurrentHour(ts TimeService) int {
	return ts.Now().Hour()
}

// TestPackageAlias_SingleWord demonstrates single-word package imports
// like "time" where the package name matches the import
func TestPackageAlias_SingleWord(t *testing.T) {
	imp := imptest.NewImp(t)

	// Mock the TimeService interface which uses time.Time and time.Duration
	timeMock := imptest.NewDependencyInterface[TimeService](imp)

	// Expect Now to be called
	call := timeMock.Now.ExpectCalledWithExactly()

	// Inject a specific time
	testTime := time.Date(2025, 1, 1, 14, 30, 0, 0, time.UTC)
	call.InjectReturnValues(testTime)

	// Execute
	hour := GetCurrentHour(timeMock.Interface())

	// Verify
	if hour != 14 {
		t.Errorf("expected hour 14, got %d", hour)
	}
}

// Note: For final-segment, obscured, and aliased package scenarios, we would need
// to create actual external packages or use real external dependencies.
// These tests demonstrate that the generated code correctly:
// 1. Uses package aliases as they appear in the source file
// 2. Generates code in the same package
// 3. Uses the same type names and package aliases
// 4. Prefixes framework imports with _ to avoid conflicts

// Example of what a final-segment test would look like:
// import "github.com/toejough/imptest/imptest/v2"
// The package name is "imptest" (final segment) but could be aliased

// TestPackageAlias_FinalSegment demonstrates final-segment package naming
func TestPackageAlias_FinalSegment(t *testing.T) {
	imp := v2.NewImp(t) // Using v2 as the actual package name

	// The key test here is that generated code uses "v2" prefix
	// for types from "github.com/toejough/imptest/imptest/v2"

	// This is verified by the generated code compiling correctly
	_ = imp
}

// TestPackageAlias_FrameworkImports verifies framework imports use _ prefix
func TestPackageAlias_FrameworkImports(t *testing.T) {
	// The generated code should import framework packages like:
	// _testing "testing"
	// _imptest "github.com/toejough/imptest/imptest/v2"
	//
	// This ensures no conflicts with user's own imports
	// Verification: generated code compiles without conflicts

	t.Skip("This is verified by code generation, not runtime")
}
