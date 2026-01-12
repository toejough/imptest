package timeconflict

import "time"

// Scheduler demonstrates using stdlib time.Time and time.Duration.
// This tests that both user code and generated code can import time without conflicts.
type Scheduler interface {
	// ScheduleAt takes a stdlib time.Time as parameter
	ScheduleAt(taskID string, when time.Time) error
	// Delay takes a stdlib time.Duration as parameter
	Delay(taskID string, duration time.Duration) error
	// NextRun returns a stdlib time.Time
	NextRun() (time.Time, error)
	// GetInterval returns a stdlib time.Duration
	GetInterval(taskID string) time.Duration
}
