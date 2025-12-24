package timeconflict_test

import (
	"testing"

	timeconflict "github.com/toejough/imptest/UAT/11-package-name-conflicts"
)

//go:generate impgen time.Timer

// TestUseTimer demonstrates testing with a package that shadows stdlib time.
//
// Key Requirements Met:
//  1. Package name conflicts: The generated code must handle the case where the user
//     has a local package named "time" that shadows the stdlib time package.
//  2. The generator should alias the stdlib packages when there's a conflict.
func TestUseTimer(t *testing.T) {
	t.Parallel()

	imp := NewTimerImp(t)

	go func() {
		result := timeconflict.UseTimer(imp.Mock)
		_ = result // Use the result to avoid unused variable warning
	}()

	imp.ExpectCallIs.Wait().ExpectArgsAre(100).InjectResult(nil)
	imp.ExpectCallIs.GetElapsed().InjectResult(42)
}
