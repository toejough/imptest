package timeconflict_test

import (
	"testing"
	"time"
)

//go:generate impgen Scheduler --dependency

func TestStdlibTimeTypes(t *testing.T) {
	t.Parallel()

	mock, imp := MockScheduler(t)

	// Test time.Time as parameter
	scheduledTime := time.Date(2025, 12, 27, 10, 30, 0, 0, time.UTC)

	go func() {
		_ = mock.ScheduleAt("task1", scheduledTime)
	}()

	imp.ScheduleAt.ExpectCalledWithExactly("task1", scheduledTime).InjectReturnValues(nil)

	// Test time.Duration as parameter
	delay := 5 * time.Minute

	go func() {
		_ = mock.Delay("task2", delay)
	}()

	imp.Delay.ExpectCalledWithExactly("task2", delay).InjectReturnValues(nil)

	// Test time.Time as return value
	expectedTime := time.Date(2025, 12, 28, 10, 30, 0, 0, time.UTC)

	go func() {
		_, _ = mock.NextRun()
	}()

	imp.NextRun.ExpectCalledWithExactly().InjectReturnValues(expectedTime, nil)

	// Test time.Duration as return value
	go func() {
		_ = mock.GetInterval("task3")
	}()

	imp.GetInterval.ExpectCalledWithExactly("task3").InjectReturnValues(10 * time.Second)
}
