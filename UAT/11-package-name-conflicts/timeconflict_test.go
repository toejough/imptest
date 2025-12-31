package timeconflict_test

import (
	"testing"

	timeconflict "github.com/toejough/imptest/UAT/11-package-name-conflicts"
)

//go:generate impgen --import-path=github.com/toejough/imptest/UAT/11-package-name-conflicts/time time.Timer --dependency

// TestUseTimer demonstrates testing with a package that shadows stdlib time.
//
// Key Requirements Met:
//  1. Package name conflicts: Uses --import-path flag to explicitly specify the local
//     time package when there's ambiguity with stdlib time.
//  2. The generator correctly resolves the local time package using the explicit path.
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
