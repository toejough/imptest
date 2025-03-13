//go:build mutation

package imptest_test

import (
	"testing"

	"github.com/gtramontina/ooze"
)

func TestMutation(t *testing.T) {
	ooze.Release(
		t,
		ooze.WithTestCommand("mage testForFail"),
		ooze.Parallel(),
		ooze.IgnoreSourceFiles("^magefiles.*|.*_string.go"),
		// ooze.WithMinimumThreshold(0.95),
		ooze.WithRepositoryRoot(".."),
		ooze.ForceColors(),
	)
}
