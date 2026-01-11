// Package time demonstrates a local package that shadows the stdlib time package.
package time //nolint:revive // Intentional stdlib shadowing for UAT

// Timer is an interface in a package that shadows the stdlib time package.
// This tests that the generator correctly aliases the stdlib time package.
type Timer interface {
	Wait(seconds int) error
	GetElapsed() int
}
