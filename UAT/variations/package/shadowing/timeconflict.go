// Package timeconflict demonstrates using an interface from a package that shadows a stdlib package.
package timeconflict

import "github.com/toejough/imptest/UAT/variations/package/shadowing/time"

// UseTimer demonstrates using an interface with a shadowed time package.
// This function will be tested with an imptest-generated mock.
func UseTimer(t time.Timer) int {
	_ = t.Wait(100)

	return t.GetElapsed()
}
