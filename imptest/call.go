// Package imptest provides impure function testing functionality.
package imptest

// This file provides Call.

import (
	"time"
)

// Call is the basic type that represents a function Call.
type (
	Call struct{}
)

// CallDeps is the set of impure dependencies for Call.
type CallDeps interface {
	After(d time.Duration) <-chan time.Time
}
