//go:build mutation

package dev

import (
	"testing"

	"github.com/gtramontina/ooze"
)

func TestMutation(t *testing.T) {
	ooze.Release(
		t,
		ooze.WithTestCommand("go test -buildvcs=false ./..."),
		ooze.Parallel(),
		ooze.IgnoreSourceFiles("^dev/.*|.*_string.go|generated_.*|.*_test.go"),
		ooze.WithMinimumThreshold(1.00),
		ooze.WithRepositoryRoot(".."),
		ooze.ForceColors(),
	)
}
