package imptest_test

import (
	"strings"

	"github.com/toejough/protest/imptest"
)

func expectPanicWith(tester imptest.Tester, message string) {
	tester.Helper()

	recoveredPanic := recover()
	if recoveredPanic != nil {
		// I don't care about the type assertion here - if the type assertion fails,
		// then I'm ok with a panic at test time.
		if !strings.Contains(recoveredPanic.(string), message) { //nolint: forcetypeassert
			tester.Fatalf(
				"The test should've failed with '%s'. Instead the failure was: %s",
				message, recoveredPanic,
			)
		}
	} else {
		tester.Fatalf(
			"The test should've panicked with '%s'. Instead the test continued!",
			message,
		)
	}
}
