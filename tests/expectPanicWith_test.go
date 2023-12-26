package protest_test

import (
	"strings"
	"testing"
)

func expectPanicWith(t *testing.T, message string) {
	t.Helper()

	recoveredPanic := recover()
	if recoveredPanic != nil {
		// I don't care about the type assertion here - if the type assertion fails,
		// then I'm ok with a panic at test time.
		if !strings.Contains(recoveredPanic.(string), message) { //nolint: forcetypeassert
			t.Fatalf(
				"The test should've failed with '%s'. Instead the failure was: %s",
				message, recoveredPanic,
			)
		}
	} else {
		t.Fatalf(
			"The test should've panicked with '%s'. Instead the test continued!",
			message,
		)
	}
}
