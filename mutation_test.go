//go:build mutation

package imptest

import (
	"testing"

	"github.com/gtramontina/ooze"
)

func TestMutation(t *testing.T) {
	ooze.Release(
		t,
		ooze.WithTestCommand("mage testForFail"),
		ooze.Parallel(),
		ooze.IgnoreSourceFiles("^magefiles.*|.*_string.go|generated_.*|.*_test.go"),
		ooze.WithMinimumThreshold(1.00),
		ooze.WithRepositoryRoot("."),
		ooze.ForceColors(),
	)
}
