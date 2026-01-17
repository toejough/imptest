// Package time demonstrates a local package that shadows the stdlib time package.
package time //nolint:revive // Intentional stdlib shadowing for UAT

type Timer interface {
	Wait(seconds int) error
	GetElapsed() int
}
