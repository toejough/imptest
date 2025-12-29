package packagealias_test

import (
	"testing"
	"time"

	"github.com/toejough/imptest/imptest"
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
	t.Parallel()

	imp := imptest.NewImp(t)

	// Mock the TimeService interface which uses time.Time and time.Duration
	timeMock := MockTimeService(imp)

	// Wrap the function under test
	result := WrapGetCurrentHour(imp, GetCurrentHour).Start(timeMock.Interface())

	// Expect Now to be called
	call := timeMock.Now.ExpectCalledWithExactly()

	// Inject a specific time
	testTime := time.Date(2025, 1, 1, 14, 30, 0, 0, time.UTC)
	call.InjectReturnValues(testTime)

	// Verify
	result.ExpectReturnsEqual(14)
}

// Note: This test demonstrates that the generated code correctly:
// 1. Uses package aliases as they appear in the source file (e.g., "time")
// 2. Generates code in the same package
// 3. Uses the same type names and package aliases
// 4. Handles single-word package imports correctly
