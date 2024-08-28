// Package imptest provides impure function testing functionality.
package imptest

// This file provides CallRelay

import (
	"time"
)

// CallRelay is intended to be used to relay calls from inside of dependencies of a
// function under test to the test.
type (
	CallRelay     struct{}
	CallRelayDeps interface {
		After(duration time.Duration) <-chan time.Time
		NewCall(function Function, args ...any) *Call
	}
)
