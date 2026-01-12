package timeconflict_test

import (
	"testing"

	localtime "github.com/toejough/imptest/UAT/variations/package/shadowing/time"
)

// This test file demonstrates the new import inference behavior.
// When the test file imports the local time package, impgen automatically
// infers that we want to mock the local time.Timer, not stdlib.

//go:generate impgen time.Timer --dependency

func TestImportInference(t *testing.T) {
	t.Parallel()

	// This test demonstrates import inference: because we imported
	// the local time package above (as localtime), impgen automatically
	// knows we want to mock local time.Timer, not stdlib time.Timer

	mock := MockTimer(t)

	// Verify we got the local time.Timer mock (which has Wait and GetElapsed methods)
	// Also verify the import is used
	var _ localtime.Timer = mock.Mock //nolint:staticcheck // Intentional compile-time interface check

	go func() {
		_ = mock.Mock.Wait(100)
		_ = mock.Mock.GetElapsed()
	}()

	mock.Method.Wait.ExpectCalledWithExactly(100).InjectReturnValues(nil)
	mock.Method.GetElapsed.ExpectCalledWithExactly().InjectReturnValues(42)
}
