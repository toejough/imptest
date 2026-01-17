package timeconflict_test

import (
	"testing"

	"github.com/toejough/imptest/UAT/variations/package/shadowing"
)

//go:generate impgen --import-path=github.com/toejough/imptest/UAT/variations/package/shadowing/time time.Timer --dependency

// TestUseTimer demonstrates testing with a package that shadows stdlib time.
//
// Key Requirements Met:
//  1. Package name conflicts: Uses --import-path flag to explicitly specify the local
//     time package when there's ambiguity with stdlib time.
//  2. The generator correctly resolves the local time package using the explicit path.
func TestUseTimer(t *testing.T) {
	t.Parallel()

	mock, imp := MockTimer(t)

	go func() {
		result := timeconflict.UseTimer(mock)
		_ = result // Use the result to avoid unused variable warning
	}()

	imp.Wait.ArgsEqual(100).Return(nil)
	imp.GetElapsed.Called().Return(42)
}
