package timeconflict

import (
	"testing"
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

	mock := MockTimer(t)

	go func() {
		result := UseTimer(mock.Mock)
		_ = result // Use the result to avoid unused variable warning
	}()

	mock.Method.Wait.ExpectCalledWithExactly(100).InjectReturnValues(nil)
	mock.Method.GetElapsed.ExpectCalledWithExactly().InjectReturnValues(42)
}
