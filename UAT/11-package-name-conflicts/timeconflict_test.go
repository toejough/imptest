package timeconflict_test

import (
	"testing"

	timeconflict "github.com/toejough/imptest/UAT/11-package-name-conflicts"
)

//go:generate impgen time.Timer --dependency

// TestUseTimer demonstrates testing with a package that shadows stdlib time.
//
// Key Requirements Met:
//  1. Package name conflicts: The generated code must handle the case where the user
//     has a local package named "time" that shadows the stdlib time package.
//  2. The generator should alias the stdlib packages when there's a conflict.
func TestUseTimer(t *testing.T) {
	t.Parallel()

	mock := MockTimer(t)

	go func() {
		result := timeconflict.UseTimer(mock.Interface())
		_ = result // Use the result to avoid unused variable warning
	}()

	mock.Wait.ExpectCalledWithExactly(100).InjectReturnValues(nil)
	mock.GetElapsed.ExpectCalledWithExactly().InjectReturnValues(42)
}
