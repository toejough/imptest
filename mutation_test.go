//go:build mutation

package main_test

import (
	"testing"

	"github.com/gtramontina/ooze"
)

func TestMutation(t *testing.T) {
	ooze.Release(
		t,
		ooze.WithTestCommand("mage testForFail"),
		ooze.Parallel(),
		ooze.IgnoreSourceFiles("^magefiles.*"),
		ooze.WithMinimumThreshold(0.75),
	)
}
